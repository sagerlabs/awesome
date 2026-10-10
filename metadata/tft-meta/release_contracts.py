#!/usr/bin/env python3
"""TFT Copilot 版本/发布契约 v1 —— Python 侧（DATA-01）。

与 Go 侧 ``tft/release/contracts`` 共用同一份 JSON 契约：
本模块提供 load / validate / build 三类能力，保证 Python 构建器
与 Go 加载器对同一份 fixture 的判定一致。

关键语义（ADR013，改动须双侧同步）：
- 四个时间字段：fetched_at / stats_window_end / effective_at / built_at
  不得互换（fetched <= built；window_end <= fetched；effective 独立）；
- manifest 不得出现在自身 files（哈希自引用禁止）；
- game_patch 是补丁真相源，禁止从 set_id 或公告标题推导；
- region/queue 未知时显式写 "unknown"，不能编造；
- schema 兼容：reader_schema_range 支持 "^MAJOR.MINOR" 与 ">=x.y,<x.y"。

错误类别与 Go 侧 ErrCategory* 一一对应，跨语言测试按类别断言。
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from datetime import datetime, timezone

SCHEMA_VERSION_V1 = "1.0.0"
MANIFEST_FILENAME = "manifest.json"

GAME = "tft"
KNOWN_REGIONS = {"cn", "na", "eu", "kr", "unknown"}
KNOWN_QUEUES = {"ranked", "normal", "unknown"}

GAME_PATCH_PATTERN = re.compile(r"^\d+\.\d+[a-z]?$")
HOTFIX_PATTERN = re.compile(r"^[a-z]$")
LOCALE_PATTERN = re.compile(r"^[a-z]{2,3}(-[A-Za-z]{2,8})*$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
MANIFEST_PATH_PATTERN = re.compile(r"^[A-Za-z0-9_][A-Za-z0-9_./-]*\.[A-Za-z0-9]+$")

# 文件 kind -> 支撑的能力
KIND_CAPABILITIES = {
    "team_comps": "base_facts",
    "localization": "base_facts",
    "champion_profiles": "champion_profiles",
    "items_priority": "items_priority",
    "patch_notes": "patch_notes",
    "aliases": "slang_nlu",
    "knowledge": "knowledge",
}
REQUIRED_KINDS = ("team_comps", "localization")
KNOWN_CAPABILITIES = {
    "base_facts", "meta_stats", "patch_notes", "champion_profiles",
    "items_priority", "slang_nlu", "knowledge",
}


class ContractError(Exception):
    """结构化契约错误。category 与 Go 侧 ErrCategory* 常量一致。"""

    def __init__(self, category: str, field: str, reason: str, path: str = ""):
        super().__init__(f"contract: {category} field={field} path={path}: {reason}")
        self.category = category
        self.field = field
        self.reason = reason
        self.path = path


# ── 时间解析 ────────────────────────────────────────────────────────────────

def parse_time(value, field: str) -> datetime:
    """解析 RFC3339 时间；零值/缺失视为非法（时间必须显式给出）。"""
    if not value:
        raise ContractError("time_order", field, "时间字段缺失或为零值")
    text = str(value).replace("Z", "+00:00")
    try:
        dt = datetime.fromisoformat(text)
    except ValueError as exc:
        raise ContractError("format", field, f"时间格式非法：{value!r}（{exc}）")
    if dt.tzinfo is None:
        raise ContractError("format", field, f"时间必须带时区：{value!r}")
    return dt


# ── schema 兼容 ─────────────────────────────────────────────────────────────

def _parse_semver(version: str):
    m = re.match(r"^v?(\d+)\.(\d+)(?:\.(\d+))?$", version or "")
    if not m:
        raise ContractError("schema", "schema_version", f"版本号非法：{version!r}")
    return tuple(int(x or 0) for x in m.groups())


def supports_schema(reader_range: str, schema_version: str) -> None:
    """判断 reader 范围是否覆盖 schema_version；非法即抛错（不静默放行）。"""
    if not reader_range:
        raise ContractError("schema", "reader_schema_range", "reader_schema_range 为空")
    sv = _parse_semver(schema_version)

    m = re.match(r"^\^(\d+)(?:\.(\d+))?(?:\.(\d+))?$", reader_range)
    if m:
        lo = tuple(int(x or 0) for x in m.groups())
        hi = (lo[0] + 1, 0, 0)
        if not (lo <= sv < hi):
            raise ContractError(
                "schema", "schema_version",
                f"schema {schema_version} 不在 {reader_range}（要求 [{lo}, {hi})）")
        return

    m = re.match(r"^>=(\d+\.\d+(?:\.\d+)?),<(\d+\.\d+(?:\.\d+)?)$", reader_range)
    if m:
        lo = _parse_semver(m.group(1))
        hi = _parse_semver(m.group(2))
        if not (lo <= sv < hi):
            raise ContractError(
                "schema", "schema_version",
                f"schema {schema_version} 不在 {reader_range}（要求 [{lo}, {hi})）")
        return

    raise ContractError(
        "schema", "reader_schema_range",
        f"语法非法：{reader_range!r}（支持 ^x.y 或 >=x.y,<x.y）")


# ── DataScope ────────────────────────────────────────────────────────────────

def validate_scope(scope: dict) -> None:
    if not isinstance(scope, dict):
        raise ContractError("scope", "scope", "scope 必须是对象")
    if scope.get("game") != GAME:
        raise ContractError("scope", "game", f"game 必须为 {GAME!r}，got {scope.get('game')!r}")
    set_id = scope.get("set_id")
    if not isinstance(set_id, int) or set_id <= 0:
        raise ContractError("scope", "set_id", f"set_id 必须为正整数，got {set_id!r}")
    patch = scope.get("game_patch", "")
    if not patch:
        raise ContractError("scope", "game_patch", "game_patch 不能为空（禁止从赛季或公告标题推导）")
    if not GAME_PATCH_PATTERN.match(patch):
        raise ContractError("scope", "game_patch", f"game_patch 格式非法：{patch!r}")
    hotfix = scope.get("hotfix_revision", "")
    if hotfix and not HOTFIX_PATTERN.match(hotfix):
        raise ContractError("scope", "hotfix_revision", f"hotfix_revision 非空时必须为单字母：{hotfix!r}")
    region = scope.get("region", "")
    if region not in KNOWN_REGIONS:
        raise ContractError("scope", "region", f"region 未知（允许 unknown/具体地区代码）：{region!r}")
    queue = scope.get("queue", "")
    if queue not in KNOWN_QUEUES:
        raise ContractError("scope", "queue", f"queue 未知（允许 ranked/normal/unknown）：{queue!r}")
    locale = scope.get("locale", "")
    if not locale or not LOCALE_PATTERN.match(locale):
        raise ContractError("scope", "locale", f"locale 必须为 BCP-47 形式：{locale!r}")
    parse_time(scope.get("effective_at"), "effective_at")


# ── 路径安全 ────────────────────────────────────────────────────────────────

def _check_path(path: str) -> None:
    if not path:
        raise ContractError("file_path", "relative_path", "relative_path 为空")
    if path in (MANIFEST_FILENAME, f"./{MANIFEST_FILENAME}"):
        raise ContractError("self_reference", "files", "manifest 不得出现在自身 files 列表（哈希自引用）")
    if not MANIFEST_PATH_PATTERN.match(path) or os.path.isabs(path):
        raise ContractError("file_path", "relative_path", f"非法路径：{path!r}")
    for seg in path.replace("\\", "/").split("/"):
        if seg in ("", ".", ".."):
            raise ContractError("file_path", "relative_path", f"路径段非法（逃逸/空段）：{path!r}")


# ── DataManifest ────────────────────────────────────────────────────────────

def _capability_supported(seen_kinds: set, capability: str) -> bool:
    if capability in ("base_facts", "meta_stats"):
        return "team_comps" in seen_kinds and "localization" in seen_kinds
    for kind, cap in KIND_CAPABILITIES.items():
        if cap == capability and kind in seen_kinds:
            return True
    return False


def validate_manifest(manifest: dict) -> None:
    if not isinstance(manifest, dict):
        raise ContractError("format", "manifest", "manifest 必须是对象")
    if not manifest.get("data_release_id"):
        raise ContractError("identity", "data_release_id", "data_release_id 不能为空")
    if not manifest.get("schema_version"):
        raise ContractError("schema", "schema_version", "schema_version 不能为空")
    supports_schema(manifest.get("reader_schema_range", ""), manifest["schema_version"])
    validate_scope(manifest.get("scope"))

    files = manifest.get("files") or []
    if not files:
        raise ContractError("file_missing", "files", "files 不能为空")
    sources = manifest.get("sources") or []
    if not sources:
        raise ContractError("identity", "sources", "sources 不能为空（至少声明一个上游来源）")

    seen_kinds, seen_paths = set(), set()
    for i, f in enumerate(files):
        path = f.get("relative_path", "")
        _check_path(path)
        if path in seen_paths:
            raise ContractError("file_path", "relative_path", f"重复路径：{path!r}")
        seen_paths.add(path)
        if not SHA256_PATTERN.match(f.get("sha256", "")):
            raise ContractError("hash", "sha256", f"sha256 必须为 64 位小写十六进制：{path!r}")
        if not isinstance(f.get("size"), int) or f["size"] < 0:
            raise ContractError("format", "size", f"size 非法：{path!r}")
        if not isinstance(f.get("entity_count"), int) or f["entity_count"] < 0:
            raise ContractError("format", "entity_count", f"entity_count 非法：{path!r}")
        kind = f.get("kind", "")
        if kind not in KIND_CAPABILITIES:
            raise ContractError("capability", "kind", f"未知文件 kind：{kind!r}")
        seen_kinds.add(kind)

    for required in REQUIRED_KINDS:
        if required not in seen_kinds:
            raise ContractError("file_missing", "kind", f"缺少必需文件 kind：{required}")

    capabilities = manifest.get("capabilities") or []
    if not capabilities:
        raise ContractError("capability", "capabilities", "capabilities 不能为空（至少 base_facts）")
    for cap in capabilities:
        if cap not in KNOWN_CAPABILITIES:
            raise ContractError("capability", "capabilities", f"未知能力：{cap!r}")
        if not _capability_supported(seen_kinds, cap):
            raise ContractError("capability", "capabilities", f"声明能力 {cap!r} 但缺少对应文件")

    built_at = parse_time(manifest.get("built_at"), "built_at")
    for i, s in enumerate(sources):
        if not s.get("provider"):
            raise ContractError("identity", "provider", f"sources[{i}].provider 为空")
        if not s.get("source_dataset_id"):
            raise ContractError("identity", "source_dataset_id", f"sources[{i}].source_dataset_id 为空")
        fetched = parse_time(s.get("fetched_at"), "fetched_at")
        if not SHA256_PATTERN.match(s.get("raw_sha256", "")):
            raise ContractError("hash", "raw_sha256", f"sources[{i}].raw_sha256 必须为 64 位小写十六进制")
        window_end = parse_time(s.get("stats_window_end"), "stats_window_end")
        window_start = s.get("stats_window_start")
        if window_start:
            if parse_time(window_start, "stats_window_start") > window_end:
                raise ContractError("time_order", "stats_window_start", f"sources[{i}] 统计窗口 start > end")
        if window_end > fetched:
            raise ContractError("time_order", "stats_window_end", f"sources[{i}] 统计窗口在抓取之后闭合")
        if fetched > built_at:
            raise ContractError("time_order", "built_at", f"sources[{i}].fetched_at 晚于 built_at")
        if not isinstance(s.get("sample_floor", 0), int) or s.get("sample_floor", 0) < 0:
            raise ContractError("format", "sample_floor", f"sources[{i}].sample_floor 不能为负")


# ── 部署 / 会话 / 路由 / 证据 ───────────────────────────────────────────────

def validate_deployment(deployment: dict) -> None:
    if not isinstance(deployment, dict):
        raise ContractError("format", "deployment", "deployment 必须是对象")
    for field in ("deployment_id", "image_digest", "app_commit", "data_release_id",
                  "prompt_version", "policy_version", "model_config_version"):
        if not deployment.get(field):
            raise ContractError("identity", field, f"{field} 不能为空")
    parse_time(deployment.get("created_at"), "created_at")


def validate_session_record(record: dict) -> None:
    if not isinstance(record, dict):
        raise ContractError("format", "session", "session 必须是对象")
    for field in ("owner_id", "session_id", "assigned_release"):
        if not record.get(field):
            raise ContractError("identity", field, f"{field} 不能为空")
    revision = record.get("state_revision", 0)
    if not isinstance(revision, int) or revision < 0:
        raise ContractError("format", "state_revision", "state_revision 不能为负")
    validate_scope(record.get("scope"))
    parse_time(record.get("expires_at"), "expires_at")


def validate_route_decision(decision: dict) -> None:
    if not isinstance(decision, dict):
        raise ContractError("format", "route_decision", "route_decision 必须是对象")
    for field in ("request_id", "subject_id", "session_id", "release_id", "backend_id"):
        if not decision.get(field):
            raise ContractError("identity", field, f"{field} 不能为空")
    revision = decision.get("policy_revision", 0)
    if not isinstance(revision, int) or revision <= 0:
        raise ContractError("format", "policy_revision", "policy_revision 必须为正")
    parse_time(decision.get("expires_at"), "expires_at")


def validate_evidence(evidence: dict) -> None:
    if not isinstance(evidence, dict):
        raise ContractError("format", "evidence", "evidence 必须是对象")
    if not evidence.get("trace_id"):
        raise ContractError("identity", "trace_id", "trace_id 不能为空")
    if not evidence.get("release_id"):
        raise ContractError("identity", "release_id", "release_id 不能为空")
    validate_scope(evidence.get("scope"))
    if not evidence.get("source_ids"):
        raise ContractError("identity", "source_ids", "source_ids 不能为空（证据必须可溯源）")
    if not evidence.get("capabilities"):
        raise ContractError("capability", "capabilities", "capabilities 不能为空")
    for cap in evidence["capabilities"]:
        if cap not in KNOWN_CAPABILITIES:
            raise ContractError("capability", "capabilities", f"未知能力：{cap!r}")
    action = evidence.get("sample_policy", {}).get("below_floor_action")
    if action not in ("degrade_basic", "reject"):
        raise ContractError("format", "below_floor_action", f"必须为 degrade_basic 或 reject：{action!r}")
    freshness = evidence.get("freshness", {})
    fetched = parse_time(freshness.get("fetched_at"), "fetched_at")
    built = parse_time(freshness.get("built_at"), "built_at")
    window_end = parse_time(freshness.get("stats_window_end"), "stats_window_end")
    if fetched > built:
        raise ContractError("time_order", "freshness", "fetched_at 晚于 built_at")
    if window_end > fetched:
        raise ContractError("time_order", "freshness", "stats_window_end 晚于 fetched_at")


# ── 构建（生成器侧） ────────────────────────────────────────────────────────

def _file_sha256(path: str) -> tuple:
    h = hashlib.sha256()
    n = 0
    with open(path, "rb") as fh:
        while True:
            chunk = fh.read(1 << 16)
            if not chunk:
                break
            h.update(chunk)
            n += len(chunk)
    return h.hexdigest(), n


def _entity_count(path: str, kind: str) -> int:
    """按 kind 粗略统计实体数（数组长度 / 对象键数）；失败返回 0。"""
    try:
        with open(path, "r", encoding="utf-8") as fh:
            data = json.load(fh)
    except (OSError, ValueError):
        return 0
    if isinstance(data, list):
        return len(data)
    if isinstance(data, dict):
        return len(data)
    return 0


def build_manifest(data_dir: str, scope: dict, sources: list,
                   capabilities: list, builder_commit: str,
                   built_at: str, files_plan: list) -> dict:
    """从 data_dir 构建 manifest（生成器侧唯一入口）。

    files_plan: [{"relative_path": "data/team_comps.json", "kind": "team_comps"}, ...]
    sha256/size/entity_count 由实际文件计算；manifest 本身不写入 files。
    """
    _check_path_ok_for_build = lambda p: _check_path(p)  # noqa: E731
    records = []
    for item in files_plan:
        rel = item["relative_path"]
        _check_path_ok_for_build(rel)
        full = os.path.join(data_dir, rel)
        if not os.path.isfile(full):
            raise ContractError("file_missing", "relative_path", f"文件不存在：{full}")
        digest, size = _file_sha256(full)
        records.append({
            "relative_path": rel,
            "sha256": digest,
            "size": size,
            "kind": item["kind"],
            "entity_count": _entity_count(full, item["kind"]),
        })

    # data_release_id：scope 摘要 + 构建时间 + 负载内容前 8 位。
    # 注意：内容摘要来自负载文件，不含 manifest 自身（自引用禁止）。
    payload_digest = hashlib.sha256(
        "\n".join(r["relative_path"] + ":" + r["sha256"] for r in sorted(records, key=lambda x: x["relative_path"])).encode()
    ).hexdigest()[:8]
    scope_key = f"{scope['game']}-s{scope['set_id']}-{scope['game_patch']}-{scope['region']}"
    release_id = f"{scope_key}-{built_at}-{payload_digest}"

    manifest = {
        "data_release_id": release_id,
        "schema_version": SCHEMA_VERSION_V1,
        "scope": scope,
        "reader_schema_range": "^1.0",
        "files": records,
        "sources": sources,
        "capabilities": capabilities,
        "builder_commit": builder_commit,
        "built_at": built_at,
    }
    validate_manifest(manifest)
    return manifest


def load_manifest(path: str) -> dict:
    with open(path, "r", encoding="utf-8") as fh:
        manifest = json.load(fh)
    validate_manifest(manifest)
    return manifest


# ── CLI ──────────────────────────────────────────────────────────────────────

def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description="TFT release contracts v1")
    sub = parser.add_subparsers(dest="command", required=True)

    p_build = sub.add_parser("build", help="从目录构建 manifest（stdout 输出 JSON）")
    p_build.add_argument("--data-dir", required=True)
    p_build.add_argument("--scope", required=True, help="scope JSON 文件路径")
    p_build.add_argument("--sources", required=True, help="sources JSON 文件路径")
    p_build.add_argument("--capabilities", required=True, help="capabilities JSON 数组文件路径")
    p_build.add_argument("--files-plan", required=True, help="files_plan JSON 文件路径")
    p_build.add_argument("--builder-commit", default="unknown")
    p_build.add_argument("--built-at", required=True)

    p_val = sub.add_parser("validate", help="校验一份 manifest.json")
    p_val.add_argument("path")

    args = parser.parse_args(argv)
    try:
        if args.command == "build":
            with open(args.scope, encoding="utf-8") as fh:
                scope = json.load(fh)
            with open(args.sources, encoding="utf-8") as fh:
                sources = json.load(fh)
            with open(args.capabilities, encoding="utf-8") as fh:
                capabilities = json.load(fh)
            with open(args.files_plan, encoding="utf-8") as fh:
                files_plan = json.load(fh)
            manifest = build_manifest(args.data_dir, scope, sources, capabilities,
                                       args.builder_commit, args.built_at, files_plan)
            json.dump(manifest, sys.stdout, ensure_ascii=False, indent=2)
            print()
        else:
            load_manifest(args.path)
            print(f"OK: {args.path}")
    except ContractError as exc:
        print(f"ERROR [{exc.category}] {exc.field}: {exc.reason}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
