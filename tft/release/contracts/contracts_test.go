package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 跨语言互通：testdata/ 由 Python 构建器（metadata/tft-meta/release_contracts.py）
// 生成/派生；本测试与 test_release_contracts.py 对同一批 fixture
// 断言相同的 pass/fail 与错误类别（DATA-01 验收要求）。

const (
	validDir   = "testdata/valid"
	invalidDir = "testdata/invalid"
)

// TestValidManifestLoadedByGo_PythonGenerated 验证 Python 生成的
// manifest 能被 Go 加载器完整读取并校验（Python→Go 正常样例互通）。
func TestValidManifestLoadedByGo_PythonGenerated(t *testing.T) {
	m, err := LoadManifest(filepath.Join(validDir, ManifestFilename))
	if err != nil {
		t.Fatalf("Python 生成的 manifest 应可被 Go 加载：%v", err)
	}
	if m.Scope.GamePatch != "18.1" || m.Scope.SetID != 18 || m.Scope.Region != "cn" {
		t.Errorf("scope 读取不符：%+v", m.Scope)
	}
	if len(m.Files) != 2 {
		t.Errorf("files 数量不符：%d", len(m.Files))
	}
	if m.ReaderSchemaRange != "^1.0" {
		t.Errorf("reader_schema_range = %q", m.ReaderSchemaRange)
	}
}

