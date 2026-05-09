# TFT Copilot

云顶之弈 AI Agent（智能体）项目。它把玩家自然语言问题解析成结构化查询，从知识库里检索当前版本阵容、英雄、装备、羁绊和版本公告，再用 LLM（大模型）生成像国服老玩家一样的简洁建议。

## 当前主链路

```text
用户输入
  -> Fast NLU / LLM NLU（快速规则或大模型意图识别）
  -> Contract（agent 与 knowledge 共享的查询协议）
  -> Knowledge（阵容、英雄、装备、羁绊、版本公告）
  -> Format LLM（把事实整理成自然语言回答）
  -> JSON / SSE 响应
```

Primary API（主接口）：

```text
POST /v1/tft/nlu
POST /v1/tft/nlu/stream
GET  /v1/tft/health
```

Legacy API（历史兼容）：

```text
POST /v1/tft/analyze
POST /v1/tft/analyze/stream
```

## 能力范围

- 阵容推荐：回答“当前版本最强阵容”“这把能玩什么”。
- 装备查询：回答“我有羊刀/珠光护手，可以玩什么”。
- 英雄垂直查询：回答“四费谁能 C”“剑魔打工强吗”“谁能抗”。
- 羁绊查询：回答“海魔人能玩吗”“未来战士怎么搭”。
- 黑话识别：通过 `tft/knowledge/data/aliases.json` 维护“羊刀、剑魔、九五”等玩家说法。
- 版本环境：可把官方更新公告写入 `tft/knowledge/data/patch_notes`，辅助解释版本变化。

## 快速启动

准备环境变量：

```bash
export LLM_PROVIDER=deepseek
export OPENAI_API_KEY=sk-xxx
export OPENAI_BASE_URL=https://api.deepseek.com
export OPENAI_MODEL=deepseek-chat
export PORT=8080
```

启动服务：

```bash
make run
```

测试主接口：

```bash
curl -N -X POST http://localhost:8080/v1/tft/nlu/stream \
  -H "Content-Type: application/json" \
  -d '{"input":"剑魔打工强吗"}'
```

## 数据更新

每次版本更新后执行：

```bash
make data
```

如果有官方版本公告 URL：

```bash
make data PATCH_NOTE_URL="https://lol.qq.com/gicp/news/662/37082726.html"
```

如果只想用本地已下载的 metadata（元数据）重新生成知识库：

```bash
make data-local
```

数据流水线由 `scripts/update_cn_knowledge.py` 负责：

```text
MetaTFT 中文数据
  -> metadata/tft-meta/data/*.json
  -> tft/knowledge/data/champions/*.json
  -> tft/knowledge/data/items/*.json
  -> tft/knowledge/data/team_comps/*.json
  -> tft/knowledge/data/champion_profiles.json
```

## 关键目录

```text
tft/agent                 Agent 编排、Fast NLU、Prompt、LLM 调用
tft/knowledge             知识库加载、查询、垂直英雄/羁绊查询
tft/knowledge/contracts   agent 与 knowledge 的公共结构体
tft/knowledge/data        当前版本知识库 JSON
tft/parser                Legacy analyze 链路的输入解析
scripts                   数据更新脚本
docs                      ADR（架构决策记录）和学习文档
```

## 质量检查

```bash
make data-check
make test
```

开启 TRACE（链路追踪）后，日志会记录 Fast Path（快速通道）是否命中、NLU 来源、LLM 调用次数、knowledge 查询命中数量和耗时：

```bash
TRACE=true make run
```
