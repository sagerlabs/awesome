#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
更新中文版 TFT knowledge 数据。

这个脚本把原来的两段流程合并为一个入口：
1. 调用 metadata/tft-meta/get_tftmeta_cn.py 获取 MetaTFT 数据和中文映射。
2. 使用 localization.json 把 ID 版 JSON 转成中文 JSON。
3. 把中文 JSON 拆分到 tft/knowledge/data，供 knowledge 模块加载。

常用命令：
  python3 scripts/update_cn_knowledge.py
  python3 scripts/update_cn_knowledge.py --skip-fetch
  python3 scripts/update_cn_knowledge.py --skip-fetch --no-clean
"""

from __future__ import annotations

import argparse
import importlib.util
import json
import os
import re
import sys
import time
from pathlib import Path
from types import ModuleType
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_FETCH_SCRIPT = REPO_ROOT / "metadata" / "tft-meta" / "get_tftmeta_cn.py"
DEFAULT_METADATA_DIR = REPO_ROOT / "metadata" / "tft-meta" / "data"
DEFAULT_KNOWLEDGE_DIR = REPO_ROOT / "tft" / "knowledge" / "data"


def load_json(path: Path) -> Any:
    if not path.exists():
        raise FileNotFoundError(f"找不到输入文件: {path}")
    with path.open("r", encoding="utf-8") as file:
        return json.load(file)


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as file:
        json.dump(data, file, ensure_ascii=False, indent=2)
        file.write("\n")


def sanitize_filename(name: str) -> str:
    name = re.sub(r'[<>:"/\\|?*]', "_", name)
    return name.replace(" ", "_")


def load_fetch_module(script_path: Path) -> ModuleType:
    if not script_path.exists():
        raise FileNotFoundError(f"找不到获取脚本: {script_path}")

    spec = importlib.util.spec_from_file_location("tftmeta_cn_fetch", script_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"无法加载获取脚本: {script_path}")

    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module

    # 原脚本内部使用 Path("./data")。临时切到脚本目录，避免误创建 repo 根目录下的 data。
    cwd = Path.cwd()
    try:
        os.chdir(script_path.parent)
        spec.loader.exec_module(module)
    finally:
        os.chdir(cwd)

    return module


def run_fetch(fetch_script: Path, metadata_dir: Path) -> None:
    print("阶段 1/3: 获取 MetaTFT 中文源数据")
    started_at = time.time()
    module = load_fetch_module(fetch_script)
    module.OUTPUT_DIR = metadata_dir
    metadata_dir.mkdir(parents=True, exist_ok=True)

    pipeline_type = getattr(module, "TFTDataPipeline", None)
    if pipeline_type is None:
        raise RuntimeError(f"{fetch_script} 中没有找到 TFTDataPipeline")

    pipeline_type().run()

    required_files = [
        metadata_dir / "comps_full.json",
        metadata_dir / "comps_for_agent.json",
        metadata_dir / "items_priority.json",
        metadata_dir / "localization.json",
    ]
    missing = [str(path) for path in required_files if not path.exists()]
    if missing:
        raise RuntimeError("获取完成后缺少必要文件:\n" + "\n".join(missing))

    # 留 1 秒余量，避免部分文件系统时间戳精度较低导致刚写入的文件被误判为旧文件。
    stale = [str(path) for path in required_files if path.stat().st_mtime < started_at - 1]
    if stale:
        raise RuntimeError("获取脚本没有刷新必要文件，避免继续使用旧数据:\n" + "\n".join(stale))


def translate_token(value: Any, id_to_cn: dict[str, str]) -> Any:
    if not isinstance(value, str):
        return value
    return id_to_cn.get(value, value)


def translate_string_list(values: Any, id_to_cn: dict[str, str]) -> Any:
    if not isinstance(values, list):
        return values
    return [translate_token(value, id_to_cn) for value in values]


def translate_priority_scores(value: Any, id_to_cn: dict[str, str]) -> Any:
    if not isinstance(value, dict):
        return value
    return {translate_token(item, id_to_cn): score for item, score in value.items()}


def translate_build(value: Any, id_to_cn: dict[str, str]) -> Any:
    if not isinstance(value, dict):
        return value

    result = dict(value)
    if "unit" in result:
        result["unit"] = translate_token(result["unit"], id_to_cn)
    if "carry" in result:
        result["carry"] = translate_token(result["carry"], id_to_cn)
    if "items" in result:
        result["items"] = translate_string_list(result["items"], id_to_cn)
    if "priority_scores" in result:
        result["priority_scores"] = translate_priority_scores(result["priority_scores"], id_to_cn)
    return result


def translate_display_names(value: Any, id_to_cn: dict[str, str]) -> Any:
    if not isinstance(value, list):
        return value

    result = []
    for item in value:
        if not isinstance(item, dict):
            result.append(item)
            continue
        translated = dict(item)
        if "name" in translated:
            translated["name"] = translate_token(translated["name"], id_to_cn)
        result.append(translated)
    return result


def translate_build_items(value: Any, id_to_cn: dict[str, str]) -> Any:
    if not isinstance(value, dict):
        return value
    return {translate_token(item, id_to_cn): detail for item, detail in value.items()}


def translate_comp(value: Any, id_to_cn: dict[str, str]) -> Any:
    if not isinstance(value, dict):
        return value

    result = dict(value)
    for field in ("units", "traits", "stars"):
        if field in result:
            result[field] = translate_string_list(result[field], id_to_cn)

    if "display_names" in result:
        result["display_names"] = translate_display_names(result["display_names"], id_to_cn)
    if "builds" in result:
        result["builds"] = [translate_build(build, id_to_cn) for build in result["builds"]]
    if "build_items" in result:
        result["build_items"] = translate_build_items(result["build_items"], id_to_cn)
    if "best_build" in result:
        result["best_build"] = translate_build(result["best_build"], id_to_cn)
    if "all_builds" in result:
        result["all_builds"] = [translate_build(build, id_to_cn) for build in result["all_builds"]]

    return result


def translate_comps_full(metadata_dir: Path, id_to_cn: dict[str, str]) -> int:
    data = load_json(metadata_dir / "comps_full.json")
    comps = data.get("comps", {})
    data["comps"] = {
        cluster_id: translate_comp(comp, id_to_cn)
        for cluster_id, comp in comps.items()
    }
    write_json(metadata_dir / "comps_full_cn.json", data)
    return len(data["comps"])


def translate_comps_for_agent(metadata_dir: Path, id_to_cn: dict[str, str]) -> int:
    data = load_json(metadata_dir / "comps_for_agent.json")
    comps = data.get("comps", [])
    data["comps"] = [translate_comp(comp, id_to_cn) for comp in comps]
    write_json(metadata_dir / "comps_for_agent_cn.json", data)
    return len(data["comps"])


def translate_items_priority(metadata_dir: Path, id_to_cn: dict[str, str]) -> int:
    data = load_json(metadata_dir / "items_priority.json")
    translated: dict[str, list[dict[str, Any]]] = {}

    for item_id, entries in data.items():
        item_name = translate_token(item_id, id_to_cn)
        translated_entries = []
        for entry in entries:
            if not isinstance(entry, dict):
                translated_entries.append(entry)
                continue
            translated_entry = dict(entry)
            if "carry" in translated_entry:
                translated_entry["carry"] = translate_token(translated_entry["carry"], id_to_cn)
            translated_entries.append(translated_entry)
        translated.setdefault(item_name, []).extend(translated_entries)

    write_json(metadata_dir / "items_priority_cn.json", translated)
    return len(translated)


def generate_cn_json(metadata_dir: Path) -> None:
    print("阶段 2/3: 生成中文版 JSON")
    localization = load_json(metadata_dir / "localization.json")
    id_to_cn = localization.get("id_to_cn", {})
    if not isinstance(id_to_cn, dict) or not id_to_cn:
        raise RuntimeError(f"localization.json 中没有可用的 id_to_cn 映射: {metadata_dir / 'localization.json'}")

    full_count = translate_comps_full(metadata_dir, id_to_cn)
    agent_count = translate_comps_for_agent(metadata_dir, id_to_cn)
    item_count = translate_items_priority(metadata_dir, id_to_cn)
    write_json(metadata_dir / "localization_cn.json", localization)

    print(f"  生成 comps_full_cn.json: {full_count} 个阵容")
    print(f"  生成 comps_for_agent_cn.json: {agent_count} 个阵容")
    print(f"  生成 items_priority_cn.json: {item_count} 个装备")


def clear_json_files(directory: Path) -> None:
    if not directory.exists():
        return
    for path in directory.glob("*.json"):
        path.unlink()


def prepare_split_dirs(knowledge_dir: Path, clean: bool) -> None:
    for name in ("team_comps", "champions", "items"):
        target = knowledge_dir / name
        target.mkdir(parents=True, exist_ok=True)
        if clean:
            clear_json_files(target)


def split_comps(input_path: Path, output_dir: Path) -> dict[str, dict[str, Any]]:
    print(f"  拆分阵容数据: {input_path}")
    data = load_json(input_path)
    raw_comps = data.get("comps", {})

    if isinstance(raw_comps, list):
        comps = {
            str(comp.get("cluster_id", index)): comp
            for index, comp in enumerate(raw_comps)
            if isinstance(comp, dict)
        }
    else:
        comps = raw_comps

    comps_dir = output_dir / "team_comps"
    count = 0
    for cluster_id, comp_data in comps.items():
        display_names = comp_data.get("display_names", []) if isinstance(comp_data, dict) else []
        comp_name = cluster_id
        if display_names:
            comp_name = display_names[0].get("name", cluster_id)

        filename = sanitize_filename(f"{cluster_id}_{comp_name}.json")
        write_json(comps_dir / filename, comp_data)
        count += 1

    print(f"    完成: {count} 个阵容文件")
    return comps


def split_champions(comps: dict[str, dict[str, Any]], output_dir: Path) -> None:
    print("  拆分英雄数据")
    champions: dict[str, dict[str, Any]] = {}

    for cluster_id, comp_data in comps.items():
        units = comp_data.get("units", [])
        builds = comp_data.get("builds", [])

        for unit in units:
            if unit not in champions:
                champions[unit] = {
                    "name": unit,
                    "appear_in_comps": [],
                    "builds": [],
                }
            champions[unit]["appear_in_comps"].append({
                "cluster_id": cluster_id,
                "comp_name": comp_data.get("name_string", cluster_id),
                "tier": comp_data.get("tier", "Unknown"),
                "avg_placement": comp_data.get("avg_placement", 0),
            })

        for build in builds:
            unit = build.get("unit")
            if not unit:
                continue
            if unit not in champions:
                champions[unit] = {
                    "name": unit,
                    "appear_in_comps": [],
                    "builds": [],
                }
            champions[unit]["builds"].append({
                "cluster_id": cluster_id,
                "comp_name": comp_data.get("name_string", cluster_id),
                "items": build.get("items", []),
                "avg_placement": build.get("avg_placement", 0),
                "count": build.get("count", 0),
                "priority_scores": build.get("priority_scores", {}),
            })

    champions_dir = output_dir / "champions"
    for champ_name, champ_data in champions.items():
        filename = sanitize_filename(f"{champ_name}.json")
        write_json(champions_dir / filename, champ_data)

    print(f"    完成: {len(champions)} 个英雄文件")


def split_items(input_path: Path, output_dir: Path) -> None:
    print(f"  拆分装备数据: {input_path}")
    data = load_json(input_path)
    items_dir = output_dir / "items"

    count = 0
    for item_name, item_data in data.items():
        filename = sanitize_filename(f"{item_name}.json")
        write_json(items_dir / filename, {
            "name": item_name,
            "priority_list": item_data,
        })
        count += 1

    print(f"    完成: {count} 个装备文件")


def split_cn_json(metadata_dir: Path, knowledge_dir: Path, clean: bool) -> None:
    print("阶段 3/3: 拆分中文版 JSON 到 knowledge")
    prepare_split_dirs(knowledge_dir, clean)
    comps = split_comps(metadata_dir / "comps_full_cn.json", knowledge_dir)
    split_champions(comps, knowledge_dir)
    split_items(metadata_dir / "items_priority_cn.json", knowledge_dir)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="更新中文版 TFT knowledge JSON 数据")
    parser.add_argument(
        "--skip-fetch",
        action="store_true",
        help="跳过 MetaTFT 网络获取，只使用 metadata/tft-meta/data 下已有 JSON",
    )
    parser.add_argument(
        "--no-clean",
        action="store_true",
        help="拆分前不清理 knowledge 下已有的阵容/英雄/装备 JSON",
    )
    parser.add_argument(
        "--fetch-script",
        type=Path,
        default=DEFAULT_FETCH_SCRIPT,
        help="MetaTFT 中文获取脚本路径",
    )
    parser.add_argument(
        "--metadata-dir",
        type=Path,
        default=DEFAULT_METADATA_DIR,
        help="metadata JSON 输入/输出目录",
    )
    parser.add_argument(
        "--knowledge-dir",
        type=Path,
        default=DEFAULT_KNOWLEDGE_DIR,
        help="knowledge 单文件 JSON 输出目录",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    fetch_script = args.fetch_script.resolve()
    metadata_dir = args.metadata_dir.resolve()
    knowledge_dir = args.knowledge_dir.resolve()

    print("=" * 60)
    print("开始更新中文版 TFT knowledge 数据")
    print(f"metadata 目录: {metadata_dir}")
    print(f"knowledge 目录: {knowledge_dir}")
    print("=" * 60)

    if not args.skip_fetch:
        run_fetch(fetch_script, metadata_dir)
    else:
        print("阶段 1/3: 已跳过 MetaTFT 网络获取")

    generate_cn_json(metadata_dir)
    split_cn_json(metadata_dir, knowledge_dir, clean=not args.no_clean)

    print("=" * 60)
    print("中文版 TFT knowledge 数据更新完成")
    print("=" * 60)


if __name__ == "__main__":
    main()
