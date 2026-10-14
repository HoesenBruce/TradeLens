package flexsync_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/flexsync"
	"github.com/tradermemos/api/internal/store"
)

const flexCSV = "ClientAccountID,Symbol,Buy/Sell,Quantity,TradePrice,DateTime,IBCommission,AssetClass,Multiplier,Put/Call\n" +
	"U1234567,AAPL,BUY,100,10,20260730;093000,-1.00,STK,1,\n" +
	"U1234567,AAPL,SELL,-100,12,20260730;100000,-1.00,STK,1,\n" +
	"U1234567,MU 260817C01030000,BUY,5,3.84,20260731;094500,-1.42,OPT,100,C\n"

// flexServer fakes IBKR's two-step Flex Web Service: SendRequest hands out a
// reference, GetStatement reports "in progress" pendingCount times, then
// serves the CSV.
type flexServer struct {
	*httptest.Server
	pendingLeft atomic.Int32
	statements  atomic.Int32
}

func newFlexServer(t *testing.T, pending int) *flexServer {
	t.Helper()
	fs := &flexServer{}
	fs.pendingLeft.Store(int32(pending))
	mux := http.NewServeMux()
	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "tok123", r.URL.Query().Get("t"))
		require.Equal(t, "q42", r.URL.Query().Get("q"))
		fmt.Fprintf(w, `<FlexStatementResponse timestamp="x"><Status>Success</Status><ReferenceCode>REF9</ReferenceCode><Url>%s/get</Url></FlexStatementResponse>`, fs.URL)
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "REF9", r.URL.Query().Get("q"))
		if fs.pendingLeft.Add(-1) >= 0 {
			fmt.Fprint(w, `<FlexStatementResponse><Status>Warn</Status><ErrorCode>1019</ErrorCode><ErrorMessage>Statement generation in progress. Please try again shortly.</ErrorMessage></FlexStatementResponse>`)
			return
		}
		fs.statements.Add(1)
		fmt.Fprint(w, flexCSV)
	})
	fs.Server = httptest.NewServer(mux)
	t.Cleanup(fs.Close)
	return fs
}

func (fs *flexServer) client() *flexsync.Client {
	return &flexsync.Client{
		BaseURL:      fs.URL + "/send",
		PollInterval: time.Millisecond,
		PollAttempts: 5,
	}
}

func TestClientFetchStatementPollsUntilReady(t *testing.T) {
	fs := newFlexServer(t, 2)
	data, err := fs.client().FetchStatement(context.Background(), "tok123", "q42")
	require.NoError(t, err)
	require.Equal(t, flexCSV, string(data))
	require.EqualValues(t, 1, fs.statements.Load())
}

func TestClientReportsFlexErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<FlexStatementResponse><Status>Fail</Status><ErrorCode>1012</ErrorCode><ErrorMessage>Token has expired.</ErrorMessage></FlexStatementResponse>`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := &flexsync.Client{BaseURL: srv.URL + "/send", PollInterval: time.Millisecond}
	_, err := c.FetchStatement(context.Background(), "tok", "q")
	require.EqualError(t, err, "external service request failed")
	require.NotContains(t, err.Error(), "Token has expired")
}

func newSyncFixture(t *testing.T) (*store.Queries, store.FlexSyncSetting) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, db.Migrate(conn))
	q := store.New(conn)
	ctx := context.Background()

	u, err := q.CreateUser(ctx, store.CreateUserParams{ID: uuid.New().String(), Email: "f@x.com", PasswordHash: "x"})
	require.NoError(t, err)
	acc, err := q.CreateAccount(ctx, store.CreateAccountParams{
		ID: uuid.New().String(), UserID: u.ID, Name: "IB",
		Broker: "ibkr", AccountType: "margin", BaseCurrency: "USD", StartingBalance: 10000,
	})
	require.NoError(t, err)
	settings, err := q.UpsertFlexSyncSettings(ctx, store.UpsertFlexSyncSettingsParams{
		AccountID: acc.ID, UserID: u.ID, Token: "tok123", QueryID: "q42", Enabled: 1,
	})
	require.NoError(t, err)
	return q, settings
}

func TestSyncImportsAndDedups(t *testing.T) {
	q, settings := newSyncFixture(t)
	fs := newFlexServer(t, 0)
	ctx := context.Background()

	res, err := flexsync.Sync(ctx, q, fs.client(), settings)
	require.NoError(t, err)
	require.Equal(t, 3, res.Inserted)
	require.Equal(t, 0, res.Skipped)

	trades, err := q.ListTrades(ctx, store.ListTradesParams{UserID: settings.UserID, AccountID: settings.AccountID})
	require.NoError(t, err)
	require.Len(t, trades, 2)
	bySymbol := map[string]store.Trade{}
	for _, tr := range trades {
		bySymbol[tr.Symbol] = tr
	}
	aapl := bySymbol["AAPL"]
	require.Equal(t, "closed", aapl.Status)
	require.InDelta(t, 198, aapl.NetPnl.Float64, 0.01) // (12−10)×100 − $2 commission

	// The OCC option symbol is normalized to underlying + contract details.
	mu, ok := bySymbol["MU"]
	require.True(t, ok, "expected an MU trade, got %v", trades)
	require.Equal(t, "open", mu.Status)
	fills, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{
		UserID: settings.UserID, AccountID: settings.AccountID,
	})
	require.NoError(t, err)
	var optDetails string
	for _, f := range fills {
		if f.InstrumentType == "option" {
			optDetails = f.Details.String
		}
	}
	require.Contains(t, optDetails, `"strike":"1030"`)
	require.Contains(t, optDetails, `"expiry":"2026-08-17"`)
	require.Contains(t, optDetails, `"option_right":"call"`)

	// Second sync of the same statement: everything is a duplicate.
	res2, err := flexsync.Sync(ctx, q, fs.client(), settings)
	require.NoError(t, err)
	require.Equal(t, 0, res2.Inserted)
	require.Equal(t, 3, res2.Skipped)

	// Both runs left committed batches for history.
	batches, err := q.ListImportBatches(ctx, settings.UserID)
	require.NoError(t, err)
	require.Len(t, batches, 2)
	for _, b := range batches {
		require.Equal(t, "committed", b.Status)
		require.Equal(t, "ibkr-flex-sync", b.Source)
	}
}

func TestSyncRejectsNonIBKRStatement(t *testing.T) {
	q, settings := newSyncFixture(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<FlexStatementResponse><Status>Success</Status><ReferenceCode>R</ReferenceCode><Url>%s/get</Url></FlexStatementResponse>`, "http://"+r.Host)
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "some,other,columns\n1,2,3\n")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := &flexsync.Client{BaseURL: srv.URL + "/send", PollInterval: time.Millisecond}
	_, err := flexsync.Sync(context.Background(), q, c, settings)
	require.ErrorContains(t, err, "Trades → Executions")
}

