"""
MetaTFT 数据爬虫 (基于真实 API)
================================
接口来源（通过抓包获得）：
  1. comps_data   - 所有阵容的完整数据（英雄/羁绊/装备构建）
  2. comps_stats  - 所有阵容的胜率/名次等统计数据
  3. comp_details - 单个阵容的详细信息（按需调用）

运行：
  pip install requests
  python scraper.py
"""

import json
import time
import logging
from typing import Optional

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
    unit: str                             # 英雄 ID，如 TFT16_Rumble
    items: list[str]                      # 装备列表（顺序即优先级）
    avg_placement: float                  # 使用该套装备的平均名次
    count: int                            # 样本数量
    score: float                          # MetaTFT 综合评分
    place_change: float                   # 与不带装备相比的名次变化（负=更好）
    priority_scores: dict[str, int] = field(default_factory=dict)
    # 示例：{"TFT_Item_Rabadons": 100, "TFT_Item_Shojin": 85}


@dataclass
class Comp:
    """完整阵容数据（合并 comps_data + comps_stats）"""

    # ── 基础信息 ──
    cluster_id: str       # MetaTFT 内部唯一 ID，如 "393000"
    tft_set: str          # 赛季，如 "TFTSet16"

    # ── 核心英雄与羁绊 ──
    units: list[str]      # 核心英雄 ID 列表，如 ["TFT16_Rumble", "TFT16_Kennen"]
    traits: list[str]     # 关键羁绊，如 ["TFT16_Yordle_4", "TFT16_Defender_1"]
    stars: list[str]      # 推荐升3星的英雄（优先级从高到低）

    # ── 阵容命名 ──
    name_string: str      # 阵容标识符，如 "TFT16_Augment_RumbleCarry"
    display_names: list[dict]  # [{"name": "TFT16_Yordle", "type": "trait", "score": 3.55}]

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
    # {"TFT_Item_Rabadons": {"count": 14680, "avg": 3.32, "pcnt": 0.398}}

    # ── 趋势 ──
    trends: list[dict] = field(default_factory=list)
    # [{"day": "2026-03-01", "count": 13088, "avg": 3.93, "pick_rate": 0.00435}]

    # ── 运营参数 ──
    levelling: str  = ""    # 推荐升级节点，如 "lvl 5"
    difficulty: float = 0.0 # 操作难度，负数=较难


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

    def fetch_comp_details(self, comp_id: str, cluster_id: str) -> Optional[dict]:
        """
        接口 3: GET /comp_details?comp={comp_id}&cluster_id={cluster_id}
        获取单个阵容详情（含强化符文推荐）
        comp_id 规则：通常为 cluster_details 的 key，如 "393000"
        cluster_id：顶层 cluster_id，如 "393"
        """
        log.info(f"Fetching comp_details: comp={comp_id} cluster={cluster_id}")
        data = self._get("comp_details", {"comp": comp_id, "cluster_id": cluster_id})
        time.sleep(0.5)
        return data


# ── 数据解析 ──────────────────────────────────────────────────────────────────

