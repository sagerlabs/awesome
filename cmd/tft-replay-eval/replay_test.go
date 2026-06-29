package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestLoadCasesSkipsUnreplayableLines(t *testing.T) {
	input := `
{"timestamp":"t1","user_input":"不对","previous_user_input":"剑魔强吗","advice_summary":"强"}
{"timestamp":"t2","user_input":"没用"}
broken
{"timestamp":"t3","user_input":"答非所问","previous_user_input":"装备给谁","advice_summary":"给兰博"}
`
	cases, skipped, err := loadCases(strings.NewReader(input))
	if err != nil {
		t.Fatalf("loadCases failed: %v", err)
	}
	if len(cases) != 2 || skipped != 2 {
		t.Fatalf("want 2 cases / 2 skipped, got %d / %d", len(cases), skipped)
	}
}

func TestJudge(t *testing.T) {
	c := replayCase{AdviceSummary: "剑魔前期打工很强，建议优先拿"}

	t.Run("纠偏回答全维度通过", func(t *testing.T) {
		v := judge(c, "上一轮可能没命中你的问题，重新看：当前版本剑魔被削弱，建议优先转其他主C。下一步先稳经济。", nil)
		if !v.Answered || !v.Acknowledged || !v.NotRepeated || !v.Actionable || !v.passed() {
			t.Fatalf("expected all dimensions to pass: %+v", v)
		}
	})

	t.Run("照搬旧回答判为重复", func(t *testing.T) {
		v := judge(c, "剑魔前期打工很强，建议优先拿，真的。", nil)
		if v.NotRepeated || v.passed() {
			t.Fatalf("verbatim repeat should fail not_repeated: %+v", v)
		}
	})

	t.Run("回放错误判为未回答", func(t *testing.T) {
		v := judge(c, "", context.DeadlineExceeded)
		if v.Answered || v.passed() {
			t.Fatalf("error should fail answered: %+v", v)
		}
	})

	t.Run("无基线时不算重复", func(t *testing.T) {
		v := judge(replayCase{}, "任意新回答，建议先看牌面。", nil)
		if !v.NotRepeated {
			t.Fatal("empty baseline should never count as repeated")
		}
	})
}

func TestReplayAllWithDemoRunnerProducesPassingReport(t *testing.T) {
	f, err := os.Open("testdata/demo_cases.jsonl")
	if err != nil {
		t.Fatalf("open demo cases: %v", err)
	}
	defer f.Close()
	cases, skipped, err := loadCases(f)
	if err != nil || skipped != 0 || len(cases) != 3 {
		t.Fatalf("demo cases should be clean: cases=%d skipped=%d err=%v", len(cases), skipped, err)
	}

	verdicts := replayAll(context.Background(), newDemoRunner(), cases)
	if len(verdicts) != 3 {
		t.Fatalf("want 3 verdicts, got %d", len(verdicts))
	}
	for i, v := range verdicts {
		if !v.passed() {
			t.Errorf("demo case %d should pass: %+v", i, v)
		}
	}

	report := buildReplayReport(verdicts, 0)
	for _, want := range []string{"回放样本：3", "通过（answered 且 not_repeated）：3/3", "各维度命中"} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q:\n%s", want, report)
		}
	}
}

func TestBuildReplayReportListsFailures(t *testing.T) {
	v := judge(replayCase{Timestamp: "t1", PreviousUserInput: "原问题", AdviceSummary: "坏回答"},
		"", context.DeadlineExceeded)
	report := buildReplayReport([]verdict{v}, 1)
	for _, want := range []string{"未通过样本", "原问题", "回放失败", "1 条无法回放已跳过"} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q:\n%s", want, report)
		}
	}
}
