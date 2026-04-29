package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sagerlabs/awesome/tft/knowledge"
)

// ToolDefinition describes a knowledge capability exposed through the MCP adapter.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Adapter maps MCP-style tool calls to the knowledge transport interface.
type Adapter struct {
	tool  knowledge.TFTKnowledgeTool
	tools []ToolDefinition
}

// NewAdapter creates an MCP-facing adapter for a knowledge tool.
func NewAdapter(tool knowledge.TFTKnowledgeTool) *Adapter {
	return &Adapter{
		tool:  tool,
		tools: defaultToolDefinitions(),
	}
}

// ListTools returns the stable tool catalog exposed by knowledge.
func (a *Adapter) ListTools() []ToolDefinition {
	out := make([]ToolDefinition, len(a.tools))
	copy(out, a.tools)
	return out
}

// CallTool validates the tool name and forwards raw JSON arguments to knowledge.
func (a *Adapter) CallTool(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if a == nil || a.tool == nil {
		return nil, fmt.Errorf("knowledge MCP adapter is not initialized")
	}
	if len(args) == 0 {
		args = []byte("{}")
	}

	var (
		resp knowledge.Response
		err  error
	)

	switch name {
	case "query_nlu":
		resp, err = a.tool.QueryNLU(knowledge.QueryRequest(args))
	case "get_comp_by_id":
		resp, err = a.tool.GetCompByID(knowledge.Request(args))
	case "get_meta_comp_by_id":
		resp, err = a.tool.GetMetaCompByID(knowledge.Request(args))
	case "get_meta_comp_by_name":
		resp, err = a.tool.GetMetaCompByName(knowledge.Request(args))
	case "search_meta_comps":
		resp, err = a.tool.SearchMetaComps(knowledge.Request(args))
	case "get_all_meta_comps":
		resp, err = a.tool.GetAllMetaComps(knowledge.Request(args))
	case "get_meta_champion_by_name":
		resp, err = a.tool.GetMetaChampionByName(knowledge.Request(args))
	case "get_all_meta_champions":
		resp, err = a.tool.GetAllMetaChampions(knowledge.Request(args))
	case "get_meta_item_by_name":
		resp, err = a.tool.GetMetaItemByName(knowledge.Request(args))
	case "get_all_meta_items":
		resp, err = a.tool.GetAllMetaItems(knowledge.Request(args))
	default:
		return nil, fmt.Errorf("unknown knowledge tool: %s", name)
	}
	if err != nil {
		return nil, err
	}

	return json.RawMessage(resp), nil
}

func defaultToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "query_nlu",
			Description: "Query TFT knowledge using an NLU-extracted game context.",
			InputSchema: objectSchema(map[string]any{
				"intent":          stringSchema(),
				"champions":       mapSchema("integer"),
				"items":           arraySchema("string"),
				"traits":          arraySchema("string"),
				"augments":        arraySchema("string"),
				"explicit_lineup": nullableStringSchema(),
				"inferred_lineup": stringSchema(),
				"playstyle":       stringSchema(),
				"game_stage":      nullableStringSchema(),
				"gold":            nullableIntegerSchema(),
				"level":           nullableIntegerSchema(),
				"hp":              nullableIntegerSchema(),
				"unit_cost":       nullableIntegerSchema(),
				"role_query":      stringSchema(),
			}, nil),
		},
		{
			Name:        "get_comp_by_id",
			Description: "Get a TFT comp summary by cluster ID.",
			InputSchema: objectSchema(map[string]any{
				"cluster_id": stringSchema(),
			}, []string{"cluster_id"}),
		},
		{
			Name:        "get_meta_comp_by_id",
			Description: "Get a MetaTFT comp by cluster ID.",
			InputSchema: objectSchema(map[string]any{
				"cluster_id": stringSchema(),
			}, []string{"cluster_id"}),
		},
		{
			Name:        "get_meta_comp_by_name",
			Description: "Get a MetaTFT comp by display name.",
			InputSchema: objectSchema(map[string]any{
				"name": stringSchema(),
			}, []string{"name"}),
		},
		{
			Name:        "search_meta_comps",
			Description: "Search MetaTFT comps by keyword.",
			InputSchema: objectSchema(map[string]any{
				"query":  stringSchema(),
				"limit":  integerSchema(),
				"offset": integerSchema(),
			}, []string{"query"}),
		},
		{
			Name:        "get_all_meta_comps",
			Description: "List all MetaTFT comps.",
			InputSchema: objectSchema(map[string]any{}, nil),
		},
		{
			Name:        "get_meta_champion_by_name",
			Description: "Get MetaTFT champion data by name.",
			InputSchema: objectSchema(map[string]any{
				"name": stringSchema(),
			}, []string{"name"}),
		},
		{
			Name:        "get_all_meta_champions",
			Description: "List all MetaTFT champions.",
			InputSchema: objectSchema(map[string]any{}, nil),
		},
		{
			Name:        "get_meta_item_by_name",
			Description: "Get MetaTFT item data by name.",
			InputSchema: objectSchema(map[string]any{
				"name": stringSchema(),
			}, []string{"name"}),
		},
		{
			Name:        "get_all_meta_items",
			Description: "List all MetaTFT items.",
			InputSchema: objectSchema(map[string]any{}, nil),
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func integerSchema() map[string]any {
	return map[string]any{"type": "integer"}
}

func nullableStringSchema() map[string]any {
	return map[string]any{"type": []string{"string", "null"}}
}

func nullableIntegerSchema() map[string]any {
	return map[string]any{"type": []string{"integer", "null"}}
}

func arraySchema(itemType string) map[string]any {
	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": itemType,
		},
	}
}

func mapSchema(valueType string) map[string]any {
	return map[string]any{
		"type": "object",
		"additionalProperties": map[string]any{
			"type": valueType,
		},
	}
}
