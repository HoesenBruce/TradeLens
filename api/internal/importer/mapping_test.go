package importer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuggestMapping(t *testing.T) {
	headers := []string{"Symbol", "銘柄名", "B/S", "Qty", "Fill Price", "Trade Date", "Commission"}
	m := SuggestMapping(headers)
	require.Equal(t, "Symbol", m["symbol"])
	require.Equal(t, "銘柄名", m["stock_name"])
	require.Equal(t, "B/S", m["side"])
	require.Equal(t, "Qty", m["quantity"])
	require.Equal(t, "Fill Price", m["price"])
	require.Equal(t, "Trade Date", m["executed_at"])
	require.Equal(t, "Commission", m["commission"])
}
