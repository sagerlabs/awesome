# TFT Knowledge Tool 接口文档

## 概述

TFT Knowledge Tool 是一个独立的知识库工具接口，设计用于：
- 作为独立的 tool/skill 使用
- 可以分割部署
- 同时支持原有数据和新的 Meta 数据
- 提供清晰的输入输出接口

## 快速开始

### 初始化

```go
import (
    "github.com/sagerlabs/awesome/tft/data"
    "github.com/sagerlabs/awesome/tft/knowledge"
)

// 1. 初始化原有数据Store
dataStore, err := data.NewStore("metadata/tft-meta/data")
if err != nil {
    panic(err)
}

// 2. 初始化Knowledge Store
knowledgeStore, err := knowledge.NewLoader("tft/knowledge/data").LoadAll()
if err != nil {
    panic(err)
}

// 3. 创建统一Tool
config := knowledge.DefaultToolConfig()
tool, err := knowledge.NewUnifiedStore(dataStore, knowledgeStore, config)
if err != nil {
    panic(err)
}
```

### 核心使用 - NLU查询

```go
import "github.com/sagerlabs/awesome/tft/agent"

// 创建Context（通常由LLM生成）
ctx := agent.Context{
    Intent:    "lineup_recommend",
    Champions: map[string]int8{"兰博": 2},
    Items:     []string{"鬼索的狂暴之刃"},
}

// 查询数据
result, err := tool.QueryNLU(ctx)
if err != nil {
    panic(err)
}

// 使用结果
fmt.Printf("匹配阵容: %d\n", len(result.MatchedComps))
fmt.Printf("匹配装备: %d\n", len(result.MatchedItems))
```

## 接口方法

### 核心方法

#### QueryNLU
NLU数据查询，这是最核心的方法。

**输入:** `agent.Context`
- Intent: 意图类型
- Champions: 英雄地图
- Items: 装备列表
- Traits: 羁绊列表
- 等等...

**输出:** `*agent.NluEnrichedContext`
- MatchedComps: 匹配的阵容
- MatchedItems: 匹配的装备信息
- 等等...

**示例:**
```go
result, err := tool.QueryNLU(ctx)
```

---

### 阵容查询

#### GetCompByClusterID
通过ClusterID查询阵容（原有数据格式）

**参数:**
- clusterID: 阵容ID，如 "394014"

**返回:**
- `*data.Comp`: 阵容数据
- `bool`: 是否找到

**示例:**
```go
comp, ok := tool.GetCompByClusterID("394014")
```

#### GetMetaCompByID
通过ClusterID查询Meta阵容（新数据格式）

**参数:**
- clusterID: 阵容ID

**返回:**
- `*models.MetaComp`: Meta阵容数据
- `bool`: 是否找到

**示例:**
```go
metaComp, ok := tool.GetMetaCompByID("394014")
```

#### GetMetaCompByName
通过名称查询Meta阵容

**参数:**
- name: 阵容名称，如 "约德尔人"

**返回:**
- `*models.MetaComp`: Meta阵容数据
- `bool`: 是否找到

**示例:**
```go
metaComp, ok := tool.GetMetaCompByName("约德尔人")
```

#### SearchMetaComps
搜索Meta阵容（关键词搜索）

**参数:**
- query: 搜索关键词

**返回:**
- `[]*models.MetaComp`: 匹配的阵容列表

**示例:**
```go
comps := tool.SearchMetaComps("约德尔")
```

#### GetAllMetaComps
获取所有Meta阵容

**返回:**
- `[]*models.MetaComp`: 所有Meta阵容列表

**示例:**
```go
allComps := tool.GetAllMetaComps()
```

#### GetCompsByUnits
通过英雄列表查询阵容（原有数据格式）

**参数:**
- unitIDs: 英雄ID列表

**返回:**
- `[]data.CompMatch`: 匹配的阵容列表

**示例:**
```go
matches := tool.GetCompsByUnits([]string{"TFT16_Rumble"})
```

#### GetCompsByTier
通过Tier查询阵容（原有数据格式）

