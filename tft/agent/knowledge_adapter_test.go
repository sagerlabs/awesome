package agent

import (
	"encoding/json"
	"testing"

	"github.com/sagerlabs/awesome/tft/knowledge"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// KnowledgeAdapter 单元测试
// 覆盖JSON序列化/反序列化路径
// =============================================================================

// mockTool 用于测试的mock实现
type mockTool struct {
	queryNLUFunc        func(req knowledge.QueryRequest) (knowledge.QueryResponse, error)
	getCompByIDFunc     func(req knowledge.Request) (knowledge.Response, error)
	getMetaCompByIDFunc func(req knowledge.Request) (knowledge.Response, error)
	// 其他方法可以根据需要添加
}

func (m *mockTool) QueryNLU(req knowledge.QueryRequest) (knowledge.QueryResponse, error) {
	if m.queryNLUFunc != nil {
		return m.queryNLUFunc(req)
	}
	return nil, nil
}

func (m *mockTool) GetCompByID(req knowledge.Request) (knowledge.Response, error) {
	if m.getCompByIDFunc != nil {
		return m.getCompByIDFunc(req)
	}
	return nil, nil
}

func (m *mockTool) GetMetaCompByID(req knowledge.Request) (knowledge.Response, error) {
	if m.getMetaCompByIDFunc != nil {
		return m.getMetaCompByIDFunc(req)
	}
	return nil, nil
}

func (m *mockTool) GetMetaCompByName(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (m *mockTool) SearchMetaComps(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (m *mockTool) GetAllMetaComps(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (m *mockTool) GetMetaChampionByName(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (m *mockTool) GetAllMetaChampions(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (m *mockTool) GetMetaItemByName(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (m *mockTool) GetAllMetaItems(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (m *mockTool) ResolveUnitID(input string) string {
	return input
}

func (m *mockTool) ResolveItemID(input string) string {
	return input
}

func (m *mockTool) IDToCN(id string) string {
	return id
}

func (m *mockTool) CNToID(cn string) string {
	return cn
}

func (m *mockTool) Reload() error {
	return nil
}

func (m *mockTool) HealthCheck() error {
	return nil
}

// =============================================================================
// 测试用例
// =============================================================================

func TestKnowledgeAdapter_QueryNLU(t *testing.T) {
	// 1. 创建测试数据
	testCtx := Context{
		Intent:    "lineup_recommend",
		Champions: map[string]int8{"兰博": 2},
		Items:     []string{"鬼索的狂暴之刃"},
	}

	// 2. 创建期望的响应
	expectedResult := NluEnrichedContext{
		Ctx: testCtx,
		MatchedComps: []contracts.CompSummary{
			{
				ClusterID: "394014",
				Name:      "约德尔人",
				Tier:      "S",
			},
		},
	}

	// 3. 创建mock tool
	mt := &mockTool{
		queryNLUFunc: func(req knowledge.QueryRequest) (knowledge.QueryResponse, error) {
			// 验证请求是否正确序列化
			var receivedCtx Context
			err := json.Unmarshal([]byte(req), &receivedCtx)
			assert.NoError(t, err)
			assert.Equal(t, testCtx.Intent, receivedCtx.Intent)
			assert.Equal(t, testCtx.Champions, receivedCtx.Champions)
			assert.Equal(t, testCtx.Items, receivedCtx.Items)

			// 返回模拟的响应
			respBytes, err := json.Marshal(expectedResult)
			assert.NoError(t, err)
			return knowledge.QueryResponse(respBytes), nil
		},
	}

	// 4. 创建adapter
	adapter := NewKnowledgeAdapter(mt)

	// 5. 调用adapter
	result, err := adapter.QueryNLU(testCtx)

	// 6. 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedResult.Ctx.Intent, result.Ctx.Intent)
	assert.Equal(t, len(expectedResult.MatchedComps), len(result.MatchedComps))
	if len(result.MatchedComps) > 0 {
		assert.Equal(t, expectedResult.MatchedComps[0].ClusterID, result.MatchedComps[0].ClusterID)
		assert.Equal(t, expectedResult.MatchedComps[0].Name, result.MatchedComps[0].Name)
		assert.Equal(t, expectedResult.MatchedComps[0].Tier, result.MatchedComps[0].Tier)
	}

	t.Log("QueryNLU JSON序列化/反序列化测试通过")
}

func TestKnowledgeAdapter_GetCompByID(t *testing.T) {
	// 1. 创建测试数据
	expectedComp := contracts.CompSummary{
		ClusterID:    "394014",
		Name:         "约德尔人",
		Tier:         "S",
		AvgPlacement: 3.9397,
	}

	// 2. 创建mock tool
	mt := &mockTool{
		getCompByIDFunc: func(req knowledge.Request) (knowledge.Response, error) {
			var request contracts.GetCompByIDRequest
			err := json.Unmarshal(req, &request)
			assert.NoError(t, err)
			assert.Equal(t, "394014", request.ClusterID)

			respBytes, err := json.Marshal(contracts.GetCompByIDResponse{Comp: &expectedComp})
			return knowledge.Response(respBytes), err
		},
	}

	// 3. 创建adapter
	adapter := NewKnowledgeAdapter(mt)

	// 4. 调用adapter
	comp, err := adapter.GetCompByID("394014")

	// 5. 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, comp)
	assert.Equal(t, expectedComp.ClusterID, comp.ClusterID)
	assert.Equal(t, expectedComp.Name, comp.Name)
	assert.Equal(t, expectedComp.Tier, comp.Tier)
	assert.InDelta(t, expectedComp.AvgPlacement, comp.AvgPlacement, 0.0001)

	t.Log("GetCompByID JSON序列化/反序列化测试通过")
}

func TestKnowledgeAdapter_GetMetaCompByID(t *testing.T) {
	// 1. 创建测试数据
	expectedComp := MetaComp{
		ClusterID:   "394014",
		NameString:  "TFT16_Augment_RumbleCarry",
		Tier:        "S",
		Units:       []string{"兰博", "凯南"},
		Description: "测试描述",
		Limit: map[string]interface{}{
			"min_level": 8,
		},
	}

	// 2. 创建mock tool（使用完整的mock）
	mt := &mockTool{
		getMetaCompByIDFunc: func(req knowledge.Request) (knowledge.Response, error) {
			var request contracts.GetMetaCompByIDRequest
			err := json.Unmarshal(req, &request)
			assert.NoError(t, err)
			assert.Equal(t, "394014", request.ClusterID)

			respBytes, err := json.Marshal(contracts.GetMetaCompByIDResponse{Comp: &expectedComp})
			return knowledge.Response(respBytes), err
		},
	}

	// 3. 创建adapter
	adapter := NewKnowledgeAdapter(mt)

	// 4. 调用adapter
	comp, err := adapter.GetMetaCompByID("394014")

	// 5. 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, comp)
	assert.Equal(t, expectedComp.ClusterID, comp.ClusterID)
	assert.Equal(t, expectedComp.NameString, comp.NameString)
	assert.Equal(t, expectedComp.Tier, comp.Tier)
	assert.Equal(t, expectedComp.Units, comp.Units)
	assert.Equal(t, expectedComp.Description, comp.Description)
	assert.NotNil(t, comp.Limit)
	assert.EqualValues(t, 8, comp.Limit["min_level"])

	t.Log("GetMetaCompByID JSON序列化/反序列化测试通过")
}

func TestKnowledgeAdapter_JSONMarshalUnmarshal(t *testing.T) {
	// 测试Context的JSON序列化/反序列化
	testCases := []struct {
		name string
		ctx  Context
	}{
		{
			name: "阵容推荐意图",
			ctx: Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"兰博": 2, "吉格斯": 1},
				Items:     []string{"鬼索的狂暴之刃", "珠光护手"},
				Traits:    []string{"约德尔人"},
			},
		},
		{
			name: "装备查询意图",
			ctx: Context{
				Intent: "item_query",
				Items:  []string{"鬼索的狂暴之刃"},
			},
		},
		{
			name: "包含游戏阶段和等级",
			ctx: Context{
				Intent:    "lineup_recommend",
				Champions: map[string]int8{"兰博": 2},
				GameStage: func(s string) *string { return &s }("4-2"),
				Level:     func(i int) *int { return &i }(8),
				HP:        func(i int) *int { return &i }(50),
				Gold:      func(i int) *int { return &i }(40),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Marshal
			data, err := json.Marshal(tc.ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, data)

			// 2. Unmarshal
			var unmarshaled Context
			err = json.Unmarshal(data, &unmarshaled)
			assert.NoError(t, err)

			// 3. 验证
			assert.Equal(t, tc.ctx.Intent, unmarshaled.Intent)
			assert.Equal(t, tc.ctx.Champions, unmarshaled.Champions)
			assert.Equal(t, tc.ctx.Items, unmarshaled.Items)
			assert.Equal(t, tc.ctx.Traits, unmarshaled.Traits)

			// 验证指针字段
			if tc.ctx.GameStage != nil {
				assert.Equal(t, *tc.ctx.GameStage, *unmarshaled.GameStage)
			}
			if tc.ctx.Level != nil {
				assert.Equal(t, *tc.ctx.Level, *unmarshaled.Level)
			}
			if tc.ctx.HP != nil {
				assert.Equal(t, *tc.ctx.HP, *unmarshaled.HP)
			}
			if tc.ctx.Gold != nil {
				assert.Equal(t, *tc.ctx.Gold, *unmarshaled.Gold)
			}
		})
	}

	t.Log("Context JSON序列化/反序列化测试通过")
}

func TestKnowledgeAdapter_MetaTypes_JSON(t *testing.T) {
	// 测试MetaComp的JSON序列化/反序列化
	t.Run("MetaComp JSON", func(t *testing.T) {
		comp := MetaComp{
			ClusterID:    "394014",
			NameString:   "TFT16_Augment_RumbleCarry",
			Tier:         "S",
			Units:        []string{"兰博", "凯南", "菲兹"},
			Traits:       []string{"约德尔人", "护卫"},
			Count:        4773,
			AvgPlacement: 3.9397,
			Top4Rate:     0.6044,
			WinRate:      0.1758,
			Description:  "这是一个测试阵容",
			Limit: map[string]interface{}{
				"min_level": 8,
				"max_gold":  50,
			},
		}

		// Marshal
		data, err := json.Marshal(comp)
		assert.NoError(t, err)

		// Unmarshal
		var unmarshaled MetaComp
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		// 验证
		assert.Equal(t, comp.ClusterID, unmarshaled.ClusterID)
		assert.Equal(t, comp.NameString, unmarshaled.NameString)
		assert.Equal(t, comp.Tier, unmarshaled.Tier)
		assert.Equal(t, comp.Units, unmarshaled.Units)
		assert.Equal(t, comp.Traits, unmarshaled.Traits)
		assert.Equal(t, comp.Count, unmarshaled.Count)
		assert.InDelta(t, comp.AvgPlacement, unmarshaled.AvgPlacement, 0.0001)
		assert.InDelta(t, comp.Top4Rate, unmarshaled.Top4Rate, 0.0001)
		assert.InDelta(t, comp.WinRate, unmarshaled.WinRate, 0.0001)
		assert.Equal(t, comp.Description, unmarshaled.Description)
		assert.EqualValues(t, comp.Limit["min_level"], unmarshaled.Limit["min_level"])
		assert.EqualValues(t, comp.Limit["max_gold"], unmarshaled.Limit["max_gold"])
	})

	// 测试MetaChampion的JSON序列化/反序列化
	t.Run("MetaChampion JSON", func(t *testing.T) {
		champ := MetaChampion{
			Name: "兰博",
			AppearInComps: []CompAppearance{
				{
					ClusterID:    "394014",
					CompName:     "约德尔人",
					Tier:         "S",
					AvgPlacement: 3.9397,
				},
			},
			Description: "这是一个测试英雄",
			Limit: map[string]interface{}{
				"priority": "high",
			},
		}

		// Marshal
		data, err := json.Marshal(champ)
		assert.NoError(t, err)

		// Unmarshal
		var unmarshaled MetaChampion
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		// 验证
		assert.Equal(t, champ.Name, unmarshaled.Name)
		assert.Equal(t, len(champ.AppearInComps), len(unmarshaled.AppearInComps))
		assert.Equal(t, champ.Description, unmarshaled.Description)
		assert.Equal(t, champ.Limit["priority"], unmarshaled.Limit["priority"])
	})

	// 测试MetaItem的JSON序列化/反序列化
	t.Run("MetaItem JSON", func(t *testing.T) {
		item := MetaItem{
			Name: "鬼索的狂暴之刃",
			PriorityList: []ItemPriority{
				{
					ClusterID:     "394014",
					CompName:      "约德尔人",
					CompTier:      "S",
					CompAvg:       3.9397,
					Carry:         "兰博",
					PriorityScore: 100,
				},
			},
			Description: "这是一个测试装备",
			Limit: map[string]interface{}{
				"max_count": 3,
			},
		}

		// Marshal
		data, err := json.Marshal(item)
		assert.NoError(t, err)

		// Unmarshal
		var unmarshaled MetaItem
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		// 验证
		assert.Equal(t, item.Name, unmarshaled.Name)
		assert.Equal(t, len(item.PriorityList), len(unmarshaled.PriorityList))
		assert.Equal(t, item.Description, unmarshaled.Description)
		assert.EqualValues(t, item.Limit["max_count"], unmarshaled.Limit["max_count"])
	})

	t.Log("Meta类型JSON序列化/反序列化测试通过")
}
