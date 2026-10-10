package contracts

import (
	"regexp"
	"time"
)

// SchemaVersionV1 当前契约版本。
const SchemaVersionV1 = "1.0.0"

// ManifestFilename 数据清单在发布包内的固定文件名。
const ManifestFilename = "manifest.json"

// 文件 kind 与能力的映射。声明能力就必须有对应 kind 的文件；
// RequiredKinds 是任何合法发布包都必须包含的最小集合。
var kindCapabilities = map[string]string{
	"team_comps":        "base_facts",
	"localization":      "base_facts",
	"champion_profiles": "champion_profiles",
	"items_priority":   "items_priority",
	"patch_notes":       "patch_notes",
	"aliases":           "slang_nlu",
	"knowledge":         "knowledge",
}

// RequiredKinds 必需文件：缺任何一个都不能作为可发布数据包。
var RequiredKinds = []string{"team_comps", "localization"}

// KnownCapabilities 能力白名单。manifest.capabilities 只能从这里取值，
// 未知能力按 ErrCategoryCapability 拒绝（不静默忽略）。
var KnownCapabilities = map[string]bool{
	"base_facts":        true,
	"meta_stats":        true,
	"patch_notes":       true,
	"champion_profiles": true,
	"items_priority":    true,
	"slang_nlu":         true,
	"knowledge":         true,
}

