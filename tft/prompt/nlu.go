package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed prompt_nlu.tmpl
var promptFS embed.FS

type PromptData struct {
	Input string
}

var nluTemplate *template.Template

func init() {
	var err error
	nluTemplate, err = template.ParseFS(promptFS, "prompt_nlu.tmpl")
	if err != nil {
		panic(fmt.Sprintf("初始化 NLU Prompt 模板失败: %v", err))
	}
}

// BuildNLUPrompt 高性能渲染函数
func BuildNLUPrompt(userInput string) (string, error) {
	data := PromptData{
		Input: userInput,
	}

	var buf bytes.Buffer
	if err := nluTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("执行 NLU 模板渲染失败: %w", err)
	}

	return buf.String(), nil
}
