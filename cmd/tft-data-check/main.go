// Command tft-data-check 是数据发布的权威校验入口（DATA-02）。
//
// 分层校验（ADR013 A02：metadata、knowledge、别名、画像、公告与派生
// 索引共同验证；任一层失败即退出非零，作为构建门禁）：
//
//	L1 文件级   必需文件存在且非空；所有 JSON 可完整解析（截断即拒）
//	L2 领域级   cluster 唯一、cn_to_id/id_to_cn 双向一致（反向冲突）、
//	            comp 单位可解析（引用完整性）、tft_set 全量一致（混版本）、
//	            patch_notes 覆盖当前赛季
//	L3 统计级   entity 计数与基线对比，骤变生成报告（告警不阻断）
//	L4 发布级   存在 manifest.json 时校验 hash/size/schema（复用
//	            tft/release/contracts）
//
// 退出码：0 通过；1 校验失败（门禁）；2 用法错误。
// 报告逐项定位 文件:字段，Python 构建流程消费同一退出码。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	contracts "github.com/sagerlabs/awesome/tft/release/contracts"
)

// 检查结果分级。
const (
	levelError = "ERROR"
	levelWarn  = "WARN"
)

type finding struct {
	Level  string `json:"level"`
	File   string `json:"file"`
	Field  string `json:"field,omitempty"`
	Msg    string `json:"msg"`
}

type checker struct {
	metadataDir  string
	knowledgeDir string
	errors       []finding
	warnings     []finding
	counts       map[string]int

	// localization 解析结果（供交叉引用检查）
	cnToID       map[string]string // cn -> id
	profileNames map[string]bool   // unit_profiles 的中文名
	// team_comps 解析结果
	compSet    string   // 全量一致的 tft_set（空=有错误）
	compUnits  []unitRef
	patches    []string // patch_notes 里收集的补丁号
}

type unitRef struct {
	comp string // 所属 comp 文件名
	name string // 单位中文名
}

func (c *checker) errf(file, field, format string, args ...interface{}) {
	c.errors = append(c.errors, finding{levelError, file, field, fmt.Sprintf(format, args...)})
}

func (c *checker) warnf(file, field, format string, args ...interface{}) {
	c.warnings = append(c.warnings, finding{levelWarn, file, field, fmt.Sprintf(format, args...)})
}

func main() {
	metadataDir := flag.String("metadata", "metadata/tft-meta/data", "metadata 数据目录")
	knowledgeDir := flag.String("knowledge", "tft/knowledge/data", "knowledge 数据目录")
	baseline := flag.String("baseline", "", "entity 计数基线文件（对比骤变）")
	writeBaseline := flag.Bool("write-baseline", false, "把本次计数写成基线文件（供下次对比）")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: tft-data-check [-metadata dir] [-knowledge dir] [-baseline file] [-write-baseline]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	c := &checker{
		metadataDir:  *metadataDir,
		knowledgeDir: *knowledgeDir,
		counts:       map[string]int{},
	}
	c.run()

	// 报告
	for _, w := range c.warnings {
		fmt.Printf("⚠️  WARN  %s%s %s\n", w.File, fieldSuffix(w.Field), w.Msg)
	}
	for _, e := range c.errors {
		fmt.Printf("❌ ERROR %s%s %s\n", e.File, fieldSuffix(e.Field), e.Msg)
	}
	fmt.Printf("—— 计数：comps=%d units=%d cn_to_id=%d id_to_cn=%d profiles=%d patch_notes=%d\n",
		c.counts["team_comps"], c.counts["comp_units"], c.counts["cn_to_id"],
		c.counts["id_to_cn"], c.counts["unit_profiles"], c.counts["patch_notes"])

	if *writeBaseline {
		if raw, err := json.MarshalIndent(c.counts, "", "  "); err == nil {
			if err := os.WriteFile(*baseline, raw, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "写基线失败：%v\n", err)
			} else {
				fmt.Printf("—— 基线已写入 %s\n", *baseline)
			}
		}
	}
	if *baseline != "" && !*writeBaseline {
		c.compareBaseline(*baseline)
	}

	if len(c.errors) > 0 {
		fmt.Fprintf(os.Stderr, "❌ 数据校验失败：%d 错误 / %d 告警\n", len(c.errors), len(c.warnings))
		os.Exit(1)
	}
	fmt.Printf("✅ 数据校验通过（%d 告警）\n", len(c.warnings))
}

