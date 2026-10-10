#!/usr/bin/env python3
"""release_contracts 的镜像测试（DATA-01）。

与 Go 侧 tft/release/contracts/contracts_test.go 对同一批 testdata
fixture 断言相同的 pass/fail 与错误类别（跨语言互通验收）。
运行：python -m unittest test_release_contracts -v（或 pytest）。
"""

from __future__ import annotations

import copy
import json
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import release_contracts as rc

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", ".."))
FIXTURE_DIR = os.path.join(REPO, "tft", "release", "contracts", "testdata")
VALID_DIR = os.path.join(FIXTURE_DIR, "valid")
INVALID_DIR = os.path.join(FIXTURE_DIR, "invalid")

# 与 Go 侧 expectedCategories 一一对应
EXPECTED_CATEGORIES = {
    "bad_schema_version.json": "schema",
    "bad_scope_region.json": "scope",
    "empty_patch.json": "scope",
    "bad_patch_format.json": "scope",
    "path_escape.json": "file_path",
    "self_reference.json": "self_reference",
    "bad_hash.json": "hash",
    "missing_required_kind.json": "file_missing",
    "bad_time_order.json": "time_order",
    "bad_reader_range.json": "schema",
    "unknown_capability.json": "capability",
    "window_after_fetch.json": "time_order",
}


def _load_json(path):
    with open(path, encoding="utf-8") as fh:
        return json.load(fh)


class TestValidFixtures(unittest.TestCase):
    def test_manifest_valid(self):
        manifest = rc.load_manifest(os.path.join(VALID_DIR, "manifest.json"))
        self.assertEqual(manifest["scope"]["game_patch"], "18.1")
        self.assertEqual(len(manifest["files"]), 2)
        self.assertEqual(manifest["reader_schema_range"], "^1.0")

    def test_deployment_valid(self):
        rc.validate_deployment(_load_json(os.path.join(VALID_DIR, "deployment.json")))

    def test_session_record_valid(self):
        rc.validate_session_record(_load_json(os.path.join(VALID_DIR, "session_record.json")))

    def test_route_decision_valid(self):
        rc.validate_route_decision(_load_json(os.path.join(VALID_DIR, "route_decision.json")))

    def test_evidence_valid(self):
        rc.validate_evidence(_load_json(os.path.join(VALID_DIR, "evidence.json")))


class TestInvalidFixtures(unittest.TestCase):
    def test_all_invalid_rejected_with_expected_category(self):
        entries = sorted(f for f in os.listdir(INVALID_DIR) if f.endswith(".json"))
        self.assertEqual(
            len(entries), len(EXPECTED_CATEGORIES),
            "invalid fixture 数量与期望表不符（新增用例须同步两侧测试）")
        for name in entries:
            with self.subTest(fixture=name):
                path = os.path.join(INVALID_DIR, name)
                with self.assertRaises(rc.ContractError) as ctx:
                    rc.load_manifest(path)
                want = EXPECTED_CATEGORIES[name]
                self.assertEqual(
                    ctx.exception.category, want,
                    f"{name} 类别不符：got {ctx.exception.category!r} want {want!r}"
                    f"（reason={ctx.exception.reason}）")


class TestBuildRoundtrip(unittest.TestCase):
    def test_build_manifest_reproduces_fixture(self):
        """从 build_inputs 重建，语义字段必须与已提交 fixture 一致。

        同输入语义产物可复现（ADR013）；差异只允许出现在
        data_release_id 的构建时间戳上（本测试用同一时间，应完全一致）。
        """
        inputs = os.path.join(VALID_DIR, "build_inputs")
        scope = _load_json(os.path.join(inputs, "scope.json"))
        sources = _load_json(os.path.join(inputs, "sources.json"))
        capabilities = _load_json(os.path.join(inputs, "capabilities.json"))
        files_plan = _load_json(os.path.join(inputs, "files_plan.json"))

        built = rc.build_manifest(
            VALID_DIR, scope, sources, capabilities,
            builder_commit="fixture0001",
            built_at="2026-09-09T09:00:00Z",
            files_plan=files_plan)

        committed = _load_json(os.path.join(VALID_DIR, "manifest.json"))
        self.assertEqual(built, committed, "同输入构建应与已提交 fixture 完全一致")

    def test_build_rejects_escape_path(self):
        scope = _load_json(os.path.join(VALID_DIR, "build_inputs", "scope.json"))
        sources = _load_json(os.path.join(VALID_DIR, "build_inputs", "sources.json"))
        with self.assertRaises(rc.ContractError) as ctx:
            rc.build_manifest(
                VALID_DIR, scope, sources, ["base_facts"],
                "c0ffee", "2026-09-09T09:00:00Z",
                files_plan=[{"relative_path": "../../etc/passwd", "kind": "team_comps"}])
        self.assertEqual(ctx.exception.category, "file_path")

    def test_build_rejects_missing_file(self):
        scope = _load_json(os.path.join(VALID_DIR, "build_inputs", "scope.json"))
        sources = _load_json(os.path.join(VALID_DIR, "build_inputs", "sources.json"))
        with self.assertRaises(rc.ContractError) as ctx:
            rc.build_manifest(
                VALID_DIR, scope, sources, ["base_facts"],
                "c0ffee", "2026-09-09T09:00:00Z",
                files_plan=[{"relative_path": "data/no_such.json", "kind": "team_comps"}])
        self.assertEqual(ctx.exception.category, "file_missing")


