package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge"
	knowledgemcp "github.com/sagerlabs/awesome/tft/knowledge/mcp"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "tft-knowledge-mcp: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	dataStore, err := data.NewStore(data.GetDataDir())
	if err != nil {
		return fmt.Errorf("load data store: %w", err)
	}

	knowledgeDir := os.Getenv("TFT_KNOWLEDGE_DIR")
	if knowledgeDir == "" {
		knowledgeDir = "tft/knowledge/data"
	}

	var knowledgeStore *knowledge.Store
	if store, err := knowledge.NewLoader(knowledgeDir).LoadAll(); err == nil {
		knowledgeStore = store
	} else {
		fmt.Fprintf(os.Stderr, "tft-knowledge-mcp: load knowledge store failed, continuing without meta data: %v\n", err)
	}

	tool, err := knowledge.NewUnifiedStore(dataStore, knowledgeStore, &knowledge.ToolConfig{
		KnowledgeDir: knowledgeDir,
		EnableMeta:   knowledgeStore != nil,
		EnableLog:    false,
	})
	if err != nil {
		return fmt.Errorf("create knowledge tool: %w", err)
	}

	adapter := knowledgemcp.NewAdapter(tool)
	server := knowledgemcp.NewStdioServer(adapter, os.Stdin, os.Stdout)
	return server.Serve(ctx)
}
