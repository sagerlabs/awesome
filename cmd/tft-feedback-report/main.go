// tft-feedback-report 汇总教练反馈闭环落盘的 rejected 样本（JSONL），
// 生成一份可人工复盘的报告：按日期/类型统计 + 最近样本明细。
//
// 这是 ADR-010 "rejected case 进入 Eval 候选" 的消费端：
//
//	go run ./cmd/tft-feedback-report                    # 读默认 data/feedback_cases.jsonl
//	go run ./cmd/tft-feedback-report -file path.jsonl   # 指定文件
//	go run ./cmd/tft-feedback-report -recent 20         # 明细条数
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	file := flag.String("file", "data/feedback_cases.jsonl", "feedback JSONL 文件路径")
	recent := flag.Int("recent", 10, "报告末尾列出最近 N 条样本明细")
	flag.Parse()

	f, err := os.Open(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tft-feedback-report: 打开 %s 失败: %v\n", *file, err)
		fmt.Fprintln(os.Stderr, "提示：该文件由服务在用户否定建议时追加生成；没有文件说明还没有 rejected 样本。")
		os.Exit(1)
	}
	defer f.Close()

	records, skipped, err := readRecords(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tft-feedback-report: 读取失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(buildReport(records, skipped, *recent))
}
