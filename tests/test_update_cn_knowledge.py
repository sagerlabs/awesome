"""scripts/update_cn_knowledge.py 纯函数的回归测试。

数据管线是知识库的源头：翻译/解析函数出错时产出的是"看起来正常的坏数据"，
比代码崩溃更难发现，所以这里锁定它们的行为。

只测不碰网络、不碰文件系统的纯函数；抓取与落盘逻辑由 make data 的
dry-run 流程覆盖。
"""

import importlib.util
import sys
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent
SCRIPT = REPO_ROOT / "scripts" / "update_cn_knowledge.py"

spec = importlib.util.spec_from_file_location("update_cn_knowledge", SCRIPT)
mod = importlib.util.module_from_spec(spec)
sys.modules["update_cn_knowledge"] = mod
spec.loader.exec_module(mod)


# ── 文本清洗 ──────────────────────────────────────────────────────────────────

def test_sanitize_filename_strips_unsafe_chars():
    assert mod.sanitize_filename('a<b>:c"d/e\\f|g?h*i j') == "a_b__c_d_e_f_g_h_i_j"


def test_normalize_text_collapses_whitespace_and_entities():
    # 注意：r"\n\s+" -> "\n" 的规则先于 r"\n{3,}" 执行，
    # 所以连续空行最终压成单个换行，而不是保留一个空行。
    raw = "&amp;前排　坦克\xa0加强  了\n\n\n\n削弱"
    assert mod.normalize_text(raw) == "&前排 坦克 加强 了\n削弱"


def test_html_fragment_to_text_strips_tags_and_scripts():
    html = (
        "<div><p>兰博加强</p><script>evil()</script>"
        "<style>.x{}</style>第一行<br/>第二行</div>"
    )
    text = mod.html_fragment_to_text(html)
    assert "兰博加强" in text
    assert "第一行\n第二行" in text
    assert "evil" not in text and ".x{}" not in text


def test_decode_html_honours_declared_gbk_charset():
    content = '<meta charset="gbk"><p>云顶之弈</p>'.encode("gbk")
    assert "云顶之弈" in mod.decode_html(content)


# ── ID -> 中文翻译 ────────────────────────────────────────────────────────────

ID_TO_CN = {
    "TFT16_Rumble": "兰博",
    "TFT_Item_GuinsoosRageblade": "鬼索的狂暴之刃",
}


def test_translate_token_translates_known_and_keeps_unknown():
    assert mod.translate_token("TFT16_Rumble", ID_TO_CN) == "兰博"
    assert mod.translate_token("TFT16_Unknown", ID_TO_CN) == "TFT16_Unknown"
    assert mod.translate_token(42, ID_TO_CN) == 42  # 非字符串原样返回


def test_translate_build_translates_carry_items_and_scores():
    build = {
        "carry": "TFT16_Rumble",
        "items": ["TFT_Item_GuinsoosRageblade", "TFT_Item_Unknown"],
        "priority_scores": {"TFT_Item_GuinsoosRageblade": 100},
        "avg_placement": 3.8,
    }
    out = mod.translate_build(build, ID_TO_CN)
    assert out["carry"] == "兰博"
    assert out["items"] == ["鬼索的狂暴之刃", "TFT_Item_Unknown"]
    assert out["priority_scores"] == {"鬼索的狂暴之刃": 100}
    assert out["avg_placement"] == 3.8
    # 不可原地修改输入
    assert build["carry"] == "TFT16_Rumble"


def test_translate_comp_translates_nested_structures():
    comp = {
        "units": ["TFT16_Rumble"],
        "traits": ["TFT16_Yordle"],
        "best_build": {"carry": "TFT16_Rumble", "items": []},
        "all_builds": [{"carry": "TFT16_Rumble"}],
        "tier": "S",
    }
    out = mod.translate_comp(comp, ID_TO_CN)
    assert out["units"] == ["兰博"]
    assert out["best_build"]["carry"] == "兰博"
    assert out["all_builds"][0]["carry"] == "兰博"
    assert out["tier"] == "S"


# ── 版本公告解析 ──────────────────────────────────────────────────────────────

@pytest.mark.parametrize(
    "title,text,expected",
    [
        ("更新总览", "任何内容", "general"),
        ("系统调整", "商店概率变化", "system"),
        ("排位变动", "胜点调整", "ranked"),
        ("强化符文", "白银阶调整", "augment"),
        ("装备调整", "神器加强", "item"),
        ("英雄改动", "弈子 4星 加强", "champion"),
        ("某个标题", "羁绊 职业调整", "trait"),
    ],
)
def test_infer_patch_note_type(title, text, expected):
    assert mod.infer_patch_note_type(title, text) == expected


def test_infer_patch_note_tags_collects_all_hits():
    tags = mod.infer_patch_note_tags("装备调整", "坦克 前排 双抗 加强，排位 胜点改动")
    assert "itemization" in tags
    assert "frontline" in tags
    assert "ranked" in tags
    assert "shop_odds" not in tags


def test_split_patch_note_details_skips_blank_and_caps_at_limit():
    text = "\n".join(["第一行", "", "  ", "第二行"] + [f"行{i}" for i in range(50)])
    details = mod.split_patch_note_details(text, limit=10)
    assert details[0] == "第一行"
    assert details[1] == "第二行"
    assert len(details) == 10
