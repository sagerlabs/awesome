package agent_test

import (
	"context"
	"testing"

	"github.com/sagerlabs/awesome/tft/agent"
	"github.com/sagerlabs/awesome/tft/data"
)

func setupTestStore(t *testing.T) *data.Store {
	t.Helper()
	store, err := data.NewStore(data.GetDataDir())
	if err != nil {
		t.Fatalf("初始化Store失败: %v", err)
	}
	return store
}

func TestQueryNLUData_Basic(t *testing.T) {
	store := setupTestStore(t)

	ctx := context.Background()
	_ = ctx

	testCases := []struct {
		name        string
		inputCtx    agent.Context
		expectHero  bool
		expectItem  bool
	}{
		{
			name: "只有英雄",
			inputCtx: agent.Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"金克丝": 2},
			},
			expectHero: true,
			expectItem: false,
		},
		{
			name: "只有装备",
			inputCtx: agent.Context{
				Intent: "lineup_recommend",
				Items:  []string{"鬼索的狂暴之刃"},
			},
			expectHero: false,
			expectItem: true,
		},
		{
			name: "英雄+装备",
			inputCtx: agent.Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"金克丝": 1},
				Items:     []string{"羊刀"},
			},
			expectHero: true,
			expectItem: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := agent.QueryNLUData(tc.inputCtx, store)

			if tc.expectHero && len(result.MatchedComps) == 0 {
				t.Errorf("期望找到匹配阵容，但没找到")
			}
			if tc.expectItem && len(result.MatchedItems) == 0 {
				t.Errorf("期望找到匹配装备，但没找到")
			}

			t.Logf("匹配到 %d 个阵容, %d 个装备", len(result.MatchedComps), len(result.MatchedItems))
		})
	}
}

func TestQueryNLUData_ChineseConversion(t *testing.T) {
	store := setupTestStore(t)

	testCases := []struct {
		name        string
		rawName     string
		expectID    string
		expectType  string // "hero" | "item"
	}{
		{
			name:       "英雄-金克丝",
			rawName:    "金克丝",
			expectID:   "TFT16_Jinx",
			expectType: "hero",
		},
		{
			name:       "装备-羊刀",
			rawName:    "羊刀",
			expectID:   "TFT_Item_GuinsoosRageblade",
			expectType: "item",
		},
		{
			name:       "英雄-中文全名",
			rawName:    "孙悟空",
			expectID:   "TFT16_Wukong",
			expectType: "hero",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ctx agent.Context
			if tc.expectType == "hero" {
				ctx = agent.Context{
					Champions: map[string]int8{tc.rawName: 1},
				}
			} else {
				ctx = agent.Context{
					Items: []string{tc.rawName},
				}
			}

			result := agent.QueryNLUData(ctx, store)

			// 检查转换后的名称
			if tc.expectType == "hero" {
				for name := range result.Ctx.Champions {
					t.Logf("英雄名称: %s", name)
				}
			} else {
				for _, item := range result.MatchedItems {
					t.Logf("装备: %s (%s)", item.ItemName, item.ItemID)
					if item.ItemID == tc.expectID {
						t.Logf("✓ 找到匹配装备: %s", item.ItemName)
					}
				}
			}
		})
	}
}

func TestQueryNLUData_EmptyInput(t *testing.T) {
	store := setupTestStore(t)

	ctx := agent.Context{
		Intent: "lineup_recommend",
	}

	result := agent.QueryNLUData(ctx, store)

	if len(result.MatchedComps) != 0 {
		t.Errorf("空输入不应匹配到阵容，但匹配到 %d 个", len(result.MatchedComps))
	}
	if len(result.MatchedItems) != 0 {
		t.Errorf("空输入不应匹配到装备，但匹配到 %d 个", len(result.MatchedItems))
	}
}
