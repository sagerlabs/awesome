package models

// AliasesFile stores player slang and short names outside Go code.
type AliasesFile struct {
	Version string            `json:"version"`
	Source  string            `json:"source,omitempty"`
	Heroes  map[string]string `json:"heroes,omitempty"`
	Items   map[string]string `json:"items,omitempty"`
	Traits  map[string]string `json:"traits,omitempty"`
}
