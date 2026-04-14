package mcp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdioServer_ToolsList(t *testing.T) {
	adapter := NewAdapter(&fakeKnowledgeTool{})
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var output bytes.Buffer

	server := NewStdioServer(adapter, input, &output)
	err := server.Serve(context.Background())

	require.NoError(t, err)
	assert.Contains(t, output.String(), `"tools"`)
	assert.Contains(t, output.String(), `"query_nlu"`)
}
