"""
FIX-02：Translator 与抓取失败路径的纯数据测试。

运行：
    /path/to/python -m unittest metadata/tft-meta/test_get_tftmeta_cn.py

不依赖网络 / pytest，stdlib unittest 即可。
"""

import os
import sys
import unittest

# 允许从 metadata/tft-meta 目录直接运行
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from get_tftmeta_cn import Translator, TFTDataPipeline  # noqa: E402


def _lookups_data(entries: list) -> dict:
    """构造最小化 lookups JSON：只含 units，items/traits 留空。"""
    return {
        "items": [],
        "units": entries,
        "traits": [],
    }


class TranslatorBasicTest(unittest.TestCase):
    def test_single_form_cn_to_id(self):
        t = Translator()
        t.apply_lookups_data(
            _lookups_data([{"apiName": "TFT16_LeeSin", "name": "盲僧", "assetNames": []}]),
            comp_unit_ids={"TFT16_LeeSin"},
        )
        self.assertEqual(t.cn_to_id["盲僧"], "TFT16_LeeSin")
        self.assertEqual(t.id_to_cn["TFT16_LeeSin"], "盲僧")

    def test_no_comp_unit_ids_falls_back_to_first_api_name(self):
        t = Translator()
        t.apply_lookups_data(
            _lookups_data([{"apiName": "TFT16_LeeSin", "name": "盲僧", "assetNames": []}]),
            comp_unit_ids=None,
        )
        # 没有 comp_unit_ids 信号时，主 ID 退化为第一个 entry 的 apiName
        self.assertEqual(t.cn_to_id["盲僧"], "TFT16_LeeSin")


class TranslatorMultiFormTest(unittest.TestCase):
    """多形态（如伊莉丝人形态 + 蜘蛛形态）的 canonical primary 选取。"""

    def test_elise_picks_comp_actual_id(self):
        """当 comps 实际用 DA_18_Elise，lookup 给出 TFT18_Elise / TFT18_EliseSpider
        两条 entry 时，cn_to_id[伊莉丝] 必须是 comps 在用的 ID。"""
        t = Translator()
        entries = [
            {"apiName": "TFT18_Elise", "name": "伊莉丝", "assetNames": ["DA_18_Elise"]},
            {"apiName": "TFT18_EliseSpider", "name": "伊莉丝", "assetNames": ["DA_18_EliseSpider"]},
        ]
        t.apply_lookups_data(_lookups_data(entries), comp_unit_ids={"DA_18_Elise"})
        # 主 ID 应选 DA_18_Elise（comps 实际用）
        self.assertEqual(t.cn_to_id["伊莉丝"], "DA_18_Elise",
                         f"期望 DA_18_Elise，实际 {t.cn_to_id['伊莉丝']}")
        # 蜘蛛形态应仍能被反向解析
        self.assertEqual(t.id_to_cn["TFT18_EliseSpider"], "伊莉丝")
        self.assertEqual(t.id_to_cn["DA_18_EliseSpider"], "伊莉丝")

    def test_elise_picks_spider_id_when_comps_uses_spider(self):
        """反之亦然：comps 用蜘蛛形态时主 ID 应为蜘蛛。"""
        t = Translator()
        entries = [
            {"apiName": "TFT18_Elise", "name": "伊莉丝", "assetNames": ["DA_18_Elise"]},
            {"apiName": "TFT18_EliseSpider", "name": "伊莉丝", "assetNames": ["DA_18_EliseSpider"]},
        ]
        t.apply_lookups_data(_lookups_data(entries), comp_unit_ids={"DA_18_EliseSpider"})
        self.assertEqual(t.cn_to_id["伊莉丝"], "DA_18_EliseSpider")

    def test_no_comp_ids_picks_first_api_name(self):
        """comps_unit_ids 为空时退化为第一个 entry 的 apiName。"""
        t = Translator()
        entries = [
            {"apiName": "TFT18_Elise", "name": "伊莉丝", "assetNames": ["DA_18_Elise"]},
            {"apiName": "TFT18_EliseSpider", "name": "伊莉丝", "assetNames": ["DA_18_EliseSpider"]},
        ]
        t.apply_lookups_data(_lookups_data(entries), comp_unit_ids=set())
        # 没人被 comps 用 → 退化为第一个 entry 的 apiName
        self.assertEqual(t.cn_to_id["伊莉丝"], "TFT18_Elise")

    def test_order_does_not_change_primary(self):
        """同一 name 多个 entry 时，主 ID 不应随 entry 顺序变化。"""
        order_a = [
            {"apiName": "TFT18_Elise", "name": "伊莉丝", "assetNames": ["DA_18_Elise"]},
            {"apiName": "TFT18_EliseSpider", "name": "伊莉丝", "assetNames": ["DA_18_EliseSpider"]},
        ]
        order_b = list(reversed(order_a))

        t_a = Translator()
        t_a.apply_lookups_data(_lookups_data(order_a), comp_unit_ids={"DA_18_Elise"})
        t_b = Translator()
        t_b.apply_lookups_data(_lookups_data(order_b), comp_unit_ids={"DA_18_Elise"})

        self.assertEqual(t_a.cn_to_id["伊莉丝"], t_b.cn_to_id["伊莉丝"],
                         f"顺序变化不应改变主 ID：a={t_a.cn_to_id} b={t_b.cn_to_id}")


