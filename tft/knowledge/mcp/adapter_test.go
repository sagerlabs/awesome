package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sagerlabs/awesome/tft/knowledge"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeKnowledgeTool struct {
	getMetaCompByID func(req knowledge.Request) (knowledge.Response, error)
	listMetaComps   func(req knowledge.Request) (knowledge.Response, error)
}

func (f *fakeKnowledgeTool) QueryNLU(req knowledge.QueryRequest) (knowledge.QueryResponse, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetCompByID(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetMetaCompByID(req knowledge.Request) (knowledge.Response, error) {
	return f.getMetaCompByID(req)
}

func (f *fakeKnowledgeTool) GetMetaCompByName(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) SearchMetaComps(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetAllMetaComps(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) ListMetaComps(req knowledge.Request) (knowledge.Response, error) {
	if f.listMetaComps != nil {
		return f.listMetaComps(req)
	}
	return nil, nil
}

func (f *fakeKnowledgeTool) GetCompPlan(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetMetaChampionByName(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetAllMetaChampions(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetChampionBuilds(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetMetaItemByName(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetAllMetaItems(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetItemFits(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) GetTraitInsight(req knowledge.Request) (knowledge.Response, error) {
	return nil, nil
}

func (f *fakeKnowledgeTool) ResolveUnitID(input string) string {
	return input
}

func (f *fakeKnowledgeTool) ResolveItemID(input string) string {
	return input
}

func (f *fakeKnowledgeTool) IDToCN(id string) string {
	return id
}

func (f *fakeKnowledgeTool) CNToID(cn string) string {
	return cn
}

func (f *fakeKnowledgeTool) Reload() error {
	return nil
}

func (f *fakeKnowledgeTool) HealthCheck() error {
	return nil
}

func TestAdapter_ListTools(t *testing.T) {
	adapter := NewAdapter(&fakeKnowledgeTool{})

	tools := adapter.ListTools()

	require.NotEmpty(t, tools)
	assert.Equal(t, "query_nlu", tools[0].Name)
	assert.Contains(t, toolNames(tools), "tft_list_meta_comps")
	assert.Contains(t, toolNames(tools), "tft_get_comp_plan")
}

func TestAdapter_QueryNLUSchemaIncludesVerticalFields(t *testing.T) {
	adapter := NewAdapter(&fakeKnowledgeTool{})

	tools := adapter.ListTools()

	require.NotEmpty(t, tools)
	queryTool := tools[0]
	properties, ok := queryTool.InputSchema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "unit_cost")
	assert.Contains(t, properties, "role_query")
}

func TestAdapter_CallTool(t *testing.T) {
	tool := &fakeKnowledgeTool{
		getMetaCompByID: func(req knowledge.Request) (knowledge.Response, error) {
			var request contracts.GetMetaCompByIDRequest
			err := json.Unmarshal(req, &request)
			require.NoError(t, err)
			assert.Equal(t, "394014", request.ClusterID)

			respBytes, err := json.Marshal(contracts.GetMetaCompByIDResponse{
				Comp: &contracts.MetaComp{
					ClusterID: "394014",
					Tier:      "S",
				},
			})
			require.NoError(t, err)
			return knowledge.Response(respBytes), nil
		},
	}
	adapter := NewAdapter(tool)

	result, err := adapter.CallTool(context.Background(), "get_meta_comp_by_id", json.RawMessage(`{"cluster_id":"394014"}`))

	require.NoError(t, err)
	var resp contracts.GetMetaCompByIDResponse
	require.NoError(t, json.Unmarshal(result, &resp))
	require.NotNil(t, resp.Comp)
	assert.Equal(t, "394014", resp.Comp.ClusterID)
	assert.Equal(t, "S", resp.Comp.Tier)
}

func TestAdapter_CallSemanticTool(t *testing.T) {
	tool := &fakeKnowledgeTool{
		listMetaComps: func(req knowledge.Request) (knowledge.Response, error) {
			var request contracts.ListMetaCompsRequest
			err := json.Unmarshal(req, &request)
			require.NoError(t, err)
			assert.Equal(t, 2, request.Limit)

			respBytes, err := json.Marshal(contracts.ListMetaCompsResponse{
				Metadata: &contracts.KnowledgeMetadata{Source: "MetaTFT", SampleCount: 100},
				Comps: []map[string]any{
					{"name": "狂战士千珏", "tier": "S"},
				},
			})
			require.NoError(t, err)
			return knowledge.Response(respBytes), nil
		},
	}
	adapter := NewAdapter(tool)

	result, err := adapter.CallTool(context.Background(), "tft_list_meta_comps", json.RawMessage(`{"limit":2}`))

	require.NoError(t, err)
	var resp contracts.ListMetaCompsResponse
	require.NoError(t, json.Unmarshal(result, &resp))
	require.Len(t, resp.Comps, 1)
	assert.Equal(t, "狂战士千珏", resp.Comps[0]["name"])
}

func toolNames(tools []ToolDefinition) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}
