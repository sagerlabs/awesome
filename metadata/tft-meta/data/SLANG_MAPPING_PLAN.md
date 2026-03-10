# 黑话与官方昵称对照表设计计划

## 背景
用户输入通常包含大量游戏黑话和昵称，NLU提取出来的内容也是黑话形式，需要转换成官方标准名称才能正确检索。

## 问题示例
| 类型 | 用户输入/黑话 | 官方名称/ID |
|------|--------------|------------|
| 英雄 | "女枪" | "厄运小姐" / "TFT16_MissFortune" |
| 英雄 | "羊刀" | "鬼索的狂暴之刃" / "TFT_Item_GuinsoosRageblade" |
| 英雄 | "红buff" | "红色增益" / "TFT_Item_RedBuff" |
| 羁绊 | "九五" | (需要映射到对应阵容) |

## 对照表设计

### 文件结构
```json
{
  "version": "1.0",
  "updated_at": "2026-03-10",
  "heroes": { ... },
  "items": { ... },
  "traits": { ... },
  "lineups": { ... }
}
```

### 字段说明

#### heroes - 英雄黑话映射
```json
{
  "heroes": {
    "女枪": ["厄运小姐", "TFT16_MissFortune"],
    "金克丝": ["金克丝", "TFT16_Jinx"],
    "jinx": ["金克丝", "TFT16_Jinx"],
    "女警": ["凯特琳", "TFT16_Caitlyn"],
    "寒冰": ["艾希", "TFT16_Ashe"],
    "奶妈": ["索拉卡", "TFT16_Soraka"],
    "龙王": ["索尔", "TFT16_AurelionSol"]
  }
}
```
- key: 黑话/昵称（支持大小写不敏感匹配）
- value[0]: 官方中文名
- value[1]: 官方ID

#### items - 装备黑话映射
```json
{
  "items": {
    "羊刀": ["鬼索的狂暴之刃", "TFT_Item_GuinsoosRageblade"],
    "羊": ["鬼索的狂暴之刃", "TFT_Item_GuinsoosRageblade"],
    "红buff": ["红色增益", "TFT_Item_RedBuff"],
    "红爸爸": ["红色增益", "TFT_Item_RedBuff"],
    "血手": ["斯塔提克的缚炉之斧", "TFT_Item_..."],
    "青龙刀": ["朔极之矛", "TFT_Item_SpearOfShojin"],
    "无尽": ["无尽之刃", "TFT_Item_InfinityEdge"]
  }
}
```

#### traits - 羁绊黑话映射
```json
{
  "traits": {
    "九五": ["九五至尊", "TFT16_..."],
    "法师": ["法师", "TFT16_Sorcerer"],
    "枪手": ["枪手", "TFT16_Gunslinger"]
  }
}
```

#### lineups - 阵容黑话映射
```json
{
  "lineups": {
    "梅尔九五": ["梅尔九五", "cluster_id": "393000"},
    "九五": ["九五至尊", "cluster_ids": ["393000", "393001"]]
  }
}
```

## 实现计划

### Phase 1: 基础映射函数
在 `tft/data/` 下新增 `slang.go`：

```go
package data

type SlangMapping struct {
    Version   string              `json:"version"`
    Heroes  map[string][]string  `json:"heroes"`
    Items   map[string][]string  `json:"items"`
    Traits  map[string][]string  `json:"traits"`
    Lineups map[string]LineupMap `json:"lineups"`
}

type LineupMap struct {
    Name       string   `json:"name"`
    ClusterIDs []string `json:"cluster_ids"`
}

// SlangMapper 黑话映射器
type SlangMapper struct {
    mapping SlangMapping
}

func NewSlangMapper(dataDir string) (*SlangMapper, error)
func (m *SlangMapper) NormalizeHero(input string) (string, string) // 返回中文名, ID
func (m *SlangMapper) NormalizeItem(input string) (string, string)
func (m *SlangMapper) NormalizeTrait(input string) (string, string)
func (m *SlangMapper) NormalizeAll(ctx *Context) *Context
```

### Phase 2: 集成到NLU流程
在 `agent/NluAnalyze() 调用后，对提取结果进行标准化：

```go
result, err := a.nluRunnable.Invoke(...)
normalizedCtx := slangMapper.NormalizeAll(&result.Ctx)
```

### Phase 3: 数据收集
- 收集常用黑话
- 持续更新SLANG_MAPPING.json
- 支持社区贡献

## 现阶段方案
在黑话对照表完成前，先使用以下方案：

1. 用户输入标准名称（官方中文名或ID）
2. 利用现有的 `store.ResolveUnitID() 和 `store.ResolveItemID()` 进行解析
3. NLU提取结果先尝试用现有localization.json进行中英转换