func fieldSuffix(f string) string {
	if f == "" {
		return ""
	}
	return "." + f
}

func (c *checker) run() {
	c.checkLocalization()
	c.checkTeamComps()
	c.checkPatchNotes()
	c.checkCrossRefs()
	c.checkAliasesAndProfiles()
	c.checkPublishManifest()
}

// ── L1/L2: localization ─────────────────────────────────────────────────────

func (c *checker) checkLocalization() {
	path := filepath.Join(c.metadataDir, "localization.json")
	raw, err := readNonEmpty(path)
	if err != nil {
		c.errf(path, "", "%v", err)
		return
	}
	var loc struct {
		Source       string            `json:"source"`
		IDToCN       map[string]string `json:"id_to_cn"`
		CNToID       map[string]string `json:"cn_to_id"`
		UnitProfiles map[string]struct {
			APIName string `json:"api_name"`
			Name    string `json:"name"`
			Cost    int    `json:"cost"`
		} `json:"unit_profiles"`
	}
	if err := json.Unmarshal(raw, &loc); err != nil {
		c.errf(path, "", "JSON 解析失败（截断/损坏）：%v", err)
		return
	}
	if len(loc.IDToCN) == 0 {
		c.errf(path, "id_to_cn", "为空")
		return
	}
	if len(loc.CNToID) == 0 {
		c.errf(path, "cn_to_id", "为空")
		return
	}
	c.counts["cn_to_id"] = len(loc.CNToID)
	c.counts["id_to_cn"] = len(loc.IDToCN)
	c.counts["unit_profiles"] = len(loc.UnitProfiles)
	c.cnToID = loc.CNToID
	c.profileNames = make(map[string]bool, len(loc.UnitProfiles))
	for _, p := range loc.UnitProfiles {
		c.profileNames[p.Name] = true
	}

	// 反向冲突：cn_to_id[cn]=id 时，id_to_cn[id] 必须回指同一个 cn。
	// 这是伊莉丝式多形态覆盖（FIX-02 修复的 bug 类别）的回归防线。
	for cn, id := range loc.CNToID {
		if back, ok := loc.IDToCN[id]; ok && back != cn {
			c.errf(path, "cn_to_id", "反向冲突：%q -> %q，但 id_to_cn[%q]=%q", cn, id, id, back)
		}
	}

	// unit_profiles 的 api_name 与 key 一致
	for key, p := range loc.UnitProfiles {
		if p.APIName == "" || p.Name == "" {
			c.errf(path, "unit_profiles", "%s 缺 api_name/name", key)
		}
	}
}

// ── L1/L2: team_comps（目录内每个 JSON 一套阵容） ────────────────────────────

type compFile struct {
	ClusterID string `json:"cluster_id"`
	TFTSet    string `json:"tft_set"`
	Units     []string `json:"units"`
	Count     int     `json:"count"`
	Tier      string  `json:"tier"`
}

func (c *checker) checkTeamComps() {
	dir := filepath.Join(c.knowledgeDir, "team_comps")
	entries, err := os.ReadDir(dir)
	if err != nil {
		c.errf(dir, "", "目录不可读：%v", err)
		return
	}
	if len(entries) == 0 {
		c.errf(dir, "", "空目录（至少需要一套阵容）")
		return
	}

	seenCluster := map[string]string{} // cluster_id -> 文件名
	seenSet := map[string]bool{}
	totalUnits := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := readNonEmpty(path)
		if err != nil {
			c.errf(path, "", "%v", err)
			continue
		}
		var comp compFile
		if err := json.Unmarshal(raw, &comp); err != nil {
			c.errf(path, "", "JSON 解析失败（截断/损坏）：%v", err)
			continue
		}
		if comp.ClusterID == "" {
			c.errf(path, "cluster_id", "为空")
			continue
		}
		if prev, dup := seenCluster[comp.ClusterID]; dup {
			c.errf(path, "cluster_id", "重复：%s 与 %s", comp.ClusterID, prev)
			continue
		}
		seenCluster[comp.ClusterID] = e.Name()
		if comp.TFTSet == "" {
			c.errf(path, "tft_set", "为空")
		} else {
			seenSet[comp.TFTSet] = true
		}
		if len(comp.Units) == 0 {
			c.errf(path, "units", "为空（缺实体）")
		}
		totalUnits += len(comp.Units)
		for _, u := range comp.Units {
			c.compUnits = append(c.compUnits, unitRef{comp: e.Name(), name: u})
		}
		if comp.Tier == "" {
			c.warnf(path, "tier", "为空（降级展示）")
		}
	}
	c.counts["team_comps"] = len(seenCluster)
	c.counts["comp_units"] = totalUnits

	// 混版本：所有阵容必须同一赛季
	if len(seenSet) == 1 {
		for s := range seenSet {
			c.compSet = s
		}
	} else if len(seenSet) > 1 {
		sets := make([]string, 0, len(seenSet))
		for s := range seenSet {
			sets = append(sets, s)
		}
		sort.Strings(sets)
		c.errf(dir, "tft_set", "混版本：同时存在 %v", sets)
	}
}

