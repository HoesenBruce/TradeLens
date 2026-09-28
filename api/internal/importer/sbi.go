package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"maps"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

type SBICashImport struct {
	Headers      []string
	Rows         []map[string]string
	Transactions []JSONCashTx
	Errors       []RowError
}

var sbiFields = map[string]string{
	"symbol":          "銘柄コード",
	"stock_name":      "銘柄",
	"side":            "取引",
	"quantity":        "約定数量",
	"price":           "約定単価",
	"executed_at":     "約定日",
	"fees":            "手数料/諸経費等",
	"instrument_type": "=stock",
}

// ReadSBITradeCSV decodes SBI's CP932 export and removes its report preamble.
func ReadSBITradeCSV(data []byte) (headers []string, rows []map[string]string, ok bool, err error) {
	decoded, err := decodeSBI(data)
	if err != nil {
		return nil, nil, false, nil
	}
	if !bytes.Contains(decoded, []byte("約定履歴照会")) {
		return nil, nil, false, nil
	}

	r := csv.NewReader(bytes.NewReader(decoded))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, true, err
	}
	headerAt := -1
	for i, record := range records {
		if hasSBIHeader(record) {
			headerAt = i
			break
		}
	}
	if headerAt < 0 {
		return nil, nil, true, fmt.Errorf("SBI execution header not found")
	}

	headers = trimHeaders(records[headerAt])
	for _, record := range records[headerAt+1:] {
		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}
		row := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(record) {
				row[header] = record[i]
			}
		}
		rows = append(rows, row)
	}
	return headers, rows, true, nil
}

// ReadSBICashCSV parses SBI's 円貨入出金明細 export into the existing cash ledger.
func ReadSBICashCSV(data []byte) (SBICashImport, bool, error) {
	decoded, err := decodeSBI(data)
	if err != nil {
		return SBICashImport{}, false, nil
	}
	if !bytes.Contains(decoded, []byte("円貨入出金明細")) {
		return SBICashImport{}, false, nil
	}

	r := csv.NewReader(bytes.NewReader(decoded))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return SBICashImport{}, true, err
	}
	headerAt := -1
	for i, record := range records {
		if hasSBIHeaderFields(record, []string{"入出金日", "取引", "区分", "摘要", "出金額", "入金額"}) {
			headerAt = i
			break
		}
	}
	if headerAt < 0 {
		return SBICashImport{}, true, fmt.Errorf("SBI cash transaction header not found")
	}

	out := SBICashImport{Headers: trimHeaders(records[headerAt]), Errors: []RowError{}}
	loc, _ := time.LoadLocation("Asia/Tokyo")
	for i, record := range records[headerAt+1:] {
		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}
		row := make(map[string]string, len(out.Headers))
		for j, header := range out.Headers {
			if j < len(record) {
				row[header] = strings.TrimSpace(record[j])
			}
		}
		out.Rows = append(out.Rows, row)

		tx, err := parseSBICashRow(row, loc)
		if err != nil {
			out.Errors = append(out.Errors, RowError{Row: i + 1, Message: err.Error()})
			continue
		}
		out.Transactions = append(out.Transactions, tx)
	}
	return out, true, nil
}

func decodeSBI(data []byte) ([]byte, error) {
	if utf8.Valid(data) {
		return data, nil
	}
	decoded, _, err := transform.Bytes(japanese.ShiftJIS.NewDecoder(), data)
	return decoded, err
}

func parseSBICashRow(row map[string]string, loc *time.Location) (JSONCashTx, error) {
	date, err := time.ParseInLocation("2006/01/02", row["入出金日"], loc)
	if err != nil {
		return JSONCashTx{}, fmt.Errorf("invalid cash transaction date %q", row["入出金日"])
	}
	outflow, outErr := strconv.ParseFloat(strings.ReplaceAll(row["出金額"], ",", ""), 64)
	inflow, inErr := strconv.ParseFloat(strings.ReplaceAll(row["入金額"], ",", ""), 64)
	if outErr != nil || inErr != nil || (outflow == 0) == (inflow == 0) {
		return JSONCashTx{}, fmt.Errorf("invalid cash transaction amount")
	}
	if (inflow > 0 && row["取引"] != "入金") || (outflow > 0 && row["取引"] != "出金") {
		return JSONCashTx{}, fmt.Errorf("cash transaction direction does not match amount")
	}

	typ := "adjustment"
	switch row["区分"] {
	case "金融機関からの入金":
		typ = "deposit"
	case "金融機関への出金", "各商品取引口座への振替出金":
		typ = "withdrawal"
	case "利金・配当金":
		typ = "dividend"
	}
	amount := inflow - outflow
	return JSONCashTx{
		Type: typ, Amount: amount, Currency: "JPY",
		OccurredAt: date.UTC(), Note: strings.TrimSpace(row["区分"] + " · " + row["摘要"]),
	}, nil
}

func hasSBIHeader(record []string) bool {
	return hasSBIHeaderFields(record, []string{"約定日", "銘柄コード", "取引", "約定数量", "約定単価"})
}

func hasSBIHeaderFields(record, required []string) bool {
	present := make(map[string]bool, len(record))
	for _, value := range record {
		present[strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))] = true
	}
	for _, field := range required {
		if !present[field] {
			return false
		}
	}
	return true
}

func trimHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, header := range headers {
		out[i] = strings.TrimSpace(strings.TrimPrefix(header, "\ufeff"))
	}
	return out
}

