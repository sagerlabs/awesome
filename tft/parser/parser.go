package parser

import (
	"github.com/sagerlabs/awesome/tft/data"
	"strings"
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
		aliases: defaultAliases(),
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

// defaultAliases 内置的民间外号映射表
// 版本更新时按需补充，无需重新爬取数据
func defaultAliases() map[string]string {
	return map[string]string{
		// 英雄外号
		"瞎子": "盲僧",
		"蛐蛐": "螳螂",
		"猴子": "孙悟空",
		"小炮": "崔斯特",
		"兔子": "璐璐",
		"熊":  "波比",
		"小鱼": "提莫",
		"企鹅": "小布",
		"轮子": "兰博",
		"电球": "肯能",

		// 装备外号
		"大帽子":   "拉巴顿死亡之帽",
		"帽子":    "拉巴顿死亡之帽",
		"破败":    "古神狂暴之刃",
		"鬼书":    "古神狂暴之刃",
		"蓝buff": "蓝色电池",
		"反甲":    "冰霜之心",
		"日炎":    "日炎圣盾",
		"鞋":     "疾步之靴",
	}
}
