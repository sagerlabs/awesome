package knowledge

import (
	"sort"
	"strings"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

// buildPatchNoteInsights 构建与查询相关的公告信息（DATA-03 重写）。
//
// 模式选择：
//   - current 模式（默认，PatchHint 为空）：只检索当前生效补丁的公告，
//     不回退旧公告——旧赛季/旧补丁内容不再混入当前回答；
//   - 历史模式（PatchHint 非空且命中公告）：只检索该补丁的公告，
//     IsHistorical=true，调用方应向用户说明这是历史版本内容。
//
// 当前补丁 = 公告集合中版本号最大者（语义化比较，见 CurrentPatch）。
// 如果当前公告没有相关章节，宁可返回空也不捞旧公告
//（ADR013 A08：不用旧公告事实冒充当前版本信息）。
func (s *UnifiedStore) buildPatchNoteInsights(ctx contracts.QueryNLURequest) []contracts.PatchNoteInsight {
	if s.knowledgeStore == nil {
		return nil
	}

	currentPatch := s.knowledgeStore.CurrentPatch()
	if currentPatch == "" {
		return nil
	}

	// 模式判定：历史模式要求 PatchHint 精确命中一份已加载公告
	targetPatch := currentPatch
	if hint := strings.TrimSpace(ctx.PatchHint); hint != "" && hint != currentPatch {
		if _, ok := s.knowledgeStore.patchNotes[hint]; ok {
			targetPatch = hint
		}
		// hint 不命中任何公告时保持 current 模式（不猜、不模糊匹配）
	}

	note, ok := s.knowledgeStore.patchNotes[targetPatch]
	if !ok {
		return nil
	}
	isCurrent := targetPatch == currentPatch

	queries := patchNoteQueries(ctx)
	preferredTags := patchNotePreferredTags(ctx)
	var insights []contracts.PatchNoteInsight
	for _, section := range note.Sections {
		if !patchNoteSectionRelevant(section, queries, preferredTags) {
			continue
		}
		insights = append(insights, patchNoteSectionToInsight(note, section, isCurrent))
		if len(insights) >= 5 {
			break
		}
	}
	return insights
}

func patchNoteQueries(ctx contracts.QueryNLURequest) []string {
	var queries []string
	for champion := range ctx.Champions {
		queries = append(queries, champion)
	}
	queries = append(queries, ctx.Items...)
	queries = append(queries, ctx.Traits...)
	if ctx.ExplicitLineup != nil {
		queries = append(queries, *ctx.ExplicitLineup)
	}
	if ctx.Playstyle != "" {
		queries = append(queries, ctx.Playstyle)
	}
	return compactStrings(queries)
}

func patchNotePreferredTags(ctx contracts.QueryNLURequest) map[string]struct{} {
	tags := make(map[string]struct{})
	switch ctx.Intent {
	case "item_query":
		tags["itemization"] = struct{}{}
		tags["frontline"] = struct{}{}
	case "trait_query":
		tags["mechanic"] = struct{}{}
	case "augment_query":
		tags["augment"] = struct{}{}
	case "vertical_query", "champion_query":
		tags["combat"] = struct{}{}
		tags["shop_odds"] = struct{}{}
	case "playstyle_query", "lineup_recommend", "":
		// 版本类泛问题（intent 为空）也命中 general/更新总览章节
		tags["mechanic"] = struct{}{}
		tags["shop_odds"] = struct{}{}
		tags["loot"] = struct{}{}
		tags["itemization"] = struct{}{}
		tags["general"] = struct{}{}
	}
	return tags
}

func patchNoteSectionRelevant(section models.PatchNoteSection, queries []string, preferredTags map[string]struct{}) bool {
	haystack := section.Title + "\n" + section.Summary + "\n" + strings.Join(section.Details, "\n")
	for _, query := range queries {
		if query != "" && strings.Contains(haystack, query) {
			return true
		}
	}
	for _, tag := range section.ImpactTags {
		if _, ok := preferredTags[tag]; ok {
			return true
		}
	}
	// 版本总览类章节（无标签）在 intent 为空的泛问题下也算相关，
	// 让"当前是什么版本"能命中最新公告的总览。
	if len(section.ImpactTags) == 0 {
		if _, ok := preferredTags["general"]; ok {
			return true
		}
	}
	return false
}

func patchNoteSectionToInsight(note *models.PatchNote, section models.PatchNoteSection, isCurrent bool) contracts.PatchNoteInsight {
	return contracts.PatchNoteInsight{
		Patch:         note.Patch,
		Title:         note.Title,
		Source:        note.Source,
		SourceURL:     note.SourceURL,
		PublishedAt:   note.PublishedAt,
		SectionType:   section.Type,
		SectionTitle:  section.Title,
		Summary:       section.Summary,
		ImpactTags:    cloneStrings(section.ImpactTags),
		Details:       limitStrings(cloneStrings(section.Details), 6),
		IsHistorical:  !isCurrent,
	}
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// sortPatchNotesDesc 按补丁号从新到旧排序公告（语义化）。
func sortPatchNotesDesc(notes []*models.PatchNote) {
	sort.Slice(notes, func(i, j int) bool {
		return ComparePatch(notes[i].Patch, notes[j].Patch) > 0
	})
}
