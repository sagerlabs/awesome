package agent_test

// FIX-01：业务行为测试已与现场生产数据解耦。
// 历史依赖 `data.NewStore(metadata/tft-meta/data)` 的耦合测试，
// 一旦赛季轮换就会因为"金克丝"等角色消失而失败。
// 本文件改用 tft/agent/fixtures_test.go 中的稳定内存数据集。
//
// 现行赛季结构契约测试（数量、引用、Chinese ID 覆盖）已迁到
// nlu_data_contract_test.go，由 Code Q 角色独立维护。

import (
	"context"
	"testing"

	"github.com/sagerlabs/awesome/tft/agent"
)

func TestQueryNLUData_Basic(t *testing.T) {
	store := newFixtureStore()
	ctx := context.Background()
	_ = ctx

	testCases := []struct {
		name         string
		inputCtx     agent.Context
		expectHero   int // 期望命中的阵容数
		expectItem   int // 期望命中的装备数
		expectCNHero string // 期望规范化后的中文名（空表示不检查）
	}{
		{
			name: "只有英雄-金克丝",
			inputCtx: agent.Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"金克丝": 2},
			},
			expectHero:   1, // FX_Jinx_S 含 Jinx
			expectItem:   0,
			expectCNHero: "金克丝",
		},
		{
			name: "只有装备-羊刀",
			inputCtx: agent.Context{
				Intent: "lineup_recommend",
				Items:  []string{"鬼索的狂暴之刃"},
			},
			// 纯装备查询：item 自身命中 1 项，关联 comp（FX_Jinx_S）同步返回 1。
			// 这是 knowledge/internal_query.go 的既定行为，不应被测试覆盖。
			expectHero: 1,
			expectItem: 1,
		},
		{
			name: "英雄+装备",
			inputCtx: agent.Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"金克丝": 1},
				Items:     []string{"鬼索的狂暴之刃"},
			},
			expectHero: 1,
			expectItem: 1,
		},
		{
			// FIX-01 新增：空英雄 map 不应 panic，应走 top-comps 兜底
			name: "空英雄-兜底S档",
			inputCtx: agent.Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{},
			},
			expectHero: 2, // S+A 两个兜底，不包含 B
			expectItem: 0,
		},
		{
			// FIX-01 新增：未识别英雄，不命中任何阵容，不应 panic
			name: "未知英雄-无命中",
			inputCtx: agent.Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"不存在的英雄XYZ": 2},
			},
			expectHero: 0,
			expectItem: 0,
		},
		{
			// FIX-01 新增：英雄+装备都不可识别
			name: "英雄+装备都未知",
			inputCtx: agent.Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"假人A": 1},
				Items:     []string{"假装备"},
			},
			expectHero: 0,
			expectItem: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := agent.QueryNLUData(tc.inputCtx, store)

			if got := len(result.MatchedComps); got != tc.expectHero {
				t.Errorf("MatchedComps 数量 = %d, 期望 %d", got, tc.expectHero)
			}
			if got := len(result.MatchedItems); got != tc.expectItem {
				t.Errorf("MatchedItems 数量 = %d, 期望 %d", got, tc.expectItem)
			}

			// 规范化后的中文名检查
			if tc.expectCNHero != "" {
				if _, ok := result.Ctx.Champions[tc.expectCNHero]; !ok {
					t.Errorf("规范化后丢失中文名 %q, 得到 %v", tc.expectCNHero, result.Ctx.Champions)
				}
			}

			t.Logf("matched_comps=%d matched_items=%d", len(result.MatchedComps), len(result.MatchedItems))
		})
	}
}

func TestQueryNLUData_ChineseConversion(t *testing.T) {
	store := newFixtureStore()

	testCases := []struct {
		name       string
		rawName    string
		expectID   string
		expectType string // "hero" | "item"
	}{
		{
			name:       "英雄-金克丝",
			rawName:    "金克丝",
			expectID:   "TFTFX_Jinx",
			expectType: "hero",
		},
		{
			name:       "英雄-孙悟空",
			rawName:    "孙悟空",
			expectID:   "TFTFX_Wukong",
			expectType: "hero",
		},
		{
			name:       "装备-鬼索",
			rawName:    "鬼索的狂暴之刃",
			expectID:   "TFTFX_Item_GuinsoosRageblade",
			expectType: "item",
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

			if tc.expectType == "hero" {
				if _, ok := result.Ctx.Champions[tc.rawName]; !ok {
					t.Fatalf("规范化后未保留原中文名 %q，得到 %v", tc.rawName, result.Ctx.Champions)
				}
			} else {
				found := false
				for _, item := range result.MatchedItems {
					if item.ItemID == tc.expectID {
						found = true
						if item.ItemName == "" {
							t.Errorf("装备 %s 缺少中文名", tc.expectID)
						}
						break
					}
				}
				if !found {
					t.Errorf("未找到装备 ID %s", tc.expectID)
				}
			}
		})
	}
}
