package coach

// LegacyAvailable is the build's confidentiality boundary, not an operator toggle.
// Re-enablement requires a separate v2 implementation and D2-approved candidate.
func LegacyAvailable() bool { return false }
