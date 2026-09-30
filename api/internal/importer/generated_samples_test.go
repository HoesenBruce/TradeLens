package importer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedPublicJSONSamples(t *testing.T) {
	for _, path := range []string{"../../../web/public/sample-json-import.json", "../../../docs/demo/tradermemos-demo-trades.json"} {
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(path)
			require.NoError(t, err)
			got, err := ParseJSONImport(body)
			require.NoError(t, err)
			require.Empty(t, got.Result.Errors)
			require.NotEmpty(t, got.Result.Executions)
			require.Equal(t, "journal_trades", got.Format)
		})
	}
}
