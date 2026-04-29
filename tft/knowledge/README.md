# TFT知识库模块

云顶之弈（Teamfight Tactics）知识库模块，为TFT Copilot提供增强的知识检索和RAG推理能力。

> **当前状态**：独立模块，位于 `tft/knowledge/` 目录下
> **未来集成选项**：
> 1. 作为Tool集成到现有TFT Agent中
> 2. 写成OpenClaw Skills
> 3. 作为独立API服务提供

---

## 🎯 模块特性

- ✅ **极简架构** - JSON/YAML文件管理数据，无需复杂数据库
- ✅ **版本控制** - 数据即代码，Git管理完整历史
- ✅ **混合检索** - 关键词检索 + 语义检索（Qdrant）
- ✅ **RAG推理** - 基于知识的智能问答
- ✅ **易于集成** - 预留多种集成方式

---

## 📁 目录结构

```
knowledge/
├── models/                   # 数据模型
│   ├── champion.go          # 英雄模型
│   ├── item.go              # 装备模型
│   ├── trait.go             # 羁绊模型
│   ├── team_comp.go         # 阵容模型
│   └── knowledge.go         # 知识文档模型
├── data/                     # 数据文件
│   ├── champions/            # 英雄数据 (JSON)
│   │   └── ahri.json
│   ├── items/                # 装备数据 (JSON)
│   │   └── rabadons_deathcap.json
│   ├── traits/               # 羁绊数据 (JSON)
│   │   └── astral.json
│   ├── team_comps/           # 阵容策略 (YAML)
│   │   └── ap_carry.yaml
│   └── knowledge/            # 知识文档 (Markdown)
│       └── economy_guide.md
├── docs/                     # 文档
│   └── ROADMAP.md           # 开发路线图
├── deploy/                   # 部署配置
│   └── docker/               # Docker相关
└── README.md                 # 本文档
```

---

## 📊 数据模型

### Champion（英雄）
- 基础信息：ID、名称、费用
- 属性：生命值、法力值、攻击力等
- 技能：名称、描述、法力消耗
- 分析：优点、缺点、最佳装备、羁绊组合

### Item（装备）
- 基础信息：ID、名称、类型
- 合成配方：基础装备组合
- 效果：属性加成、特殊效果
- 适用英雄：推荐使用的英雄列表

### Trait（羁绊）
- 基础信息：ID、名称、描述
- 断点：不同数量的效果
- 分析：优点、缺点、最佳搭配

### TeamComp（阵容）
- 基础信息：ID、名称、版本、等级
- 英雄列表：角色、装备、优先级
- 羁绊列表：名称、数量
- 策略：前期、中期、后期、经济
- 站位：详细的站位说明

### KnowledgeDoc（知识文档）
- 基础信息：ID、标题、分类、标签
- 内容：Markdown格式
- 元数据：版本、作者、创建/更新时间

---

## 🚀 快速开始

### 1. 查看示例数据

```bash
cd knowledge/data/

# 查看英雄数据
cat champions/ahri.json

# 查看装备数据
cat items/rabadons_deathcap.json

# 查看阵容策略
cat team_comps/ap_carry.yaml

# 查看知识文档
cat knowledge/economy_guide.md
```

### 2. 添加新数据

按照现有格式创建新的JSON/YAML/Markdown文件即可。

---

## 📋 开发路线图

详细的开发计划请参考 [docs/ROADMAP.md](./docs/ROADMAP.md)。

### 近期计划
- Phase 2: 数据加载器 + 内存索引
- Phase 3: Qdrant集成 + 向量检索
- Phase 4-5: 混合检索 + RAG引擎
- Phase 6: API接口 + MVP发布

### 未来集成选项
- 作为Tool集成到现有TFT Agent
- 写成OpenClaw Skills
- 作为独立API服务

---

## 🛠️ 技术栈

| 层级 | 技术选型 |
|------|---------|
| 语言 | Go 1.21+ |
| Web框架 | Gin |
| 向量DB | Qdrant |
| 缓存 | Redis |
| 全文索引 | bleve |
| Embedding | OpenAI text-embedding-3-small |
| LLM | Claude 3 / GPT-4 |

---

## 📖 相关文档

- [ROADMAP.md](./docs/ROADMAP.md) - 详细开发路线图
- [现有TFT代码](../) - 现有的TFT Copilot实现

---

## 🤝 贡献指南

1. 在 `knowledge/data/` 下添加新数据
2. 遵循现有数据格式
3. 提交Pull Request

---

## 📄 许可证

与主项目保持一致。
