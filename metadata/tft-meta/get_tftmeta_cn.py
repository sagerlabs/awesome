"""
MetaTFT 数据爬虫 (基于真实 API) - 中文版本
================================
接口来源（通过抓包获得）：
  1. comps_data   - 所有阵容的完整数据（英雄/羁绊/装备构建）
  2. comps_stats  - 所有阵容的胜率/名次等统计数据
  3. comp_details - 单个阵容的详细信息（按需调用）
  4. lookups      - 中英文对照表

运行：
  pip install requests
  python get_tftmeta_cn.py
"""

import json
import time
import logging
from typing import Optional, Any

import requests
from pathlib import Path
from dataclasses import dataclass, asdict, field

# ── 配置 ──────────────────────────────────────────────────────────────────────

OUTPUT_DIR = Path("./data")
OUTPUT_DIR.mkdir(exist_ok=True)

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger(__name__)

BASE_URL = "https://api-hc.metatft.com/tft-comps-api"

HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/124.0.0.0 Safari/537.36"
    ),
    "Accept": "application/json",
    "Referer": "https://www.metatft.com/",
    "Origin": "https://www.metatft.com",
}

# API 参数
QUEUE_ID = 1100   # 1100=排位  1090=双人火焰
RANKS    = "CHALLENGER,GRANDMASTER,MASTER,DIAMOND,EMERALD,PLATINUM"
DAYS     = 3


# ── 数据结构 ──────────────────────────────────────────────────────────────────

@dataclass
class ItemBuild:
    """单个英雄的最优装备组合"""
    unit: str                             # 英雄 ID（中文）
    items: list[str]                      # 装备列表（中文，顺序即优先级）
    avg_placement: float                  # 使用该套装备的平均名次
    count: int                            # 样本数量
    score: float                          # MetaTFT 综合评分
    place_change: float                   # 与不带装备相比的名次变化（负=更好）
    priority_scores: dict[str, int] = field(default_factory=dict)
    # 示例：{"灭世者的死亡之帽": 100, "朔极之矛": 85}


@dataclass
class Comp:
    """完整阵容数据（合并 comps_data + comps_stats）"""

    # ── 基础信息 ──
    cluster_id: str       # MetaTFT 内部唯一 ID，如 "393000"
    tft_set: str          # 赛季，如 "TFTSet16"

    # ── 核心英雄与羁绊 ──
    units: list[str]      # 核心英雄列表（中文）
    traits: list[str]     # 关键羁绊（中文）
    stars: list[str]      # 推荐升3星的英雄（中文，优先级从高到低）

    # ── 阵容命名 ──
    name_string: str      # 阵容标识符（中文）
    display_names: list[dict]  # [{"name": "约德尔", "type": "trait", "score": 3.55}]  name已替换为中文

    # ── 统计数据 ──
    count: int            # 总场次
    avg_placement: float  # 平均名次（越低越好，满分=1.0）
    top4_rate: float      # 进前4率
    win_rate: float       # 第一名率
    tier: str             # S/A/B/C

    # ── 装备构建 ──
    builds: list[ItemBuild]   # 各 carry 的最优装备方案，按 score 降序

    # ── 装备出现频率（全阵容维度）──
    build_items: dict[str, dict] = field(default_factory=dict)
    # {"灭世者的死亡之帽": {"count": 14680, "avg": 3.32, "pcnt": 0.398}}

    # ── 趋势 ──
    trends: list[dict] = field(default_factory=list)
    # [{"day": "2026-03-01", "count": 13088, "avg": 3.93, "pick_rate": 0.00435}]

    # ── 运营参数 ──
    levelling: str  = ""    # 推荐升级节点，如 "lvl 5"
    difficulty: float = 0.0 # 操作难度，负数=较难


# ── 中文翻译器 ────────────────────────────────────────────────────────────────

