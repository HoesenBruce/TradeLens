package coach

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validNewsAnalysis = `{"summary":"Supply expansion","category":"industry","assets":[{"asset_type":"stock","symbol":"285A","market":"JP","exchange":"TSE","display_name":"Test","direction":"bullish","confidence":65,"reasoning":"Potential demand","catalysts":"Orders","risks":"Execution","horizons":[1,5]},{"asset_type":"etf","symbol":"SPY","market":"US","exchange":"NYSE","display_name":"S&P","direction":"neutral","confidence":0,"reasoning":"Limited exposure","catalysts":"","risks":"","horizons":[20]}]}`

func TestNewsAnalysisRejectsEntireMalformedResult(t *testing.T) {
	cases := []string{`{}`, `null`, validNewsAnalysis + `{}`, strings.Replace(validNewsAnalysis, `"confidence":65`, `"confidence":null`, 1), strings.Replace(validNewsAnalysis, `"risks":"Execution",`, "", 1), strings.Replace(validNewsAnalysis, `"market":"JP"`, `"market":null`, 1), strings.Replace(validNewsAnalysis, `"confidence":65`, `"confidence":101`, 1), strings.Replace(validNewsAnalysis, `[1,5]`, `[2]`, 1), strings.Replace(validNewsAnalysis, `[1,5]`, `[1,1]`, 1), strings.Replace(validNewsAnalysis, `"neutral"`, `"unknown"`, 1), strings.Replace(validNewsAnalysis, `"summary":`, `"extra":true,"summary":`, 1)}
	for _, raw := range cases {
		got, err := parseNewsAnalysis(raw)
		require.Error(t, err, raw)
		require.Empty(t, got.Assets)
	}
}
