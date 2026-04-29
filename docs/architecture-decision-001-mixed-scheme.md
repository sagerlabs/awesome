# 架构决策记录：采用混合方案解耦knowledge和agent

## 日期
2026-03-25

## Implementation Status（实现状态）

Status: Superseded（已被后续方案替代） / Historical（历史记录）

Checklist:

- [x] `knowledge` 对外保持字节流 transport（传输）接口。
- [x] `agent` 侧通过 adapter（适配器）调用 `knowledge`。
- [x] 后续 ADR002 已将共享语义进一步收敛到 `contracts`（公共协议）。
- [ ] 本 ADR 中“agent 自己持有类型再转换”的做法不再作为最新推荐方案。

Evidence（证据）:

- `tft/knowledge/tool.go`
- `tft/agent/knowledge_adapter.go`
- `tft/knowledge/contracts/query_nlu.go`
- `docs/architecture-decision-002-knowledge-mcp-contract-boundary.md`

## 背景

在设计TFT Knowledge Tool时，我们遇到了包依赖的问题：

### 问题描述
1. **循环依赖风险**：`knowledge` 包直接引用了 `agent` 包
2. **难以独立部署**：无法将 `knowledge` 作为独立的 tool/skill 分割出去
3. **耦合度高**：两个包紧密绑定，难以单独演进

### 初始架构
```
knowledge/
    ↓ 引用
agent/
    ↓ 引用
data/
```

## 可选方案评估

### 方案A：提取公共类型到common包
**设计**：
- 创建 `tft/common` 包
- 将 `Context`、`NluEnrichedContext` 等公共类型移到common
- knowledge和agent都引用common

**优点**：
- ✅ 清晰的依赖关系
- ✅ 类型安全
- ✅ 避免循环依赖

**缺点**：
- ❌ 需要重构较多代码
- ❌ common包可能变成"垃圾场"
- ❌ 仍然有一定程度的耦合

**决策**：❌ 不采用（暂时）

---

### 方案B：在knowledge中定义接口
**设计**：
- knowledge包中定义 `QueryContext`、`QueryResult` 接口
- 不直接依赖agent的具体类型

**优点**：
- ✅ knowledge不依赖agent

**缺点**：
- ❌ 需要做类型转换
- ❌ 接口可能不够完整
- ❌ 使用体验差

**决策**：❌ 不采用

---

### 方案C：反转依赖方向
**设计**：
- agent依赖knowledge
- knowledge不依赖agent
- knowledge定义自己的类型，agent做转换

**优点**：
- ✅ knowledge完全独立

**缺点**：
- ❌ 需要写大量转换代码
- ❌ agent层会变厚
- ❌ 维护成本高

**决策**：❌ 不采用

---

### 方案D：字节流接口（纯松耦合）
**设计**：
- 所有接口都返回 `[]byte`
- 调用方自己Unmarshal

**优点**：
- ✅ 零依赖
- ✅ 完全独立
- ✅ 语言无关

**缺点**：
- ❌ 失去类型安全
- ❌ 使用不便
- ❌ 错误处理难
- ❌ 文档不足

**决策**：❌ 不采用（太极端）

---

### 方案E：混合方案（最终选择 ⭐）

## 最终选择：混合方案

### 设计思路

**核心原则**：
1. **knowledge包零依赖** - 不引用agent、data等包
2. **接口使用字节流** - 保证完全独立
3. **agent包做类型转换** - 提供类型安全的使用体验
4. **保持现有功能** - 不破坏现有代码

### 架构设计

```
┌─────────────────────────────────────────────────┐
│                    agent                       │
│  (Context, NluEnrichedContext)                │
│         ↓ (Marshal/Unmarshal)                 │
│      []byte / []byte                           │
└─────────────────┬─────────────────────────────┘
                  │
         ┌────────▼─────────┐
         │  knowledge       │
         │  (零依赖!)        │
         │  QueryNLU([]byte)│
         │  → []byte         │
         └────────┬─────────┘
                  │
         ┌────────▼─────────┐
         │  data           │
         │  knowledge.Store │
         └──────────────────┘
```

### 具体实现

#### 1. knowledge包接口（零依赖）

```go
// tft/knowledge/tool.go
package knowledge

// QueryRequest 查询请求（字节流）
type QueryRequest []byte

// QueryResponse 查询响应（字节流）
type QueryResponse []byte

// TFTKnowledgeTool 接口定义
type TFTKnowledgeTool interface {
    QueryNLU(req QueryRequest) (QueryResponse, error)
    
    // 其他方法也用字节流
    GetCompByID(clusterID string) ([]byte, error)
    GetChampionByName(name string) ([]byte, error)
    GetItemByName(name string) ([]byte, error)
}
```

