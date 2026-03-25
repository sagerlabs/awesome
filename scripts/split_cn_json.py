#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
拆分中文版JSON数据到knowledge目录
- 阵容：每个阵容一个JSON文件
- 英雄：每个英雄一个JSON文件
- 装备：每个装备一个JSON文件
"""

import json
import os
import re
from pathlib import Path

def sanitize_filename(name):
    """清理文件名，移除非法字符"""
    # 替换非法字符
    name = re.sub(r'[<>:"/\\|?*]', '_', name)
    # 移除空格，用下划线代替
    name = name.replace(' ', '_')
    return name

def split_comps(input_path, output_dir):
    """拆分阵容数据"""
    print(f"正在拆分阵容数据: {input_path}")
    
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    comps_dir = output_dir / 'team_comps'
    comps_dir.mkdir(parents=True, exist_ok=True)
    
    comps = data.get('comps', {})
    count = 0
    
    for cluster_id, comp_data in comps.items():
        # 生成文件名：使用cluster_id和显示名称
        display_names = comp_data.get('display_names', [])
        comp_name = cluster_id
        if display_names:
            # 取第一个显示名称
            comp_name = display_names[0].get('name', cluster_id)
        
        filename = sanitize_filename(f"{cluster_id}_{comp_name}.json")
        output_path = comps_dir / filename
        
        # 写入单个阵容文件
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(comp_data, f, ensure_ascii=False, indent=2)
        
        count += 1
    
    print(f"  拆分完成: {count} 个阵容文件")
    return comps

def split_champions(comps, output_dir):
    """从阵容数据中提取英雄信息"""
    print(f"正在拆分英雄数据...")
    
    champions_dir = output_dir / 'champions'
    champions_dir.mkdir(parents=True, exist_ok=True)
    
    # 收集所有英雄
    champions = {}
    
    for cluster_id, comp_data in comps.items():
        units = comp_data.get('units', [])
        builds = comp_data.get('builds', [])
        
        for unit in units:
            if unit not in champions:
                champions[unit] = {
                    'name': unit,
                    'appear_in_comps': [],
                    'builds': []
                }
            champions[unit]['appear_in_comps'].append({
                'cluster_id': cluster_id,
                'comp_name': comp_data.get('name_string', cluster_id),
                'tier': comp_data.get('tier', 'Unknown'),
                'avg_placement': comp_data.get('avg_placement', 0)
            })
        
        # 收集该英雄的出装
        for build in builds:
            unit = build.get('unit')
            if unit:
                if unit not in champions:
                    champions[unit] = {
                        'name': unit,
                        'appear_in_comps': [],
                        'builds': []
                    }
                champions[unit]['builds'].append({
                    'cluster_id': cluster_id,
                    'comp_name': comp_data.get('name_string', cluster_id),
                    'items': build.get('items', []),
                    'avg_placement': build.get('avg_placement', 0),
                    'count': build.get('count', 0),
                    'priority_scores': build.get('priority_scores', {})
                })
    
    # 写入每个英雄文件
    count = 0
    for champ_name, champ_data in champions.items():
        filename = sanitize_filename(f"{champ_name}.json")
        output_path = champions_dir / filename
        
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(champ_data, f, ensure_ascii=False, indent=2)
        
        count += 1
    
    print(f"  拆分完成: {count} 个英雄文件")

def split_items(input_path, output_dir):
    """拆分装备数据"""
    print(f"正在拆分装备数据: {input_path}")
    
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    items_dir = output_dir / 'items'
    items_dir.mkdir(parents=True, exist_ok=True)
    
    count = 0
    for item_name, item_data in data.items():
        filename = sanitize_filename(f"{item_name}.json")
        output_path = items_dir / filename
        
        # 包装一下，加入装备名称
        output_data = {
            'name': item_name,
            'priority_list': item_data
        }
        
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(output_data, f, ensure_ascii=False, indent=2)
        
        count += 1
    
    print(f"  拆分完成: {count} 个装备文件")

def main():
    # 基础路径
    base_dir = Path('/root/.openclaw/workspace/awesome')
    metadata_dir = base_dir / 'metadata' / 'tft-meta' / 'data'
    knowledge_dir = base_dir / 'tft' / 'knowledge' / 'data'
    
    # 输入文件
    comps_file = metadata_dir / 'comps_full_cn.json'
    items_file = metadata_dir / 'items_priority_cn.json'
    
    print("=" * 50)
    print("开始拆分中文版JSON数据")
    print("=" * 50)
    
    # 1. 拆分阵容
    comps = split_comps(comps_file, knowledge_dir)
    
    # 2. 拆分英雄（从阵容数据中提取）
    split_champions(comps, knowledge_dir)
    
    # 3. 拆分装备
    split_items(items_file, knowledge_dir)
    
    print("=" * 50)
    print("所有数据拆分完成！")
    print(f"输出目录: {knowledge_dir}")
    print("=" * 50)

if __name__ == '__main__':
    main()
