package agent

import (
	"context"
	"fmt"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/parser"
	"github.com/sagerlabs/awesome/tft/tool"
)

// InputParser keeps graph input parsing injectable without exposing parser internals.
type InputParser interface {
	Parse(raw string) (*data.UserInput, error)
}

// HeroCompsQuerier describes the hero-to-comp tool used by the baseline graph.
type HeroCompsQuerier interface {
	Query(ctx context.Context, in *data.HeroCompsInput) (*data.HeroCompsOutput, error)
}

// ItemFitQuerier describes the item-to-comp tool used by the baseline graph.
type ItemFitQuerier interface {
	Query(ctx context.Context, in *data.ItemFitInput) (*data.ItemFitOutput, error)
}

// CompTierQuerier describes the meta-tier tool used by the baseline graph.
type CompTierQuerier interface {
	QueryTopTier(ctx context.Context) (*data.CompTierOutput, error)
}

// IntersectionComputer merges tool observations into final recommendations.
type IntersectionComputer interface {
	Compute(in *data.IntersectionInput) (*data.IntersectionOutput, error)
}

// PromptBuilder turns structured recommendations into the LLM prompt.
type PromptBuilder interface {
	BuildRecommendationPrompt(in *data.IntersectionOutput) string
}

type defaultPromptBuilder struct{}

func (defaultPromptBuilder) BuildRecommendationPrompt(in *data.IntersectionOutput) string {
	return BuildPrompt(in)
}

// ToolRegistry is the stable catalog of tools available to the graph runtime.
type ToolRegistry struct {
	HeroComps    HeroCompsQuerier
	ItemFit      ItemFitQuerier
	CompTier     CompTierQuerier
	Intersection IntersectionComputer
}

// Validate makes graph assembly fail fast instead of failing inside a request.
func (r *ToolRegistry) Validate() error {
	if r == nil {
		return fmt.Errorf("tool registry is nil")
	}
	if r.HeroComps == nil {
		return fmt.Errorf("hero comps tool is nil")
	}
	if r.ItemFit == nil {
		return fmt.Errorf("item fit tool is nil")
	}
	if r.CompTier == nil {
		return fmt.Errorf("comp tier tool is nil")
	}
	if r.Intersection == nil {
		return fmt.Errorf("intersection tool is nil")
	}
	return nil
}

// Names returns the bounded tool catalog for docs, debugging and future planners.
func (r *ToolRegistry) Names() []string {
	return []string{"hero_comps", "item_fit", "comp_tier", "intersection"}
}

// GraphRuntime contains dependencies that may vary by environment or test.
// The graph topology stays static; runtime dependencies are injected here.
type GraphRuntime struct {
	Parser        InputParser
	Tools         *ToolRegistry
	PromptBuilder PromptBuilder
	Planner       ToolPlanner
	Executor      *RecommendationExecutor
}

// Validate checks the runtime before graph compilation.
func (r *GraphRuntime) Validate() error {
	if r == nil {
		return fmt.Errorf("graph runtime is nil")
	}
	if r.Parser == nil {
		return fmt.Errorf("input parser is nil")
	}
	if err := r.Tools.Validate(); err != nil {
		return err
	}
	if r.PromptBuilder == nil {
		return fmt.Errorf("prompt builder is nil")
	}
	if r.Planner == nil {
		return fmt.Errorf("tool planner is nil")
	}
	if r.Executor == nil {
		return fmt.Errorf("recommendation executor is nil")
	}
	return nil
}

// NewDefaultGraphRuntime creates the production runtime for the baseline graph.
func NewDefaultGraphRuntime(store *data.Store) *GraphRuntime {
	registry := &ToolRegistry{
		HeroComps:    tool.NewHeroCompsTool(store),
		ItemFit:      tool.NewItemFitTool(store),
		CompTier:     tool.NewCompTierTool(store),
		Intersection: tool.NewIntersectionCalc(store),
	}
	runtime := &GraphRuntime{
		Parser:        parser.NewInputParser(store),
		Tools:         registry,
		PromptBuilder: defaultPromptBuilder{},
		Planner:       StaticToolPlanner{},
	}
	runtime.Executor = NewRecommendationExecutor(runtime)
	return runtime
}

// ToolPlan is intentionally small: it documents which bounded tools are used.
type ToolPlan struct {
	UseHeroComps bool
	UseItemFit   bool
	UseCompTier  bool
}

// ToolPlanner is the future seam for model-assisted tool selection.
type ToolPlanner interface {
	Plan(input *data.UserInput) ToolPlan
}

// StaticToolPlanner keeps the MVP deterministic while making planning explicit.
type StaticToolPlanner struct{}

func (StaticToolPlanner) Plan(input *data.UserInput) ToolPlan {
	return ToolPlan{
		UseHeroComps: true,
		UseItemFit:   true,
		UseCompTier:  true,
	}
}

// RecommendationExecutor runs the bounded tool plan outside the Eino graph.
type RecommendationExecutor struct {
	runtime *GraphRuntime
}

func NewRecommendationExecutor(runtime *GraphRuntime) *RecommendationExecutor {
	return &RecommendationExecutor{runtime: runtime}
}

// Execute computes recommendations using the same runtime dependencies as the graph.
func (e *RecommendationExecutor) Execute(ctx context.Context, rawInput string) (*data.IntersectionOutput, error) {
	if e == nil || e.runtime == nil {
		return nil, fmt.Errorf("recommendation executor is nil")
	}
	if e.runtime.Parser == nil || e.runtime.Tools == nil || e.runtime.Planner == nil {
		return nil, fmt.Errorf("recommendation executor runtime is incomplete")
	}

	userInput, err := e.runtime.Parser.Parse(rawInput)
	if err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	plan := e.runtime.Planner.Plan(userInput)

	var heroComps data.HeroCompsOutput
	if plan.UseHeroComps {
		out, err := e.runtime.Tools.HeroComps.Query(ctx, &data.HeroCompsInput{
			Heroes: userInput.Heroes,
			TopN:   5,
		})
		if err != nil {
			return nil, fmt.Errorf("hero comps query: %w", err)
		}
		if out != nil {
			heroComps = *out
		}
	}

	var itemFits data.ItemFitOutput
	if plan.UseItemFit {
		out, err := e.runtime.Tools.ItemFit.Query(ctx, &data.ItemFitInput{Items: userInput.Items})
		if err != nil {
			return nil, fmt.Errorf("item fit query: %w", err)
		}
		if out != nil {
			itemFits = *out
		}
	}

	var compTiers data.CompTierOutput
	if plan.UseCompTier {
		out, err := e.runtime.Tools.CompTier.QueryTopTier(ctx)
		if err != nil {
			return nil, fmt.Errorf("comp tier query: %w", err)
		}
		if out != nil {
			compTiers = *out
		}
	}

	result, err := e.runtime.Tools.Intersection.Compute(&data.IntersectionInput{
		HeroComps: heroComps,
		ItemFits:  itemFits,
		CompTiers: compTiers,
		UserInput: *userInput,
	})
	if err != nil {
		return nil, fmt.Errorf("intersection compute: %w", err)
	}
	return result, nil
}
