package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// 评测维度来自 ADR-010 Phase 4 的定义。规则判定只覆盖能机械验证的部分：
//
//	answered     回放没有报错且产出非空回答
//	acknowledged 新回答承认上一轮可能没命中（出现纠偏开场词）
//	not_repeated 新回答没有照搬当时的坏回答（与 advice_summary 的字符重叠率低）
//	actionable   新回答包含可执行指引（行动类措辞）
//
// "回答质量是否真的更好"无法用规则判定，那一档留给人工抽查或后续 LLM judge。
type verdict struct {
	Case         replayCase
	NewAnswer    string
	Err          error
	Answered     bool
	Acknowledged bool
	NotRepeated  bool
	Actionable   bool
}

func (v verdict) passed() bool {
	return v.Answered && v.NotRepeated
}

var acknowledgeMarkers = []string{
	"没命中", "重新", "刚才的回答", "上一轮", "之前的建议", "换个角度", "纠正",
}

var actionMarkers = []string{
	"建议", "优先", "下一步", "先", "可以", "应该", "推荐",
}

func judge(c replayCase, answer string, err error) verdict {
	v := verdict{Case: c, NewAnswer: answer, Err: err}
	if err != nil || strings.TrimSpace(answer) == "" {
		return v
	}
	v.Answered = true
	v.Acknowledged = containsAny(answer, acknowledgeMarkers)
	v.Actionable = containsAny(answer, actionMarkers)
	// 与坏回答的重叠率 < 0.6 视为没有照搬。摘要可能被截断过，
	// 所以用字符级 bigram 重叠而不是整句相等。
	v.NotRepeated = overlapRatio(answer, c.AdviceSummary) < 0.6
	return v
}

func containsAny(text string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

// overlapRatio 计算 old 的字符 bigram 有多大比例出现在 new 里。
// old 为空时返回 0（没有基线可比，视为未重复）。
func overlapRatio(newText, oldText string) float64 {
	oldGrams := bigrams(oldText)
	if len(oldGrams) == 0 {
		return 0
	}
	newGrams := bigrams(newText)
	hit := 0
	for g := range oldGrams {
		if newGrams[g] {
			hit++
		}
	}
	return float64(hit) / float64(len(oldGrams))
}

func bigrams(s string) map[string]bool {
	runes := []rune(strings.TrimSpace(s))
	grams := make(map[string]bool)
	for i := 0; i+1 < len(runes); i++ {
		grams[string(runes[i:i+2])] = true
	}
	return grams
}

// buildReplayReport 汇总所有判定结果为人工可读报告。
func buildReplayReport(verdicts []verdict, skipped int) string {
	var sb strings.Builder
	sb.WriteString("# 自动回放评测报告\n\n")
	sb.WriteString(fmt.Sprintf("回放样本：%d", len(verdicts)))
	if skipped > 0 {
		sb.WriteString(fmt.Sprintf("（另有 %d 条无法回放已跳过）", skipped))
	}
	sb.WriteString("\n")

	if len(verdicts) == 0 {
		sb.WriteString("\n没有可回放的样本。\n")
		return sb.String()
	}

	counts := map[string]int{}
	for _, v := range verdicts {
		if v.passed() {
			counts["passed"]++
		}
		for name, ok := range map[string]bool{
			"answered": v.Answered, "acknowledged": v.Acknowledged,
			"not_repeated": v.NotRepeated, "actionable": v.Actionable,
		} {
			if ok {
				counts[name]++
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\n通过（answered 且 not_repeated）：%d/%d\n", counts["passed"], len(verdicts)))
	sb.WriteString("\n## 各维度命中\n\n")
	for _, name := range []string{"answered", "acknowledged", "not_repeated", "actionable"} {
		sb.WriteString(fmt.Sprintf("- %s：%d/%d\n", name, counts[name], len(verdicts)))
	}

	// 失败明细放前面：报告的目的就是找问题
	var failed []verdict
	for _, v := range verdicts {
		if !v.passed() {
			failed = append(failed, v)
		}
	}
	sort.Slice(failed, func(i, j int) bool {
		return failed[i].Case.Timestamp < failed[j].Case.Timestamp
	})
	if len(failed) > 0 {
		sb.WriteString("\n## 未通过样本\n")
		for _, v := range failed {
			sb.WriteString(fmt.Sprintf("\n### %s\n", v.Case.Timestamp))
			sb.WriteString(fmt.Sprintf("- 原问题：%s\n", v.Case.PreviousUserInput))
			sb.WriteString(fmt.Sprintf("- 当时坏回答：%s\n", v.Case.AdviceSummary))
			if v.Err != nil {
				sb.WriteString(fmt.Sprintf("- 回放失败：%v\n", v.Err))
				continue
			}
			sb.WriteString(fmt.Sprintf("- 新回答摘要：%s\n", truncateRunes(v.NewAnswer, 120)))
			sb.WriteString(fmt.Sprintf("- 判定：answered=%v acknowledged=%v not_repeated=%v actionable=%v\n",
				v.Answered, v.Acknowledged, v.NotRepeated, v.Actionable))
		}
	}
	return sb.String()
}

func truncateRunes(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "..."
}
