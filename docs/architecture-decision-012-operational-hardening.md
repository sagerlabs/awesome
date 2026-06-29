# ADR-012: 运维化加固（容器、限流、健康探针、legacy 退场）

## 日期

2026-06-11

## Status

Accepted and implemented

## 背景

ADR-001 ~ 011 解决的是"答得对不对"：边界、知识、反馈。项目走到可部署阶段后，
缺的是"跑得稳不稳"：没有镜像、没有限流、健康检查只回 `{"status":"ok"}`、
版本信息注入到不存在的符号被链接器静默丢弃、legacy `/analyze` 链路无限期陪跑。

## 决策

### 1. 容器化

多阶段 Dockerfile：builder 编译 CGO-free 静态二进制，运行时 Alpine + 非 root
用户，knowledge / metadata 数据按仓库布局打进镜像，内置 HEALTHCHECK。
`docker-compose.yml` 全部配置走环境变量。数据目录支持
`TFT_DATA_DIR` / `TFT_KNOWLEDGE_DIR` 覆盖（默认仓库布局），挂载外部数据时不需要
迁就镜像内路径。

### 2. 限流：自研 per-IP 令牌桶，不引第三方库

`tft/ratelimit.go` 约 100 行：每 IP 一个桶，rps 回填 + burst 突发，空闲 IP
定时清理保证内存有界。`RATE_LIMIT_RPS<=0` 时退化为 passthrough，默认关闭。

之所以不引 `golang.org/x/time/rate` 或限流库：逻辑足够小且需要按 IP 分桶 +
清理，自研版本可读、可注入时钟测试，避免为小功能扩大依赖面。

### 3. 健康探针带版本与数据状态

`/v1/tft/health` 返回 `status / version / git_commit / build_time / comp_count`；
数据未加载时返回 503 `degraded`，可直接用于容器与负载均衡探针。
版本变量收敛到 `tft` 包（`tft.Version` 等），修复了 Makefile / release.yml
向不存在的 `main.Version` 注入、被静默丢弃的问题。

### 4. legacy 链路退场路线

- 响应头：`Deprecation: true`、`Sunset: 2026-12-31`、`Link` 指向 `/v1/tft/nlu`。
- `DISABLE_LEGACY_GRAPH=true` 跳过 legacy graph 编译，禁用后接口返回 410，
  `Analyze/AnalyzeStream` 以哨兵错误 `ErrLegacyGraphDisabled` 快速失败。
- Sunset 日期之后移除 `tft/parser` 与 legacy graph 代码。

## 明确推迟的事项

**存储类型收敛**：`data.Store`（legacy/交集推荐用）与 `knowledge.UnifiedStore`
（NLU 用）仍是两套并行类型。核实结论：每进程 `data.Store` 只加载一次并被
`UnifiedStore` 复用，不存在双份内存；真正的债是两套查询索引和两套类型语义。
合并属于大重构，收益在 legacy 链路移除（见上文 Sunset）之后才显现，
届时随 `tft/parser` 一起处理，本 ADR 不展开。

## 验收

- 镜像可一键 `make docker-build` / `make compose-up`。
- 限流、健康探针、410 行为均有 httptest / 单测锁定（`tft/ratelimit_test.go`、
  `tft/handler_test.go`）。
- `/health` 能报告真实版本号（ldflags 注入实测生效）。