// ParseSBIRows maps SBI cash and margin executions onto canonical fills.
func ParseSBIRows(rows []map[string]string, mapping map[string]string, sourceTZ string) ParseResult {
	if sourceTZ == "" {
		sourceTZ = "Asia/Tokyo"
	}
	fields := maps.Clone(sbiFields)
	for key, value := range mapping {
		if value = strings.TrimSpace(value); value != "" {
			fields[key] = value
		}
	}
	generic := NewGeneric(fields).WithSourceTZ(sourceTZ).WithDateOnlyTime(9, 0)
	occurrences := map[string]int{}
	result := ParseResult{Format: "executions"}

	for i, row := range rows {
		transaction := strings.TrimSpace(row[fields["side"]])
		legs := []string{}
		if transaction == "現引" {
			legs = []string{"信用返済売", "株式現物買"}
		} else if transaction == "現渡" {
			legs = []string{"株式現物売", "信用返済買"}
		} else if _, _, supported := sbiSemantics(transaction); supported {
			legs = []string{transaction}
		} else {
			result.Errors = append(result.Errors, RowError{Row: i + 1, Message: fmt.Sprintf("unsupported SBI transaction %q", transaction)})
			continue
		}
		key := strings.Join([]string{
			row[fields["executed_at"]], row[fields["symbol"]], transaction, row[fields["quantity"]],
			row[fields["price"]], row[fields["fees"]], row["市場"],
		}, "|")
		occurrences[key]++
		conversionID := ""
		if transaction == "現引" {
			conversionID = fmt.Sprintf("sbi|genbiki|%s|%d", key, occurrences[key])
		}

		for legIndex, leg := range legs {
			legRow := maps.Clone(row)
			legRow[fields["side"]] = leg
			if (transaction == "現引" || transaction == "現渡") && legIndex == 1 {
				legRow[fields["fees"]] = "--" // charge the event once, on its first leg
				if transaction == "現渡" && fields["commission"] != "" {
					legRow[fields["commission"]] = "--"
				}
			}
			parsed := generic.ParseRows([]map[string]string{legRow})
			if len(parsed.Errors) > 0 {
				parsed.Errors[0].Row = i + 1
				result.Errors = append(result.Errors, parsed.Errors[0])
				break
			}

			execution := parsed.Executions[0]
			execution.PositionType, execution.PositionEffect, _ = sbiSemantics(leg)
			if transaction == leg && execution.PositionEffect == PositionReduce {
				raw := strings.TrimSpace(strings.ReplaceAll(row["平均取得価額"], ",", ""))
				if raw != "" && raw != "--" && raw != "-" {
					basis, err := strconv.ParseFloat(raw, 64)
					if err != nil || math.IsNaN(basis) || math.IsInf(basis, 0) || basis < 0 {
						result.Errors = append(result.Errors, RowError{Row: i + 1, Message: "invalid reported acquisition basis"})
						break
					}
					execution.ReportedCloseBasis = &basis
				}
			}
			if transaction == "現渡" {
				execution.EventType = "position_settlement"
				execution.SettlementType = "genwatashi"
				execution.SettlementID = fmt.Sprintf("sbi|genwatashi|%s|%d", key, occurrences[key])
				if legIndex == 0 {
					invalid := false
					for field, target := range map[string]**float64{"受渡金額/決済損益": &execution.SettlementProceeds, "実現損益": &execution.ReportedRealizedPnl, "平均取得価額": &execution.ReportedCloseBasis} {
						raw := strings.TrimSpace(row[field])
						if raw == "" || raw == "--" || raw == "-" {
							continue
						}
						v, err := sbiPnLNumber(raw)
						if err != nil || (field != "実現損益" && v < 0) {
							result.Errors = append(result.Errors, RowError{Row: i + 1, Message: "invalid genwatashi " + field})
							invalid = true
							break
						}
						*target = &v
					}
					if invalid {
						break
					}
				}
			}
			if conversionID != "" {
				execution.EventType = "position_conversion"
				execution.ConversionType = "genbiki"
				execution.ConversionID = conversionID
			}
			if transaction == leg && execution.PositionEffect == PositionReduce && execution.PositionType != PositionCash {
				raw := strings.TrimSpace(strings.ReplaceAll(row["受渡金額/決済損益"], ",", ""))
				if raw != "" && raw != "--" && raw != "-" {
					pnl, err := strconv.ParseFloat(raw, 64)
					if err != nil || math.IsNaN(pnl) || math.IsInf(pnl, 0) {
						result.Errors = append(result.Errors, RowError{Row: i + 1, Message: "invalid margin settlement P&L"})
						break
					}
					execution.ReportedRealizedPnl = &pnl
				}
			}
			execution.LotKey = "sbi:" + strings.ReplaceAll(execution.PositionType, "_", "-")
			// ponytail: SBI exports dates but no times; tiny offsets preserve its row and transfer-leg order.
			execution.ExecutedAt = execution.ExecutedAt.Add(time.Duration(i*2+legIndex) * time.Microsecond)
			execution.DedupKey = fmt.Sprintf("sbi|%s|%d|%d", key, occurrences[key], legIndex)
			result.Executions = append(result.Executions, execution)
		}
	}
	return result
}

func sbiSemantics(transaction string) (positionType, effect string, supported bool) {
	switch transaction {
	case "株式現物買", "現物買":
		return PositionCash, PositionIncrease, true
	case "株式現物売", "現物売":
		return PositionCash, PositionReduce, true
	case "信用新規買":
		return PositionMarginLong, PositionIncrease, true
	case "信用返済売":
		return PositionMarginLong, PositionReduce, true
	case "信用新規売":
		return PositionMarginShort, PositionIncrease, true
	case "信用返済買":
		return PositionMarginShort, PositionReduce, true
	default:
		return "", "", false
	}
}
