package store

import "encoding/json"

// Capabilities is stored as JSON text so both database backends use the same schema.
func (a Account) MarshalJSON() ([]byte, error) {
	type account Account
	capabilities := []string{}
	if a.Capabilities != "" {
		if err := json.Unmarshal([]byte(a.Capabilities), &capabilities); err != nil {
			return nil, err
		}
	}
	return json.Marshal(struct {
		account
		Capabilities []string `json:"capabilities"`
	}{account(a), capabilities})
}
