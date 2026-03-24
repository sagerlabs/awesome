
import requests
import json

# 获取 comps_data
url = "https://api-hc.metatft.com/tft-comps-api/comps_data?queue=1100"
resp = requests.get(url, timeout=15)
data = resp.json()

cluster_details = data["results"]["data"]["cluster_details"]

# 收集所有阵容的 avg_placement
avg_placements = []
for cid, comp in cluster_details.items():
    avg = comp.get("overall", {}).get("avg", 0)
    if avg > 0:
        avg_placements.append((cid, avg))

# 按 avg_placement 排序
avg_placements.sort(key=lambda x: x[1])

print("="*80)
print("阵容 avg_placement 分布（按升序排列）")
print("="*80)
print(f"{'Rank':<6} {'Cluster':<10} {'Avg Placement':<15} {'Name':<30}")
print("-"*80)

for i, (cid, avg) in enumerate(avg_placements[:50], 1):
    comp = cluster_details[cid]
    name = comp.get("name_string", "")[:30]
    print(f"{i:<6} {cid:<10} {avg:<15.4f} {name:<30}")

print("\n" + "="*80)
print("关键阈值分析")
print("="*80)
print(f"第 10 名: {avg_placements[9][1]:.4f}")
print(f"第 20 名: {avg_placements[19][1]:.4f}")
print(f"第 30 名: {avg_placements[29][1]:.4f}")
print(f"第 40 名: {avg_placements[39][1]:.4f}")
print(f"第 50 名: {avg_placements[49][1]:.4f}")

print("\n" + "="*80)
print("全部阵容数量:", len(avg_placements))
print("="*80)

# 查看每个阵容的完整数据，看看有没有 tier 字段
print("\n" + "="*80)
print("检查是否有 tier 相关字段")
print("="*80)
first_comp = cluster_details[avg_placements[0][0]]
print("第一个阵容的所有字段:")
for key in sorted(first_comp.keys()):
    print(f"  {key}: {type(first_comp[key]).__name__}")