func TestRecordOutcome(t *testing.T) {
	q, settings := newSyncFixture(t)
	ctx := context.Background()

	flexsync.RecordOutcome(ctx, q, settings, flexsync.Result{Inserted: 3, Skipped: 1, Rows: 4}, nil)
	row, err := q.GetFlexSyncSettings(ctx, store.GetFlexSyncSettingsParams{AccountID: settings.AccountID, UserID: settings.UserID})
	require.NoError(t, err)
	require.True(t, row.LastSyncedAt.Valid)
	require.Contains(t, row.LastStatus, "3 new")
	require.Empty(t, row.LastError)

	flexsync.RecordOutcome(ctx, q, settings, flexsync.Result{}, fmt.Errorf("token expired"))
	row, err = q.GetFlexSyncSettings(ctx, store.GetFlexSyncSettingsParams{AccountID: settings.AccountID, UserID: settings.UserID})
	require.NoError(t, err)
	require.Equal(t, "error", row.LastStatus)
	require.Equal(t, "external service request failed", row.LastError)
	require.NotContains(t, row.LastError, "token expired")
}

func TestClientSafeErrorIdentity(t *testing.T) {
	fs := newFlexServer(t, 100)
	c := fs.client()
	_, err := c.FetchStatement(context.Background(), "tok123", "q42")
	require.ErrorIs(t, err, flexsync.ErrStatementPending)
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		ctx, cancel := context.WithCancel(context.Background())
		if cause == context.DeadlineExceeded {
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}
		cancel()
		_, err = c.FetchStatement(ctx, "tok123", "q42")
		require.ErrorIs(t, err, cause)
		require.False(t, strings.Contains(err.Error(), "tok123"), "diagnostic marker exposed")
	}
}

type diagnosticFlexTransport func(*http.Request) (*http.Response, error)

func (f diagnosticFlexTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestClientDiagnosticFailureMatrix(t *testing.T) {
	const marker = "synthetic-diagnostic-marker-298"
	for _, mode := range []string{"status", "body", "malformed", "transport", "timeout", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			client := &flexsync.Client{BaseURL: "http://synthetic.invalid/send", HTTP: &http.Client{Transport: diagnosticFlexTransport(func(r *http.Request) (*http.Response, error) {
				switch mode {
				case "transport":
					return nil, fmt.Errorf("%s %s", r.URL, marker)
				case "timeout":
					return nil, fmt.Errorf("%w %s", context.DeadlineExceeded, marker)
				case "cancel":
					return nil, fmt.Errorf("%w %s", context.Canceled, marker)
				}
				status := 200
				body := `<FlexStatementResponse><Status>Fail</Status><ErrorMessage>` + marker + `</ErrorMessage></FlexStatementResponse>`
				if mode == "status" {
					status = 401
				}
				if mode == "malformed" {
					body = "<" + marker
				}
				return &http.Response{StatusCode: status, Status: marker, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			_, err := client.FetchStatement(context.Background(), marker, "synthetic")
			require.True(t, err != nil, "failure must reach provider")
			var logs bytes.Buffer
			slog.New(slog.NewJSONHandler(&logs, nil)).Warn("sync failed", "err", err)
			require.False(t, strings.Contains(err.Error()+logs.String(), marker), "broker diagnostic marker exposed")
			if mode == "timeout" {
				require.True(t, errors.Is(err, context.DeadlineExceeded))
			}
			if mode == "cancel" {
				require.True(t, errors.Is(err, context.Canceled))
			}
		})
	}
}
