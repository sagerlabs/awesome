package models

// ChampionProfilesFile 是英雄画像配置文件的顶层结构。
// 它只存放费用这类知识库统计文件缺失、但查询时需要的稳定元信息。
type ChampionProfilesFile struct {
	Version     string                      `json:"version"`
	Source      string                      `json:"source"`
	GeneratedAt string                      `json:"generated_at,omitempty"`
	Champions   map[string]*ChampionProfile `json:"champions"`
}

// ChampionProfile 描述单个英雄的轻量画像。
type ChampionProfile struct {
	Name    string   `json:"name,omitempty"`
	APIName string   `json:"api_name,omitempty"`
	Cost    int      `json:"cost,omitempty"`
	Traits  []string `json:"traits,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}
