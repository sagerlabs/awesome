#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从 OP.GG MCP 拉取最小 TFT meta 数据，并转换成本项目 knowledge JSON。

MVP 范围：
1. 拉取 OP.GG 当前 meta decks，默认最多保留前 20 套。
2. 只处理这些阵容里的英雄和装备。
3. 生成 final/early/middle 棋盘、核心英雄、核心装备、平均名次、前四率、吃鸡率、样本量。
4. early/middle 有就存，没有就留空，不让 LLM 编造。

常用命令：
  python3 scripts/update_opgg_mcp_mvp.py
  python3 scripts/update_opgg_mcp_mvp.py --input-response /tmp/opgg_response.json --dry-run
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import time
import urllib.request
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_ENDPOINT = "https://mcp-api.op.gg/mcp"
DEFAULT_KNOWLEDGE_DIR = REPO_ROOT / "tft" / "knowledge" / "data"
DEFAULT_LOCALIZATION = REPO_ROOT / "metadata" / "tft-meta" / "data" / "localization.json"
DEFAULT_SOURCE = "OP.GG MCP"


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


def compact_name(name: str) -> str:
    return re.sub(r"\s+", "", name or "").strip()


def load_localization(path: Path) -> dict[str, Any]:
    data = load_json(path)
    return {
        "id_to_cn": data.get("id_to_cn", {}),
        "cn_to_id": data.get("cn_to_id", {}),
        "unit_profiles": data.get("unit_profiles", {}),
    }


def cn_name(value: Any, id_to_cn: dict[str, str]) -> str:
    if not isinstance(value, str):
        return ""
    return id_to_cn.get(value, value)


def localized_deck_name(deck: dict[str, Any]) -> str:
    names = deck.get("name", {})
    if isinstance(names, dict):
        return names.get("zh_CN") or names.get("en_US") or next(iter(names.values()), "")
    if isinstance(names, str):
        return names
    return deck.get("id", "")


def split_display_names(name: str) -> list[dict[str, Any]]:
    parts = [part for part in re.split(r"\s+", name.strip()) if part]
    if not parts and name:
        parts = [name]
    return [{"name": part, "type": "opgg", "score": 1} for part in parts]


def tier_from_opgg(op_tier: str) -> str:
    upper = (op_tier or "").strip().upper()
    if upper == "OP":
        return "S"
    if upper in {"S", "A", "B", "C", "D"}:
        return upper
    return upper or "Unknown"


def tft_set_from_deck(deck: dict[str, Any]) -> str:
    team_code = deck.get("teamCode")
    if isinstance(team_code, str):
        match = re.search(r"(TFTSet\d+)", team_code)
        if match:
            return match.group(1)
    for unit in deck.get("units", []):
        key = unit.get("key")
        if isinstance(key, str):
            match = re.match(r"TFT(\d+)_", key)
            if match:
                return f"TFTSet{match.group(1)}"
    return ""


def score_for_priority(priority: Any) -> int:
    if isinstance(priority, int) and priority > 0:
        return max(40, 110 - priority * 10)
    return 50


def clean_items(items: Any, id_to_cn: dict[str, str]) -> list[str]:
    if not isinstance(items, list):
        return []
    result = []
    for item in items:
        if not isinstance(item, str) or not item:
            continue
        result.append(cn_name(item, id_to_cn))
    return result


def convert_traits(traits: Any, id_to_cn: dict[str, str]) -> list[str]:
    if not isinstance(traits, list):
        return []
    result = []
    for trait in traits:
        if not isinstance(trait, dict):
            continue
        name = cn_name(trait.get("key"), id_to_cn)
        if not name:
            continue
        count = trait.get("numUnits")
        if isinstance(count, int) and count > 0:
            result.append(f"{name} ({count})")
        else:
            result.append(name)
    return result