**参数:**
- tiers: Tier列表，如 "S", "A"

**返回:**
- `[]*data.Comp`: 阵容列表

**示例:**
```go
comps := tool.GetCompsByTier("S", "A")
```

---

### 英雄查询

#### GetMetaChampionByName
通过名称查询Meta英雄

**参数:**
- name: 英雄名称，如 "兰博"

**返回:**
- `*models.MetaChampion`: Meta英雄数据
- `bool`: 是否找到

**示例:**
```go
champ, ok := tool.GetMetaChampionByName("兰博")
```

#### GetAllMetaChampions
获取所有Meta英雄

**返回:**
- `[]*models.MetaChampion`: 所有Meta英雄列表

**示例:**
```go
allChamps := tool.GetAllMetaChampions()
```

#### GetChampionBestBuild
获取英雄最佳装备（原有数据格式）

**参数:**
- clusterID: 阵容ID
- unitName: 英雄名称

**返回:**
- `*data.BuildInfo`: 装备信息
- `bool`: 是否找到

**示例:**
```go
build, ok := tool.GetChampionBestBuild("394014", "兰博")
```

---

### 装备查询

#### GetMetaItemByName
通过名称查询Meta装备

**参数:**
- name: 装备名称，如 "鬼索的狂暴之刃"

**返回:**
- `*models.MetaItem`: Meta装备数据
- `bool`: 是否找到

**示例:**
```go
item, ok := tool.GetMetaItemByName("鬼索的狂暴之刃")
```

#### GetAllMetaItems
获取所有Meta装备

**返回:**
- `[]*models.MetaItem`: 所有Meta装备列表

**示例:**
```go
allItems := tool.GetAllMetaItems()
```

#### GetItemFitEntries
查询装备适配的阵容（原有数据格式）

**参数:**
- itemID: 装备ID

**返回:**
- `[]data.ItemFitEntry`: 适配阵容列表

**示例:**
```go
entries := tool.GetItemFitEntries("TFT_Item_GuinsoosRageblade")
```

#### GetItemFitByItems
批量装备查询（原有数据格式）

**参数:**
- itemIDs: 装备ID列表

**返回:**
- `[]data.ItemFitResult`: 适配结果列表

**示例:**
```go
results := tool.GetItemFitByItems([]string{"TFT_Item_GuinsoosRageblade"})
```

---

### 名称解析和转换

#### ResolveUnitID
解析英雄输入：支持中文名、ID、模糊匹配

**参数:**
- input: 输入字符串

**返回:**
- `string`: 标准TFT ID，找不到返回空

**示例:**
```go
unitID := tool.ResolveUnitID("兰博")
// 返回: "TFT16_Rumble"
```

#### ResolveItemID
解析装备输入：支持中文名、ID、模糊匹配

**参数:**
- input: 输入字符串

**返回:**
- `string`: 标准TFT ID，找不到返回空

**示例:**
```go
itemID := tool.ResolveItemID("羊刀")
// 返回: "TFT_Item_GuinsoosRageblade"
```

#### IDToCN
将ID转换为中文名

**参数:**
- id: TFT ID

**返回:**
- `string`: 中文名，找不到返回原ID

**示例:**
```go
cnName := tool.IDToCN("TFT16_Rumble")
// 返回: "兰博"
```

#### CNToID
将中文名转换为ID

**参数:**
- cn: 中文名

**返回:**
- `string`: TFT ID，找不到返回原字符串

**示例:**
```go
id := tool.CNToID("兰博")
// 返回: "TFT16_Rumble"
```

---

### 数据管理

#### Reload
重新加载数据（热更新）

**返回:**
- `error`: 错误信息

**示例:**
```go
err := tool.Reload()
```

#### HealthCheck
健康检查

**返回:**
- `error`: 错误信息，nil表示健康

**示例:**
```go
err := tool.HealthCheck()
if err != nil {
    fmt.Printf("健康检查失败: %v\n", err)
}
```

---

## 配置

### ToolConfig

