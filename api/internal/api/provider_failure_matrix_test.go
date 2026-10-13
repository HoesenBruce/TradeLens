package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/alerts"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/flexsync"
	"github.com/tradermemos/api/internal/jobs"
	"github.com/tradermemos/api/internal/ocr"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func noProviderSecrets(t *testing.T, output string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		require.False(t, strings.Contains(output, secret), "provider diagnostic exposed a sentinel")
	}
}

func TestCrossUserIsolationProviderConfigurationDiagnostics(t *testing.T) {
	f := newIsolationFixture(t)
	owner := registerAndLogin(t, f.s, "owner@diagnostic.invalid")
	member := registerAndLogin(t, f.s, "member@diagnostic.invalid")
	pat := isolationObject(t, f.s, "POST", "/access-tokens", `{"name":"member"}`, member, 201)["token"].(string)
	q := store.NewForDriver(f.conn, "sqlite")
	const secret = "diagnostic-owner-key-244"
	var calls atomic.Int64
	var logs bytes.Buffer
	client := &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("owner key did not reach models provider")
		}
		return &http.Response{StatusCode: 401, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Authorization Bearer ` + secret + `"}}`))}, nil
	})}
	cfg := ocr.VisionConfig{Enabled: true, BaseURL: "http://private-provider.invalid", APIKey: secret, HTTPClient: client}
	s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Logger: slog.New(slog.NewTextHandler(&logs, nil)), CoachDefaults: cfg, OCR: ocr.NewService(cfg, nil)})
	for _, name := range []string{"ocr", "coach"} {
		for _, action := range []string{"test", "models"} {
			path := "/api/v1/settings/" + name + "/" + action
			for _, token := range []string{member, pat} {
				count := calls.Load()
				before := f.databaseState(t)
				require.Equal(t, 403, do(s, "POST", path, `{}`, token).Code)
				require.Equal(t, count, calls.Load())
				require.Equal(t, before, f.databaseState(t))
			}
			count := calls.Load()
			response := do(s, "POST", path, `{}`, owner)
			require.Equal(t, 200, response.Code)
			require.Greater(t, calls.Load(), count)
			noProviderSecrets(t, response.Body.String()+logs.String(), secret, "Authorization", "private-provider.invalid")
		}
	}
}

func TestCrossUserIsolationAsyncProviderErrors(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "owner", "ISOA", 2)
	b := f.seed(t, "member", "ISOB", 9876)
	q := store.NewForDriver(f.conn, "sqlite")
	const secret = "async-provider-secret-244"
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	var calls atomic.Int64
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if strings.Contains(r.URL.Path, "flex") {
			if r.URL.Query().Get("t") != secret {
				t.Error("IBKR credential did not reach provider")
			}
			fmt.Fprintf(w, `<FlexStatementResponse><Status>Fail</Status><ErrorCode>1012</ErrorCode><ErrorMessage>Authorization Bearer %s</ErrorMessage></FlexStatementResponse>`, secret)
		} else {
			http.Redirect(w, r, "/private?token="+secret, 302)
		}
	}))
	t.Cleanup(remote.Close)
	isolationObject(t, f.s, "PUT", "/accounts/"+b.accounts[0]+"/flex-sync", isolationJSON(t, map[string]any{"enabled": true, "query_id": "synthetic", "token": secret}), b.session, 200)
	client := &flexsync.Client{BaseURL: remote.URL + "/flex", HTTP: remote.Client()}
	before := f.databaseState(t)
	require.NoError(t, jobs.NewFlexSync(q, client, time.Hour, logger).Run(context.Background()))
	require.EqualValues(t, 1, calls.Load())
	row, err := q.GetFlexSyncSettings(context.Background(), store.GetFlexSyncSettingsParams{AccountID: b.accounts[0], UserID: b.id})
	require.NoError(t, err)
	noProviderSecrets(t, row.LastError+logs.String(), secret, "Authorization")
	after := f.databaseState(t)
	delete(before, "flex_sync_settings")
	delete(after, "flex_sync_settings")
	require.Equal(t, before, after)
	// Only this synthetic channel is enabled; the dispatcher never contacts fixture URLs.
	_, err = f.conn.Exec("UPDATE alert_channels SET enabled=0")
	require.NoError(t, err)
	ch := isolationObject(t, f.s, "POST", "/settings/alert-channels", isolationJSON(t, map[string]any{"kind": "webhook", "target": remote.URL + "/notify?token=" + secret}), b.session, 200)["id"].(string)
	_, err = f.conn.Exec("UPDATE trades SET closed_at=? WHERE id=?", time.Now().UTC(), b.trade)
	require.NoError(t, err)
	isolationObject(t, f.s, "PUT", "/settings/risk-rules", `{"max_risk_per_trade":1}`, b.session, 200)
	isolationObject(t, f.s, "PUT", "/settings/alerts", `{"enabled":true,"rule_risk":true,"loss_streak_n":3,"prop_warn_pct":0.8,"unreviewed_days":7,"timezone":"UTC"}`, b.session, 200)
	count := calls.Load()
	require.NoError(t, alerts.NewService(q, logger, true).EvaluateUser(context.Background(), b.id))
	require.Greater(t, calls.Load(), count, "async notification failure must actually be triggered")
	var stored string
	require.NoError(t, f.conn.QueryRow("SELECT last_error FROM alert_channels WHERE id=?", ch).Scan(&stored))
	require.NotEmpty(t, stored)
	noProviderSecrets(t, stored+logs.String(), secret)
	noProviderSecrets(t, do(f.s, "GET", "/api/v1/alerts/events", "", b.pat).Body.String(), secret)
	noProviderSecrets(t, do(f.s, "GET", "/api/v1/flex-sync", "", a.pat).Body.String(), secret)
}