class Translator:
    """翻译器：把 TFT ID 替换为中文"""

    def __init__(self):
        self.id_to_cn: dict[str, str] = {}
        self.cn_to_id: dict[str, str] = {}
        self.trait_translations: dict[str, str] = {}

    def load_from_lookups(self, tft_set: str = "TFTSet16") -> bool:
        """从 MetaTFT lookups 接口加载翻译"""
        try:
            url = f"https://data.metatft.com/lookups/{tft_set}_latest_zh_cn.json"
            log.info(f"加载翻译表: {url}")
            resp = requests.get(url, timeout=15)
            resp.raise_for_status()
            data = resp.json()

            # 解析装备
            items = data.get("items", [])
            for item in items:
                api_name = item.get("apiName", "")
                name = item.get("name", "")
                if api_name and name:
                    self.id_to_cn[api_name] = name
                    self.cn_to_id[name] = api_name

            # 解析英雄
            units = data.get("units", [])
            for unit in units:
                api_name = unit.get("apiName", "")
                name = unit.get("name", "")
                if api_name and name:
                    self.id_to_cn[api_name] = name
                    self.cn_to_id[name] = api_name

            # 解析羁绊
            traits = data.get("traits", [])
            for trait in traits:
                api_name = trait.get("apiName", "")
                name = trait.get("name", "")
                if api_name and name:
                    self.id_to_cn[api_name] = name
                    self.cn_to_id[name] = api_name
                    # 也处理带数量的羁绊，比如 "TFT16_Yordle_4" -> "约德尔 (4)"
                    for i in range(1, 10):
                        trait_with_num = f"{api_name}_{i}"
                        self.id_to_cn[trait_with_num] = f"{name} ({i})"

            log.info(f"翻译表加载完成，共 {len(self.id_to_cn)} 条")
            return True
        except Exception as e:
            log.warning(f"翻译表加载失败: {e}")
            return False

    def t(self, s: str) -> str:
        """翻译单个字符串，如果找不到就返回原字符串"""
        if not s:
            return s
        return self.id_to_cn.get(s, s)

    def t_list(self, lst: list[str]) -> list[str]:
        """翻译列表"""
        return [self.t(item) for item in lst]

    def t_dict_keys(self, d: dict) -> dict:
        """翻译字典的 key"""
        return {self.t(k): v for k, v in d.items()}


# ── API 客户端 ────────────────────────────────────────────────────────────────

class MetaTFTClient:

    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update(HEADERS)

    def _get(self, endpoint: str, params: dict) -> Optional[dict]:
        url = f"{BASE_URL}/{endpoint}"
        try:
            resp = self.session.get(url, params=params, timeout=15)
            resp.raise_for_status()
            return resp.json()
        except requests.HTTPError as e:
            log.error(f"HTTP {e.response.status_code}: {url}")
        except Exception as e:
            log.error(f"Request failed: {e}")
        return None

    def fetch_comps_data(self) -> dict:
        """
        接口 1: GET /comps_data?queue=1100
        返回所有阵容完整信息
        结构: results.data.cluster_details -> {cluster_id: 阵容详情}
        """
        log.info("Fetching comps_data ...")
        data = self._get("comps_data", {"queue": QUEUE_ID})
        return data.get("results", {}).get("data", {}) if data else {}

    def fetch_comps_stats(self) -> list[dict]:
        """
        接口 2: GET /comps_stats?queue=1100&patch=current&days=3&rank=...
        返回所有阵容名次分布（用于计算胜率/进4率）
        结构: results -> [{cluster, places: [1名,2名,...,8名,总], count}]
        """
        log.info("Fetching comps_stats ...")
        data = self._get("comps_stats", {
            "queue":                   QUEUE_ID,
            "patch":                   "current",
            "days":                    DAYS,
            "rank":                    RANKS,
            "permit_filter_adjustment":"true",
        })
        return data.get("results", []) if data else []


# ── 数据解析 ──────────────────────────────────────────────────────────────────