class TranslatorAssetNamesTest(unittest.TestCase):
    """assetNames 别名索引。"""

    def test_asset_names_written_to_id_to_cn(self):
        t = Translator()
        t.apply_lookups_data(
            _lookups_data([
                {"apiName": "TFT18_Ahri", "name": "阿狸", "assetNames": ["DA_18_Ahri"]},
            ]),
            comp_unit_ids={"DA_18_Ahri"},
        )
        # 正向
        self.assertEqual(t.id_to_cn["TFT18_Ahri"], "阿狸")
        self.assertEqual(t.id_to_cn["DA_18_Ahri"], "阿狸")
        # 反向
        self.assertEqual(t.cn_to_id["阿狸"], "DA_18_Ahri")

    def test_duplicate_asset_first_wins(self):
        """同一 asset 在多个 entry 出现时，第一次写入的 name 应获胜。"""
        t = Translator()
        # 异常场景：两个不同 name 共用同一 asset（数据坏）
        # 期望 setdefault 行为：先到的 name 保留
        t.apply_lookups_data(
            _lookups_data([
                {"apiName": "TFT18_A", "name": "甲", "assetNames": ["DA_18_X"]},
                {"apiName": "TFT18_B", "name": "乙", "assetNames": ["DA_18_X"]},
            ]),
            comp_unit_ids={"DA_18_X"},
        )
        # 甲先出现，DA_18_X 应映射到甲
        self.assertEqual(t.id_to_cn["DA_18_X"], "甲")


class PipelineFetchFailureTest(unittest.TestCase):
    """抓取 None/空响应时不应崩溃。"""

    def test_run_returns_silently_on_none_response(self):
        """fetch_comps_data 返回 None 时，run() 应打印错误并 return，不抛异常。"""
        pipe = TFTDataPipeline()

        class _FakeClient:
            def fetch_comps_data(self):
                return None
            def fetch_comps_stats(self):
                return {}

        pipe.client = _FakeClient()
        # 不应抛 AttributeError
        try:
            pipe.run()
        except Exception as e:  # pragma: no cover
            self.fail(f"run() 在 None 响应下抛异常: {e}")

    def test_run_returns_silently_on_empty_dict(self):
        pipe = TFTDataPipeline()

        class _FakeClient:
            def fetch_comps_data(self):
                return {}
            def fetch_comps_stats(self):
                return {}

        pipe.client = _FakeClient()
        try:
            pipe.run()
        except Exception as e:  # pragma: no cover
            self.fail(f"run() 在空字典响应下抛异常: {e}")


if __name__ == "__main__":
    unittest.main()
