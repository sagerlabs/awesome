package data_test

// FIX-02：data/loader.go 的 ResolveUnitID / CNToID 多形态 ID 解析测试。
//
// 这些测试与生产数据解耦，使用 NewStoreFromRaw 构造固定输入。
// 关键覆盖：
//   1. 中文名精确解析（伊莉丝 → 实际阵容 ID 而非变换形态）
//   2. 多形态别名解析（伊莉丝 ↔ TFT18_Elise / DA_18_Elise / TFT18_EliseSpider）
//   3. 不限前缀的裸 ID 解析（TFT16_*、DA_18_* 都可被 ResolveUnitID 接受）
//   4. 反向：id→中文 应回退到中文名
//
// 避免改动 production fixtures：单独构造最小化伊莉丝场景。

import (
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
)

const (
	eliseNormalID  = "DA_18_Elise"        // comps_data 实际单位
	eliseBaseID    = "TFT18_Elise"         // lookup apiName（人形态）
	eliseSpiderID  = "TFT18_EliseSpider"   // lookup apiName（蜘蛛形态）
	eliseAsset1    = "DA_18_Elise"
	eliseAsset2    = "DA_18_EliseSpider"
	eliseName      = "伊莉丝"
	eliseCompID    = "FX_Elise_Comp"
)

func newEliseStore() *data.Store {
	comps := []data.Comp{
		{
			ClusterID: eliseCompID,
			Name:      "CovenMorgana",
			Tier:      "D",
			Count:     500,
			Units:     []string{eliseNormalID, "DA_18_Morgana"},
			BestBuild: data.BuildInfo{Carry: "DA_18_Morgana"},
		},
	}
	// 模拟翻译表：comps 用 DA_18_Elise，lookup apiName 是 TFT18_Elise + 蜘蛛形态
	loc := data.LocalizationFile{
		IDToCN: map[string]string{
			eliseBaseID:   eliseName,
			eliseSpiderID: eliseName,
			eliseAsset1:   eliseName,
			eliseAsset2:   eliseName,
			"DA_18_Morgana": "莫甘娜",
		},
		// canonical 选了 comps 实际用的 DA_18_Elise（数据驱动主 ID）
		CNToID: map[string]string{
			eliseName:   eliseNormalID,
			"莫甘娜":       "DA_18_Morgana",
		},
	}
	return data.NewStoreFromRaw(comps, data.ItemsFile{}, loc)
}

func TestResolveUnitID_ChineseReturnsCompID(t *testing.T) {
	store := newEliseStore()
	got := store.ResolveUnitID(eliseName)
	if got != eliseNormalID {
		t.Fatalf("ResolveUnitID(%q) = %q, 期望 %q", eliseName, got, eliseNormalID)
	}
}

func TestResolveUnitID_AssetNameReturnsCompID(t *testing.T) {
	store := newEliseStore()
	// DA_18_Elise 出现在 compsByUnit 中，ResolveUnitID 应直接接受
	got := store.ResolveUnitID(eliseAsset1)
	if got != eliseAsset1 {
		t.Fatalf("ResolveUnitID(%q) = %q, 期望 %q", eliseAsset1, got, eliseAsset1)
	}
}

func TestResolveUnitID_BaseFormReturnsCompID(t *testing.T) {
	store := newEliseStore()
	// TFT18_Elise 不在 compsByUnit 中（comps 用 DA_18_Elise），
	// 但 ResolveUnitID 应该回退到中文名再解析，避免漏命中
	got := store.ResolveUnitID(eliseBaseID)
	if got != eliseNormalID {
		t.Fatalf("ResolveUnitID(%q) = %q, 期望 %q（通过中文名回退到主 ID）",
			eliseBaseID, got, eliseNormalID)
	}
}

func TestResolveUnitID_SpiderFormReturnsCompID(t *testing.T) {
	store := newEliseStore()
	got := store.ResolveUnitID(eliseSpiderID)
	if got != eliseNormalID {
		t.Fatalf("ResolveUnitID(%q) = %q, 期望 %q（蜘蛛形态也应能命中正形态）",
			eliseSpiderID, got, eliseNormalID)
	}
}

func TestGetCompsByUnits_ResolvesEliseCorrectly(t *testing.T) {
	store := newEliseStore()
	// 通过 ResolveUnitID("伊莉丝") 拿到的 ID 应能命中 comp
	id := store.ResolveUnitID(eliseName)
	if id == "" {
		t.Fatal("ResolveUnitID(伊莉丝) 返回空")
	}
	matches := store.GetCompsByUnits([]string{id})
	if len(matches) != 1 {
		t.Fatalf("GetCompsByUnits([%q]) = %d, 期望 1", id, len(matches))
	}
	if matches[0].Comp.ClusterID != eliseCompID {
		t.Fatalf("命中了错误的 comp: %s", matches[0].Comp.ClusterID)
	}
}

func TestIDToCN_RoundTrip(t *testing.T) {
	store := newEliseStore()
	if got := store.IDToCN(eliseNormalID); got != eliseName {
		t.Errorf("IDToCN(%q) = %q, 期望 %q", eliseNormalID, got, eliseName)
	}
	// 中文回 ID
	if got := store.CNToID(eliseName); got != eliseNormalID {
		t.Errorf("CNToID(%q) = %q, 期望 %q", eliseName, got, eliseNormalID)
	}
}

func TestResolveUnitID_UnknownReturnsEmpty(t *testing.T) {
	store := newEliseStore()
	if got := store.ResolveUnitID("不存在的英雄"); got != "" {
		t.Errorf("未知英雄应返回空，实际 %q", got)
	}
}