var (
	sha256Pattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	manifestPathRx = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./-]*\.[A-Za-z0-9]+$`)
)

// FileRecord 描述发布包内的一个不可变数据负载文件。
//
// relative_path 必须是包内相对路径：禁止绝对路径、`..` 逃逸、
// 反斜杠（统一正斜杠）。manifest.json 本身不得出现在 files 中
//（哈希自引用禁止，见包文档）。
type FileRecord struct {
	RelativePath string `json:"relative_path" yaml:"relative_path"`
	SHA256       string `json:"sha256" yaml:"sha256"`
	Size         int64  `json:"size" yaml:"size"`
	Kind         string `json:"kind" yaml:"kind"`
	EntityCount  int    `json:"entity_count" yaml:"entity_count"`
}

// SourceRecord 描述一个上游数据来源及其统计口径。
//
// 语义（ADR013）：多来源分别记录；未知统计地区/段位保留 unknown
// 不能编造；未知补丁的统计不能作为 current 排名证据（该判断属于
// DATA-03 的运行时规则，这里保证口径字段本身合法）。
type SourceRecord struct {
	Provider         string    `json:"provider" yaml:"provider"`
	SourceDatasetID  string    `json:"source_dataset_id" yaml:"source_dataset_id"`
	SourceURL        string    `json:"source_url,omitempty" yaml:"source_url,omitempty"`
	SourceVersion    string    `json:"source_version,omitempty" yaml:"source_version,omitempty"`
	FetchedAt        time.Time `json:"fetched_at" yaml:"fetched_at"`
	RawSHA256        string    `json:"raw_sha256" yaml:"raw_sha256"`
	StatsRegion      string    `json:"stats_region" yaml:"stats_region"`
	StatsQueue       string    `json:"stats_queue" yaml:"stats_queue"`
	StatsWindowStart time.Time `json:"stats_window_start,omitempty" yaml:"stats_window_start,omitempty"`
	StatsWindowEnd   time.Time `json:"stats_window_end" yaml:"stats_window_end"`
	// SampleFloor 该来源进入排名/推荐的最小样本门槛（如 200）。
	// 低样本事实可以保留展示，但是否有排名资格由此门槛声明。
	SampleFloor int `json:"sample_floor" yaml:"sample_floor"`
}

// DataManifest 是一个数据发布包的清单：版本身份 + scope + 文件 + 来源。
//
// 注意：data_release_id 编码 scope 摘要与构建时间，但清单内容
//（本结构）不参与 FileRecord 哈希；built_at/builder_commit 是构建
// run 信息，与负载 content hash 分离（避免自引用，见包文档）。
type DataManifest struct {
	DataReleaseID     string         `json:"data_release_id" yaml:"data_release_id"`
	SchemaVersion     string         `json:"schema_version" yaml:"schema_version"`
	Scope             DataScope      `json:"scope" yaml:"scope"`
	ReaderSchemaRange string         `json:"reader_schema_range" yaml:"reader_schema_range"`
	Files             []FileRecord   `json:"files" yaml:"files"`
	Sources           []SourceRecord `json:"sources" yaml:"sources"`
	Capabilities      []string       `json:"capabilities" yaml:"capabilities"`
	BuilderCommit     string         `json:"builder_commit" yaml:"builder_commit"`
	BuiltAt           time.Time      `json:"built_at" yaml:"built_at"`
}

// Validate 校验清单结构与语义。校验内容 hash 与文件实际存在性属于
// DATA-02（严格验证器），这里保证清单自身的契约正确性。
func (m *DataManifest) Validate() error {
	if m == nil {
		return newErr(ErrCategoryFormat, "manifest", "", "manifest 为 nil")
	}
	if m.DataReleaseID == "" {
		return newErr(ErrCategoryIdentity, "data_release_id", "", "data_release_id 不能为空")
	}
	if m.SchemaVersion == "" {
		return newErr(ErrCategorySchema, "schema_version", "", "schema_version 不能为空")
	}
	if err := SupportsSchema(m.ReaderSchemaRange, m.SchemaVersion); err != nil {
		return newErr(ErrCategorySchema, "reader_schema_range", "",
			"reader_schema_range 非法：%v", err)
	}
	if err := m.Scope.Validate(); err != nil {
		return err
	}

	if len(m.Files) == 0 {
		return newErr(ErrCategoryFileMissing, "files", "", "files 不能为空")
	}
	if len(m.Sources) == 0 {
		return newErr(ErrCategoryIdentity, "sources", "", "sources 不能为空（至少声明一个上游来源）")
	}

	// 文件记录
	seenKind := map[string]bool{}
	seenPath := map[string]bool{}
	for i := range m.Files {
		f := &m.Files[i]
		if f.RelativePath == "" {
			return newErr(ErrCategoryFilePath, "relative_path", "", "files[%d] relative_path 为空", i)
		}
		if f.RelativePath == ManifestFilename || f.RelativePath == "./"+ManifestFilename {
			return newErr(ErrCategorySelfReference, "files", f.RelativePath,
				"manifest 不得出现在自身 files 列表（哈希自引用）")
		}
		if !manifestPathRx.MatchString(f.RelativePath) || containsEscape(f.RelativePath) {
			return newErr(ErrCategoryFilePath, "relative_path", f.RelativePath,
				"非法路径（要求包内相对路径，禁止绝对路径/..逃逸/反斜杠）")
		}
		if seenPath[f.RelativePath] {
			return newErr(ErrCategoryFilePath, "relative_path", f.RelativePath, "重复路径")
		}
		seenPath[f.RelativePath] = true

		if !sha256Pattern.MatchString(f.SHA256) {
			return newErr(ErrCategoryHash, "sha256", f.RelativePath, "sha256 必须为 64 位小写十六进制")
		}
		if f.Size < 0 {
			return newErr(ErrCategoryFormat, "size", f.RelativePath, "size 不能为负")
		}
		if f.EntityCount < 0 {
			return newErr(ErrCategoryFormat, "entity_count", f.RelativePath, "entity_count 不能为负")
		}
		if _, ok := kindCapabilities[f.Kind]; !ok {
			return newErr(ErrCategoryCapability, "kind", f.RelativePath, "未知文件 kind：%q", f.Kind)
		}
		seenKind[f.Kind] = true
	}

	// 必需文件
	for _, k := range RequiredKinds {
		if !seenKind[k] {
			return newErr(ErrCategoryFileMissing, "kind", "",
				"缺少必需文件 kind：%s", k)
		}
	}

	// 能力声明
	if len(m.Capabilities) == 0 {
		return newErr(ErrCategoryCapability, "capabilities", "", "capabilities 不能为空（至少 base_facts）")
	}
	for _, c := range m.Capabilities {
		if !KnownCapabilities[c] {
			return newErr(ErrCategoryCapability, "capabilities", "", "未知能力：%q", c)
		}
		if !hasCapabilityFile(seenKind, c) {
			return newErr(ErrCategoryCapability, "capabilities", "",
				"声明能力 %q 但缺少对应文件", c)
		}
	}

	// 来源记录与时间语义
	for i := range m.Sources {
		s := &m.Sources[i]
		if s.Provider == "" {
			return newErr(ErrCategoryIdentity, "provider", "", "sources[%d].provider 为空", i)
		}
		if s.SourceDatasetID == "" {
			return newErr(ErrCategoryIdentity, "source_dataset_id", "", "sources[%d].source_dataset_id 为空", i)
		}
		if s.FetchedAt.IsZero() {
			return newErr(ErrCategoryTimeOrder, "fetched_at", "", "sources[%d].fetched_at 为零值", i)
		}
		if !sha256Pattern.MatchString(s.RawSHA256) {
			return newErr(ErrCategoryHash, "raw_sha256", "", "sources[%d].raw_sha256 必须为 64 位小写十六进制", i)
		}
		if s.StatsWindowEnd.IsZero() {
			return newErr(ErrCategoryTimeOrder, "stats_window_end", "", "sources[%d].stats_window_end 为零值", i)
		}
		if !s.StatsWindowStart.IsZero() && s.StatsWindowStart.After(s.StatsWindowEnd) {
			return newErr(ErrCategoryTimeOrder, "stats_window_start", "",
				"sources[%d] 统计窗口 start > end", i)
		}
		if s.StatsWindowEnd.After(s.FetchedAt) {
			return newErr(ErrCategoryTimeOrder, "stats_window_end", "",
				"sources[%d] 统计窗口在抓取之后闭合（stats_window_end > fetched_at）", i)
		}
		if s.SampleFloor < 0 {
			return newErr(ErrCategoryFormat, "sample_floor", "", "sources[%d].sample_floor 不能为负", i)
		}
	}

	// built_at 语义
	if m.BuiltAt.IsZero() {
		return newErr(ErrCategoryTimeOrder, "built_at", "", "built_at 为零值")
	}
	for i := range m.Sources {
		if m.Sources[i].FetchedAt.After(m.BuiltAt) {
			return newErr(ErrCategoryTimeOrder, "built_at", "",
				"sources[%d].fetched_at 晚于 built_at（不能在构建之后抓取）", i)
		}
	}
	return nil
}

// hasCapabilityFile 判断 seenKind 中的文件是否支撑能力 c。
func hasCapabilityFile(seenKind map[string]bool, c string) bool {
	if c == "base_facts" || c == "meta_stats" {
		// base_facts/meta_stats 都由 team_comps+localization 支撑
		return seenKind["team_comps"] && seenKind["localization"]
	}
	for kind, cap := range kindCapabilities {
		if cap == c && seenKind[kind] {
			return true
		}
	}
	return false
}

// containsEscape 检查路径段级逃逸（".."、反斜杠）。
func containsEscape(p string) bool {
	if len(p) == 0 || p[0] == '/' {
		return true
	}
	for _, seg := range splitPath(p) {
		if seg == ".." || seg == "." || seg == "" {
			return true
		}
	}
	return false
}

func splitPath(p string) []string {
	var out []string
	cur := ""
	for _, r := range p {
		if r == '/' || r == '\\' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}