def convert_trait_markers(traits: Any, id_to_cn: dict[str, str]) -> list[dict[str, Any]]:
    if not isinstance(traits, list):
        return []
    result = []
    for trait in traits:
        if not isinstance(trait, dict):
            continue
        name = cn_name(trait.get("key"), id_to_cn)
        if not name:
            continue
        marker = {"name": name}
        count = trait.get("numUnits")
        if isinstance(count, int) and count > 0:
            marker["count"] = count
        result.append(marker)
    return result


def convert_board_unit(unit: dict[str, Any], id_to_cn: dict[str, str], key_field: str) -> dict[str, Any]:
    name = cn_name(unit.get(key_field), id_to_cn)
    result: dict[str, Any] = {"name": name}
    items = clean_items(unit.get("items"), id_to_cn)
    if items:
        result["items"] = items
    if unit.get("isCore") is True:
        result["is_core"] = True
    priority = unit.get("priority")
    if isinstance(priority, int):
        result["priority"] = priority
    cell = unit.get("cell")
    if isinstance(cell, dict):
        result["cell"] = cell
    return result


def convert_snapshot(snapshot: Any, id_to_cn: dict[str, str]) -> dict[str, Any] | None:
    if not isinstance(snapshot, dict):
        return None
    units = []
    for unit in snapshot.get("units", []):
        if isinstance(unit, dict):
            converted = convert_board_unit(unit, id_to_cn, "characterId")
            if converted.get("name"):
                units.append(converted)
    traits = convert_trait_markers(snapshot.get("traits"), id_to_cn)
    if not units and not traits:
        return None
    result: dict[str, Any] = {
        "level": str(snapshot.get("level", "")),
        "units": units,
        "traits": traits,
    }
    if isinstance(snapshot.get("play"), int):
        result["play"] = snapshot["play"]
    if isinstance(snapshot.get("win"), int):
        result["win"] = snapshot["win"]
    if isinstance(snapshot.get("lose"), int):
        result["lose"] = snapshot["lose"]
    return result


def levelling_from_badges(deck: dict[str, Any]) -> str:
    for badge in deck.get("badge", []):
        if isinstance(badge, dict) and badge.get("key") == "reroll" and isinstance(badge.get("value"), int):
            return f"lvl {badge['value']}"
    unit_count = len(deck.get("units", []))
    if unit_count >= 9:
        return "fast 9"
    if unit_count >= 8:
        return "fast 8"
    return ""


def difficulty_from_badges(deck: dict[str, Any]) -> float:
    for badge in deck.get("badge", []):
        if isinstance(badge, dict) and badge.get("key") == "difficulty" and isinstance(badge.get("value"), (int, float)):
            return float(badge["value"])
    return 0


