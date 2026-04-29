package agent

import (
	"context"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInputParser struct {
	input *data.UserInput
}

func (p fakeInputParser) Parse(raw string) (*data.UserInput, error) {
	if p.input != nil {
		out := *p.input
		out.Raw = raw
		return &out, nil
	}
	return &data.UserInput{Raw: raw}, nil
}

type fakeHeroCompsTool struct {
	called bool
}

func (t *fakeHeroCompsTool) Query(ctx context.Context, in *data.HeroCompsInput) (*data.HeroCompsOutput, error) {
	t.called = true
	return &data.HeroCompsOutput{}, nil
}

type fakeItemFitTool struct {
	called bool
}

func (t *fakeItemFitTool) Query(ctx context.Context, in *data.ItemFitInput) (*data.ItemFitOutput, error) {
	t.called = true
	return &data.ItemFitOutput{}, nil
}

type fakeCompTierTool struct {
	called bool
}

func (t *fakeCompTierTool) QueryTopTier(ctx context.Context) (*data.CompTierOutput, error) {
	t.called = true
	return &data.CompTierOutput{}, nil
}

type fakeIntersectionTool struct {
	input *data.IntersectionInput
}

func (t *fakeIntersectionTool) Compute(in *data.IntersectionInput) (*data.IntersectionOutput, error) {
	t.input = in
	return &data.IntersectionOutput{
		Recommendations: []data.Recommendation{
			{Confidence: 0.9, ConfidenceDesc: "高"},
		},
		UserInput: in.UserInput,
	}, nil
}

func TestGraphRuntimeValidateRequiresDependencies(t *testing.T) {
	err := (&GraphRuntime{}).Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "input parser")
}

func TestRecommendationExecutorUsesRuntimeTools(t *testing.T) {
	heroTool := &fakeHeroCompsTool{}
	itemTool := &fakeItemFitTool{}
	tierTool := &fakeCompTierTool{}
	intersection := &fakeIntersectionTool{}

	runtime := &GraphRuntime{
		Parser: fakeInputParser{input: &data.UserInput{
			Heroes: []string{"TFT17_Belveth"},
			Items:  []string{"TFT_Item_GuinsoosRageblade"},
		}},
		Tools: &ToolRegistry{
			HeroComps:    heroTool,
			ItemFit:      itemTool,
			CompTier:     tierTool,
			Intersection: intersection,
		},
		PromptBuilder: defaultPromptBuilder{},
		Planner:       StaticToolPlanner{},
	}
	runtime.Executor = NewRecommendationExecutor(runtime)

	out, err := runtime.Executor.Execute(context.Background(), "卑尔维斯 羊刀")

	require.NoError(t, err)
	require.Len(t, out.Recommendations, 1)
	assert.True(t, heroTool.called)
	assert.True(t, itemTool.called)
	assert.True(t, tierTool.called)
	require.NotNil(t, intersection.input)
	assert.Equal(t, "卑尔维斯 羊刀", intersection.input.UserInput.Raw)
}

func TestToolRegistryNamesAreStable(t *testing.T) {
	names := (&ToolRegistry{}).Names()

	assert.Equal(t, []string{"hero_comps", "item_fit", "comp_tier", "intersection"}, names)
}
