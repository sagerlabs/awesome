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
		Metadata: &contracts.KnowledgeMetadata{
			Source:      "MetaTFT",
			Version:     "TFT17",
			UpdatedAt:   "2026-05-09",
			SampleCount: 900,
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
				Plan: &contracts.CompPlan{
					Final: contracts.BoardSnapshot{
						Level: "8",
						Units: []contracts.BoardUnit{
							{Name: "兰博", Items: []string{"鬼索的狂暴之刃"}, IsCore: true},
							{Name: "凯南"},
						},
						Traits: []contracts.TraitMarker{{Name: "约德尔人", Count: 4}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"约德尔人兰博", "知识库元信息", "样本量：900", "样本场次：25060", "运营节奏：5级节奏", "成型棋盘：8级", "兰博（鬼索的狂暴之刃）"} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestBuildNluFormatPromptShowsNormalizedTerms(t *testing.T) {
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "羊刀给谁？",
		Ctx: contracts.QueryNLURequest{
			Intent: "item_query",
			Items:  []string{"鬼索的狂暴之刃"},
		},
		NormalizedTerms: []contracts.NormalizedTerm{
			{Type: "item", Raw: "羊刀", Normalized: "鬼索的狂暴之刃"},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"已识别黑话", "羊刀 => 鬼索的狂暴之刃", "装备：鬼索的狂暴之刃"} {
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

func TestBuildNluFormatPromptIncludesVerticalChampionData(t *testing.T) {
	cost := 4
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "四费卡谁最厉害，谁能C，谁能抗？",
		Ctx: contracts.QueryNLURequest{
			Intent:    "vertical_query",
			UnitCost:  &cost,
			RoleQuery: "all",
		},
		MatchedChampions: []contracts.ChampionInsight{
			{
				Name:             "拉克丝",
				Cost:             4,
				Role:             "主C",
				Tags:             []string{"能C"},
				BestAvgPlacement: 3.10,
				BestComps: []contracts.CompSummary{
					{Name: "法师拉克丝", Tier: "S", AvgPlacement: 3.20, Top4Rate: 0.66, WinRate: 0.22},
				},
				BestBuilds: []contracts.BuildInfo{
					{Items: []string{"珠光护手", "朔极之矛"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"查询费用：4费卡", "垂直英雄数据", "拉克丝（4费，主C）", "定位判断：能C", "不能把未命中的英雄塞进答案"} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestBuildNluFormatPromptTreatsWorkQueryAsTransition(t *testing.T) {
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "剑魔打工强吗？",
		Ctx: contracts.QueryNLURequest{
			Intent:    "champion_query",
			Champions: map[string]int8{"亚托克斯": 1},
			RoleQuery: "work",
		},
		NormalizedTerms: []contracts.NormalizedTerm{
			{Type: "champion", Raw: "剑魔", Normalized: "亚托克斯"},
		},
		MatchedChampions: []contracts.ChampionInsight{
			{
				Name:             "亚托克斯",
				Cost:             1,
				Role:             "主C",
				Tags:             []string{"能C"},
				BestAvgPlacement: 4.18,
				BestComps: []contracts.CompSummary{
					{Name: "海魔人卑尔维斯", Tier: "S", AvgPlacement: 4.18, Top4Rate: 0.58, WinRate: 0.10},
				},
				BestBuilds: []contracts.BuildInfo{
					{Items: []string{"饮血剑", "泰坦的坚决"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"查询定位：打工/过渡", "打工问题说明", "亚托克斯（1费）", "可携带装备数据", "不要默认说能当主C"} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
	for _, text := range []string{"亚托克斯（1费，主C）", "定位判断：能C", "推荐装备："} {
		if strings.Contains(prompt, text) {
			t.Fatalf("prompt should not contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestBuildNluFormatPromptIncludesTraitData(t *testing.T) {
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "法师羁绊现在能玩吗？",
		Ctx: contracts.QueryNLURequest{
			Intent: "trait_query",
			Traits: []string{"法师"},
		},
		MatchedTraits: []contracts.TraitInsight{
			{
				Name:        "法师",
				Activations: []string{"法师 (3)", "法师 (5)"},
				Units:       []string{"拉克丝", "安妮"},
				BestComps: []contracts.CompSummary{
					{Name: "法师拉克丝", Tier: "S", AvgPlacement: 3.20, Top4Rate: 0.66, WinRate: 0.22},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"羁绊数据", "命中档位：法师 (3)、法师 (5)", "代表阵容：法师拉克丝", "不要编造解锁规则"} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestBuildNluFormatPromptIncludesPatchNotes(t *testing.T) {
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "现在坦克装备还强吗？",
		Ctx: contracts.QueryNLURequest{
			Intent: "item_query",
			Items:  []string{"棘刺背心"},
		},
		PatchNotes: []contracts.PatchNoteInsight{
			{
				Patch:        "17.1",
				Source:       "Tencent LOL",
				PublishedAt:  "2026-04-15 19:04:36",
				SectionTitle: "装备",
				Summary:      "坦克装备整体削弱",
				ImpactTags:   []string{"itemization", "frontline"},
				Details:      []string{"【棘刺背心】额外生命值：9%->6%"},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"官方版本环境", "17.1 - 装备", "坦克装备整体削弱", "版本原因可以引用官方版本环境"} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestBuildNluFormatPromptIncludesDecisionPolicyHints(t *testing.T) {
	stage := "3-2"
	level := 6
	hp := 40
	gold := 50
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "我现在3-2，6级，40血，50金币，千珏两星羊刀水银，能冲吗？",
		Ctx: contracts.QueryNLURequest{
			Intent:    "lineup_recommend",
			GameStage: &stage,
			Level:     &level,
			HP:        &hp,
			Gold:      &gold,
			Champions: map[string]int8{"千珏": 2},
			Items:     []string{"鬼索的狂暴之刃", "水银"},
		},
		MatchedComps: []contracts.CompSummary{
			{
				Name:         "狂战士千珏",
				Tier:         "S",
				AvgPlacement: 3.02,
				Top4Rate:     0.80,
				WinRate:      0.22,
				Plan: &contracts.CompPlan{
					Early:  &contracts.BoardSnapshot{Level: "5"},
					Middle: &contracts.BoardSnapshot{Level: "7"},
					Final:  contracts.BoardSnapshot{Level: "9", Units: []contracts.BoardUnit{{Name: "千珏"}}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"局内决策提示", "3-2 是常见启动点", "血量中低", "经济健康", "按可执行性排序", "early/middle/final"} {
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
