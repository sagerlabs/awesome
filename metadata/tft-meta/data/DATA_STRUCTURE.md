# TFT 数据文件结构说明

## 三个文件结构关系与字段说明

### 整体关系

```
comps_for_agent.json (精简版，Agent用)
        ↓ cluster_id关联
comps_full.json (完整版，带display_names等)
        ↓ item关联
items_priority.json (装备→阵容反向索引)
```

---

### 1. `comps_for_agent.json` - Agent使用的精简阵容数据

**结构：**
```json
{
  "meta": { "tft_set": "TFTSet16", "cluster_id": "393" },
  "comps": [ ... ]  // 数组格式
}
```

**comps数组元素字段：**

| 字段 | 作用 | 示例 |
|------|------|------|
| `cluster_id` | 阵容唯一ID | `"393003"` |
| `name` | 阵容名称 | `"TFT16_Zaun, TFT16_Warwick, TFT16_Wukong"` |
| `tier` | 阵容等级 | `"S"` / `"A"` |
| `avg_placement` | 平均排名（越低越好） | `3.5921` |
| `top4_rate` | 前4率 | `0.6991` |
| `win_rate` | 吃鸡率 | `0.152` |
| `count` | 样本数 | `3592` |
| `units` | 阵容包含的英雄 | `["TFT16_DrMundo", ...]` |
| `traits` | 阵容激活的羁绊 | `["TFT16_Brawler_1", ...]` |
| `stars` | 推荐追3星的英雄 | `["TFT16_Jinx", ...]` |
| `levelling` | 升级节奏 | `"Standard"` |
| `difficulty` | 难度系数 | `-0.09` |
| `best_build` | 最优出装 | `{carry, items, priority_scores}` |
| `all_builds` | 所有可选出装 | 数组，每个元素含score、avg_placement |

---

### 2. `comps_full.json` - 完整版阵容数据（包含display_names等）

**结构：**
```json
{
  "meta": { "tft_set": "TFTSet16", "cluster_id": "393", "total": 66, ... },
  "comps": { "393003": { ... }, ... }  // 对象格式，key=cluster_id
}
```

**与comps_for_agent的相同字段：**
- `cluster_id`, `tft_set`, `units`, `traits`, `stars`
- `count`, `avg_placement`, `top4_rate`, `win_rate`, `tier`

**comps_full独有的字段：**

| 字段 | 作用 |
|------|------|
| `name_string` | 同comps_for_agent的name |
| `display_names` | 显示名称列表（带type和score），用于前端展示 |
| `builds` | 同comps_for_agent的all_builds，但字段名是`unit`而非`carry` |

---

### 3. `items_priority.json` - 装备优先级反向索引

**结构：**
```json
{
  "TFT_Item_Bloodthirster": [ ... ],
  "TFT_Item_GuardianAngel": [ ... ],
  ...
}
```

**每个装备数组元素字段：**

| 字段 | 作用 |
|------|------|
| `cluster_id` | 关联的阵容ID |
| `comp_name` | 阵容名称 |
| `comp_tier` | 阵容等级 |
| `comp_avg` | 阵容平均排名 |
| `carry` | 该装备给哪个英雄 |
| `priority_score` | 装备优先级分数（100最高） |

---

### 关联关系总结

1. **cluster_id 是主键**：三个文件都用 `cluster_id` 作为阵容的唯一标识
2. **comps_for_agent vs comps_full**：
   - 相同的阵容数据，只是格式不同（数组 vs 对象）
   - comps_full包含更多展示用字段（display_names）
   - comps_for_agent是精简版，专门给Agent使用
3. **items_priority 是反向索引**：从装备角度出发，快速查"这个装备适合哪些阵容"
