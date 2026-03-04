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
│       ├── scraper.py              # MetaTFT + CDragon 数据爬虫
│       ├── debug_localization.py   # 汉化数据调试工具
│       └── data/
│           ├── comps_for_agent.json    # 阵容数据（Eino Tool 使用）
│           ├── items_priority.json     # 装备优先级索引
│           └── localization.json       # 英雄/装备中文名映射
├── tft/
│   ├── handler.go              # HTTP 路由入口
│   ├── data/
│   │   ├── types.go            # 所有结构体定义
│   │   └── loader.go           # Store 数据加载与查询
│   ├── tool/
│   │   ├── parser.go           # 用户输入标准化（别名/中文名/ID）
│   │   ├── hero_comps.go       # Tool1：英雄 → 推荐阵容
│   │   ├── item_fit.go         # Tool2：装备 → 适配阵容
│   │   └── comp_tier.go        # Tool3：阵容强度 + 交集计算
│   └── agent/
│       ├── model.go            # LLM 初始化（多 Provider）
│       ├── graph.go            # Eino Graph 编排
│       └── agent.go            # 对外入口
├── sse/                        # SSE 框架（已有）
├── main.go
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
python scraper.py
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
  POST /tft/analyze         普通接口
  POST /tft/analyze/stream  流式接口（SSE）
  GET  /tft/health          健康检查
────────────────────────────────
```

---

## API 接口

### POST /tft/analyze

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

### POST /tft/analyze/stream

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
python scraper.py
```

爬虫会自动：
1. 从 MetaTFT API 拉取最新阵容数据（胜率/名次/装备优先级）
2. 从 Community Dragon 拉取最新英雄/装备中文名
3. 过滤样本数 < 200 的低质量阵容
4. 生成 Eino 所需的四个 JSON 文件

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
