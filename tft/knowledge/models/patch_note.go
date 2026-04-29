package models

// PatchNote 表示一次官方版本公告。
type PatchNote struct {
	Patch       string             `json:"patch"`
	Title       string             `json:"title"`
	Source      string             `json:"source"`
	SourceURL   string             `json:"source_url"`
	PublishedAt string             `json:"published_at"`
	FetchedAt   string             `json:"fetched_at,omitempty"`
	Sections    []PatchNoteSection `json:"sections"`
}

// PatchNoteSection 表示版本公告中的一个结构化章节。
type PatchNoteSection struct {
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	ImpactTags []string `json:"impact_tags,omitempty"`
	Details    []string `json:"details,omitempty"`
}
