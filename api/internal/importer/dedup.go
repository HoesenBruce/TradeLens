package importer

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

func DedupHash(symbol, side string, qty, price float64, at time.Time) string {
	return DedupHashOccurrence(symbol, side, qty, price, at, 0)
}

func dedupHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:16])
}

// DedupHashOccurrence preserves the legacy identity for the first fill.
func DedupHashOccurrence(symbol, side string, qty, price float64, at time.Time, occurrence int) string {
	raw := fmt.Sprintf("%s|%s|%.4f|%.6f|%d", symbol, side, qty, price, at.UTC().Unix())
	if occurrence > 0 {
		raw += "|#" + strconv.Itoa(occurrence)
	}
	return dedupHash(raw)
}

// DedupOccurrence reads the string-valued execution details used by all writers.
func DedupOccurrence(details map[string]string) int {
	n, err := strconv.Atoi(details["occ"])
	if err != nil || n < 0 {
		return 0
	}
	return n
}
