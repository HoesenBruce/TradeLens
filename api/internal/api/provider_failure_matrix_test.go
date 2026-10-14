package api_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
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

func (f *isolationFixture) otherProviderStatus(t *testing.T, userID string) [32]byte {
	t.Helper()
	var records []string
	for _, table := range []string{"alert_channels", "flex_sync_settings"} {
		rows, err := f.conn.Query("SELECT * FROM "+table+" WHERE user_id != ?", userID)
		require.NoError(t, err)
		cols, err := rows.Columns()
		require.NoError(t, err)
		for rows.Next() {
			values := make([]any, len(cols))
			dest := make([]any, len(cols))
			for i := range values {
				dest[i] = &values[i]
			}
			require.NoError(t, rows.Scan(dest...))
			records = append(records, table+isolationJSON(t, values))
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
	sort.Strings(records)
	return sha256.Sum256([]byte(strings.Join(records, "\n")))
}

func TestCrossUserIsolationProviderConfigurationDiagnostics(t *testing.T) {
	f := newIsolationFixture(t)
	owner := registerAndLogin(t, f.s, "owner@diagnostic.invalid")
	member := registerAndLogin(t, f.s, "member@diagnostic.invalid")
	pat := isolationObject(t, f.s, "POST", "/access-tokens", `{"name":"member"}`, member, 201)["token"].(string)
	q := store.NewForDriver(f.conn, "sqlite")
	const secret = "diagnostic-owner-key-244"
	ownerPAT := isolationObject(t, f.s, "POST", "/access-tokens", `{"name":"owner-matrix"}`, owner, 201)["token"].(string)
	var mode string
	var calls atomic.Int64
	var logs bytes.Buffer
	client := &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("owner key did not reach models provider")
		}
		switch mode {
		case "transport":
			return nil, fmt.Errorf("%s %s", r.URL, secret)
		case "timeout":
			return nil, fmt.Errorf("%w %s", context.DeadlineExceeded, secret)
		case "cancel":
			return nil, fmt.Errorf("%w %s", context.Canceled, secret)
		}
		status, body := 200, `{"error":{"message":"Authorization Bearer `+secret+`"}}`
		if mode == "status" {
			status = 401
		}
		if mode == "malformed" {
			body = "{\"" + secret
		}
		return &http.Response{StatusCode: status, Status: secret, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	cfg := ocr.VisionConfig{Enabled: true, BaseURL: "http://private-provider.invalid", APIKey: secret, HTTPClient: client}
	s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Logger: slog.New(slog.NewTextHandler(&logs, nil)), CoachDefaults: cfg, OCR: ocr.NewService(cfg, nil)})
	for _, failure := range []string{"status", "embedded", "malformed", "transport", "timeout", "cancel"} {
		mode = failure
		for _, name := range []string{"ocr", "coach"} {
			for _, action := range []string{"test", "models"} {
				path := "/api/v1/settings/" + name + "/" + action
				t.Logf("case=config/%s/%s/%s", failure, name, action)
				for _, token := range []string{member, pat} {
					count := calls.Load()
					before := f.databaseState(t)
					require.Equal(t, 403, do(s, "POST", path, `{}`, token).Code)
					require.Equal(t, count, calls.Load())
					require.Equal(t, before, f.databaseState(t))
				}
				for _, ownerToken := range []string{owner, ownerPAT} {
					count := calls.Load()
					before := f.databaseState(t)
					response := do(s, "POST", path, `{}`, ownerToken)
					if name == "coach" {
						require.Equal(t, 503, response.Code)
						require.Equal(t, count, calls.Load(), "isolated Coach must not contact provider")
					} else {
						require.Equal(t, 200, response.Code)
						require.Greater(t, calls.Load(), count)
					}
					noProviderSecrets(t, response.Body.String()+fmt.Sprint(response.Header())+logs.String(), secret, "Authorization", "private-provider.invalid")
					require.Equal(t, before, f.databaseState(t))
				}
			}
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
	otherBefore := f.otherProviderStatus(t, b.id)
	before := f.databaseState(t)
	require.NoError(t, jobs.NewFlexSync(q, client, time.Hour, logger).Run(context.Background()))
	require.EqualValues(t, 1, calls.Load())
	row, err := q.GetFlexSyncSettings(context.Background(), store.GetFlexSyncSettingsParams{AccountID: b.accounts[0], UserID: b.id})
	require.NoError(t, err)
	noProviderSecrets(t, row.LastError+logs.String(), secret, "Authorization")
	require.Equal(t, otherBefore, f.otherProviderStatus(t, b.id))
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
	notifyBefore := f.databaseState(t)
	otherNotifyBefore := f.otherProviderStatus(t, b.id)
	require.NoError(t, alerts.NewService(q, logger, true).EvaluateUser(context.Background(), b.id))
	require.Greater(t, calls.Load(), count, "async notification failure must actually be triggered")
	notifyAfter := f.databaseState(t)
	// Evaluation creates owned alert events even when delivery fails.
	for _, table := range []string{"alert_events", "alert_channels"} {
		delete(notifyBefore, table)
		delete(notifyAfter, table)
	}
	require.Equal(t, notifyBefore, notifyAfter)
	require.Equal(t, otherNotifyBefore, f.otherProviderStatus(t, b.id))
	var stored string
	require.NoError(t, f.conn.QueryRow("SELECT last_error FROM alert_channels WHERE id=?", ch).Scan(&stored))
	require.NotEmpty(t, stored)
	noProviderSecrets(t, stored+logs.String(), secret)
	noProviderSecrets(t, do(f.s, "GET", "/api/v1/alerts/events", "", b.pat).Body.String(), secret)
	noProviderSecrets(t, do(f.s, "GET", "/api/v1/flex-sync", "", a.pat).Body.String(), secret)
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
		otherBefore := f.otherProviderStatus(t, b.id)
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
		require.Equal(t, otherBefore, f.otherProviderStatus(t, b.id))
		count := calls.Load()
		deniedBefore := f.databaseState(t)
		require.Equal(t, 404, do(s, "POST", "/api/v1/accounts/"+b.accounts[0]+"/flex-sync/run", `{}`, a.pat).Code)
		require.Equal(t, 404, do(s, "POST", "/api/v1/settings/alert-channels/"+ch+"/test", `{}`, a.pat).Code)
		require.Equal(t, count, calls.Load())
		require.Equal(t, deniedBefore, f.databaseState(t))
		// Legacy status rows are hidden on read without rewriting credential/configuration history.
		_, err = f.conn.Exec("UPDATE flex_sync_settings SET last_error=? WHERE account_id=?", ibkrSecret, b.accounts[0])
		require.NoError(t, err)
		_, err = f.conn.Exec("UPDATE alert_channels SET last_error=? WHERE id=?", webhookSecret, ch)
		require.NoError(t, err)
		legacyBefore := f.databaseState(t)
		noProviderSecrets(t, do(s, "GET", "/api/v1/flex-sync", "", token).Body.String(), ibkrSecret)
		var channels []map[string]any
		read := do(s, "GET", "/api/v1/settings/alert-channels", "", token)
		require.Equal(t, 200, read.Code)
		require.NoError(t, json.Unmarshal(read.Body.Bytes(), &channels))
		for _, channel := range channels {
			noProviderSecrets(t, channel["last_error"].(string), webhookSecret)
		}
		require.Equal(t, legacyBefore, f.databaseState(t))
		_, err = f.conn.Exec("UPDATE flex_sync_settings SET last_status='imported 3',last_error='' WHERE account_id=?", b.accounts[0])
		require.NoError(t, err)
		_, err = f.conn.Exec("UPDATE alert_channels SET last_status='ok',last_error='' WHERE id=?", ch)
		require.NoError(t, err)
		successBefore := f.databaseState(t)
		flex := isolationObject(t, s, "GET", "/accounts/"+b.accounts[0]+"/flex-sync", "", token, 200)
		require.True(t, flex["last_error"] == nil && flex["last_status"] == "imported 3", "empty error or successful summary changed")
		read = do(s, "GET", "/api/v1/settings/alert-channels", "", token)
		require.NoError(t, json.Unmarshal(read.Body.Bytes(), &channels))
		for _, channel := range channels {
			if channel["id"] == ch {
				require.True(t, channel["last_error"] == "" && channel["last_status"] == "ok", "success channel state changed")
			}
		}
		require.Equal(t, successBefore, f.databaseState(t))
		remote.Close()
	}
}

func TestCrossUserIsolationProviderFailureMatrix(t *testing.T) {
	f := newIsolationFixture(t)
	owner := registerAndLogin(t, f.s, "owner@ocr-matrix.invalid")
	member := registerAndLogin(t, f.s, "member@ocr-matrix.invalid")
	pat := isolationObject(t, f.s, "POST", "/access-tokens", `{"name":"matrix"}`, member, 201)["token"].(string)
	q := store.NewForDriver(f.conn, "sqlite")
	const marker = "synthetic-diagnostic-marker-298"
	for _, mode := range []string{"status", "embedded", "malformed", "transport", "timeout", "cancel"} {
		for _, identity := range []struct{ name, token string }{{"owner", owner}, {"jwt", member}, {"pat", pat}} {
			t.Run(mode+"/"+identity.name, func(t *testing.T) {
				var logs bytes.Buffer
				var calls atomic.Int64
				client := &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Header.Get("Authorization") != "Bearer "+marker {
						t.Error("configured key did not reach provider")
					}
					switch mode {
					case "transport":
						return nil, fmt.Errorf("transport %s: %s", r.URL, marker)
					case "timeout":
						return nil, fmt.Errorf("%w: %s", context.DeadlineExceeded, marker)
					case "cancel":
						return nil, fmt.Errorf("%w: %s", context.Canceled, marker)
					}
					status := 200
					body := isolationJSON(t, map[string]any{"error": map[string]string{"message": marker}})
					if mode == "status" {
						status = 401
					}
					if mode == "malformed" {
						body = "{\"" + marker
					}
					return &http.Response{StatusCode: status, Status: marker, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}
				cfg := ocr.VisionConfig{Enabled: true, BaseURL: "http://private-provider.invalid", APIKey: marker, HTTPClient: client}
				s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Logger: slog.New(slog.NewTextHandler(&logs, nil)), OCR: ocr.NewService(cfg, nil)})
				before := f.databaseState(t)
				response := httptest.NewRecorder()
				s.Echo.ServeHTTP(response, multipartFileReq(t, "/api/v1/ocr/parse", identity.token, "fixture.png", "fictional-image", nil))
				want := 502
				if mode == "timeout" {
					want = 504
				}
				require.Equal(t, want, response.Code)
				require.Positive(t, calls.Load(), "failure must reach provider")
				noProviderSecrets(t, response.Body.String()+fmt.Sprint(response.Header())+logs.String(), marker, "private-provider.invalid")
				require.Equal(t, before, f.databaseState(t), "failed parse must not mutate durable state")
			})
		}
	}
}