class DataParser:

    # 装备优先级评分：列表第1位=100，第2位=85，以此递减
    _PRIORITY_DECAY = [100, 85, 72, 60, 50, 42, 35]

    def __init__(self, translator: Translator):
        self.translator = translator

    def parse_stats(self, raw_stats: list[dict]) -> dict[str, dict]:
        """
        解析 comps_stats，构建 cluster_id -> 统计数据 映射
        places 格式：[1名数, 2名数, 3名数, 4名数, 5名数, 6名数, 7名数, 8名数, 总数]
        """
        result = {}
        for item in raw_stats:
            cid    = item.get("cluster", "")
            places = item.get("places", [])
            count  = item.get("count", 0)
            if not cid or count == 0 or len(places) < 8:
                continue

            top4    = sum(places[:4])
            weighted = sum((i + 1) * places[i] for i in range(8))

            result[cid] = {
                "count":         count,
                "avg_placement": round(weighted / count, 4),
                "top4_rate":     round(top4 / count, 4),
                "win_rate":      round(places[0] / count, 4),
            }
        return result

    def parse_comp(self, cid: str, raw: dict, stats: Optional[dict]) -> Comp:
        """解析单个阵容的完整数据"""
        t = self.translator

        # 装备构建解析
        builds = []
        for b in raw.get("builds", []):
            items = b.get("buildName", [])
            builds.append(ItemBuild(
                unit            = t.t(b.get("unit", "")),
                items           = t.t_list(items),
                avg_placement   = b.get("avg", 0.0),
                count           = b.get("count", 0),
                score           = b.get("score", 0.0),
                place_change    = b.get("place_change", 0.0),
                priority_scores = self._build_priority_scores(t.t_list(items)),
            ))
        builds.sort(key=lambda x: x.score, reverse=True)

        # 统计数据（优先用 comps_stats，兜底用 comps_data 的 overall）
        overall       = raw.get("overall", {})
        count         = stats["count"]          if stats else overall.get("count", 0)
        avg_placement = stats["avg_placement"]  if stats else overall.get("avg", 4.5)
        top4_rate     = stats["top4_rate"]       if stats else 0.0
        win_rate      = stats["win_rate"]         if stats else 0.0

        # 处理 display_names，把 name 替换为中文
        display_names = raw.get("name", [])
        translated_display_names = []
        for dn in display_names:
            new_dn = dn.copy()
            if "name" in new_dn:
                new_dn["name"] = t.t(new_dn["name"])
            translated_display_names.append(new_dn)

        # 处理 build_items：key 翻译为中文，value 里的 itemNames 也翻译为中文
        build_items_translated = {}
        raw_build_items = raw.get("build_items", {})
        for item_id, item_data in raw_build_items.items():
            cn_item_name = t.t(item_id)
            # 复制 item_data，然后翻译 itemNames
            item_data_translated = item_data.copy() if isinstance(item_data, dict) else item_data
            if isinstance(item_data_translated, dict) and "itemNames" in item_data_translated:
                item_data_translated["itemNames"] = t.t(item_data_translated["itemNames"])
            build_items_translated[cn_item_name] = item_data_translated

        return Comp(
            cluster_id    = cid,
            tft_set       = "",
            units         = t.t_list([u.strip() for u in raw.get("units_string", "").split(",") if u.strip()]),
            traits        = t.t_list([t.strip() for t in raw.get("traits_string", "").split(",") if t.strip()]),
            stars         = t.t_list(raw.get("stars", [])),
            name_string   = t.t(raw.get("name_string", "")),
            display_names = translated_display_names,
            count         = count,
            avg_placement = avg_placement,
            top4_rate     = top4_rate,
            win_rate      = win_rate,
            tier          = self._placement_to_tier(avg_placement),
            builds        = builds,
            build_items   = build_items_translated,
            trends        = self._parse_trends(raw.get("trends", [])),
            levelling     = raw.get("levelling", ""),
            difficulty    = raw.get("difficulty", 0.0),
        )

    def _build_priority_scores(self, items: list[str]) -> dict[str, int]:
        """根据装备排列顺序生成优先级评分（第1位=100分）"""
        scores = {}
        for i, name in enumerate(items):
            scores[name] = (
                self._PRIORITY_DECAY[i]
                if i < len(self._PRIORITY_DECAY)
                else max(30, 35 - (i - len(self._PRIORITY_DECAY)) * 5)
            )
        return scores

    def _placement_to_tier(self, avg: float) -> str:
        if avg <= 4.25: return "S"
        if avg <= 4.52: return "A"
        if avg <= 4.78: return "B"
        if avg <= 5.10: return "C"
        return "D"

    def _parse_trends(self, raw: list[dict]) -> list[dict]:
        return [
            {
                "day":       t.get("day", "")[:10],
                "count":     t.get("count", 0),
                "avg":       t.get("avg", 0.0),
                "pick_rate": round(t.get("pick", 0.0), 6),
            }
            for t in raw
        ]


