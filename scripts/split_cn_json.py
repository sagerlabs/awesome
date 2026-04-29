#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
兼容旧入口：只执行“生成中文 JSON + 拆分到 knowledge”，不重新联网获取。

新入口请使用：
  python3 scripts/update_cn_knowledge.py
"""

from __future__ import annotations

import runpy
import sys
from pathlib import Path


def main() -> None:
    script = Path(__file__).with_name("update_cn_knowledge.py")
    sys.argv = [str(script), "--skip-fetch", *sys.argv[1:]]
    runpy.run_path(str(script), run_name="__main__")


if __name__ == "__main__":
    main()
