# TFT Copilot

基于 [Eino](https://github.com/cloudwego/eino) 框架构建的云顶之弈 AI 阵容推荐助手。输入你当前棋盘上的英雄和装备，自动分析最优阵容方向并给出运营建议。

---

## 功能特性

- **智能阵容推荐** — 根据已有英雄和装备，匹配当前版本强势阵容
- **三路并行分析** — 英雄匹配 + 装备适配 + 版本强度同步计算，交集越多置信度越高
- **中文输入支持** — 支持中文名、民间外号（"瞎子"、"大帽子"）、TFT 标准 ID 混合输入
- **流式输出** — SSE 流式推送 LLM 建议，打字机效果实时展示
- **多 LLM 支持** — OpenAI / DeepSeek / 火山引擎豆包，环境变量一键切换
- **数据自动更新** — 配套 Python 爬虫从 MetaTFT 拉取最新阵容数据，每个版本跑一次即可

---

## 项目结构

```
awesome/
├── metadata/
│   └── tft-meta/
│       ├── get_tftmeta.py         # MetaTFT 数据爬虫
│       ├── test.py                 # 汉化数据调试工具
│       └── data/
│           ├── comps_full.json         # 完整原始数据（调试用）
│           ├── comps_for_agent.json    # 阵容数据（Eino Tool 使用）
│           ├── items_priority.json     # 装备优先级索引
│           └── localization.json       # 英雄/装备中英文映射
├── tft/
│   ├── handler.go              # HTTP 路由入口
│   ├── middleware.go           # HTTP 中间件
│   ├── logger.go               # 日志工具
│   ├── data/
│   │   ├── types.go            # 所有结构体定义
│   │   └── loader.go           # Store 数据加载与查询
│   ├── parser/
│   │   └── parser.go           # 用户输入标准化（别名/中文名/ID）
│   ├── tool/
│   │   ├── hero_comp.go        # Tool1：英雄 → 推荐阵容
│   │   ├── item_fit.go         # Tool2：装备 → 适配阵容
│   │   ├── comps_tier.go       # Tool3：阵容强度 + 交集计算
│   │   └── priority_recommend.go  # Tool4：优先级推荐
│   ├── prompt/
│   │   └── nlu.go              # NLU 提示词
│   ├── knowledge/
│   │   └── models/
│   │       ├── knowledge.go    # 知识库基础
│   │       ├── champion.go     # 英雄知识模型
│   │       ├── item.go         # 装备知识模型
│   │       ├── trait.go        # 羁绊知识模型
│   │       └── team_comp.go    # 阵容知识模型
│   ├── agent/
│   │   ├── agent.go            # 对外入口
│   │   ├── model.go            # LLM 初始化（多 Provider）
│   │   ├── graph.go            # Eino Graph 编排
│   │   ├── prompt.go           # Agent 提示词
│   │   ├── context.go          # 上下文管理
│   │   ├── trace.go            # 追踪工具
│   │   ├── token_usage.go      # Token 使用统计
│   │   ├── nlu_data_query.go   # NLU 数据查询
│   │   └── agent_test.go       # Agent 测试
│   ├── trace/
│   │   └── trace.go            # 分布式追踪
│   └── sse/
│       ├── sse.go              # SSE 框架核心
│       ├── sse_test.go         # SSE 测试
│       └── examples/           # SSE 示例代码
├── desktop/
│   └── main.go                 # 桌面端入口
├── tests/                      # 测试目录
├── scripts/                    # 脚本目录
├── docs/                       # 文档目录
├── frontend/                   # 前端目录
├── main.go                     # 主入口
├── Makefile                    # 构建脚本
└── go.mod
```

---

## 快速开始

### 1. 准备数据

```bash
cd metadata/tft-meta

# 安装依赖
pip install requests

# 拉取最新阵容数据 + 生成汉化表（每个版本跑一次）
python get_tftmeta.py
```

运行后 `data/` 目录会生成四个文件：

```
data/
├── comps_full.json         # 完整原始数据（调试用）
├── comps_for_agent.json    # Eino Tool 使用的精简版
├── items_priority.json     # 装备优先级索引
└── localization.json       # 中文名映射表
```

### 2. 配置环境变量

根据你使用的 LLM Provider 选择对应配置：

**DeepSeek（推荐，中文效果好，价格低）**
```bash
export LLM_PROVIDER=deepseek
export OPENAI_API_KEY=sk-xxx
export OPENAI_BASE_URL=https://api.deepseek.com
export OPENAI_MODEL=deepseek-chat       # 可选，默认 deepseek-chat
```

**OpenAI**
```bash
export LLM_PROVIDER=openai
export OPENAI_API_KEY=sk-xxx
export OPENAI_MODEL=gpt-4o-mini         # 可选，默认 gpt-4o-mini
```

**火山引擎豆包（字节跳动，与 Eino 生态最配）**
```bash
export LLM_PROVIDER=ark
export ARK_API_KEY=xxx
export ARK_MODEL_ID=doubao-pro-32k-xxxxxx   # 推理接入点 ID，非模型名称
```

**数据目录（可选）**
```bash
export TFT_DATA_DIR=./metadata/tft-meta/data   # 默认值，一般无需修改
export PORT=8080                                # 默认 8080
```

### 3. 启动服务

```bash
go run main.go
```

启动成功后输出：

```
✅ TFT Copilot 初始化完成
🚀 服务启动: http://localhost:8080
────────────────────────────────
  POST /v1/tft/analyze         普通接口
  POST /v1/tft/analyze/stream  流式接口（SSE）
  GET  /v1/tft/health          健康检查
────────────────────────────────
```

---

## API 接口

### POST /v1/tft/analyze

普通接口，返回完整推荐结果。

**请求**
```json
{
  "input": "兰博 肯能 鬼索的狂暴之刃 大帽子"
}
```

**响应**
```json
{
  "success": true,
  "data": {
    "recommendations": [
      {
        "comp": {
          "name": "TFT16_Augment_RumbleCarry",
          "tier": "S",
          "avg_placement": 3.72,
          "top4_rate": 0.61
        },
        "confidence": 0.95,
        "confidence_desc": "高",
        "hit_sources": ["hero", "item", "tier"],
        "matched_units": ["兰博", "肯能"],
        "missing_units": ["璐璐", "波比", "提莫"],
        "matched_items": ["古神狂暴之刃", "死亡之帽"],
        "suggested_carry": "兰博"
      }
    ],
    "llm_advice": "当前棋盘非常适合走约德尔小炮路线..."
  }
}
```

### POST /v1/tft/analyze/stream

流式接口，SSE 格式逐 token 推送 LLM 建议。

**请求**（同普通接口）
```json
{
  "input": "瞎子 破败 大帽子"
}
```

**SSE 事件流**
```
: connected

event: message
data: {"type":"token","content":"当前"}

event: message
data: {"type":"token","content":"棋盘"}

event: done
data: {"type":"done"}
```

**前端接入示例**
```javascript
const resp = await fetch('/tft/analyze/stream', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ input: '兰博 肯能 大帽子' })
})

const reader = resp.body.getReader()
const decoder = new TextDecoder()

// 逐块读取 SSE 数据
while (true) {
  const { done, value } = await reader.read()
  if (done) break

  const text = decoder.decode(value)
  const lines = text.split('\n')

  for (const line of lines) {
    if (!line.startsWith('data:')) continue
    const chunk = JSON.parse(line.slice(5).trim())
    if (chunk.type === 'token') appendText(chunk.content)
    if (chunk.type === 'done') break
  }
}
```

### GET /tft/health

健康检查，服务正常时返回 `{"status":"ok"}`。

---

## 输入格式说明

支持多种输入方式，混用也没问题：

| 输入类型 | 示例 |
|---|---|
| 中文名 | `兰博 肯能 璐璐` |
| 民间外号 | `瞎子 大帽子 破败` |
| 标准 ID | `TFT16_Rumble TFT_Item_Rabadons` |
| 混合输入 | `兰博 TFT16_Kennen 大帽子` |
| 中文逗号分隔 | `兰博，肯能，鬼索的狂暴之刃` |

内置别名表（`tool/parser.go`，可按需扩充）：

| 外号 | 对应英雄/装备 |
|---|---|
| 瞎子 | 盲僧 |
| 轮子 | 兰博 |
| 电球 | 肯能 |
| 大帽子 | 拉巴顿死亡之帽 |
| 破败 | 古神狂暴之刃 |
| 蓝buff | 蓝色电池 |

---

## 推荐置信度说明

| 命中来源 | 置信度 | 含义 |
|---|---|---|
| 英雄 + 装备 + Tier | 90~95% 高 | 三路全部命中，强烈推荐 |
| 英雄 + 装备 | 65~70% 中 | 两路命中，推荐跟进 |
| 仅英雄或仅装备 | 40% 低 | 单路命中，可作为参考 |
| 无命中（兜底） | 30% 低 | 返回版本最强阵容，建议转型 |

---

## 数据更新

每次版本更新后重新跑一次爬虫即可，全程约 1~2 分钟：

```bash
cd metadata/tft-meta
python get_tftmeta.py
```

爬虫会自动：
1. 从 MetaTFT API 拉取最新阵容数据（comps_data + comps_stats）
2. 从 MetaTFT lookups 接口拉取最新英雄/装备中英文对照
3. 过滤样本数 < 200 的低质量阵容
4. 生成 Eino 所需的四个 JSON 文件

---

## API 数据结构说明

### 1. comps_data 接口
**URL**: `https://api-hc.metatft.com/tft-comps-api/comps_data?queue=1100`

返回所有阵容的完整信息：

```json
{
  "results": {
    "data": {
      "cluster_id": 394,
      "tft_set": "TFTSet16",
      "cluster_details": {
        "394000": {
          "Cluster": 394000,
          "units_string": "TFT16_Annie, TFT16_Galio, ...",
          "traits_string": "TFT16_Sorcerer_3, TFT16_Demacia_1, ...",
          "name_string": "TFT16_Sorcerer, TFT16_Lux",
          "overall": {
            "count": 238856,
            "avg": 4.0918
          },
          "stars": ["TFT16_Aphelios"],
          "builds": [
            {
              "unit": "TFT16_Annie",
              "buildName": ["TFT_Item_AdaptiveHelm", ...],
              "count": 49133,
              "avg": 3.5717,
              "score": 0.5768,
              "place_change": -0.52
            }
          ],
          "build_items": {
            "TFT_Item_JeweledGauntlet": {
              "count": 254607,
              "avg": 3.9272,
              "pcnt": 1.06594
            }
          },
          "trends": [
            {
              "day": "2026-03-17T00:00:00.000Z",
              "count": 60149,
              "avg": 3.9079,
              "pick": 0.05251
            }
          ],
          "levelling": "Fast 9",
          "difficulty": 0.024
        }
      }
    }
  }
}
```

### 2. comps_stats 接口
**URL**: `https://api-hc.metatft.com/tft-comps-api/comps_stats?queue=1100&patch=current&days=3&rank=...`

返回所有阵容的名次分布（用于计算胜率/进4率）：

```json
{
  "results": [
    {
      "cluster": "394000",
      "places": [202150, 120672, 107877, 102715, 101085, 101090, 99162, 82410, 917161],
      "count": 917161
    }
  ],
  "updated": 1774339522336,
  "tft_set": "TFTSet16",
  "queue_id": 1100,
  "cluster_id": 394
}
```

**places 数组说明**：索引 0-7 分别对应第 1-8 名的场次，索引 8 是总场次

### 3. lookups 接口（中英文对照）
**URL**: `https://data.metatft.com/lookups/TFTSet16_latest_zh_cn.json`

返回所有英雄/装备的中英文对照：

```json
{
  "items": [
    {
      "apiName": "TFT_Item_RabadonsDeathcap",
      "name": "灭世者的死亡之帽",
      "en_name": "Rabadon's Deathcap",
      "desc": "这顶不起眼的帽子可以帮助你创造...",
      "effects": {
        "AP": 50,
        "BonusDamage": 0.15
      }
    }
  ]
}
```

### Tier 计算规则
Tier 不再由 API 返回，而是根据 `avg_placement` 动态计算（基于 MetaTFT 官网规则）：

| avg_placement | Tier |
|---|---|
| ≤ 4.25 | S |
| ≤ 4.52 | A |
| ≤ 4.78 | B |
| ≤ 5.10 | C |
| > 5.10 | D |

---

## 技术架构

```
用户输入
   ↓
[InputParser]  别名解析 → 中文名 → TFT ID
   ↓
┌──────────────┬──────────────┬──────────────┐
│ HeroCompsTool│  ItemFitTool │ CompTierTool │  并行执行
│ 英雄 → 阵容  │  装备 → 阵容 │  版本强度    │
└──────────────┴──────────────┴──────────────┘
   ↓
[IntersectionCalc]  三路交集 + 置信度计算
   ↓
[LLM 推理节点]  生成自然语言运营建议
   ↓
响应（普通 JSON 或 SSE 流式）
```

整个推理流程由 [Eino](https://github.com/cloudwego/eino) Graph 编排，三个 Tool 节点自动并发执行，LLM 节点在交集计算完成后才触发。

---

## 依赖

```
github.com/cloudwego/eino
github.com/cloudwego/eino-ext
```

数据爬虫依赖：
```
pip install requests
```