# ── 输出构建 ──────────────────────────────────────────────────────────────────

class OutputBuilder:
    """将解析后的数据构建为三个输出文件（全中文 value）"""

    def __init__(self, translator: Translator):
        self.translator = translator

    def save_all(self, comps: list[Comp], tft_set: str, cluster_id: str):
        self._save_comps_full(comps, tft_set, cluster_id)
        self._save_comps_for_agent(comps, tft_set, cluster_id)
        self._save_items_priority(comps)
        self._print_summary(comps)

    def _save_comps_full(self, comps: list[Comp], tft_set: str, cluster_id: str):
        """
        comps_full_cn.json
        完整原始数据（全中文），用于调试和离线分析
        """
        path = OUTPUT_DIR / "comps_full_cn.json"
        data = {
            "meta": {
                "tft_set":    tft_set,
                "cluster_id": cluster_id,
                "total":      len(comps),
                "ranks":      RANKS,
                "days":       DAYS,
            },
            "comps": {c.cluster_id: asdict(c) for c in comps},
        }
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2))
        log.info(f"✅ comps_full_cn.json     → {path}  ({path.stat().st_size // 1024} KB)")

    def _save_comps_for_agent(self, comps: list[Comp], tft_set: str, cluster_id: str):
        """
        comps_for_agent_cn.json
        精简格式（全中文），供 Eino Tool 层直接读取
        过滤掉样本量 <200 的阵容（数据不可靠）
        """
        valid = [c for c in comps if c.count >= 200]
        path  = OUTPUT_DIR / "comps_for_agent_cn.json"
        data  = {
            "meta":  {"tft_set": tft_set, "cluster_id": cluster_id},
            "comps": [self._to_agent_format(c) for c in valid],
        }
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2))
        log.info(f"✅ comps_for_agent_cn.json → {path}  ({len(valid)} 个有效阵容)")

    def _save_items_priority(self, comps: list[Comp]):
        """
        items_priority_cn.json
        装备 -> 阵容映射（全中文），供 QueryItemFit Tool 使用
        只索引 S/A Tier 阵容，避免噪声

        结构：
        {
          "灭世者的死亡之帽": [
            {"cluster_id": "393000", "comp_name": "兰博主C", "carry": "兰博",
             "priority_score": 100, "comp_tier": "S", "comp_avg": 3.72},
            ...
          ]
        }
        """
        index: dict[str, list[dict]] = {}

        for comp in comps:
            if comp.tier not in ("S", "A"):
                continue
            for build in comp.builds:
                for item, score in build.priority_scores.items():
                    if item not in index:
                        index[item] = []
                    index[item].append({
                        "cluster_id":     comp.cluster_id,
                        "comp_name":      comp.name_string,
                        "comp_tier":      comp.tier,
                        "comp_avg":       comp.avg_placement,
                        "carry":          build.unit,
                        "priority_score": score,
                    })

        # 每个装备的推荐列表按 priority_score 降序
        for item in index:
            index[item].sort(key=lambda x: x["priority_score"], reverse=True)

        path = OUTPUT_DIR / "items_priority_cn.json"
        path.write_text(json.dumps(index, ensure_ascii=False, indent=2))
        log.info(f"✅ items_priority_cn.json  → {path}  ({len(index)} 个装备)")

    def _to_agent_format(self, comp: Comp) -> dict:
        best = comp.builds[0] if comp.builds else None
        return {
            "cluster_id":    comp.cluster_id,
            "name":          comp.name_string,
            "tier":          comp.tier,
            "avg_placement": comp.avg_placement,
            "top4_rate":     comp.top4_rate,
            "win_rate":      comp.win_rate,
            "count":         comp.count,
            "units":         comp.units,
            "traits":        comp.traits,
            "stars":         comp.stars[:3],
            "levelling":     comp.levelling,
            "difficulty":    comp.difficulty,
            "best_build": {
                "carry":           best.unit            if best else "",
                "items":           best.items           if best else [],
                "priority_scores": best.priority_scores if best else {},
                "avg_placement":   best.avg_placement   if best else 0.0,
                "place_change":    best.place_change    if best else 0.0,
            },
            "all_builds": [
                {
                    "carry":           b.unit,
                    "items":           b.items,
                    "priority_scores": b.priority_scores,
                    "score":           b.score,
                    "avg_placement":   b.avg_placement,
                }
                for b in comp.builds
            ],
        }

    def _print_summary(self, comps: list[Comp]):
        tiers = {"S": [], "A": [], "B": [], "C": []}
        for c in comps:
            tiers.get(c.tier, tiers["C"]).append(c)

        log.info(f"\n{'='*60}")
        log.info(f"版本强度速览 (近{DAYS}天 {RANKS}) - 中文版本")
        log.info(f"{'='*60}")
        for tier in ["S", "A", "B", "C"]:
            cs = tiers[tier]
            log.info(f"\n[{tier} Tier] {len(cs)} 个阵容")
            for c in cs[:5]:
                log.info(
                    f"  {c.cluster_id}  avg={c.avg_placement:.2f}  "
                    f"top4={c.top4_rate:.1%}  win={c.win_rate:.1%}  "
                    f"n={c.count:,}  {c.name_string}"
                )


