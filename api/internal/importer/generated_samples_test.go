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
			if len(got.Setups) > 0 {
				require.NotNil(t, got.Result.Executions[0].Annotation)
				require.Equal(t, "Momentum", got.Result.Executions[0].Annotation.SetupName)
				require.NotEmpty(t, got.Result.Executions[0].Annotation.Tags)
				var options int
				for _, fill := range got.Result.Executions {
					if fill.InstrumentType == "option" {
						options++
						require.Equal(t, 100.0, fill.Multiplier)
					}
				}
				require.Positive(t, options)
			}
		})
	}
}
