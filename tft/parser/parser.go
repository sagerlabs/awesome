package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sagerlabs/awesome/tft/data"
)

// InputParser 用户输入标准化
// 职责：别名解析 → 中文名 → TFT ID
// 无法识别的 token 放入 Unknown，由 LLM 兜底
type InputParser struct {
	store   *data.Store
	aliases map[string]string // 民间外号 -> 中文名，如 "瞎子" -> "盲僧"
}

func NewInputParser(store *data.Store) *InputParser {
	return &InputParser{
		store:   store,
		aliases: loadAliases(),
	}
}

// Parse 解析原始输入，返回标准化的 UserInput
//
// 支持格式：
//
//	"兰博 肯能 鬼索的狂暴之刃"
//	"瞎子，破败，大帽子"
//	"TFT16_Rumble TFT_Item_Rabadons"
func (p *InputParser) Parse(raw string) (*data.UserInput, error) {
	// 统一分隔符：中文逗号/顿号/空格都当分隔符
	replacer := strings.NewReplacer("，", " ", "、", " ", ",", " ", "\t", " ")
	normalized := replacer.Replace(raw)
	tokens := splitTokens(normalized)

	result := &data.UserInput{Raw: raw}

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		// 1. 别名转中文名
		if cn, ok := p.aliases[token]; ok {
			token = cn
		}

		// 2. 尝试解析为英雄
		if unitID := p.store.ResolveUnitID(token); unitID != "" {
			result.Heroes = append(result.Heroes, unitID)
			continue
		}

		// 3. 尝试解析为装备
		if itemID := p.store.ResolveItemID(token); itemID != "" {
			result.Items = append(result.Items, itemID)
			continue
		}

		// 4. 无法识别，放入 Unknown 交 LLM 兜底
		result.Unknown = append(result.Unknown, token)
	}

	return result, nil
}

// splitTokens 分词：按空格切分，过滤空串
func splitTokens(s string) []string {
	parts := strings.Fields(s)
	var tokens []string
	for _, p := range parts {
		if p != "" {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

type aliasesFile struct {
	Heroes map[string]string `json:"heroes"`
	Items  map[string]string `json:"items"`
	Traits map[string]string `json:"traits"`
}

// loadAliases 优先读取 knowledge 侧的 aliases.json，让旧 analyze 链路和新 NLU 链路共享同一套黑话表。
func loadAliases() map[string]string {
	aliases := make(map[string]string)

	path := os.Getenv("TFT_ALIASES_FILE")
	if path == "" {
		path = filepath.Join("tft", "knowledge", "data", "aliases.json")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return aliases
	}

	var file aliasesFile
	if err := json.Unmarshal(b, &file); err != nil {
		return aliases
	}
	mergeAliases(aliases, file.Heroes)
	mergeAliases(aliases, file.Items)
	mergeAliases(aliases, file.Traits)
	return aliases
}

func mergeAliases(dst map[string]string, src map[string]string) {
	for raw, normalized := range src {
		raw = strings.TrimSpace(raw)
		normalized = strings.TrimSpace(normalized)
		if raw == "" || normalized == "" {
			continue
		}
		dst[raw] = normalized
	}
}