func TestCrossUserIsolationProviderFailureMatrix(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "owner", "ISOA", 2)
	b := f.seed(t, "member", "ISOB", 9876)
	q := store.NewForDriver(f.conn, "sqlite")
	const key, prompt = "owner-key-244", "private-prompt-244"
	secret := key + "; Authorization: Bearer " + key + "; " + prompt
	for _, mode := range []string{"auth", "embedded", "malformed", "partial", "timeout", "cancel", "fallback"} {
		for _, identity := range []struct{ name, token string }{{"jwt", b.session}, {"pat", b.pat}, {"owner", a.session}} {
			for _, path := range []string{"/trades/" + b.trade + "/coach", "/trades/" + b.trade + "/coach/stream", "/news/" + b.news + "/analyze", "/ocr/parse"} {
				if identity.name == "owner" {
					path = strings.ReplaceAll(strings.ReplaceAll(path, b.trade, a.trade), b.news, a.news)
				}
				t.Run(mode+"/"+identity.name+path, func(t *testing.T) {
					var logs bytes.Buffer
					var calls atomic.Int64
					client := &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
						n := calls.Add(1)
						if r.Header.Get("Authorization") != "Bearer "+key {
							t.Error("configured key did not reach provider")
						}
						if mode == "timeout" {
							return nil, fmt.Errorf("%w: %s", context.DeadlineExceeded, secret)
						}
						if mode == "cancel" {
							return nil, fmt.Errorf("%w: %s", context.Canceled, secret)
						}
						status := 200
						body := isolationJSON(t, map[string]any{"error": map[string]string{"message": secret}})
						if mode == "auth" || mode == "fallback" {
							status = 401
						}
						if mode == "fallback" && n == 1 {
							status = 400
						}
						if mode == "malformed" {
							body = "{\"" + secret
						}
						if mode == "partial" {
							chunk := isolationJSON(t, map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": `{"notes":[{"tone":"tip","headline":"Partial","detail":"safe output"}],`}}}})
							body = "data: " + chunk + "\n\ndata: " + body + "\n\n"
						}
						return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
					})}
					cfg := ocr.VisionConfig{Enabled: true, BaseURL: "http://private-provider.invalid", APIKey: key, CustomPrompt: prompt, HTTPClient: client}
					s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(&logs, nil)), CoachDefaults: cfg, OCR: ocr.NewService(cfg, nil)})
					before := f.databaseState(t)
					var response *httptest.ResponseRecorder
					if path == "/ocr/parse" {
						response = httptest.NewRecorder()
						s.Echo.ServeHTTP(response, multipartFileReq(t, "/api/v1"+path, identity.token, "fixture.png", "fictional-image", nil))
					} else {
						response = do(s, "POST", "/api/v1"+path, `{}`, identity.token)
					}
					want := 200
					if strings.Contains(path, "/analyze") || path == "/ocr/parse" {
						want = 502
						if mode == "timeout" {
							want = 504
						}
					}
					require.Equal(t, want, response.Code)
					require.Positive(t, calls.Load(), "failure must reach the provider")
					if mode == "fallback" && path != "/ocr/parse" {
						require.GreaterOrEqual(t, calls.Load(), int64(2))
					}
					noProviderSecrets(t, response.Body.String()+logs.String(), key, prompt, "Authorization: Bearer", "private-provider.invalid")
					require.Equal(t, before, f.databaseState(t), "failed operations must not persist reviews or alter other resources")
					if strings.HasSuffix(path, "/stream") {
						frames := sseFrames(t, response.Body.String())
						require.NotEmpty(t, frames)
						require.Equal(t, "error", frames[len(frames)-1][0])
						if mode == "partial" {
							require.Equal(t, "note", frames[0][0])
						}
					}
					for _, followup := range []string{"/trades/" + b.trade + "/coach/reviews", "/news/" + b.news} {
						read := do(s, "GET", "/api/v1"+followup, "", b.session)
						require.Equal(t, 200, read.Code)
						noProviderSecrets(t, read.Body.String(), key, prompt)
					}
				})
			}
		}
	}
}

