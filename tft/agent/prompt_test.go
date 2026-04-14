package agent

import (
	"strings"
	"testing"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
)

func TestFormatSystemPromptHasFactBoundary(t *testing.T) {
	required := []string{
		"只能使用用户消息中",
		"不要使用模型记忆",
		"不要编造 T0/T0.5",
		"当前知识库没有命中",
		"第一句话先回答玩家最关心的结论",
		"数据来源说明放在回答末尾",
		"国服云顶老玩家",
		"S级/A级",
	}

	for _, text := range required {
		if !strings.Contains(FormatSystemPrompt, text) {
			t.Fatalf("FormatSystemPrompt should contain %q", text)
		}
	}
}

func TestBuildNluFormatPromptUsesKnowledgeFriendlyFields(t *testing.T) {
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "当前版本最强三套阵容是什么？",
		Ctx: contracts.QueryNLURequest{
			Intent: "lineup_recommend",
		},
		MatchedComps: []contracts.CompSummary{
			{
				Name:         "约德尔人兰博",
				Tier:         "S",
				AvgPlacement: 3.84,
				Top4Rate:     0.62,
				WinRate:      0.18,
				Count:        25060,
				Levelling:    "lvl 5",
				BestBuild: contracts.BuildInfo{
					Carry: "兰博",
					Items: []string{"鬼索的狂暴之刃", "灭世者的死亡之帽"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"约德尔人兰博", "样本场次：25060", "运营节奏：5级节奏"} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestBuildNluFormatPromptSeparatesItemCarrierFromCompCore(t *testing.T) {
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "我有珠光护手，可以玩什么阵容？",
		Ctx: contracts.QueryNLURequest{
			Intent: "item_query",
			Items:  []string{"珠光护手"},
		},
		MatchedItems: []contracts.MatchedItemInfo{
			{
				ItemName: "珠光护手",
				CompInfos: []contracts.ItemFitCompInfo{
					{
						ClusterID:     "394014",
						CompName:      "约德尔人兰博",
						CompTier:      "S",
						CompAvg:       3.85,
						CarryName:     "菲兹",
						PriorityScore: 100,
					},
				},
			},
		},
		MatchedComps: []contracts.CompSummary{
			{
				ClusterID:    "394014",
				Name:         "约德尔人兰博",
				Tier:         "S",
				AvgPlacement: 3.85,
				Top4Rate:     0.62,
				WinRate:      0.18,
				BestBuild: contracts.BuildInfo{
					Carry: "兰博",
					Items: []string{"鬼索的狂暴之刃", "灭世者的死亡之帽"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{
		"装备携带者不一定等于阵容名里的主核心",
		"约德尔人兰博（S级，平均排名3.85）里可给 **菲兹**",
		"本次装备适配：珠光护手可给菲兹（优先级100/100）",
		"不要说成“核心英雄”",
	} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestFormatTier(t *testing.T) {
	cases := map[string]string{
		"S":      "S级",
		"A":      "A级",
		"S Tier": "S级",
		"":       "未知强度",
	}

	for input, want := range cases {
		if got := formatTier(input); got != want {
			t.Fatalf("formatTier(%q) = %q, want %q", input, got, want)
		}
	}
}