def convert_deck(deck: dict[str, Any], metadata: dict[str, Any], id_to_cn: dict[str, str]) -> dict[str, Any]:
    deck_stat = ((deck.get("stat") or {}).get("deck") or {})
    cluster_id = str(deck.get("id") or (deck.get("stat") or {}).get("originHash") or "")
    deck_name = localized_deck_name(deck)
    display_name = compact_name(deck_name)
    units = [cn_name(unit.get("key"), id_to_cn) for unit in deck.get("units", []) if isinstance(unit, dict)]
    units = [unit for unit in units if unit]
    traits = convert_traits(deck.get("traits"), id_to_cn)

    builds = []
    build_items: dict[str, dict[str, Any]] = {}
    for unit in deck.get("units", []):
        if not isinstance(unit, dict):
            continue
        items = clean_items(unit.get("items"), id_to_cn)
        if not items:
            continue
        unit_name = cn_name(unit.get("key"), id_to_cn)
        priority = unit.get("priority")
        priority_scores = {item: score_for_priority(priority) for item in items}
        builds.append({
            "unit": unit_name,
            "items": items,
            "avg_placement": deck_stat.get("avgPlacement", 0),
            "count": deck_stat.get("compsCount", 0),
            "score": score_for_priority(priority),
            "place_change": 0,
            "priority_scores": priority_scores,
        })
        build_items[" + ".join(items)] = {
            "itemNames": " + ".join(items),
            "count": deck_stat.get("compsCount", 0),
            "avg": deck_stat.get("avgPlacement", 0),
            "pcnt": deck_stat.get("pickRate", 0),
        }

    final_units = []
    for unit in deck.get("units", []):
        if isinstance(unit, dict):
            converted = convert_board_unit(unit, id_to_cn, "key")
            if converted.get("name"):
                final_units.append(converted)

    plan: dict[str, Any] = {
        "cluster_id": cluster_id,
        "name": display_name,
        "tier": tier_from_opgg((deck.get("stat") or {}).get("opTier", "")),
        "final": {
            "level": str(len(final_units)) if final_units else "",
            "units": final_units,
            "traits": convert_trait_markers(deck.get("traits"), id_to_cn),
        },
    }
    early = convert_snapshot(deck.get("early"), id_to_cn)
    middle = convert_snapshot(deck.get("middle"), id_to_cn)
    if early:
        plan["early"] = early
    if middle:
        plan["middle"] = middle

    updated_at = metadata.get("gameStatDateTime") or time.strftime("%Y-%m-%dT%H:%M:%S%z")
    sample_count = deck_stat.get("compsCount") or metadata.get("gameStatCounts") or 0

    return {
        "cluster_id": cluster_id,
        "tft_set": tft_set_from_deck(deck),
        "metadata": {
            "version": tft_set_from_deck(deck),
            "source": DEFAULT_SOURCE,
            "updated_at": updated_at,
            "sample_count": sample_count,
        },
        "units": units,
        "traits": traits,
        "stars": [build["unit"] for build in builds if build.get("unit")],
        "name_string": display_name,
        "display_names": split_display_names(deck_name),
        "count": deck_stat.get("compsCount", 0),
        "avg_placement": deck_stat.get("avgPlacement", 0),
        "top4_rate": deck_stat.get("top4Rate", 0),
        "win_rate": deck_stat.get("winRate", 0),
        "tier": tier_from_opgg((deck.get("stat") or {}).get("opTier", "")),
        "builds": builds,
        "build_items": build_items,
        "trends": [{
            "day": updated_at[:10],
            "count": deck_stat.get("compsCount", 0),
            "avg": deck_stat.get("avgPlacement", 0),
            "pick_rate": deck_stat.get("pickRate", 0),
        }],
        "levelling": levelling_from_badges(deck),
        "difficulty": difficulty_from_badges(deck),
        "plan": plan,
        "limit": {
            "source_deck_id": cluster_id,
            "op_tier": (deck.get("stat") or {}).get("opTier", ""),
        },
    }


def generate_champions(comps: dict[str, dict[str, Any]]) -> dict[str, dict[str, Any]]:
    champions: dict[str, dict[str, Any]] = {}
    for cluster_id, comp in comps.items():
        for unit in comp.get("units", []):
            champions.setdefault(unit, {
                "name": unit,
                "appear_in_comps": [],
                "builds": [],
            })
            champions[unit]["appear_in_comps"].append({
                "cluster_id": cluster_id,
                "comp_name": comp.get("name_string", cluster_id),
                "tier": comp.get("tier", "Unknown"),
                "avg_placement": comp.get("avg_placement", 0),
            })
        for build in comp.get("builds", []):
            unit = build.get("unit")
            if not unit:
                continue
            champions.setdefault(unit, {
                "name": unit,
                "appear_in_comps": [],
                "builds": [],
            })
            champions[unit]["builds"].append({
                "cluster_id": cluster_id,
                "comp_name": comp.get("name_string", cluster_id),
                "items": build.get("items", []),
                "avg_placement": build.get("avg_placement", 0),
                "count": build.get("count", 0),
                "priority_scores": build.get("priority_scores", {}),
            })
    return champions