func TestCrossUserIsolationStoredProviderErrors(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "owner", "ISOA", 2)
	b := f.seed(t, "member", "ISOB", 9876)
	q := store.NewForDriver(f.conn, "sqlite")
	const ibkrSecret, webhookSecret = "ibkr-private-token-244", "webhook-private-secret-244"
	for _, token := range []string{b.session, b.pat} {
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))
		var calls atomic.Int64
		client := &flexsync.Client{BaseURL: "http://private-broker.invalid/send", HTTP: &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.Query().Get("t") != ibkrSecret {
				t.Error("IBKR credential did not reach provider")
			}
			return nil, fmt.Errorf("transport failure: Authorization Bearer %s", ibkrSecret)
		})}}
		remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Redirect(w, r, "/private?token="+webhookSecret, 302)
		}))
		s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Trades: trades.NewService(q), Logger: logger, FlexClient: client, Alerts: alerts.NewService(q, logger, true)})
		isolationObject(t, s, "PUT", "/accounts/"+b.accounts[0]+"/flex-sync", isolationJSON(t, map[string]any{"query_id": "synthetic", "token": ibkrSecret}), token, 200)
		before := f.databaseState(t)
		rec := do(s, "POST", "/api/v1/accounts/"+b.accounts[0]+"/flex-sync/run", `{}`, token)
		require.Equal(t, 502, rec.Code)
		require.Positive(t, calls.Load())
		noProviderSecrets(t, rec.Body.String(), ibkrSecret, "Authorization", "private-broker.invalid")
		for _, path := range []string{"/accounts/" + b.accounts[0] + "/flex-sync", "/flex-sync"} {
			read := do(s, "GET", "/api/v1"+path, "", token)
			require.Equal(t, 200, read.Code)
			noProviderSecrets(t, read.Body.String(), ibkrSecret, "Authorization", "private-broker.invalid")
		}
		ch := isolationObject(t, s, "POST", "/settings/alert-channels", isolationJSON(t, map[string]any{"kind": "webhook", "target": remote.URL + "?token=" + webhookSecret}), token, 200)["id"].(string)
		rec = do(s, "POST", "/api/v1/settings/alert-channels/"+ch+"/test", `{}`, token)
		require.Equal(t, 200, rec.Code)
		noProviderSecrets(t, rec.Body.String(), webhookSecret)
		var stored string
		require.NoError(t, f.conn.QueryRow("SELECT last_error FROM alert_channels WHERE id=?", ch).Scan(&stored))
		noProviderSecrets(t, stored+logs.String(), ibkrSecret, webhookSecret, "Authorization", "private-broker.invalid")
		row, err := q.GetFlexSyncSettings(context.Background(), store.GetFlexSyncSettingsParams{AccountID: b.accounts[0], UserID: b.id})
		require.NoError(t, err)
		noProviderSecrets(t, row.LastError, ibkrSecret)
		// Only the authenticated user's connection/channel status may change.
		after := f.databaseState(t)
		for _, table := range []string{"flex_sync_settings", "alert_channels"} {
			delete(before, table)
			delete(after, table)
		}
		require.Equal(t, before, after)
		require.Equal(t, 404, do(s, "POST", "/api/v1/accounts/"+b.accounts[0]+"/flex-sync/run", `{}`, a.pat).Code)
		require.Equal(t, 404, do(s, "POST", "/api/v1/settings/alert-channels/"+ch+"/test", `{}`, a.pat).Code)
		// Legacy status rows are hidden on read without rewriting credential/configuration history.
		_, err = f.conn.Exec("UPDATE flex_sync_settings SET last_error=? WHERE account_id=?", ibkrSecret, b.accounts[0])
		require.NoError(t, err)
		_, err = f.conn.Exec("UPDATE alert_channels SET last_error=? WHERE id=?", webhookSecret, ch)
		require.NoError(t, err)
		noProviderSecrets(t, do(s, "GET", "/api/v1/flex-sync", "", token).Body.String(), ibkrSecret)
		var channels []map[string]any
		read := do(s, "GET", "/api/v1/settings/alert-channels", "", token)
		require.Equal(t, 200, read.Code)
		require.NoError(t, json.Unmarshal(read.Body.Bytes(), &channels))
		for _, channel := range channels {
			noProviderSecrets(t, channel["last_error"].(string), webhookSecret)
		}
		remote.Close()
	}
}
