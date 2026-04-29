# TFT知识库服务 - Roadmap

基于方案A（极简版）的云顶之弈知识库服务开发计划。

> **注意**：本知识库目前作为独立模块存在于 `tft/knowledge/` 目录下。
> 未来可以考虑：
> 1. 作为Tool集成到现有TFT Agent中
> 2. 写成OpenClaw Skills
> 3. 作为独立API服务提供

---

## 🎯 架构概览

```
数据层 (Git管理) → 数据同步脚本 → Qdrant (向量检索) → Go服务 → API
```

---

## 📅 详细Roadmap

### Phase 0: 项目初始化（Week 1）
- [x] **0.1 项目结构创建**
  - [x] 目录结构设计
  - [x] 数据模型定义
  - [x] 示例数据创建

- [ ] **0.2 基础设施搭建**
  - [ ] Docker Compose编排（Qdrant + Redis）
  - [ ] 开发环境配置
  - [ ] Makefile优化

### Phase 1: 数据模型设计（Week 1）
- [x] **1.1 数据结构定义**
  - [x] Champion（英雄）JSON Schema
  - [x] Item（装备）JSON Schema
  - [x] Trait（羁绊）JSON Schema
  - [x] TeamComp（阵容）YAML Schema
  - [x] KnowledgeDoc（知识文档）结构

- [x] **1.2 数据目录创建**
  ```
  knowledge/data/
  ├── champions/          # 英雄数据 (JSON)
  ├── items/              # 装备数据 (JSON)
  ├── traits/             # 羁绊数据 (JSON)
  ├── team_comps/         # 阵容策略 (YAML)
  └── knowledge/          # 知识文档 (Markdown)
  ```

### Phase 2: 数据加载与内存索引（Week 1-2）
- [ ] **2.1 数据加载器**
  - [ ] JSON文件加载器
  - [ ] YAML文件加载器
  - [ ] Markdown文档加载器
  - [ ] 数据验证与标准化

- [ ] **2.2 内存索引**
  - [ ] 英雄索引（按费用、羁绊、名称）
  - [ ] 装备索引（按名称、合成路径）
  - [ ] 羁绊索引（按名称、所需数量）
  - [ ] 全文索引（bleve）

### Phase 3: Qdrant向量检索集成（Week 2）
- [ ] **3.1 Qdrant客户端封装**
  - [ ] 连接管理
  - [ ] Collection管理
  - [ ] 向量CRUD操作
  - [ ] 批量写入优化

- [ ] **3.2 Embedding服务**
  - [ ] OpenAI Embedding API封装
  - [ ] 本地Embedding模型支持（可选）
  - [ ] 批量Embedding生成
  - [ ] Embedding缓存（Redis）

- [ ] **3.3 数据同步脚本**
  - [ ] 读取数据文件 → 生成Embedding → 写入Qdrant
  - [ ] 增量同步逻辑
  - [ ] 版本管理
  - [ ] 同步状态追踪

### Phase 4: 混合检索引擎（Week 2-3）
- [ ] **4.1 关键词检索**
  - [ ] 内存索引查询
  - [ ] 全文搜索（bleve）
  - [ ] 过滤、排序、分页

- [ ] **4.2 语义检索**
  - [ ] Qdrant向量搜索
  - [ ] 相似度阈值控制
  - [ ] 结果重排序

- [ ] **4.3 混合检索策略**
  - [ ] 多路召回
  - [ ] 结果融合
  - [ ]  reranking（可选）

### Phase 5: RAG推理引擎（Week 3-4）
- [ ] **5.1 LLM集成**
  - [ ] Claude API封装
  - [ ] GPT API封装
  - [ ] Prompt模板管理
  - [ ] 流式输出支持

- [ ] **5.2 RAG链实现**
  - [ ] Query理解（意图识别）
  - [ ] 检索策略动态选择
  - [ ] 上下文构建
  - [ ] 回答生成

- [ ] **5.3 推理优化**
  - [ ] 对话历史管理
  - [ ] Cache层（Redis）
  - [ ] 超时控制
  - [ ] 降级策略

### Phase 6: API接口层（Week 4）
- [ ] **6.1 HTTP REST API**
  - [ ] Gin路由
  - [ ] 请求验证
  - [ ] 分页、排序、过滤
  - [ ] API文档（Swagger）

- [ ] **6.2 核心接口**
  - [ ] `/api/v1/champions` - 英雄查询
  - [ ] `/api/v1/items` - 装备查询
  - [ ] `/api/v1/traits` - 羁绊查询
  - [ ] `/api/v1/team-comps` - 阵容查询
  - [ ] `/api/v1/search` - 混合检索
  - [ ] `/api/v1/chat` - RAG对话

### Phase 7: 未来集成选项（可选）
- [ ] **7.1 作为Tool集成到现有Agent**
  - [ ] 封装为Eino Tool
  - [ ] 集成到现有TFT Agent Graph中
  - [ ] 提供知识库查询能力

- [ ] **7.2 写成OpenClaw Skills**
  - [ ] 创建Skill目录结构
  - [ ] 编写SKILL.md文档
  - [ ] 封装知识库功能为Skills

- [ ] **7.3 独立API服务**
  - [ ] 完整的HTTP/gRPC API
  - [ ] Docker部署
  - [ ] 现有系统通过API调用

---

## 📁 项目目录结构

```
tft/
├── knowledge/                    # 知识库模块（本目录）
│   ├── models/                   # 数据模型
│   ├── data/                     # 数据文件
│   │   ├── champions/            # 英雄数据 (JSON)
│   │   ├── items/                # 装备数据 (JSON)
│   │   ├── traits/               # 羁绊数据 (JSON)
│   │   ├── team_comps/           # 阵容策略 (YAML)
│   │   └── knowledge/            # 知识文档 (Markdown)
│   ├── docs/                     # 文档
│   │   └── ROADMAP.md           # 本文档
│   └── deploy/                   # 部署配置
│       └── docker/               # Docker相关
├── (现有tft代码...)              # 现有TFT Copilot代码保持不变
```

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
| 配置 | Viper |
| 日志 | Zap |
| 监控 | Prometheus |
| 部署 | Docker + Docker Compose |

---

## 🚀 快速开始（MVP）

1. **Phase 0-1**: 项目初始化 + 数据模型 ✅
2. **Phase 2**: 数据加载 + 内存索引
3. **Phase 3**: Qdrant集成 + 向量检索
4. **Phase 4-5**: 检索 + RAG
5. **Phase 6**: API接口 + MVP发布
6. **Phase 7**: 考虑集成方案（Tool/Skills/独立API）

**目标**：4周内跑通MVP版本

---

## 📊 数据格式

### 英雄数据 (JSON)

```json
{
  "id": "ahri",
  "name": "阿狸",
  "cost": 3,
  "traits": ["灵能使", "星之守护者"],
  "stats": {
    "health": 700,
    "mana": 50,
    "attackDamage": 40
  },
  "ability": {
    "name": "灵魄突袭",
    "description": "阿狸释放灵魄，对敌人造成魔法伤害..."
  },
  "analysis": {
    "strengths": ["爆发高", "范围伤害"],
    "weaknesses": ["怕控制", "依赖装备"],
    "best_items": ["大天使之杖", "帽子", "法爆"],
    "synergies": ["灵能使", "星之守护者"]
  }
}
```

更多示例数据请参考 `knowledge/data/` 目录。

---

## 📖 相关文档

- [现有TFT代码](../../) - 现有的TFT Copilot实现