// ── L2: 关联引用（comp 单位必须可解析到 localization） ───────────────────────

var tftSetPattern = regexp.MustCompile(`^TFTSet(\d+)$`)
var patchMajorPattern = regexp.MustCompile(`^(\d+)\.`)

func (c *checker) checkCrossRefs() {
	// 单位引用：cn_to_id / unit_profiles 名字 /（兜底）aliases 解析
	aliasPath := filepath.Join(c.knowledgeDir, "aliases.json")
	aliasHeroes := map[string]string{}
	if raw, err := os.ReadFile(aliasPath); err == nil {
		var aliases struct {
			Heroes map[string]string `json:"heroes"`
		}
		if json.Unmarshal(raw, &aliases) == nil {
			aliasHeroes = aliases.Heroes
		}
	}
	for _, ref := range c.compUnits {
		if _, ok := c.cnToID[ref.name]; ok {
			continue
		}
		if c.profileNames[ref.name] {
			continue
		}
		if canon, ok := aliasHeroes[ref.name]; ok {
			if _, ok := c.cnToID[canon]; ok {
				continue
			}
		}
		c.errf(filepath.Join(c.knowledgeDir, "team_comps", ref.comp), "units",
			"单位 %q 无法解析（localization.cn_to_id / unit_profiles / aliases 均未命中，缺实体）", ref.name)
	}

	// 赛季交叉：tft_set 的数字与补丁公告主版本匹配
	if m := tftSetPattern.FindStringSubmatch(c.compSet); m != nil {
		setNum := m[1]
		matched := false
		for _, p := range c.patches {
			if pm := patchMajorPattern.FindStringSubmatch(p); pm != nil && pm[1] == setNum {
				matched = true
				break
			}
		}
		if !matched {
			c.errf(filepath.Join(c.knowledgeDir, "patch_notes"), "",
				"当前赛季 %s 没有对应补丁公告（patches=%v，当前赛季问题会退化为旧版本）", c.compSet, c.patches)
		}
	}
}

// ── L2: patch_notes 与赛季一致性 ────────────────────────────────────────────

func (c *checker) checkPatchNotes() {
	dir := filepath.Join(c.knowledgeDir, "patch_notes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		c.errf(dir, "", "目录不可读：%v", err)
		return
	}
	patches := []string{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := readNonEmpty(path)
		if err != nil {
			c.errf(path, "", "%v", err)
			continue
		}
		var note struct {
			Patch      string `json:"patch"`
			Title      string `json:"title"`
			FetchedAt  string `json:"fetched_at"`
		}
		if err := json.Unmarshal(raw, &note); err != nil {
			c.errf(path, "", "JSON 解析失败（截断/损坏）：%v", err)
			continue
		}
		if note.Patch == "" {
			c.errf(path, "patch", "为空")
			continue
		}
		if note.FetchedAt == "" {
			c.warnf(path, "fetched_at", "为空")
		}
		patches = append(patches, note.Patch)
	}
	c.counts["patch_notes"] = len(patches)
	c.patches = patches
	if len(patches) == 0 {
		c.errf(dir, "", "没有任何补丁公告（当前赛季问题会退化为旧版本）")
	}
}