def generate_items(comps: dict[str, dict[str, Any]]) -> dict[str, dict[str, Any]]:
    items: dict[str, dict[str, Any]] = {}
    for cluster_id, comp in comps.items():
        for build in comp.get("builds", []):
            carry = build.get("unit")
            priority_scores = build.get("priority_scores", {})
            for item in build.get("items", []):
                items.setdefault(item, {"name": item, "priority_list": []})
                score = priority_scores.get(item, 50)
                items[item]["priority_list"].append({
                    "cluster_id": cluster_id,
                    "comp_name": comp.get("name_string", cluster_id),
                    "comp_tier": comp.get("tier", "Unknown"),
                    "comp_avg": comp.get("avg_placement", 0),
                    "carry": carry,
                    "priority_score": score,
                })
    for item in items.values():
        item["priority_list"].sort(key=lambda row: (-row.get("priority_score", 0), row.get("comp_avg", 99)))
    return items


def generate_champion_profiles(champions: dict[str, dict[str, Any]], localization: dict[str, Any], version: str) -> dict[str, Any]:
    unit_profiles = localization.get("unit_profiles", {})
    id_to_cn = localization.get("id_to_cn", {})
    by_name = {}
    for api_name, profile in unit_profiles.items():
        if not isinstance(profile, dict):
            continue
        name = profile.get("name") or id_to_cn.get(api_name)
        if name:
            by_name[name] = profile

    generated = {}
    for name in sorted(champions):
        profile = by_name.get(name)
        if not profile:
            continue
        cost = profile.get("cost")
        if not isinstance(cost, int) or cost <= 0 or cost > 7:
            continue
        traits = profile.get("traits", [])
        if not isinstance(traits, list):
            traits = []
        generated[name] = {
            "name": name,
            "api_name": profile.get("api_name", ""),
            "cost": cost,
            "traits": traits,
        }

    return {
        "version": version,
        "source": DEFAULT_SOURCE,
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "champions": generated,
    }


def clear_json_files(directory: Path) -> None:
    if not directory.exists():
        return
    for path in directory.glob("*.json"):
        path.unlink()


def write_knowledge(knowledge_dir: Path, comps: dict[str, dict[str, Any]], localization: dict[str, Any], clean: bool) -> None:
    for dirname in ("team_comps", "champions", "items"):
        target = knowledge_dir / dirname
        target.mkdir(parents=True, exist_ok=True)
        if clean:
            clear_json_files(target)

    for cluster_id, comp in comps.items():
        filename = sanitize_filename(f"{cluster_id}_{comp.get('name_string', cluster_id)}.json")
        write_json(knowledge_dir / "team_comps" / filename, comp)

    champions = generate_champions(comps)
    for name, champion in champions.items():
        write_json(knowledge_dir / "champions" / f"{sanitize_filename(name)}.json", champion)

    items = generate_items(comps)
    for name, item in items.items():
        write_json(knowledge_dir / "items" / f"{sanitize_filename(name)}.json", item)

    first_version = next((comp.get("tft_set", "") for comp in comps.values() if comp.get("tft_set")), "")
    write_json(knowledge_dir / "champion_profiles.json", generate_champion_profiles(champions, localization, first_version))

    print(f"  写入阵容: {len(comps)}")
    print(f"  写入英雄: {len(champions)}")
    print(f"  写入装备: {len(items)}")


def parse_mcp_response(response: dict[str, Any]) -> dict[str, Any]:
    if "error" in response:
        raise RuntimeError(f"OP.GG MCP error: {response['error']}")
    result = response.get("result", {})
    content = result.get("content", [])
    if not content:
        raise RuntimeError("OP.GG MCP response missing result.content")
    text = content[0].get("text") if isinstance(content[0], dict) else None
    if not isinstance(text, str):
        raise RuntimeError("OP.GG MCP response content is not text")
    return json.loads(text)


