"""
汉化数据调试脚本
================
单独运行，不影响主爬虫
用途：探查 CDragon 各接口的真实数据结构，找到正确的字段名

运行：
    python debug_localization.py
"""

import json
import requests
from pathlib import Path

BASE = "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/zh_cn/v1"

def inspect(url: str, label: str):
    print(f"\n{'='*60}")
    print(f"[{label}]  {url}")
    print('='*60)

    resp = requests.get(url, timeout=15)
    print(f"HTTP {resp.status_code}")
    if resp.status_code != 200:
        print("请求失败，跳过")
        return

    raw = resp.json()

    # ── 顶层类型 ──
    if isinstance(raw, list):
        print(f"顶层类型: list，共 {len(raw)} 条")
        # 打印前3条，看字段
        for i, item in enumerate(raw[:3]):
            print(f"\n  第{i+1}条: {json.dumps(item, ensure_ascii=False)[:300]}")

        # 统计所有出现的 key
        all_keys = set()
        for item in raw:
            if isinstance(item, dict):
                all_keys.update(item.keys())
        print(f"\n  所有字段名: {sorted(all_keys)}")

        # 找含 TFT 的字段值
        print("\n  含 'TFT' 的字段样本（前5条）:")
        count = 0
        for item in raw:
            if not isinstance(item, dict):
                continue
            for k, v in item.items():
                if isinstance(v, str) and "TFT" in v:
                    print(f"    [{k}] = {v}")
                    count += 1
                    if count >= 5:
                        break
            if count >= 5:
                break

    elif isinstance(raw, dict):
        print(f"顶层类型: dict，共 {len(raw)} 个 key")
        print(f"顶层 keys（前20个）: {list(raw.keys())[:20]}")

        # 递归打印第一个值的结构
        first_key = list(raw.keys())[0]
        first_val = raw[first_key]
        print(f"\n  第一个 key: {first_key!r}")
        print(f"  对应值类型: {type(first_val).__name__}")
        print(f"  对应值内容: {json.dumps(first_val, ensure_ascii=False)[:300]}")

        # 找含 TFT16 的 key 或值
        print("\n  含 'TFT16' 的条目（前5个）:")
        count = 0
        for k, v in raw.items():
            if "TFT16" in str(k) or "TFT16" in str(v)[:100]:
                print(f"    key={k!r}  val={json.dumps(v, ensure_ascii=False)[:150]}")
                count += 1
                if count >= 5:
                    break

    # 保存完整原始数据供离线分析
    out = Path(f"debug_{label.replace(' ', '_')}.json")
    out.write_text(json.dumps(raw, ensure_ascii=False, indent=2))
    print(f"\n  完整数据已保存 → {out}")


if __name__ == "__main__":
    # 1. 当前用的英雄接口（返回0条，说明字段名不对）
    inspect(f"{BASE}/tftchampions.json", "tftchampions")

    # 2. 装备接口（有1064条但没有TFT16）
    inspect(f"{BASE}/tftitems.json", "tftitems")

    # 3. 尝试其他可能的路径
    candidates = [
        "https://raw.communitydragon.org/latest/cdragon/tft/zh_cn.json",
        f"{BASE}/tft-data.json",
        "https://raw.communitydragon.org/latest/game/assets/characters/tft/units/zh_cn.json",
    ]
    for url in candidates:
        try:
            r = requests.get(url, timeout=10)
            status = r.status_code
            preview = str(r.text[:200]) if status == 200 else "N/A"
            print(f"\n[候选路径] {url}")
            print(f"  HTTP {status}  预览: {preview}")
        except Exception as e:
            print(f"\n[候选路径] {url}  错误: {e}")