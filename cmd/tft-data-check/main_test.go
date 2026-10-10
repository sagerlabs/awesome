package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// DATA-02 验收：空文件/空目录/截断 JSON/缺实体/反向 ID 冲突/重复
// cluster/混版本全部阻止；正确小包可加载；报告可定位文件/字段。

// buildValidDataset 在临时目录构造一个最小合法数据集。
func buildValidDataset(t *testing.T) (metadataDir, knowledgeDir string) {
	t.Helper()
	root := t.TempDir()
	metadataDir = filepath.Join(root, "metadata")
	knowledgeDir = filepath.Join(root, "knowledge")
	for _, d := range []string{metadataDir, filepath.Join(knowledgeDir, "team_comps"), filepath.Join(knowledgeDir, "champions"), filepath.Join(knowledgeDir, "items"), filepath.Join(knowledgeDir, "patch_notes")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string, v interface{}) {
		t.Helper()
		raw, _ := json.Marshal(v)
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(filepath.Join(metadataDir, "localization.json"), map[string]interface{}{
		"source": "test",
		"id_to_cn": map[string]string{
			"DA_18_Ahri":  "阿狸",
			"DA_18_Elise": "伊莉丝",
		},
		"cn_to_id": map[string]string{
			"阿狸":   "DA_18_Ahri",
			"伊莉丝": "DA_18_Elise",
		},
		"unit_profiles": map[string]interface{}{
			"DA_18_Ahri": map[string]interface{}{"api_name": "DA_18_Ahri", "name": "阿狸", "cost": 4, "traits": []string{}},
		},
	})
	write(filepath.Join(knowledgeDir, "team_comps", "900001_测试.json"), map[string]interface{}{
		"cluster_id": "900001",
		"tft_set":    "TFTSet18",
		"units":       []string{"阿狸", "伊莉丝"},
		"count":       500,
		"tier":        "S",
	})
	write(filepath.Join(knowledgeDir, "team_comps", "900002_测试2.json"), map[string]interface{}{
		"cluster_id": "900002",
		"tft_set":    "TFTSet18",
		"units":       []string{"阿狸"},
		"count":       300,
		"tier":        "A",
	})
	write(filepath.Join(knowledgeDir, "patch_notes", "18.1.json"), map[string]interface{}{
		"patch": "18.1", "title": "18.1 公告",
		"source": "test", "fetched_at": "2026-09-01T14:17:46+0800",
		"sections": []interface{}{},
	})
	write(filepath.Join(knowledgeDir, "aliases.json"), map[string]interface{}{
		"version": "1.0", "source": "test",
		"heroes": map[string]string{"狐狸": "阿狸"},
	})
	write(filepath.Join(knowledgeDir, "champion_profiles.json"), map[string]interface{}{
		"阿狸": map[string]interface{}{"name": "阿狸"},
	})
	write(filepath.Join(knowledgeDir, "champions", "阿狸.json"), map[string]interface{}{"name": "阿狸"})
	write(filepath.Join(knowledgeDir, "items", "蓝霸符.json"), map[string]interface{}{"name": "蓝霸符"})
	return metadataDir, knowledgeDir
}

func runChecker(t *testing.T, metadataDir, knowledgeDir string) *checker {
	t.Helper()
	c := &checker{metadataDir: metadataDir, knowledgeDir: knowledgeDir, counts: map[string]int{}}
	c.run()
	return c
}

func errorContains(t *testing.T, c *checker, substr string) {
	t.Helper()
	for _, e := range c.errors {
		if strings.Contains(e.File+e.Field+e.Msg, substr) {
			return
		}
	}
	t.Errorf("错误列表中未找到 %q；errors=%+v", substr, c.errors)
}

func TestValidDatasetPasses(t *testing.T) {
	md, kd := buildValidDataset(t)
	c := runChecker(t, md, kd)
	if len(c.errors) != 0 {
		t.Fatalf("合法数据集应通过，errors=%+v", c.errors)
	}
	if c.counts["team_comps"] != 2 || c.counts["cn_to_id"] != 2 {
		t.Errorf("计数不符：%+v", c.counts)
	}
}

func TestEmptyFileRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	if err := os.WriteFile(filepath.Join(kd, "aliases.json"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	c := runChecker(t, md, kd)
	errorContains(t, c, "空文件")
}

func TestEmptyDirRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	entries, _ := os.ReadDir(filepath.Join(kd, "team_comps"))
	for _, e := range entries {
		os.Remove(filepath.Join(kd, "team_comps", e.Name()))
	}
	c := runChecker(t, md, kd)
	errorContains(t, c, "空目录")
}

func TestTruncatedJSONRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	path := filepath.Join(kd, "team_comps", "900001_测试.json")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, raw[:len(raw)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	c := runChecker(t, md, kd)
	errorContains(t, c, "JSON 解析失败")
}

func TestMissingEntityRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	writeJSON(t, filepath.Join(kd, "team_comps", "900001_测试.json"), map[string]interface{}{
		"cluster_id": "900001", "tft_set": "TFTSet18", "units": []string{},
	})
	c := runChecker(t, md, kd)
	errorContains(t, c, "units")
}

func TestReverseIDConflictRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	// 模拟伊莉丝式多形态覆盖：cn_to_id 指向 A，id_to_cn 却指向 B
	writeJSON(t, filepath.Join(md, "localization.json"), map[string]interface{}{
		"id_to_cn": map[string]string{
			"DA_18_Elise":        "蜘蛛女皇",
			"DA_18_EliseSpider": "蜘蛛形态",
		},
		"cn_to_id": map[string]string{
			"伊莉丝": "DA_18_EliseSpider",
		},
		"unit_profiles": map[string]interface{}{},
	})
	c := runChecker(t, md, kd)
	errorContains(t, c, "反向冲突")
}

func TestDuplicateClusterRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	writeJSON(t, filepath.Join(kd, "team_comps", "900003_重复.json"), map[string]interface{}{
		"cluster_id": "900001", "tft_set": "TFTSet18", "units": []string{"阿狸"},
	})
	c := runChecker(t, md, kd)
	errorContains(t, c, "重复")
}

func TestMixedSetRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	writeJSON(t, filepath.Join(kd, "team_comps", "900003_旧赛季.json"), map[string]interface{}{
		"cluster_id": "900003", "tft_set": "TFTSet17", "units": []string{"阿狸"},
	})
	c := runChecker(t, md, kd)
	errorContains(t, c, "混版本")
}

func TestMissingPatchNotesRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	os.Remove(filepath.Join(kd, "patch_notes", "18.1.json"))
	c := runChecker(t, md, kd)
	errorContains(t, c, "补丁公告")
}

func TestBaselineSpikeWarns(t *testing.T) {
	md, kd := buildValidDataset(t)
	c := runChecker(t, md, kd)
	// 基线有 100 套阵容，现在 2 套 -> 骤变告警
	base := map[string]int{"team_comps": 100, "cn_to_id": 2}
	basePath := filepath.Join(t.TempDir(), "baseline.json")
	raw, _ := json.Marshal(base)
	os.WriteFile(basePath, raw, 0o644)
	c.compareBaseline(basePath)
	found := false
	for _, w := range c.warnings {
		if strings.Contains(w.Msg, "骤变") {
			found = true
		}
	}
	if !found {
		t.Errorf("应产生骤变告警，warnings=%+v", c.warnings)
	}
}

func TestCrossRefUnresolvableUnitRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	// units 里放一个 localization 解析不到的名字
	writeJSON(t, filepath.Join(kd, "team_comps", "900003_幽灵.json"), map[string]interface{}{
		"cluster_id": "900003", "tft_set": "TFTSet18", "units": []string{"不存在的英雄"},
	})
	c := runChecker(t, md, kd)
	errorContains(t, c, "无法解析")
}

func TestCrossRefAliasResolutionPasses(t *testing.T) {
	md, kd := buildValidDataset(t)
	// aliases 里的黑话应能兜底解析（狐狸 -> 阿狸）
	writeJSON(t, filepath.Join(kd, "team_comps", "900003_黑话.json"), map[string]interface{}{
		"cluster_id": "900003", "tft_set": "TFTSet18", "units": []string{"狐狸"},
	})
	c := runChecker(t, md, kd)
	if len(c.errors) != 0 {
		t.Fatalf("别名兜底应通过，errors=%+v", c.errors)
	}
}

func TestCrossRefMissingSeasonPatchRejected(t *testing.T) {
	md, kd := buildValidDataset(t)
	// 赛季 18 但公告只有 17.x → 拒绝
	os.Remove(filepath.Join(kd, "patch_notes", "18.1.json"))
	writeJSON(t, filepath.Join(kd, "patch_notes", "17.7.json"), map[string]interface{}{
		"patch": "17.7", "title": "17.7 公告", "fetched_at": "2026-08-01T00:00:00+0800",
	})
	c := runChecker(t, md, kd)
	errorContains(t, c, "没有对应补丁公告")
}

func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
