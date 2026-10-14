package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tradermemos/api/internal/store"
)

func TestProviderExpoErrorBoundary(t *testing.T) {
	const secret = "expo-private-token-244"
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"data":[{"status":"error","message":"Authorization Bearer ` + secret + `","details":{"error":"InvalidCredentials"}}]}`))
	}))
	t.Cleanup(srv.Close)
	original := expoPushURL
	expoPushURL = srv.URL
	t.Cleanup(func() { expoPushURL = original })
	errs, err := testService().sendExpo(context.Background(), []string{secret}, Event{Title: "synthetic"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(errs) != 1 || errs[0] == nil {
		t.Fatal("provider failure did not reach the ticket boundary")
	}
	if strings.Contains(errs[0].Error(), secret) || strings.Contains(errs[0].Error(), "Authorization") {
		t.Fatal("provider ticket disclosed secret")
	}
}

func TestProviderWebhookErrorBoundary(t *testing.T) {
	const secret = "webhook-private-secret-244"
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "/private?token="+secret, 302)
	}))
	t.Cleanup(srv.Close)
	err := testService().sendWebhook(context.Background(), srv.URL+"?token="+secret, Event{Title: "synthetic"}, time.Now())
	if err == nil || calls != 1 {
		t.Fatal("redirect failure was not triggered")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("webhook error disclosed private URL token")
	}
}

func testService() *Service {
	// httptest servers listen on loopback, so the webhook client must allow
	// private addresses here.
	return &Service{httpc: http.DefaultClient, webhookc: newWebhookClient(true), now: time.Now}
}

func TestWebhookClientBlocksPrivateTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("guarded client must not reach a loopback server")
	}))
	defer srv.Close()
	s := &Service{httpc: http.DefaultClient, webhookc: newWebhookClient(false), now: time.Now}
	err := s.sendWebhook(context.Background(), srv.URL, Event{Title: "t"}, time.Now())
	if err == nil {
		t.Fatal("want dial refusal for loopback webhook target")
	}
}

func TestSendWebhookPayload(t *testing.T) {
	var got webhookPayload
	var title string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		title = r.Header.Get("X-Title")
	}))
	defer srv.Close()

	ev := Event{Rule: RuleDailyLoss, Title: "Daily loss limit hit", Body: "Down 600."}
	if err := testService().sendWebhook(context.Background(), srv.URL, ev, time.Unix(1754500000, 0)); err != nil {
		t.Fatal(err)
	}
	if got.Source != "tradermemos" || got.Rule != RuleDailyLoss {
		t.Errorf("payload metadata wrong: %+v", got)
	}
	// Discord (`content`) and Slack (`text`) fields must mirror title+body.
	want := "Daily loss limit hit — Down 600."
	if got.Content != want || got.Text != want {
		t.Errorf("content/text = %q / %q, want %q", got.Content, got.Text, want)
	}
	if title != "Daily loss limit hit" {
		t.Errorf("X-Title = %q", title)
	}
}

func TestSendWebhookErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	err := testService().sendWebhook(context.Background(), srv.URL, Event{Title: "t"}, time.Now())
	if err == nil {
		t.Fatal("5xx must surface as an error")
	}
}

func TestSendExpoTicketErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msgs []expoMessage
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &msgs); err != nil || len(msgs) != 2 {
			t.Errorf("want a 2-message batch, got %s", body)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"status":"ok"},
			{"status":"error","message":"gone","details":{"error":"DeviceNotRegistered"}}
		]}`))
	}))
	defer srv.Close()
	orig := expoPushURL
	expoPushURL = srv.URL
	defer func() { expoPushURL = orig }()

	errs, err := testService().sendExpo(context.Background(), []string{"ExponentPushToken[a]", "ExponentPushToken[b]"}, Event{Title: "t", Body: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if errs[0] != nil {
		t.Errorf("first token should deliver, got %v", errs[0])
	}
	if !errors.Is(errs[1], errDeviceNotRegistered) {
		t.Errorf("second token should be DeviceNotRegistered, got %v", errs[1])
	}
}

type providerStatusStore struct {
	store.Querier
	statuses []store.UpdateAlertChannelStatusParams
	disabled []string
}

func (q *providerStatusStore) ListEnabledAlertChannels(context.Context, string) ([]store.AlertChannel, error) {
	return []store.AlertChannel{{ID: "dead", Kind: "expo", Target: "synthetic"}, {ID: "live", Kind: "expo", Target: "synthetic-live"}}, nil
}
func (q *providerStatusStore) UpdateAlertChannelStatus(_ context.Context, p store.UpdateAlertChannelStatusParams) error {
	q.statuses = append(q.statuses, p)
	return nil
}
func (q *providerStatusStore) DisableAlertChannel(_ context.Context, id string) error {
	q.disabled = append(q.disabled, id)
	return nil
}
func TestProviderExpoDispatcherCleanup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"status":"error","message":"synthetic-diagnostic-marker-298","details":{"error":"DeviceNotRegistered"}},{"status":"ok"}]}`))
	}))
	defer srv.Close()
	old := expoPushURL
	expoPushURL = srv.URL
	t.Cleanup(func() { expoPushURL = old })
	q := &providerStatusStore{}
	var logs bytes.Buffer
	s := NewService(q, slog.New(slog.NewJSONHandler(&logs, nil)), true)
	if err := s.dispatch(context.Background(), "owned", []Event{{Title: "synthetic"}}); err != nil {
		t.Fatal("dispatch failed")
	}
	if len(q.disabled) != 1 || q.disabled[0] != "dead" || len(q.statuses) != 2 {
		t.Fatal("dead token cleanup or outcome recording changed")
	}
	if q.statuses[0].LastStatus != "error" || q.statuses[1].LastStatus != "ok" || q.statuses[1].LastError != "" {
		t.Fatal("ticket outcomes changed")
	}
	if strings.Contains(q.statuses[0].LastError+logs.String(), "synthetic-diagnostic-marker-298") {
		t.Fatal("dispatcher disclosed diagnostic marker")
	}
}

type diagnosticTransport func(*http.Request) (*http.Response, error)

func (f diagnosticTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestProviderNotificationFailureMatrix(t *testing.T) {
	const marker = "synthetic-diagnostic-marker-298"
	for _, mode := range []string{"status", "malformed", "transport", "timeout", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			client := &http.Client{Transport: diagnosticTransport(func(r *http.Request) (*http.Response, error) {
				switch mode {
				case "transport":
					return nil, fmt.Errorf("%s %s", r.URL, marker)
				case "timeout":
					return nil, fmt.Errorf("%w %s", context.DeadlineExceeded, marker)
				case "cancel":
					return nil, fmt.Errorf("%w %s", context.Canceled, marker)
				}
				status := 200
				body := "{\"" + marker
				if mode == "status" {
					status = 401
				}
				return &http.Response{StatusCode: status, Status: marker, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			s := &Service{httpc: client, webhookc: client, now: time.Now}
			_, expoErr := s.sendExpo(context.Background(), []string{marker}, Event{Title: "synthetic"})
			if expoErr == nil {
				t.Fatal("Expo failure was not triggered")
			}
			webhookErr := s.sendWebhook(context.Background(), "http://synthetic.invalid/?token="+marker, Event{Title: "synthetic"}, time.Now())
			for _, err := range []error{expoErr, webhookErr} {
				// Webhooks ignore successful response bodies; only Expo decodes them.
				if err == nil {
					if mode != "malformed" {
						t.Fatal("failure was not triggered")
					}
					continue
				}
				var logs bytes.Buffer
				slog.New(slog.NewJSONHandler(&logs, nil)).Warn("provider failed", "err", err)
				if strings.Contains(err.Error()+logs.String(), marker) {
					t.Fatal("notification diagnostic marker exposed")
				}
				if mode == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal("timeout identity changed")
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal("cancel identity changed")
				}
			}
		})
	}
}
