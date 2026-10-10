package knowledge

import (
	"testing"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

// DATA-03 验收：同一问题 current 模式不混入过期公告；
// 历史模式可查询旧公告；版本号语义化比较。

func buildPatchNoteStore(t *testing.T) *Store {
	t.Helper()
	s := NewStore()
	// 三份公告：17.7（旧）、18.1（旧）、18.4（当前）
	s.AddPatchNote(&models.PatchNote{
		Patch: "17.7", Title: "17.7 公告", PublishedAt: "2026-08-01 10:00:00",
		Sections: []models.PatchNoteSection{
			{Type: "general", Title: "星神仍在持续", Summary: "第 17 赛季星神内容仍在持续"},
		},
	})
	s.AddPatchNote(&models.PatchNote{
		Patch: "18.1", Title: "18.1 公告", PublishedAt: "2026-08-27 10:00:00",
		Sections: []models.PatchNoteSection{
			{Type: "general", Title: "18.1 更新总览", Summary: "自然之力赛季上线"},
		},
	})
	s.AddPatchNote(&models.PatchNote{
		Patch: "18.4", Title: "18.4 公告", PublishedAt: "2026-10-09 10:00:00",
		Sections: []models.PatchNoteSection{
			{Type: "general", Title: "18.4 更新总览", Summary: "战斗进场与离场时间各减少 1 秒"},
			{Type: "combat", Title: "魔女削弱", Summary: "魔女收菜奖励下调"},
		},
	})
	return s
}

func TestCurrentPatch(t *testing.T) {
	s := buildPatchNoteStore(t)
	if got := s.CurrentPatch(); got != "18.4" {
		t.Errorf("CurrentPatch = %q, want 18.4", got)
	}
}

func TestComparePatch(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"18.4", "18.1", 1},
		{"18.10", "18.4", 1},  // 语义化：10 > 4，非字典序
		{"18.1", "17.7", 1},
		{"17.7", "18.1", -1},
		{"18.4", "18.4", 0},
		{"18.1", "18.1", 0},
	}
	for _, c := range cases {
		if got := ComparePatch(c.a, c.b); got != c.want {
			t.Errorf("ComparePatch(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// current 模式：泛问题（intent 空）只命中 18.4 的章节，
// 17.7 的"星神仍在持续"绝不出现（用户踩坑的原始 bug）。
func TestPatchNoteInsightsCurrentModeNoStaleLeak(t *testing.T) {
	us := &UnifiedStore{knowledgeStore: buildPatchNoteStore(t)}
	insights := us.buildPatchNoteInsights(contracts.QueryNLURequest{})
	if len(insights) == 0 {
		t.Fatal("current 模式应命中当前公告章节")
	}
	for _, ins := range insights {
		if ins.Patch != "18.4" {
			t.Errorf("current 模式混入旧公告 %q（%s）", ins.Patch, ins.SectionTitle)
		}
		if ins.IsHistorical {
			t.Errorf("current 模式的条目不应标记历史")
		}
		if contains(ins.Summary, "星神") || contains(ins.SectionTitle, "星神") {
			t.Errorf("17.7 的旧内容泄漏进 current 回答")
		}
	}
}

// 历史模式：PatchHint="17.7" 命中旧公告，且标记 IsHistorical=true。
func TestPatchNoteInsightsHistoricalMode(t *testing.T) {
	us := &UnifiedStore{knowledgeStore: buildPatchNoteStore(t)}
	insights := us.buildPatchNoteInsights(contracts.QueryNLURequest{PatchHint: "17.7"})
	if len(insights) == 0 {
		t.Fatal("历史模式应命中 17.7 公告")
	}
	for _, ins := range insights {
		if ins.Patch != "17.7" {
			t.Errorf("历史模式应只返回 17.7，got %q", ins.Patch)
		}
		if !ins.IsHistorical {
			t.Errorf("历史模式条目 IsHistorical 应为 true")
		}
	}
	if !contains(insights[0].Summary, "星神") {
		t.Errorf("历史模式应能查到旧公告内容")
	}
}

// PatchHint 命中当前版本 → 等同 current 模式；命中不存在的版本 → 不猜，回落 current。
func TestPatchNoteInsightsHintEdgeCases(t *testing.T) {
	us := &UnifiedStore{knowledgeStore: buildPatchNoteStore(t)}

	// hint = 当前版本
	insights := us.buildPatchNoteInsights(contracts.QueryNLURequest{PatchHint: "18.4"})
	if len(insights) == 0 || insights[0].IsHistorical {
		t.Errorf("hint=18.4 应等同 current 模式")
	}

	// hint = 不存在的版本 → 回落 current，不模糊匹配
	insights = us.buildPatchNoteInsights(contracts.QueryNLURequest{PatchHint: "99.9"})
	for _, ins := range insights {
		if ins.Patch != "18.4" {
			t.Errorf("不存在的 hint 应回落 current，got %q", ins.Patch)
		}
	}
}

// 当前公告无相关章节时返回空，不回退旧公告。
func TestPatchNoteInsightsNoFallbackToStale(t *testing.T) {
	s := buildPatchNoteStore(t)
	// 只保留 17.7 有"星神"章节；把 18.4 的章节改成不相关的标签
	s.AddPatchNote(&models.PatchNote{
		Patch: "18.4", Title: "18.4 公告", PublishedAt: "2026-10-09 10:00:00",
		Sections: []models.PatchNoteSection{
			{Type: "general", Title: "个性化内容", Summary: "上架 58 款个性化道具", ImpactTags: []string{"cosmetic"}},
		},
	})
	us := &UnifiedStore{knowledgeStore: s}
	// item_query 意图：preferredTags = itemization/frontline，均不匹配
	insights := us.buildPatchNoteInsights(contracts.QueryNLURequest{Intent: "item_query", Items: []string{"不存在的装备"}})
	for _, ins := range insights {
		if ins.Patch != "18.4" {
			t.Errorf("无匹配时应返回空而不是回退旧公告，got patch=%q", ins.Patch)
		}
	}
}

// 版本元信息：buildKnowledgeMetadata 声明当前补丁与样本门槛。
func TestKnowledgeMetadataCarriesPatchAndFloor(t *testing.T) {
	us := &UnifiedStore{knowledgeStore: buildPatchNoteStore(t)}
	md := us.buildKnowledgeMetadata()
	if md == nil {
		t.Fatal("metadata 不应为 nil")
	}
	if md.GamePatch != "18.4" {
		t.Errorf("GamePatch = %q, want 18.4（agent 据此回答当前版本）", md.GamePatch)
	}
	if md.SampleFloor != MinSampleFloorForStats {
		t.Errorf("SampleFloor = %d, want %d", md.SampleFloor, MinSampleFloorForStats)
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