#### 2. agent包转换层

```go
// tft/agent/knowledge_adapter.go
package agent

import (
    "encoding/json"
    "github.com/sagerlabs/awesome/tft/knowledge"
)

// KnowledgeAdapter knowledge的适配器
// 提供类型安全的接口，内部做字节流转换
type KnowledgeAdapter struct {
    tool knowledge.TFTKnowledgeTool
}

// NewKnowledgeAdapter 创建适配器
func NewKnowledgeAdapter(tool knowledge.TFTKnowledgeTool) *KnowledgeAdapter {
    return &KnowledgeAdapter{tool: tool}
}

// QueryNLU 类型安全的查询方法
func (a *KnowledgeAdapter) QueryNLU(ctx Context) (*NluEnrichedContext, error) {
    // 1. Marshal agent.Context → []byte
    reqBytes, err := json.Marshal(ctx)
    if err != nil {
        return nil, err
    }
    
    // 2. 调用knowledge（字节流接口）
    respBytes, err := a.tool.QueryNLU(knowledge.QueryRequest(reqBytes))
    if err != nil {
        return nil, err
    }
    
    // 3. Unmarshal []byte → agent.NluEnrichedContext
    var result NluEnrichedContext
    if err := json.Unmarshal([]byte(respBytes), &result); err != nil {
        return nil, err
    }
    
    return &result, nil
}
```

#### 3. knowledge内部实现（仍然使用类型安全）

```go
// tft/knowledge/unified_store.go
// 内部实现仍然使用类型，只在接口边界做JSON转换

func (s *UnifiedStore) QueryNLU(req QueryRequest) (QueryResponse, error) {
    // 1. Unmarshal请求
    var ctx agent.Context // 内部可以引用agent（只在实现层，不在接口层）
    if err := json.Unmarshal([]byte(req), &ctx); err != nil {
        return nil, err
    }
    
    // 2. 内部调用（类型安全）
    result := agent.QueryNLUData(ctx, s.dataStore)
    
    // 3. Marshal响应
    respBytes, err := json.Marshal(result)
    if err != nil {
        return nil, err
    }
    
    return QueryResponse(respBytes), nil
}
```

## 选择理由

### 为什么选择混合方案？

#### 1. ✅ knowledge包零依赖（接口层）
- 接口定义完全不依赖其他包
- 可以轻松分割成独立的服务
- 甚至可以用其他语言重写

#### 2. ✅ 保持类型安全（使用层）
- agent包通过适配器提供类型安全的接口
- 编译时类型检查
- IDE自动补全和重构支持

#### 3. ✅ 渐进式迁移
- 不需要一次性重构所有代码
- 可以逐步迁移
- 风险可控

#### 4. ✅ 灵活性高
- 以后可以轻松替换knowledge实现
- 可以加缓存、监控等中间层
- 便于A/B测试

#### 5. ✅ 文档清晰
- 类型定义在agent包，文档自然
- 接口定义在knowledge包，职责清晰

## 权衡与妥协

### 接受的缺点

1. **有一次JSON序列化/反序列化开销**
   - 影响：很小（相比LLM调用可以忽略）
   - 缓解：以后可以加缓存

2. **需要维护转换代码**
   - 影响：需要写一些Marshal/Unmarshal代码
   - 缓解：可以生成，或者用反射简化

3. **knowledge实现层仍然依赖agent**
   - 影响：实现层不能完全独立
   - 缓解：只在实现层，接口层是干净的

### 为什么可以接受这些缺点？

1. **性能影响可忽略** - 相比LLM调用，JSON开销可以忽略
2. **转换代码简单** - 都是标准的json.Marshal/Unmarshal
3. **渐进式改进** - 以后可以进一步解耦实现层

## 后续演进计划

### Phase 1（当前）
- ✅ 接口层零依赖
- ✅ agent包提供适配器
- ✅ 保持现有功能

### Phase 2（未来）
- 将knowledge实现层对agent的依赖也移除
- 在knowledge包中定义自己的类型
- agent包只做转换

### Phase 3（最终目标）
- knowledge完全独立
- 可以作为独立服务部署
- 通过HTTP/gRPC调用

## 经验总结