class TestSupportsSchema(unittest.TestCase):
    def test_matrix(self):
        cases = [
            ("^1.0", "1.0.0", True), ("^1.0", "1.9.3", True),
            ("^1.0", "2.0.0", False), ("^1.0", "0.9.0", False),
            ("^1", "1.4.2", True),
            (">=1.0,<1.5", "1.4.99", True), (">=1.0,<1.5", "1.5.0", False),
            (">=1.2,<2.0", "1.1.9", False),
            ("latest", "1.0.0", False), ("*", "1.0.0", False),
            ("^1.0", "", False),
        ]
        for rng, ver, ok in cases:
            with self.subTest(rng=rng, ver=ver):
                try:
                    rc.supports_schema(rng, ver)
                    self.assertTrue(ok, f"supports_schema({rng!r},{ver!r}) 应拒绝")
                except rc.ContractError:
                    self.assertFalse(ok, f"supports_schema({rng!r},{ver!r}) 应通过")

    def test_go_python_category_agreement_on_schema(self):
        """schema 类错误在两侧都归因为 schema 类别（互通的类别契约）。"""
        manifest = _load_json(os.path.join(VALID_DIR, "manifest.json"))
        bad = copy.deepcopy(manifest)
        bad["schema_version"] = "0.9.0"
        with self.assertRaises(rc.ContractError) as ctx:
            rc.validate_manifest(bad)
        self.assertEqual(ctx.exception.category, "schema")


class TestJsonSchemaFiles(unittest.TestCase):
    """JSON Schema 权威文档存在性 + 自身是合法 JSON。

    schema 文件的深度校验（fixture 全量过 schema）属于 DATA-02
    校验器接线；DATA-01 保证 schema 文件存在且可解析。
    """

    def test_schema_files_exist_and_parse(self):
        schema_dir = os.path.join(REPO, "tft", "release", "contracts", "schema")
        expected = [
            "data_scope.schema.json",
            "data_manifest.schema.json",
            "deployment_manifest.schema.json",
            "evidence_envelope.schema.json",
            "session_record.schema.json",
            "route_decision.schema.json",
        ]
        for name in expected:
            with self.subTest(schema=name):
                data = _load_json(os.path.join(schema_dir, name))
                self.assertIn("$schema", data)
                self.assertIn("properties", data)


class TestReviewRegression(unittest.TestCase):
    """review 发现的对称疏漏回归（与 Go TestReviewRegression_DATA01 一致）。"""

    def test_evidence_missing_effective_at_rejected(self):
        evidence = _load_json(os.path.join(VALID_DIR, "evidence.json"))
        del evidence["freshness"]["effective_at"]
        with self.assertRaises(rc.ContractError) as ctx:
            rc.validate_evidence(evidence)
        self.assertEqual(ctx.exception.category, "time_order")

    def test_stats_scope_invalid_rejected(self):
        manifest = _load_json(os.path.join(VALID_DIR, "manifest.json"))
        for field, value in (("stats_region", ""), ("stats_region", "火星"), ("stats_queue", "")):
            with self.subTest(field=field, value=value):
                bad = copy.deepcopy(manifest)
                bad["sources"][0][field] = value
                with self.assertRaises(rc.ContractError) as ctx:
                    rc.validate_manifest(bad)
                self.assertEqual(ctx.exception.category, "scope")

    def test_stats_scope_global_unknown_not_overkilled(self):
        manifest = _load_json(os.path.join(VALID_DIR, "manifest.json"))
        manifest["sources"][0]["stats_region"] = "global"
        manifest["sources"][0]["stats_queue"] = "unknown"
        rc.validate_manifest(manifest)


if __name__ == "__main__":
    unittest.main(verbosity=2)
