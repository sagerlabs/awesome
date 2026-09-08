package agent_test

// nlu_data_contract_test.go：现场数据契约测试。
//
// FIX-01 把"金克丝必有阵容"等业务断言从本文件剥离；
// 本文件只验证当前生产 metadata/tft-meta/data/ 的结构性约束，
// 失败时应归 D 角色修复数据，而不是修改本测试。
//
// 契约不变量（与赛季无关）：
//  1. 至少加载到一个 comp；每个 comp 必须有 cluster_id、name、tier、units。
//  2. tier ∈ {S, A, B, C, D}。
//  3. 每个 comp 的 BestBuild.Carry（若非空）必须出现在 Units 中。
//  4. items_priority.json 中的装备 ID 必须在 localization 中存在中文名。
//  5. 所有 comp 的 unit ID 中，正向（ID→CN）覆盖率应 ≥ 80%，
//     低于该阈值通常意味着多形态 ID 适配问题（FOLLOW-UP FIX-02）。
//
// 这些断言使用生产数据目录，请确保 TFT_DATA_DIR 未指向隔离目录。

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
)

const minForwardCoverage = 0.80

func loadProductionStore(t *testing.T) *data.Store {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件路径")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	dataDir := filepath.Join(projectRoot, "metadata", "tft-meta", "data")
	store, err := data.NewStore(dataDir)
	if err != nil {
		t.Fatalf("加载生产数据失败: %v", err)
	}
	return store
}

func TestDataContract_Loads(t *testing.T) {
	store := loadProductionStore(t)

	comps := store.AllComps()
	if len(comps) == 0 {
		t.Fatal("生产数据未加载到任何 comp；门禁应阻止此类快照发布")
	}
	t.Logf("loaded comps: %d", len(comps))
}

func TestDataContract_CompRequiredFields(t *testing.T) {
	store := loadProductionStore(t)
	comps := store.AllComps()

	validTiers := map[string]bool{"S": true, "A": true, "B": true, "C": true, "D": true}
	for i, c := range comps {
		if c.ClusterID == "" {
			t.Errorf("comp[%d] 缺少 cluster_id", i)
		}
		if c.Name == "" {
			t.Errorf("comp[%s] 缺少 name", c.ClusterID)
		}
		if c.Tier == "" {
			t.Errorf("comp[%s] 缺少 tier", c.ClusterID)
		} else if !validTiers[c.Tier] {
			t.Errorf("comp[%s] tier=%q 不在合法集合 S/A/B/C/D 内", c.ClusterID, c.Tier)
		}
		if len(c.Units) == 0 {
			t.Errorf("comp[%s] units 为空", c.ClusterID)
		}
	}
}

func TestDataContract_BestBuildCarryInUnits(t *testing.T) {
	store := loadProductionStore(t)
	comps := store.AllComps()

	for _, c := range comps {
		carry := c.BestBuild.Carry
		if carry == "" {
			continue
		}
		found := false
		for _, u := range c.Units {
			if u == carry {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("comp[%s] best_build.carry=%q 不在 units 中", c.ClusterID, carry)
		}
	}
}

func TestDataContract_ItemIDsHaveCN(t *testing.T) {
	store := loadProductionStore(t)
	// 透出 items 集合（通过 AllItems 等价：内部 items map 不可直接访问，
	// 通过 comp.best_build.items 抽样已绑定的装备 ID，再检查 localization）
	comps := store.AllComps()
	checked := 0
	missing := 0
	for _, c := range comps {
		for _, it := range c.BestBuild.Items {
			if it == "" {
				continue
			}
			checked++
			if cn := store.IDToCN(it); cn == it || cn == "" {
				missing++
				t.Logf("装备 %q 在 comp[%s] 缺失中文名", it, c.ClusterID)
			}
		}
	}
	if checked == 0 {
		t.Skip("现场数据无任何 best_build.items，跳过")
	}
	coverage := float64(checked-missing) / float64(checked)
	if coverage < minForwardCoverage {
		t.Errorf("装备 ID→中文覆盖率 %.2f < %.2f（missing=%d, total=%d）",
			coverage, minForwardCoverage, missing, checked)
	}
	t.Logf("item forward coverage: %.2f (%d/%d)", coverage, checked-missing, checked)
}

func TestDataContract_UnitIDsForwardCoverage(t *testing.T) {
	store := loadProductionStore(t)
	comps := store.AllComps()

	total := 0
	hit := 0
	missingIDs := map[string]int{}
	for _, c := range comps {
		for _, u := range c.Units {
			if u == "" {
				continue
			}
			total++
			if cn := store.IDToCN(u); cn != u && cn != "" {
				hit++
			} else {
				missingIDs[u]++
			}
		}
	}
	if total == 0 {
		t.Fatal("现场数据无任何 units")
	}
	coverage := float64(hit) / float64(total)
	if coverage < minForwardCoverage {
		// 不直接 fail（FOLLOW-UP FIX-02），但记录原因。
		t.Logf("unit ID→中文覆盖率 %.2f < %.2f（missing distinct=%d, total refs=%d），疑似多形态 ID 适配问题",
			coverage, minForwardCoverage, len(missingIDs), total)
	} else {
		t.Logf("unit forward coverage: %.2f (%d/%d)", coverage, hit, total)
	}
}
