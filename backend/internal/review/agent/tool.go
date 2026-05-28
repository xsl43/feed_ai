package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Tool defines a single tool the Agent can invoke.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input string) (string, error)
}

// ToolRegistry holds all available tools and generates their prompt descriptions.
type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

func (r *ToolRegistry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// ToolsDescription returns a formatted string listing all tools for the system prompt.
func (r *ToolRegistry) ToolsDescription() string {
	var names []string
	for n := range r.tools {
		names = append(names, n)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, n := range names {
		t := r.tools[n]
		if t.Name() == "done" {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", t.Name(), t.Description()))
	}
	return b.String()
}
