package agent

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

type fastAliasFile struct {
	Heroes map[string]string `json:"heroes"`
	Items  map[string]string `json:"items"`
	Traits map[string]string `json:"traits"`
}

// FastNLUExtractor handles high-confidence common TFT queries without an LLM call.
type FastNLUExtractor struct {
	champions []string
	items     []string
	traits    []string
}

// NewFastNLUExtractor builds a small dictionary from knowledge metadata.
func NewFastNLUExtractor(adapter *KnowledgeAdapter) *FastNLUExtractor {
	extractor := &FastNLUExtractor{}
	if adapter == nil {
		extractor.addDefaultAliases()
		extractor.prepare()
		return extractor
	}

	if champions, err := adapter.GetAllMetaChampions(); err == nil {
		for _, champion := range champions {
			extractor.addChampion(champion.Name)
		}
	}
	if items, err := adapter.GetAllMetaItems(); err == nil {
		for _, item := range items {
			extractor.addItem(item.Name)
		}
	}
	if comps, err := adapter.GetAllMetaComps(); err == nil {
		for _, comp := range comps {
			for _, trait := range comp.Traits {
				extractor.addTrait(normalizeFastTraitName(trait))
			}
		}
	}

	extractor.addAliasesFromFile("tft/knowledge/data/aliases.json")
	extractor.addDefaultAliases()
	extractor.prepare()
	return extractor
}

func (e *FastNLUExtractor) TryParse(raw string) (Context, bool) {
	input := strings.TrimSpace(raw)
	if input == "" {
		return Context{}, false
	}

	ctx := Context{
		Champions: make(map[string]int8),
	}

	champions := e.matchTerms(input, e.champions)
	for _, champion := range champions {
		ctx.Champions[champion] = 1
	}
	ctx.Items = e.matchTerms(input, e.items)
	ctx.Traits = e.matchTerms(input, e.traits)

	if cost, ok := parseFastUnitCost(input); ok {
		ctx.UnitCost = &cost
	}
	if stage, ok := parseFastStage(input); ok {
		ctx.GameStage = &stage
	}
	ctx.RoleQuery = parseFastRole(input)
	ctx.Playstyle = parseFastPlaystyle(input)

	ctx.Intent = inferFastIntent(input, ctx)
	if ctx.Intent == "" {
		return Context{}, false
	}

	if len(ctx.Champions) == 0 {
		ctx.Champions = nil
	}
	return ctx, true
}

func (e *FastNLUExtractor) addDefaultAliases() {
	for _, champion := range []string{"剑魔", "炸弹人", "女枪", "龙王", "铁男", "卡牌"} {
		e.addChampion(champion)
	}
	for _, item := range []string{"羊刀", "青龙刀", "法爆", "帽子", "反甲", "板甲", "日炎", "科技枪", "饮血", "泰坦", "无尽", "轻语"} {
		e.addItem(item)
	}
	for _, trait := range []string{"机甲", "未来"} {
		e.addTrait(trait)
	}
}

func (e *FastNLUExtractor) addAliasesFromFile(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var file fastAliasFile
	if err := json.Unmarshal(b, &file); err != nil {
		return
	}
	for raw := range file.Heroes {
		e.addChampion(raw)
	}
	for raw := range file.Items {
		e.addItem(raw)
	}
	for raw := range file.Traits {
		e.addTrait(raw)
	}
}

func (e *FastNLUExtractor) addChampion(name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		e.champions = append(e.champions, name)
	}
}

func (e *FastNLUExtractor) addItem(name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		e.items = append(e.items, name)
	}
}

func (e *FastNLUExtractor) addTrait(name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		e.traits = append(e.traits, name)
	}
}

func (e *FastNLUExtractor) prepare() {
	e.champions = uniqueFastTerms(e.champions)
	e.items = uniqueFastTerms(e.items)
	e.traits = uniqueFastTerms(e.traits)
}

func (e *FastNLUExtractor) matchTerms(input string, terms []string) []string {
	var matches []string
	for _, term := range terms {
		if strings.Contains(input, term) {
			matches = append(matches, term)
		}
	}
	return matches
}

func uniqueFastTerms(terms []string) []string {
	seen := make(map[string]struct{}, len(terms))
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	sort.Slice(out, func(i, j int) bool {
		if len([]rune(out[i])) != len([]rune(out[j])) {
			return len([]rune(out[i])) > len([]rune(out[j]))
		}
		return out[i] < out[j]
	})
	return out
}

func inferFastIntent(input string, ctx Context) string {
	hasChampion := len(ctx.Champions) > 0
	hasItem := len(ctx.Items) > 0
	hasTrait := len(ctx.Traits) > 0

	if hasItem && containsAny(input, "给谁", "适合谁", "适合哪个", "怎么出装", "带什么", "给什么", "装备") {
		return "item_query"
	}
	if hasTrait && !hasChampion && !hasItem {
		return "trait_query"
	}
	if hasChampion {
		return "champion_query"
	}
	if ctx.UnitCost != nil || ctx.RoleQuery != "" {
		return "vertical_query"
	}
	if ctx.Playstyle != "" {
		return "playstyle_query"
	}
	if containsAny(input, "最强阵容", "当前版本", "能玩什么", "玩什么", "阵容推荐") {
		return "lineup_recommend"
	}
	return ""
}

func parseFastRole(input string) string {
	switch {
	case containsAny(input, "打工", "过渡", "前期", "二阶段"):
		return "work"
	case containsAny(input, "能抗", "前排", "坦克", "肉"):
		return "tank"
	case containsAny(input, "主C", "主c", "能C", "能c", "输出"):
		return "carry"
	case containsAny(input, "谁最强", "最厉害"):
		return "all"
	default:
		return ""
	}
}

func parseFastPlaystyle(input string) string {
	switch {
	case containsAny(input, "九五", "95", "高费"):
		return "高费阵容"
	case containsAny(input, "赌狗", "追三", "追3"):
		return "低费追三星"
	case containsAny(input, "连胜"):
		return "连胜运营"
	case containsAny(input, "连败"):
		return "连败运营"
	default:
		return ""
	}
}

func parseFastUnitCost(input string) (int, bool) {
	costWords := map[string]int{
		"一费": 1, "1费": 1,
		"二费": 2, "2费": 2,
		"三费": 3, "3费": 3,
		"四费": 4, "4费": 4,
		"五费": 5, "5费": 5,
		"七费": 7, "7费": 7,
	}
	for word, cost := range costWords {
		if strings.Contains(input, word) {
			return cost, true
		}
	}
	return 0, false
}

func parseFastStage(input string) (string, bool) {
	switch {
	case strings.Contains(input, "二阶段"):
		return "2阶段", true
	case strings.Contains(input, "三阶段"):
		return "3阶段", true
	case strings.Contains(input, "四阶段"):
		return "4阶段", true
	default:
		return "", false
	}
}

func normalizeFastTraitName(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.Index(name, "("); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	if idx := strings.Index(name, "（"); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	return name
}

func containsAny(input string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(input, needle) {
			return true
		}
	}
	return false
}
