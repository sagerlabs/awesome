package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// record 镜像 tft/agent 写入 JSONL 的字段。文件格式即契约：
// 字段名与 agent.feedbackRecord 的 json tag 保持一致。
type record struct {
	Timestamp         string `json:"timestamp"`
	Date              string `json:"date"`
	Type              string `json:"type"`
	UserInput         string `json:"user_input"`
	PreviousUserInput string `json:"previous_user_input"`
	AdviceSummary     string `json:"advice_summary"`
	Reason            string `json:"reason"`
}

// readRecords 逐行解析 JSONL。坏行跳过并计数而不是中断：
// 报告工具的职责是尽量呈现数据，少量损坏不应让整个复盘失败。
func readRecords(r io.Reader) (records []record, skipped int, err error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec record
		if json.Unmarshal([]byte(line), &rec) != nil || rec.UserInput == "" {
			skipped++
			continue
		}
		records = append(records, rec)
	}
	return records, skipped, scanner.Err()
}

// buildReport 生成纯文本报告：总量、按日期分布、最近明细。
func buildReport(records []record, skipped int, recentN int) string {
	var sb strings.Builder
	sb.WriteString("# Coach Feedback 复盘报告\n\n")
	sb.WriteString(fmt.Sprintf("样本总数：%d", len(records)))
	if skipped > 0 {
		sb.WriteString(fmt.Sprintf("（另有 %d 条损坏行已跳过）", skipped))
	}
	sb.WriteString("\n")

	if len(records) == 0 {
		sb.WriteString("\n还没有 rejected 样本。说明回答都被接受，或者流量还不够。\n")
		return sb.String()
	}

	// 按日期统计，日期升序输出
	byDate := make(map[string]int)
	for _, rec := range records {
		byDate[rec.Date]++
	}
	dates := make([]string, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	sb.WriteString("\n## 按日期分布\n\n")
	for _, d := range dates {
		sb.WriteString(fmt.Sprintf("- %s：%d 条\n", d, byDate[d]))
	}

	// 最近 N 条明细（文件是追加写入的，末尾即最新）
	if recentN > len(records) {
		recentN = len(records)
	}
	sb.WriteString(fmt.Sprintf("\n## 最近 %d 条样本（复盘清单）\n", recentN))
	for _, rec := range records[len(records)-recentN:] {
		sb.WriteString(fmt.Sprintf("\n### %s\n", rec.Timestamp))
		sb.WriteString(fmt.Sprintf("- 上轮问题：%s\n", rec.PreviousUserInput))
		sb.WriteString(fmt.Sprintf("- 上轮回答摘要：%s\n", rec.AdviceSummary))
		sb.WriteString(fmt.Sprintf("- 用户否定：%s\n", rec.UserInput))
		if rec.Reason != "" {
			sb.WriteString(fmt.Sprintf("- 判定原因：%s\n", rec.Reason))
		}
		sb.WriteString("- 复盘结论：待填写（命中问题/知识缺失/表达问题）\n")
	}

	return sb.String()
}
