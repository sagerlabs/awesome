package knowledge

import (
	"sort"
	"strings"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

func (s *UnifiedStore) buildPatchNoteInsights(ctx contracts.QueryNLURequest) []contracts.PatchNoteInsight {
	if s.knowledgeStore == nil {
		return nil
	}

	notes := s.knowledgeStore.GetAllPatchNotes()
	if len(notes) == 0 {
		return nil
	}
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].PublishedAt > notes[j].PublishedAt
	})

	queries := patchNoteQueries(ctx)
	preferredTags := patchNotePreferredTags(ctx)
	var insights []contracts.PatchNoteInsight
	for _, note := range notes {
		for _, section := range note.Sections {
			if !patchNoteSectionRelevant(section, queries, preferredTags) {
				continue
			}
			insights = append(insights, patchNoteSectionToInsight(note, section))
			if len(insights) >= 5 {
				return insights
			}
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
		tags["mechanic"] = struct{}{}
		tags["shop_odds"] = struct{}{}
		tags["loot"] = struct{}{}
		tags["itemization"] = struct{}{}
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
	return false
}

func patchNoteSectionToInsight(note *models.PatchNote, section models.PatchNoteSection) contracts.PatchNoteInsight {
	return contracts.PatchNoteInsight{
		Patch:        note.Patch,
		Title:        note.Title,
		Source:       note.Source,
		SourceURL:    note.SourceURL,
		PublishedAt:  note.PublishedAt,
		SectionType:  section.Type,
		SectionTitle: section.Title,
		Summary:      section.Summary,
		ImpactTags:   cloneStrings(section.ImpactTags),
		Details:      limitStrings(cloneStrings(section.Details), 6),
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