1. **接口边界很重要** - 在包边界使用简单的数据结构（[]byte, map等）
2. **不要过度设计** - 混合方案虽然不是"最纯"的，但最实用
3. **渐进式重构** - 不需要一步到位，可以分阶段进行
4. **权衡是必要的** - 没有完美的方案，只有最合适的方案

---

# 架构决策记录：移除对knowledge/models包的引用

## 日期
2026-03-25

## 背景

在 `knowledge/unified_store.go` 中，我们遇到了包引用问题：

### 问题描述
1. **不必要的包引用**：`knowledge/unified_store.go` 引用了 `github.com/sagerlabs/awesome/tft/knowledge/models`
2. **实际使用场景**：只在返回空JSON数组时使用了 `models.MetaComp`、`models.MetaChampion`、`models.MetaItem`
3. **包依赖问题**：造成了不必要的包依赖，可能导致模块解析问题

### 错误信息
```
github.com/sagerlabs/awesome/tft/knowledge/models: module github.com/sagerlabs/awesome@latest found (v1.0.3), but does not contain package github.com/sagerlabs/awesome/tft/knowledge/models
```

## 可选方案评估

### 方案A：保持引用，解决模块问题
**设计**：
- 保持对 `knowledge/models` 包的引用
- 解决模块解析问题（可能需要检查目录结构、go.mod等）

**优点**：
- ✅ 类型安全
- ✅ 代码改动最小

**缺点**：
- ❌ 模块问题可能难以定位
- ❌ 仍然有不必要的包依赖
- ❌ 可能需要调整目录结构

**决策**：❌ 不采用

---

### 方案B：将models包中的类型移到knowledge包
**设计**：
- 将 `MetaComp`、`MetaChampion`、`MetaItem` 等类型从 `knowledge/models` 移到 `knowledge` 包
- 移除对 `knowledge/models` 包的引用

**优点**：
- ✅ 类型安全
- ✅ 减少包依赖层级

**缺点**：
- ❌ 需要移动和修改较多代码
- ❌ 可能影响其他引用 `knowledge/models` 的代码
- ❌ 类型定义分散

**决策**：❌ 不采用

---

### 方案C：使用空interface{}代替（最终选择 ⭐）
**设计**：
- 移除对 `knowledge/models` 包的引用
- 将 `[]*models.MetaComp` 改为 `[]interface{}`
- 将 `[]*models.MetaChampion` 改为 `[]interface{}`
- 将 `[]*models.MetaItem` 改为 `[]interface{}`

**优点**：
- ✅ 完全移除不必要的包引用
- ✅ 代码改动最小
- ✅ `json.Marshal()` 可以接受任何类型
- ✅ 不影响功能（只是返回空数组）

**缺点**：
- ⚠️ 失去编译时类型检查（但只在返回空数组时使用）
- ⚠️ 看起来不够"优雅"

**权衡**：
- 只在返回空JSON数组时使用这些类型
- `json.Marshal()` 不关心具体类型
- 风险很低，收益很大

**决策**：✅ 采用

## 实现细节

### 修改的文件
`tft/knowledge/unified_store.go`

### 具体修改
1. **移除import**：
   ```go
   // 移除
   import "github.com/sagerlabs/awesome/tft/knowledge/models"
   ```

2. **修改空数组返回**：
   ```go
   // 修改前
   return json.Marshal([]*models.MetaComp{})
   
   // 修改后
   return json.Marshal([]interface{}{})
   ```

3. **应用到所有方法**：
   - `SearchMetaComps()`
   - `GetAllMetaComps()`
   - `GetAllMetaChampions()`
   - `GetAllMetaItems()`

## 结果

### 包依赖简化
```
knowledge包（实现层）
    ↓ 引用
agent包
data包
    ✅ 不再引用 knowledge/models包！
```

### 功能不变
- ✅ 所有功能正常工作
- ✅ JSON序列化不受影响
- ✅ 空数组返回正确

### 风险控制
- ⚠️ 只在返回空数组时使用 `interface{}`
- ⚠️ 这些方法在 `knowledgeStore == nil` 时才会被调用
- ⚠️ 正常情况下不会走到这些分支

## 经验总结

1. **审视每一个import** - 问问自己：真的需要这个包吗？
2. **json.Marshal很灵活** - 它可以接受任何类型，不一定非要用具体类型
3. **简单就是美** - 有时候最"不优雅"的方案反而是最实用的
4. **风险收益比** - 评估改动的风险和收益，选择平衡点

## 参考资料

- https://golang.org/pkg/encoding/json/#Marshal
- https://dave.cheney.net/2016/08/20/solid-go-design