// TestValidManifestVerifyFiles 内容 hash 与大小校验（Python 算的
// sha256 必须与 Go 实算一致——生成器/加载器 hash 语义互通）。
func TestValidManifestVerifyFiles(t *testing.T) {
	m, err := LoadManifest(filepath.Join(validDir, ManifestFilename))
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if err := m.VerifyFiles(validDir); err != nil {
		t.Fatalf("VerifyFiles 失败：%v", err)
	}

	// 篡改一个字节后必须失败
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range m.Files {
		raw, _ := os.ReadFile(filepath.Join(validDir, filepath.FromSlash(f.RelativePath)))
		tampered := append(raw, ' ')
		if err := os.WriteFile(filepath.Join(tmp, filepath.FromSlash(f.RelativePath)), tampered, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.VerifyFiles(tmp); err == nil {
		t.Fatal("篡改内容后 VerifyFiles 应失败")
	}
}

// expectedCategories 与 Python 侧测试的期望一一对应（文件名 -> 错误类别）。
var expectedCategories = map[string]string{
	"bad_schema_version.json":  ErrCategorySchema,
	"bad_scope_region.json":    ErrCategoryScope,
	"empty_patch.json":         ErrCategoryScope,
	"bad_patch_format.json":    ErrCategoryScope,
	"path_escape.json":         ErrCategoryFilePath,
	"self_reference.json":      ErrCategorySelfReference,
	"bad_hash.json":            ErrCategoryHash,
	"missing_required_kind.json": ErrCategoryFileMissing,
	"bad_time_order.json":      ErrCategoryTimeOrder,
	"bad_reader_range.json":    ErrCategorySchema,
	"unknown_capability.json":  ErrCategoryCapability,
	"window_after_fetch.json":   ErrCategoryTimeOrder,
}

// TestInvalidManifestsRejected 每个 invalid fixture 必须被拒绝，
// 且错误类别与期望一致（未知/不兼容 schema 和非法 scope 拒绝）。
func TestInvalidManifestsRejected(t *testing.T) {
	entries, err := os.ReadDir(invalidDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(expectedCategories) {
		t.Fatalf("invalid fixture 数量不符：got %d want %d（新增用例须同步 expectedCategories 与 Python 测试）", len(entries), len(expectedCategories))
	}
	for _, e := range entries {
		name := e.Name()
		want, ok := expectedCategories[name]
		if !ok {
			t.Errorf("%s 缺少期望类别（补 expectedCategories）", name)
			continue
		}
		_, err := LoadManifest(filepath.Join(invalidDir, name))
		if err == nil {
			t.Errorf("%s 应被拒绝", name)
			continue
		}
		ce, ok := err.(*ContractError)
		if !ok {
			t.Errorf("%s 错误类型应为 *ContractError，got %T", name, err)
			continue
		}
		if ce.Category != want {
			t.Errorf("%s 错误类别 = %q, want %q（reason=%s）", name, ce.Category, want, ce.Reason)
		}
	}
}

// TestValidDeploymentSessionRouteEvidence 其余协议的合法样例。
func TestValidDeploymentSessionRouteEvidence(t *testing.T) {
	load := func(name string, v interface{}) {
		raw, err := os.ReadFile(filepath.Join(validDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, v); err != nil {
			t.Fatalf("%s 解析失败：%v", name, err)
		}
	}

	var d DeploymentManifest
	load("deployment.json", &d)
	if err := d.Validate(); err != nil {
		t.Errorf("deployment 校验失败：%v", err)
	}

	var s SessionRecord
	load("session_record.json", &s)
	if err := s.Validate(); err != nil {
		t.Errorf("session 校验失败：%v", err)
	}

	var r RouteDecision
	load("route_decision.json", &r)
	if err := r.Validate(); err != nil {
		t.Errorf("route_decision 校验失败：%v", err)
	}

	var e EvidenceEnvelope
	load("evidence.json", &e)
	if err := e.Validate(); err != nil {
		t.Errorf("evidence 校验失败：%v", err)
	}

	// 部署与数据组合兼容检查
	m, err := LoadManifest(filepath.Join(validDir, ManifestFilename))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.ComposeWith(m); err != nil {
		t.Errorf("ComposeWith 失败：%v", err)
	}
	// 数据 release 不匹配的组合必须拒绝
	d2 := d
	d2.DataReleaseID = "tft-s18-18.1-cn-99999999T000000Z-deadbeef"
	if err := d2.ComposeWith(m); err == nil {
		t.Error("release 不匹配的组合应被拒绝")
	}
}

// TestSupportsSchema 版本兼容矩阵。
func TestSupportsSchema(t *testing.T) {
	cases := []struct {
		rng, ver string
		ok       bool
	}{
		{"^1.0", "1.0.0", true},
		{"^1.0", "1.9.3", true},
		{"^1.0", "2.0.0", false},
		{"^1.0", "0.9.0", false},
		{"^1", "1.4.2", true},
		{">=1.0,<1.5", "1.4.99", true},
		{">=1.0,<1.5", "1.5.0", false},
		{">=1.2,<2.0", "1.1.9", false},
		{"latest", "1.0.0", false},  // 语法非法
		{"*", "1.0.0", false},        // 通配符不允许
		{"^1.0", "", false},          // 空版本
	}
	for _, c := range cases {
		err := SupportsSchema(c.rng, c.ver)
		if (err == nil) != c.ok {
			t.Errorf("SupportsSchema(%q, %q) ok=%v, want %v (err=%v)", c.rng, c.ver, err == nil, c.ok, err)
		}
	}
}

// TestScopeEdgeCases scope 边界：hotfix、unknown region、
// 非法 locale、零 effective_at。
func TestScopeEdgeCases(t *testing.T) {
	valid := DataScope{
		Game: "tft", SetID: 18, GamePatch: "18.1a", HotfixRevision: "a",
		Region: "unknown", Queue: "unknown", Locale: "zh-CN",
		EffectiveAt: mustTime("2026-09-10T00:00:00Z"),
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("合法 scope 被拒绝：%v", err)
	}

	bad := []func(*DataScope){
		func(s *DataScope) { s.Game = "lol" },
		func(s *DataScope) { s.SetID = 0 },
		func(s *DataScope) { s.GamePatch = "" },
		func(s *DataScope) { s.GamePatch = "S18 18.1" },
		func(s *DataScope) { s.HotfixRevision = "12" },
		func(s *DataScope) { s.Region = "mars" },
		func(s *DataScope) { s.Queue = "aram" },
		func(s *DataScope) { s.Locale = "中文" },
		func(s *DataScope) { s.EffectiveAt = mustTime("0001-01-01T00:00:00Z") },
	}
	for i, mutate := range bad {
		s := valid
		mutate(&s)
		if err := s.Validate(); err == nil {
			t.Errorf("case %d：非法 scope 应被拒绝", i)
		} else if ce, ok := err.(*ContractError); !ok || ce.Category != ErrCategoryScope {
			t.Errorf("case %d：错误类别应为 scope，got %v", i, err)
		}
	}
}

// TestSamePatchInRegion 同补丁同地区判断（灰度分桶前置）。
func TestSamePatchInRegion(t *testing.T) {
	a := &DataScope{Game: "tft", SetID: 18, GamePatch: "18.1", Region: "cn"}
	b := &DataScope{Game: "tft", SetID: 18, GamePatch: "18.1", Region: "cn"}
	c := &DataScope{Game: "tft", SetID: 18, GamePatch: "18.1", HotfixRevision: "a", Region: "cn"}
	d := &DataScope{Game: "tft", SetID: 18, GamePatch: "18.1", Region: "na"}
	if !a.SamePatchInRegion(b) {
		t.Error("同补丁同地区应匹配")
	}
	if a.SamePatchInRegion(c) {
		t.Error("热修不同应视为不同补丁")
	}
	if a.SamePatchInRegion(d) {
		t.Error("不同地区不应匹配")
	}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