def fetch_meta_decks(endpoint: str, timeout: int) -> dict[str, Any]:
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "tft_list_meta_decks",
            "arguments": {},
        },
    }
    request = urllib.request.Request(
        endpoint,
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Content-Type": "application/json",
            "Accept": "application/json, text/event-stream",
            "User-Agent": "tft-copilot-opgg-mcp-mvp/1.0",
        },
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=timeout) as response:
        body = response.read().decode("utf-8")
    return parse_mcp_response(json.loads(body))


def load_meta_decks_from_response(path: Path) -> dict[str, Any]:
    response = load_json(path)
    if "data" in response and "metadata" in response:
        return response
    return parse_mcp_response(response)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="从 OP.GG MCP 生成最小 TFT knowledge JSON")
    parser.add_argument("--endpoint", default=DEFAULT_ENDPOINT, help="OP.GG MCP endpoint")
    parser.add_argument("--limit", type=int, default=20, help="最多保留多少套 meta 阵容")
    parser.add_argument("--timeout", type=int, default=30, help="网络请求超时时间，单位秒")
    parser.add_argument("--input-response", type=Path, default=None, help="使用已保存的 OP.GG MCP 响应 JSON")
    parser.add_argument("--localization", type=Path, default=DEFAULT_LOCALIZATION, help="本地中文映射 localization.json")
    parser.add_argument("--knowledge-dir", type=Path, default=DEFAULT_KNOWLEDGE_DIR, help="knowledge 输出目录")
    parser.add_argument("--no-clean", action="store_true", help="写入前不清理 team_comps/champions/items")
    parser.add_argument("--dry-run", action="store_true", help="只解析和统计，不写入文件")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    localization = load_localization(args.localization.resolve())
    id_to_cn = localization["id_to_cn"]
    if not id_to_cn:
        raise RuntimeError(f"localization 缺少 id_to_cn: {args.localization}")

    print("=" * 60)
    print("开始更新 OP.GG MCP MVP knowledge 数据")
    print(f"knowledge 目录: {args.knowledge_dir.resolve()}")
    print(f"目标数量: {args.limit}")
    print("=" * 60)

    if args.input_response:
        payload = load_meta_decks_from_response(args.input_response.resolve())
        print(f"阶段 1/3: 使用本地 OP.GG MCP 响应 {args.input_response}")
    else:
        print("阶段 1/3: 请求 OP.GG MCP tft_list_meta_decks")
        payload = fetch_meta_decks(args.endpoint, args.timeout)

    raw_decks = payload.get("data", [])
    metadata = payload.get("metadata", {})
    if not isinstance(raw_decks, list) or not raw_decks:
        raise RuntimeError("OP.GG MCP 没有返回可用 meta decks")

    selected = raw_decks[: max(args.limit, 0)]
    if len(selected) < args.limit:
        print(f"  提醒: OP.GG 当前只返回 {len(selected)} 套，少于目标 {args.limit} 套")

    print("阶段 2/3: 转换为本项目 knowledge schema")
    comps = {}
    for deck in selected:
        if not isinstance(deck, dict):
            continue
        comp = convert_deck(deck, metadata, id_to_cn)
        if comp.get("cluster_id"):
            comps[comp["cluster_id"]] = comp
    if not comps:
        raise RuntimeError("没有转换出任何阵容")

    if args.dry_run:
        champions = generate_champions(comps)
        items = generate_items(comps)
        print("阶段 3/3: dry-run，跳过写入")
        print(f"  阵容: {len(comps)}")
        print(f"  英雄: {len(champions)}")
        print(f"  装备: {len(items)}")
        print(f"  数据时间: {metadata.get('gameStatDateTime', '')}")
        return

    print("阶段 3/3: 写入 knowledge JSON")
    write_knowledge(args.knowledge_dir.resolve(), comps, localization, clean=not args.no_clean)
    print("=" * 60)
    print("OP.GG MCP MVP knowledge 数据更新完成")
    print("=" * 60)


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"错误: {exc}", file=sys.stderr)
        sys.exit(1)