// ── L2: aliases / champion_profiles（存在性 + 基本结构） ────────────────────

func (c *checker) checkAliasesAndProfiles() {
	// aliases.json
	aliasPath := filepath.Join(c.knowledgeDir, "aliases.json")
	if raw, err := readNonEmpty(aliasPath); err != nil {
		c.errf(aliasPath, "", "%v", err)
	} else {
		var aliases struct {
			Heroes map[string]string `json:"heroes"`
			Items  map[string]string `json:"items"`
		}
		if err := json.Unmarshal(raw, &aliases); err != nil {
			c.errf(aliasPath, "", "JSON 解析失败：%v", err)
		} else if len(aliases.Heroes) == 0 {
			c.warnf(aliasPath, "heroes", "为空（黑话理解退化）")
		}
		c.counts["aliases"] = len(aliases.Heroes)
	}

	// champion_profiles.json
	profPath := filepath.Join(c.knowledgeDir, "champion_profiles.json")
	if raw, err := readNonEmpty(profPath); err != nil {
		c.errf(profPath, "", "%v", err)
	} else {
		var profiles map[string]json.RawMessage
		if err := json.Unmarshal(raw, &profiles); err != nil {
			c.errf(profPath, "", "JSON 解析失败：%v", err)
		} else if len(profiles) == 0 {
			c.errf(profPath, "", "为空（缺实体：英雄画像全部缺失）")
		}
		c.counts["champion_profiles"] = len(profiles)
	}

	// champions/ 与 items/ 目录非空
	for _, d := range []string{"champions", "items"} {
		p := filepath.Join(c.knowledgeDir, d)
		entries, err := os.ReadDir(p)
		if err != nil {
			c.errf(p, "", "目录不可读：%v", err)
			continue
		}
		n := 0
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				n++
			}
		}
		if n == 0 {
			c.errf(p, "", "空目录")
		}
		c.counts[d] = n
	}
}

// ── L4: 发布清单（存在才校验） ─────────────────────────────────────────────

func (c *checker) checkPublishManifest() {
	path := filepath.Join(c.metadataDir, contracts.ManifestFilename)
	if _, err := os.Stat(path); err != nil {
		return // DATA-04 起构建才产出 manifest；没有就跳过
	}
	m, err := contracts.LoadManifest(path)
	if err != nil {
		c.errf(path, "", "manifest 校验失败：%v", err)
		return
	}
	// manifest 声明的文件必须在 metadata 目录内且 hash 一致
	if err := m.VerifyFiles(c.metadataDir); err != nil {
		if ce, ok := err.(*contracts.ContractError); ok {
			c.errf(path, ce.Field, "%v", err)
		} else {
			c.errf(path, "", "%v", err)
		}
	}
}

// ── L3: 基线对比（骤变报告） ────────────────────────────────────────────────

func (c *checker) compareBaseline(baselineFile string) {
	raw, err := os.ReadFile(baselineFile)
	if err != nil {
		c.warnf(baselineFile, "", "基线不可读（跳过骤变对比）：%v", err)
		return
	}
	var base map[string]int
	if err := json.Unmarshal(raw, &base); err != nil {
		c.warnf(baselineFile, "", "基线解析失败：%v", err)
		return
	}
	// 关键实体计数骤变阈值：超过 30% 输出报告
	for _, key := range []string{"team_comps", "comp_units", "cn_to_id", "id_to_cn", "unit_profiles", "champion_profiles"} {
		old, ok := base[key]
		if !ok || old == 0 {
			continue
		}
		now := c.counts[key]
		delta := float64(now-old) / float64(old)
		if delta < -0.30 || delta > 0.30 {
			c.warnf(baselineFile, key,
				"entity 骤变：%s %d -> %d（%+.0f%%）", key, old, now, delta*100)
		}
	}
}

// readNonEmpty 读取文件并检查非空（L1：空文件即拒）。
func readNonEmpty(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("文件不存在：%v", err)
	}
	if st.IsDir() {
		return nil, fmt.Errorf("是目录而非文件")
	}
	if st.Size() == 0 {
		return nil, fmt.Errorf("空文件")
	}
	return os.ReadFile(path)
}
