package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/api"
)

func TestAccountKindCapabilitiesRoundTrip(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "metadata@x.com")
	create := do(s, http.MethodPost, "/api/v1/accounts", `{"name":"SBI","broker":"sbi","account_kind":"brokerage","capabilities":["cash","margin"]}`, token)
	require.Equal(t, http.StatusCreated, create.Code, create.Body.String())
	var account struct {
		ID           string   `json:"id"`
		Kind         string   `json:"account_kind"`
		Capabilities []string `json:"capabilities"`
	}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &account))
	require.Equal(t, "brokerage", account.Kind)
	require.Equal(t, []string{"cash", "margin"}, account.Capabilities)

	update := do(s, http.MethodPut, "/api/v1/accounts/"+account.ID, `{"capabilities":["margin"]}`, token)
	require.Equal(t, http.StatusOK, update.Code, update.Body.String())
	read := do(s, http.MethodGet, "/api/v1/accounts/"+account.ID, "", token)
	require.Equal(t, http.StatusOK, read.Code)
	require.NoError(t, json.Unmarshal(read.Body.Bytes(), &account))
	require.Equal(t, []string{"margin"}, account.Capabilities)

	bad := do(s, http.MethodPut, "/api/v1/accounts/"+account.ID, `{"capabilities":[]}`, token)
	require.Equal(t, http.StatusBadRequest, bad.Code)

	legacy := do(s, http.MethodPost, "/api/v1/accounts", `{"name":"Replay","account_type":"backtest"}`, token)
	require.Equal(t, http.StatusCreated, legacy.Code, legacy.Body.String())
	require.NoError(t, json.Unmarshal(legacy.Body.Bytes(), &account))
	require.Equal(t, api.AccountTypeBacktest, account.Kind)
	require.Empty(t, account.Capabilities)
}
