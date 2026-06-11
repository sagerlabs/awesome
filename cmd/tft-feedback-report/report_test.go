package main

import (
	"strings"
	"testing"
)

const sampleJSONL = `
{"timestamp":"2026-06-10T01:00:00Z","date":"2026-06-10","type":"advice_rejected","user_input":"不对","previous_user_input":"剑魔打工强吗","advice_summary":"能打工"}
not-valid-json
{"timestamp":"2026-06-11T02:00:00Z","date":"2026-06-11","type":"advice_rejected","user_input":"答非所问","previous_user_input":"最强阵容","advice_summary":"推荐了旧版本","reason":"用户明确指出"}

{"timestamp":"2026-06-11T03:00:00Z","date":"2026-06-11","type":"advice_rejected","user_input":"没用","previous_user_input":"装备给谁","advice_summary":"给兰博"}
`

func TestReadRecordsSkipsBadLinesAndBlanks(t *testing.T) {
	records, skipped, err := readRecords(strings.NewReader(sampleJSONL))
	if err != nil {
		t.Fatalf("readRecords failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("want 3 records, got %d", len(records))
	}
	if skipped != 1 {
		t.Fatalf("want 1 skipped bad line, got %d", skipped)
	}
	if records[0].UserInput != "不对" || records[2].AdviceSummary != "给兰博" {
		t.Fatalf("records parsed out of order: %+v", records)
	}
}

func TestBuildReportAggregatesByDateAndListsRecent(t *testing.T) {
	records, skipped, _ := readRecords(strings.NewReader(sampleJSONL))
	out := buildReport(records, skipped, 2)

	for _, want := range []string{
		"样本总数：3",
		"1 条损坏行已跳过",
		"2026-06-10：1 条",
		"2026-06-11：2 条",
		"最近 2 条样本",
		"答非所问", // 最近2条应含第2、3条
		"给兰博",
		"判定原因：用户明确指出",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "剑魔打工强吗") {
		t.Error("oldest record should be excluded when recentN=2")
	}
}

func TestBuildReportEmpty(t *testing.T) {
	out := buildReport(nil, 0, 10)
	if !strings.Contains(out, "还没有 rejected 样本") {
		t.Fatalf("empty report should explain there is no data:\n%s", out)
	}
}