# ── 主流程 ────────────────────────────────────────────────────────────────────

class TFTDataPipeline:

    def __init__(self):
        self.client   = MetaTFTClient()
        self.translator = Translator()
        self.builder  = None  # 等翻译表加载后初始化
        self.parser   = None  # 等翻译表加载后初始化

    def run(self):
        # 0. 先加载翻译表
        raw_data  = self.client.fetch_comps_data()
        tft_set = raw_data.get("tft_set", "TFTSet16") if raw_data else "TFTSet16"
        self.translator.load_from_lookups(tft_set)

        # 初始化依赖翻译表的组件
        self.parser = DataParser(self.translator)
        self.builder = OutputBuilder(self.translator)

        # 1. 获取两个核心接口
        if not raw_data:
            log.error("comps_data 接口返回为空，终止")
            return

        raw_stats = self.client.fetch_comps_stats()

        cluster_id_top  = str(raw_data.get("cluster_id", ""))
        cluster_details = raw_data.get("cluster_details", {})

        # 保存原始的 localization.json（方便调试）
        self._save_localization()

        # 2. 解析统计数据
        stats_map = self.parser.parse_stats(raw_stats)
        log.info(f"TFT Set: {tft_set} | cluster: {cluster_id_top} | {len(stats_map)} 个阵容有统计数据")

        # 3. 解析所有阵容
        comps: list[Comp] = []
        for cid, raw_comp in cluster_details.items():
            try:
                comp = self.parser.parse_comp(cid, raw_comp, stats_map.get(cid))
                comp.tft_set = tft_set
                comps.append(comp)
            except Exception as e:
                log.warning(f"解析阵容 {cid} 失败: {e}")

        log.info(f"共解析 {len(comps)} 个阵容")
        comps.sort(key=lambda c: c.avg_placement)

        # 4. 保存阵容数据（三个文件，全中文）
        self.builder.save_all(comps, tft_set, cluster_id_top)

    def _save_localization(self):
        """保存原始的 localization.json（方便调试）"""
        output = {
            "source":   "MetaTFT",
            "id_to_cn": self.translator.id_to_cn,
            "cn_to_id": self.translator.cn_to_id,
        }

        path = OUTPUT_DIR / "localization_cn.json"
        path.write_text(json.dumps(output, ensure_ascii=False, indent=2))
        log.info(f"✅ localization_cn.json   → {path}  ({len(self.translator.id_to_cn)} 条翻译)")


if __name__ == "__main__":
    TFTDataPipeline().run()
