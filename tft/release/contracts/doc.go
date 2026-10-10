// Package contracts 定义 TFT Copilot 的版本与发布契约 v1（DATA-01）。
//
// 本包是 ADR013「版本化数据、应用边界与可回滚发布」的协议落地：
// scope、manifest、source/files、deployment identity、evidence identity、
// 路由身份。只做契约与校验，不做线上发布实现（那是 DATA-04/OPS-* 的范围）。
//
// # 依赖方向
//
// 本包不依赖 agent、数据库或传输实现；业务查询侧（knowledge 等）可以引用
// 本包的版本结构，反向禁止——版本结构只在此维护一份，其他任务不得复制 DTO。
//
// # 时间语义（ADR013 强制）
//
// 四个时间字段含义不同，禁止互相冒充：
//
//	fetched_at          上游数据被抓取的时刻
//	stats_window_end    统计样本窗口的结束时刻（窗口 start 可选）
//	effective_at        游戏版本（补丁/热修）在目标 region 生效的时刻
//	built_at            数据包构建完成的时刻
//
// 约束：fetched_at <= built_at；stats_window_end <= fetched_at
// （样本窗口必须在抓取之前闭合）。effective_at 与统计时间无关——新补丁
// 已生效不代表统计已更新，反之亦然。
//
// # content hash 与 build run ID（ADR013：避免哈希自引用）
//
// FileRecord.sha256 只对不可变数据负载定义。manifest 本身不得出现在
// files 列表中（自引用禁止）；构建 run ID / 构建时间（built_at、
// builder_commit）是清单字段而不是被哈希的负载。同输入重复构建的
// 语义内容必须可复现。
//
// # 精确补丁的真相源
//
// DataScope.game_patch 是补丁版本的唯一真相源，必须显式提供，格式
// ^\d+\.\d+[a-z]?$。禁止从 set_id 推导补丁，禁止从公告标题/赛季名盲猜
// 补丁号；公告数据只是知识内容，不参与 scope 推导。
//
// # 版本与兼容策略
//
// schema_version 语义化：MAJOR 破坏性变更（删字段/改语义），MINOR 新增
// 可选字段，PATCH 描述修正。reader_schema_range 声明应用读取范围，
// 支持 "^MAJOR.MINOR"（锁定 major）与 ">=MAJOR.MINOR,<MAJOR.MINOR"
// （显式范围）两种写法；未知/不兼容 schema 一律拒绝加载（ErrSchema）。
package contracts
