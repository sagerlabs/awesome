# OP.GG MCP MVP Data Pipeline

这条管线用于把 OP.GG MCP（Model Context Protocol，模型上下文协议）里的 TFT meta decks 转换成本项目的本地 knowledge JSON。

## 目标

最小 MVP 只做这些事：

- 拉取 OP.GG 当前 meta decks，默认最多保留前 20 套。
- 只处理这些阵容里的英雄和装备。
- 生成 `final`（成型棋盘）、核心英雄、核心装备、平均名次、前四率、吃鸡率、样本量。
- 如果 OP.GG 返回 `early/middle`（前期/中期棋盘），就写入；没有就留空，不让 LLM 编造。

## 使用方式

直接请求 OP.GG MCP 并写入当前 knowledge：

```bash
make data-opgg-mvp
```

指定数量：

```bash
make data-opgg-mvp LIMIT=20
```

只校验，不写文件：

```bash
python3 scripts/update_opgg_mcp_mvp.py --dry-run
```

使用已保存的 OP.GG MCP 响应做离线校验：

```bash
python3 scripts/update_opgg_mcp_mvp.py \
  --input-response /path/to/opgg_response.json \
  --dry-run
```

写到临时目录，不覆盖当前 knowledge：

```bash
python3 scripts/update_opgg_mcp_mvp.py \
  --input-response /path/to/opgg_response.json \
  --knowledge-dir /private/tmp/opgg_knowledge_mvp
```

## 当前限制

- OP.GG 当前 `tft_list_meta_decks` 实际返回 10 套阵容；脚本支持 `--limit 20`，但不会为了凑数编造额外阵容。
- 这条管线依赖本地 `metadata/tft-meta/data/localization.json` 做中文名映射。
- 输出会覆盖 `team_comps`、`champions`、`items` 下的 JSON；如果想保留旧文件，使用 `--no-clean`。
- OP.GG 数据适合作为学习和 MVP 数据源；如果后续商业化，需要确认数据使用条款。

## 数据流

```text
OP.GG MCP tft_list_meta_decks
  -> scripts/update_opgg_mcp_mvp.py
  -> tft/knowledge/data/team_comps
  -> tft/knowledge/data/champions
  -> tft/knowledge/data/items
  -> UnifiedStore
  -> Agent / 本地 MCP tools
```

核心原则：

```text
外部 MCP 负责提供结构化原始数据
本项目数据管线负责转换和落库
knowledge 负责稳定查询
LLM 只负责表达，不负责补事实
```
