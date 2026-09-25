package importer

import "time"

const (
	PositionCash        = "cash"
	PositionMarginLong  = "margin_long"
	PositionMarginShort = "margin_short"
	PositionIncrease    = "increase"
	PositionReduce      = "reduce"
)

// ParsedExecution is a broker-agnostic fill produced by an Importer.
type ParsedExecution struct {
	ExternalID          string
	DedupKey            string // optional broker-stable identity when fill fields are not unique
	Symbol              string
	StockName           string // broker-provided company/security name, when available
	InstrumentType      string
	OptionRight         string   // call|put when instrument is option
	Strike              string   // option strike, decimal string ("120", "37.5")
	Expiry              string   // option expiry, YYYY-MM-DD
	Side                string   // buy|sell
	PositionType        string   // cash|margin_long|margin_short when source supplies it
	PositionEffect      string   // increase|reduce when source supplies it
	ReportedRealizedPnl *float64 // broker-reported close result, when the source identifies one
	ReportedCloseBasis  *float64 // broker-reported average acquisition price for a close
	EventType           string   // position_conversion when the source identifies a conversion
	ConversionType      string   // broker conversion kind, e.g. genbiki
	ConversionID        string   // links the source and destination legs of one conversion
	Quantity            float64
	Price               float64
	Fees                float64
	Commission          float64
	ExecutedAt          time.Time
	SourceTimePrecision string  // "date" when a broker supplied no clock time
	Multiplier          float64 // 1 stock, 100 option; 0 means "default to 1"
	// LotKey isolates overlapping same-symbol round-trips (journal imports).
	// Stored in executions.details as {"lot":"..."}.
	LotKey string
	// Annotation is set on the opening fill of a journal-trade row so callers
	// can attach journal/tags/setup after regroup (trade id = opening fill id).
	Annotation *TradeAnnotation
}

// TradeAnnotation carries journal metadata from a trade-level CSV row.
type TradeAnnotation struct {
	Notes      string
	SetupName  string
	Confidence *int64
	Target     *float64
	Stop       *float64
	Emotion    string
	Tags       []TagRef
	Dividends  float64
}

// TagRef is a tag to create/attach during journal import.
type TagRef struct {
	Name string
	Kind string // mistake|custom
}

type RowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type ParseResult struct {
	Executions []ParsedExecution
	Errors     []RowError
	Format     string // "executions" | "journal_trades"
}

type Importer interface {
	Detect(headers []string) bool
	ParseRows(rows []map[string]string) ParseResult
	Name() string
}

// DefaultMultiplier returns the conventional multiplier for an instrument type.
func DefaultMultiplier(instrumentType string) float64 {
	switch instrumentType {
	case "option":
		return 100
	default:
		return 1
	}
}