class DataParser:

    # 装备优先级评分：列表第1位=100，第2位=85，以此递减
    _PRIORITY_DECAY = [100, 85, 72, 60, 50, 42, 35]

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

        # 装备构建解析
        builds = []
        for b in raw.get("builds", []):
            items = b.get("buildName", [])
            builds.append(ItemBuild(
                unit            = b.get("unit", ""),
                items           = items,
                avg_placement   = b.get("avg", 0.0),
                count           = b.get("count", 0),
                score           = b.get("score", 0.0),
                place_change    = b.get("place_change", 0.0),
                priority_scores = self._build_priority_scores(items),
            ))
        builds.sort(key=lambda x: x.score, reverse=True)

        # 统计数据（优先用 comps_stats，兜底用 comps_data 的 overall）
        overall       = raw.get("overall", {})
        count         = stats["count"]          if stats else overall.get("count", 0)
        avg_placement = stats["avg_placement"]  if stats else overall.get("avg", 4.5)
        top4_rate     = stats["top4_rate"]       if stats else 0.0
        win_rate      = stats["win_rate"]         if stats else 0.0

        return Comp(
            cluster_id    = cid,
            tft_set       = "",
            units         = [u.strip() for u in raw.get("units_string", "").split(",") if u.strip()],
            traits        = [t.strip() for t in raw.get("traits_string", "").split(",") if t.strip()],
            stars         = raw.get("stars", []),
            name_string   = raw.get("name_string", ""),
            display_names = raw.get("name", []),
            count         = count,
            avg_placement = avg_placement,
            top4_rate     = top4_rate,
            win_rate      = win_rate,
            tier          = self._placement_to_tier(avg_placement),
            builds        = builds,
            build_items   = raw.get("build_items", {}),
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
        if avg <= 3.8: return "S"
        if avg <= 4.1: return "A"
        if avg <= 4.4: return "B"
        return "C"

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
    """将解析后的数据构建为三个输出文件"""

    def save_all(self, comps: list[Comp], tft_set: str, cluster_id: str):
        self._save_comps_full(comps, tft_set, cluster_id)
        self._save_comps_for_agent(comps, tft_set, cluster_id)
        self._save_items_priority(comps)
        self._print_summary(comps)

    def _save_comps_full(self, comps: list[Comp], tft_set: str, cluster_id: str):
        """
        comps_full.json
        完整原始数据，用于调试和离线分析
        """
        path = OUTPUT_DIR / "comps_full.json"
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
        log.info(f"✅ comps_full.json     → {path}  ({path.stat().st_size // 1024} KB)")

    def _save_comps_for_agent(self, comps: list[Comp], tft_set: str, cluster_id: str):
        """
        comps_for_agent.json
        精简格式，供 Eino Tool 层直接读取
        过滤掉样本量 <200 的阵容（数据不可靠）
        """
        valid = [c for c in comps if c.count >= 200]
        path  = OUTPUT_DIR / "comps_for_agent.json"
        data  = {
            "meta":  {"tft_set": tft_set, "cluster_id": cluster_id},
            "comps": [self._to_agent_format(c) for c in valid],
        }
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2))
        log.info(f"✅ comps_for_agent.json → {path}  ({len(valid)} 个有效阵容)")

    def _save_items_priority(self, comps: list[Comp]):
        """
        items_priority.json
        装备 -> 阵容映射，供 QueryItemFit Tool 使用
        只索引 S/A Tier 阵容，避免噪声

        结构：
        {
          "TFT_Item_Rabadons": [
            {"cluster_id": "393000", "comp_name": "...", "carry": "TFT16_Rumble",
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

        path = OUTPUT_DIR / "items_priority.json"
        path.write_text(json.dumps(index, ensure_ascii=False, indent=2))
        log.info(f"✅ items_priority.json  → {path}  ({len(index)} 个装备)")

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
        log.info(f"版本强度速览 (近{DAYS}天 {RANKS})")
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

class LocalizationBuilder:
    """
    从 Community Dragon 自动生成 ID ↔ 中文名 映射表

    真实数据结构（通过 debug_localization.py 确认）：

    tftchampions.json 每条结构：
    {
      "name": "TFT16_LeeSin",               <- 顶层 name 是英雄 ID，不是中文名
      "character_record": {
        "character_id": "TFT16_LeeSin",
        "display_name": "盲僧",             <- 中文名在这里
        ...
      }
    }

    tftitems.json 每条结构：
    {
      "name": "鬼索的狂暴之刃",             <- 中文名
      "nameId": "TFTTutorial_Item_GuinsoosRageblade",  <- 前缀是 TFTTutorial_Item_
      ...
    }
    装备 nameId 前缀是 TFTTutorial_Item_，需替换为 TFT_Item_ 与 metatft 数据对齐
    """

    BASE = "https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/zh_cn/v1"

    CHAMPION_URL = f"{BASE}/tftchampions.json"
    ITEM_URL     = f"{BASE}/tftitems.json"

    def build(self) -> bool:
        """
        拉取 CDragon 中文数据，生成 localization.json
        返回 True=成功  False=失败（不影响主流程）
        """
        try:
            id_to_cn: dict[str, str] = {}

            self._parse_champions(id_to_cn)
            self._parse_items(id_to_cn)

            if not id_to_cn:
                log.warning("未解析到任何映射，请检查 CDragon 接口结构是否变化")
                return False

            # 反向映射：中文名 → ID（供 InputParser 查表）
            cn_to_id = {cn: tft_id for tft_id, cn in id_to_cn.items()}

            output = {
                "source":   "CommunityDragon/latest",
                "id_to_cn": id_to_cn,   # "TFT16_LeeSin" -> "盲僧"
                "cn_to_id": cn_to_id,   # "盲僧"         -> "TFT16_LeeSin"
            }

            path = OUTPUT_DIR / "localization.json"
            path.write_text(json.dumps(output, ensure_ascii=False, indent=2))
            log.info(f"✅ localization.json   → {path}  ({len(id_to_cn)} 条有效映射)")
            return True

        except Exception as e:
            log.warning(f"汉化表生成失败（不影响主流程）: {e}")
            return False

    def _parse_champions(self, id_to_cn: dict):
        """
        英雄：中文名在 character_record.display_name
        只保留 TFT16_ 开头的当前赛季英雄
        """
        log.info(f"从 CDragon 拉取英雄数据: {self.CHAMPION_URL}")
        resp = requests.get(self.CHAMPION_URL, timeout=15)
        resp.raise_for_status()

        count = 0
        for entry in resp.json():
            record   = entry.get("character_record", {})
            tft_id   = record.get("character_id", "")
            cn_name  = record.get("display_name", "").strip()

            if not tft_id or not cn_name:
                continue
            # 只取当前赛季 TFT16_ 开头的英雄，过滤历史赛季和系统单位
            if not tft_id.startswith("TFT16_"):
                continue

            id_to_cn[tft_id] = cn_name
            count += 1

        log.info(f"  → 新增 {count} 条英雄映射")

    def _parse_items(self, id_to_cn: dict):
        """
        装备：nameId 前缀是 TFTTutorial_Item_，替换为 TFT_Item_ 与 metatft 对齐
        过滤掉重复条目（同名装备可能出现多次，取第一条）
        """
        log.info(f"从 CDragon 拉取装备数据: {self.ITEM_URL}")
        resp = requests.get(self.ITEM_URL, timeout=15)
        resp.raise_for_status()

        count = 0
        seen  = set()
        for entry in resp.json():
            raw_id  = entry.get("nameId", "")
            cn_name = entry.get("name", "").strip()

            if not raw_id or not cn_name:
                continue
            # 只处理 TFT 相关装备
            if not raw_id.startswith("TFT"):
                continue
            # TFTTutorial_Item_GuinsoosRageblade -> TFT_Item_GuinsoosRageblade
            tft_id = raw_id.replace("TFTTutorial_Item_", "TFT_Item_")

            # 去重：同一 ID 只取第一条
            if tft_id in seen:
                continue
            seen.add(tft_id)

            id_to_cn[tft_id] = cn_name
            count += 1

        log.info(f"  → 新增 {count} 条装备映射")


class TFTDataPipeline:

    def __init__(self):
        self.client   = MetaTFTClient()
        self.parser   = DataParser()
        self.builder  = OutputBuilder()
        self.localizer = LocalizationBuilder()

    def run(self):
        # 1. 生成汉化表（失败也继续，不阻塞主流程）
        self.localizer.build()

        # 2. 获取两个核心接口
        raw_data  = self.client.fetch_comps_data()
        raw_stats = self.client.fetch_comps_stats()

        if not raw_data:
            log.error("comps_data 接口返回为空，终止")
            return

        tft_set         = raw_data.get("tft_set", "")
        cluster_id_top  = str(raw_data.get("cluster_id", ""))
        cluster_details = raw_data.get("cluster_details", {})

        # 3. 解析统计数据
        stats_map = self.parser.parse_stats(raw_stats)
        log.info(f"TFT Set: {tft_set} | cluster: {cluster_id_top} | {len(stats_map)} 个阵容有统计数据")

        # 4. 解析所有阵容
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

        # 5. 保存阵容数据（三个文件）
        self.builder.save_all(comps, tft_set, cluster_id_top)


if __name__ == "__main__":
    TFTDataPipeline().run()