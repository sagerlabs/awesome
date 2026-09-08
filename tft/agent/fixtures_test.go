package agent_test

// fixtures_test.go 为 FIX-01 提供稳定、与生产数据解耦的内存数据集。
// 测试通过 data.NewStoreFromRaw 直接构造 data.Store，
// 不再依赖 metadata/tft-meta/data/ 下的 JSON 现场文件。
//
// 设计原则：
//   - 角色与 ID 命名沿用历史测试中"金克丝"系列代号，便于追溯；
//   - 中文→ID、ID→中文双向可解析；
//   - 至少一个阵容绑定 Guinsoo 装备，以便验证英雄+装备查询路径；
//   - 故意包含一个 S/A/B 三档阵容，覆盖 shouldReturnTopComps 排序。
//
// 注意：本文件只应被 *_test.go 引用，不允许被生产代码 import。

import "github.com/sagerlabs/awesome/tft/data"

// fixtureChampionID/fixtureItemID 统一前缀，避免与生产 API ID 混淆。
const (
	fixtureJinxID    = "TFTFX_Jinx"
	fixtureWukongID  = "TFTFX_Wukong"
	fixtureHeimerID  = "TFTFX_Heimerdinger"
	fixtureRumbleID  = "TFTFX_Rumble"
	fixtureTaricID   = "TFTFX_Taric"
	fixtureGuinsooID = "TFTFX_Item_GuinsoosRageblade"
	fixtureRabadonID = "TFTFX_Item_RabadonsDeathcap"
	fixtureSunfireID = "TFTFX_Item_SunfireCape"
)

// newFixtureStore 返回一个内存数据仓库，金克丝、孙悟空等角色稳定可解析。
// 不依赖任何网络或磁盘文件，可在隔离环境运行。
func newFixtureStore() *data.Store {
	comps := []data.Comp{
		{
			// S 档 JinxCarry：金克丝主C，绑定 Guinsoo+Rabadon
			ClusterID:    "FX_Jinx_S",
			Name:         "JinxCarry",
			Tier:         "S",
			AvgPlacement: 3.50,
			Top4Rate:     0.65,
			WinRate:      0.22,
			Count:        1500,
			Units:        []string{fixtureJinxID, fixtureHeimerID, fixtureRumbleID},
			Traits:       []string{"TFTFX_Yordle_2"},
			Stars:        []string{fixtureJinxID},
			Levelling:    "lvl 8",
			BestBuild: data.BuildInfo{
				Carry: fixtureJinxID,
				Items: []string{fixtureGuinsooID, fixtureRabadonID},
				PriorityScores: map[string]int{
					fixtureGuinsooID: 100,
					fixtureRabadonID: 85,
				},
				AvgPlacement: 3.40,
			},
		},
		{
			// A 档 WukongFlex：孙悟空+塔里克，与金克丝解耦
			ClusterID:    "FX_Wukong_A",
			Name:         "WukongFlex",
			Tier:         "A",
			AvgPlacement: 4.10,
			Top4Rate:     0.55,
			WinRate:      0.14,
			Count:        900,
			Units:        []string{fixtureWukongID, fixtureTaricID},
			Traits:       []string{"TFTFX_Brawler_2"},
			BestBuild: data.BuildInfo{
				Carry: fixtureWukongID,
				Items: []string{fixtureSunfireID},
				PriorityScores: map[string]int{
					fixtureSunfireID: 80,
				},
				AvgPlacement: 4.00,
			},
		},
		{
			// B 档 TaricSolo：单核塔里克，专门用于 top-comps 排序
			ClusterID:    "FX_Taric_B",
			Name:         "TaricSolo",
			Tier:         "B",
			AvgPlacement: 4.60,
			Top4Rate:     0.45,
			WinRate:      0.07,
			Count:        600,
			Units:        []string{fixtureTaricID},
			BestBuild: data.BuildInfo{
				Carry: fixtureTaricID,
				Items: []string{fixtureSunfireID},
			},
		},
	}

	items := data.ItemsFile{
		fixtureGuinsooID: {
			{
				ClusterID:     "FX_Jinx_S",
				CompName:      "JinxCarry",
				CompTier:      "S",
				CompAvg:       3.50,
				Carry:         fixtureJinxID,
				PriorityScore: 100,
			},
		},
		fixtureRabadonID: {
			{
				ClusterID:     "FX_Jinx_S",
				CompName:      "JinxCarry",
				CompTier:      "S",
				CompAvg:       3.50,
				Carry:         fixtureJinxID,
				PriorityScore: 85,
			},
		},
		fixtureSunfireID: {
			{
				ClusterID:     "FX_Wukong_A",
				CompName:      "WukongFlex",
				CompTier:      "A",
				CompAvg:       4.10,
				Carry:         fixtureWukongID,
				PriorityScore: 80,
			},
			{
				ClusterID:     "FX_Taric_B",
				CompName:      "TaricSolo",
				CompTier:      "B",
				CompAvg:       4.60,
				Carry:         fixtureTaricID,
				PriorityScore: 50,
			},
		},
	}

	loc := data.LocalizationFile{
		Source: "fixture://fix-01",
		IDToCN: map[string]string{
			fixtureJinxID:    "金克丝",
			fixtureWukongID:  "孙悟空",
			fixtureHeimerID:  "黑默丁格",
			fixtureRumbleID:  "兰博",
			fixtureTaricID:   "塔里克",
			fixtureGuinsooID: "鬼索的狂暴之刃",
			fixtureRabadonID: "灭世者的死亡之帽",
			fixtureSunfireID: "日炎斗篷",
		},
		CNToID: map[string]string{
			"金克丝":         fixtureJinxID,
			"孙悟空":         fixtureWukongID,
			"黑默丁格":        fixtureHeimerID,
			"兰博":          fixtureRumbleID,
			"塔里克":         fixtureTaricID,
			"鬼索的狂暴之刃":     fixtureGuinsooID,
			"灭世者的死亡之帽": fixtureRabadonID,
			"日炎斗篷":        fixtureSunfireID,
		},
	}

	return data.NewStoreFromRaw(comps, items, loc)
}