```go
type ToolConfig struct {
    // 数据目录
    DataDir      string // 原有数据目录，默认: "metadata/tft-meta/data"
    KnowledgeDir string // Knowledge数据目录，默认: "tft/knowledge/data"

    // 是否启用Meta数据
    EnableMeta bool // 默认: true

    // 日志配置
    EnableLog bool   // 是否启用日志，默认: true
    LogLevel  string // 日志级别，默认: "info"
}
```

### 默认配置

```go
config := knowledge.DefaultToolConfig()
```

---

## 数据模型

### MetaComp
Meta阵容数据（来自拆分后的JSON）

```go
type MetaComp struct {
    ClusterID    string                 // 阵容ID
    TFTSet       string                 // TFT赛季
    Units        []string               // 英雄列表
    Traits       []string               // 羁绊列表
    NameString   string                 // 名称字符串
    DisplayNames []DisplayName          // 显示名称列表
    Count        int                    // 样本数
    AvgPlacement float64                // 平均名次
    Top4Rate     float64                // 前4率
    WinRate      float64                // 胜率
    Tier         string                 // 等级: S/A/B/C
    Builds       []CompBuild            // 出装方案
    Trends       []Trend                // 趋势数据
    Levelling    string                 // 升级节点
    Difficulty   float64                // 难度
    
    // 预留字段
    Description  string                 // 描述
    Limit        map[string]interface{} // 限制条件
}
```

### MetaChampion
Meta英雄数据

```go
type MetaChampion struct {
    Name          string                 // 英雄名称
    AppearInComps []CompAppearance       // 出现的阵容列表
    Builds        []ChampionBuild        // 出装方案
    
    // 预留字段
    Description   string                 // 描述
    Limit         map[string]interface{} // 限制条件
}
```

### MetaItem
Meta装备数据

```go
type MetaItem struct {
    Name         string                 // 装备名称
    PriorityList []ItemPriority         // 优先级列表
    
    // 预留字段
    Description  string                 // 描述
    Limit        map[string]interface{} // 限制条件
}
```

---

## 使用场景

### 场景1: 阵容推荐

```go
// 用户输入: "兰博 鬼索"
ctx := agent.Context{
    Intent:    "lineup_recommend",
    Champions: map[string]int8{"兰博": 1},
    Items:     []string{"鬼索的狂暴之刃"},
}

result, _ := tool.QueryNLU(ctx)

for _, comp := range result.MatchedComps {
    fmt.Printf("推荐阵容: %s (Tier: %s)\n", comp.Name, comp.Tier)
}
```

### 场景2: 装备查询

```go
// 查询某个装备适合的阵容
item, ok := tool.GetMetaItemByName("鬼索的狂暴之刃")
if ok {
    for _, priority := range item.PriorityList {
        fmt.Printf("适合: %s (Carry: %s)\n", priority.CompName, priority.Carry)
    }
}
```

### 场景3: 英雄出装查询

```go
// 查询某个英雄的出装
champ, ok := tool.GetMetaChampionByName("兰博")
if ok {
    for _, build := range champ.Builds {
        fmt.Printf("出装: %v\n", build.Items)
    }
}
```

---

## 错误处理

所有方法都遵循Go的错误处理惯例：

```go
result, err := tool.QueryNLU(ctx)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}
```

常见错误：
- 数据加载失败
- 数据文件不存在
- 健康检查失败

---

## 线程安全

`UnifiedStore` 使用 `sync.RWMutex` 保证线程安全：
- 读操作使用 `RLock()`
- 写操作使用 `Lock()`

可以在多个goroutine中安全使用。

---

## 扩展性

接口设计支持后续扩展：

1. **新增查询方法** - 在接口中添加新方法
2. **替换数据源** - 创建新的 `TFTKnowledgeTool` 实现
3. **添加缓存层** - 包装现有实现添加缓存
4. **A/B测试** - 同时运行多个实现对比效果

---

## 相关文件

- `tft/knowledge/tool.go` - 接口定义
- `tft/knowledge/unified_store.go` - 统一实现
- `tft/knowledge/models/meta.go` - Meta数据模型
- `docs/context-knowledge-connection.md` - 架构说明
