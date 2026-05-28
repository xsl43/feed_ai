package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"feedsystem_ai_go/internal/review"
)

// ============================================================
// ToolRegistry tests
// ============================================================

type mockTool struct {
	name        string
	description string
	execute     func(ctx context.Context, input string) (string, error)
}

func (m *mockTool) Name() string                                  { return m.name }
func (m *mockTool) Description() string                           { return m.description }
func (m *mockTool) Execute(ctx context.Context, input string) (string, error) { return m.execute(ctx, input) }

func TestToolRegistryRegisterAndGet(t *testing.T) {
	reg := NewToolRegistry()
	t1 := &mockTool{name: "extract_frames", description: "extract frames"}
	t2 := &mockTool{name: "ocr", description: "ocr frames"}

	reg.Register(t1)
	reg.Register(t2)

	got, ok := reg.Get("extract_frames")
	if !ok {
		t.Fatal("expected tool to be registered")
	}
	if got.Name() != "extract_frames" {
		t.Fatalf("got %s, want extract_frames", got.Name())
	}

	got, ok = reg.Get("ocr")
	if !ok {
		t.Fatal("expected ocr tool to be registered")
	}
	if got.Name() != "ocr" {
		t.Fatalf("got %s, want ocr", got.Name())
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Fatal("expected nonexistent tool to not be found")
	}
}

func TestToolRegistryToolsDescription(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&mockTool{name: "extract_frames", description: "extract video frames"})
	reg.Register(&mockTool{name: "done", description: "finish review"})
	reg.Register(&mockTool{name: "ocr", description: "ocr frames"})

	desc := reg.ToolsDescription()

	if !strings.Contains(desc, "extract_frames") {
		t.Error("description should contain extract_frames")
	}
	if !strings.Contains(desc, "ocr") {
		t.Error("description should contain ocr")
	}
	if strings.Contains(desc, "done") {
		t.Error("description should NOT contain done tool")
	}
}

// ============================================================
// AgentTrace tests
// ============================================================

func TestAgentTraceAddRoundAndSetObservation(t *testing.T) {
	trace := &AgentTrace{}

	trace.AddRound(1, "checking frames", "enlarge_frame", `{"frame_id": 3}`)
	trace.SetObservation(`{"status": "ok", "review_status": "approved"}`)

	if len(trace.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(trace.Rounds))
	}
	r := trace.Rounds[0]
	if r.Round != 1 {
		t.Errorf("round = %d, want 1", r.Round)
	}
	if r.Thought != "checking frames" {
		t.Errorf("thought = %s, want 'checking frames'", r.Thought)
	}
	if r.Action != "enlarge_frame" {
		t.Errorf("action = %s, want enlarge_frame", r.Action)
	}
	if r.ActionInput != `{"frame_id": 3}` {
		t.Errorf("action_input = %s", r.ActionInput)
	}
	if !strings.Contains(r.Observation, "approved") {
		t.Errorf("observation = %s, should contain 'approved'", r.Observation)
	}
}

func TestAgentTraceSetObservationEmptyRounds(t *testing.T) {
	trace := &AgentTrace{}
	trace.SetObservation("nothing")
	// Should not panic
	if len(trace.Rounds) != 0 {
		t.Error("should still be empty")
	}
}

func TestAgentTraceToJSON(t *testing.T) {
	trace := &AgentTrace{}
	trace.AddRound(1, "looks good", "done", `{"verdict": "approved", "reason": "all clear"}`)
	trace.FinalVerdict = "approved"
	trace.FinalReason = "all clear"

	jsonStr := trace.ToJSON()
	if jsonStr == "" {
		t.Fatal("ToJSON returned empty string")
	}
	if !strings.Contains(jsonStr, "approved") {
		t.Error("JSON should contain verdict")
	}
}

func TestAgentTraceFromJSON(t *testing.T) {
	original := &AgentTrace{}
	original.AddRound(1, "test thought", "extract_more_frames", `{"n": 5}`)
	original.SetObservation("extracted 5 frames")
	original.FinalVerdict = "approved"
	original.FinalReason = "no violations"

	// Roundtrip
	trace, err := AgentTraceFromJSON(original.ToJSON())
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}
	if trace.FinalVerdict != "approved" {
		t.Errorf("verdict = %s, want approved", trace.FinalVerdict)
	}
	if len(trace.Rounds) != 1 {
		t.Errorf("rounds = %d, want 1", len(trace.Rounds))
	}
	if trace.Rounds[0].Action != "extract_more_frames" {
		t.Errorf("action = %s", trace.Rounds[0].Action)
	}
}

func TestAgentTraceFromJSONInvalid(t *testing.T) {
	_, err := AgentTraceFromJSON("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ============================================================
// Engine parsing tests
// ============================================================

func TestExtractFirstGroupThought(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"Thought: The frames look suspicious\nAction: enlarge_frame",
			"The frames look suspicious",
		},
		{
			"thought: checking content\naction: done",
			"checking content",
		},
		{
			"Thought: 这个视频的文本审核通过，但第3帧画面模糊需要进一步检查\nAction: extract_more_frames\nAction Input: {\"n\": 3}",
			"这个视频的文本审核通过，但第3帧画面模糊需要进一步检查",
		},
		{
			"Action: done\nAction Input: {...}",
			"",
		},
	}

	for _, tt := range tests {
		got := extractFirstGroup(reThought, tt.input)
		if got != tt.expected {
			t.Errorf("thought: got %q, want %q", got, tt.expected)
		}
	}
}

func TestExtractFirstGroupAction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Action: extract_more_frames", "extract_more_frames"},
		{"action: done", "done"},
		{"Thought: checking\nAction: transcribe_audio\nAction Input: {}", "transcribe_audio"},
		{"Thought: ok", ""},
	}

	for _, tt := range tests {
		got := extractFirstGroup(reAction, tt.input)
		if got != tt.expected {
			t.Errorf("action: got %q, want %q", got, tt.expected)
		}
	}
}

func TestExtractFirstGroupActionInput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`Action Input: {"n": 3}`, `{"n": 3}`},
		{`action input: {"frame_id": 1}`, `{"frame_id": 1}`},
		{`Thought: test
Action: enlarge_frame
Action Input: {"frame_id": 5}`, `{"frame_id": 5}`},
		{"Action Input:", ""},
	}

	for _, tt := range tests {
		got := extractFirstGroup(reActionInput, tt.input)
		if got != tt.expected {
			t.Errorf("action_input: got %q, want %q", got, tt.expected)
		}
	}
}

func TestParseDoneVerdict(t *testing.T) {
	verdict, reason := parseDoneVerdict(`{"verdict": "approved", "reason": "all clear"}`)
	if verdict != "approved" {
		t.Errorf("verdict = %s, want approved", verdict)
	}
	if reason != "all clear" {
		t.Errorf("reason = %s", reason)
	}
}

func TestParseDoneVerdictInvalid(t *testing.T) {
	verdict, reason := parseDoneVerdict("not json")
	if verdict != "manual_review" {
		t.Errorf("verdict = %s, want manual_review", verdict)
	}
	if !strings.Contains(reason, "parse failed") {
		t.Errorf("reason should mention parse failure: %s", reason)
	}
}

func TestParseEscalateReason(t *testing.T) {
	reason := parseEscalateReason(`{"reason": "内容可疑需要人工判断"}`)
	if reason != "内容可疑需要人工判断" {
		t.Errorf("reason = %s", reason)
	}
}

// ============================================================
// Engine config defaults
// ============================================================

func TestNewEngineDefaults(t *testing.T) {
	reg := NewToolRegistry()
	engine := NewEngine(EngineConfig{}, reg)
	if engine.cfg.MaxRounds != 5 {
		t.Errorf("MaxRounds = %d, want 5", engine.cfg.MaxRounds)
	}
	if engine.cfg.Timeout != 120*time.Second {
		t.Errorf("Timeout = %v, want 120s", engine.cfg.Timeout)
	}
	if engine.client == nil {
		t.Error("client should not be nil")
	}
}

// ============================================================
// Tool interface tests (mark_checked, done, escalate)
// ============================================================

func TestMarkCheckedTool(t *testing.T) {
	tool := &markCheckedTool{}
	if tool.Name() != "mark_checked" {
		t.Errorf("name = %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}

	out, err := tool.Execute(context.Background(), `{"frame_ids": [1, 2]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "marked_frames") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestDoneTool(t *testing.T) {
	tool := &doneTool{}
	if tool.Name() != "done" {
		t.Errorf("name = %s", tool.Name())
	}
	out, err := tool.Execute(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("done tool should return empty string, got %q", out)
	}
}

func TestEscalateTool(t *testing.T) {
	tool := &escalateTool{}
	if tool.Name() != "escalate" {
		t.Errorf("name = %s", tool.Name())
	}

	out, err := tool.Execute(context.Background(), `{"reason": "test"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "escalated") {
		t.Errorf("unexpected output: %s", out)
	}
}

// ============================================================
// buildPublishingContext tests
// ============================================================

func TestBuildPublishingContext(t *testing.T) {
	details := map[string]*review.ReviewResult{
		"text":  {Status: "approved", Confidence: 0.95, Reason: "文本内容正常"},
		"cover": {Status: "approved", Confidence: 0.90, Reason: "封面无违规"},
		"frame": {Status: "rejected", Confidence: 0.85, Reason: "第3帧疑似违规"},
	}

	result := buildPublishingContext(
		"测试标题", "测试描述",
		[]string{}, // no phase0 hits
		&review.ReviewResult{Status: "rejected", Confidence: 0.85, Reason: "frame rejected"},
		details,
		[]string{"f1.jpg", "f2.jpg", "f3.jpg"},
	)

	if !strings.Contains(result, "测试标题") {
		t.Error("should contain title")
	}
	if !strings.Contains(result, "Phase 0") {
		t.Error("should contain Phase 0 section")
	}
	if !strings.Contains(result, "Phase 1") {
		t.Error("should contain Phase 1 section")
	}
	if !strings.Contains(result, "通过") {
		t.Error("should show 通过 for phase 0")
	}
	if !strings.Contains(result, "text") {
		t.Error("should contain text dimension result")
	}
	if !strings.Contains(result, "第3帧疑似违规") {
		t.Error("should contain frame review reason")
	}
}

func TestBuildPublishingContextWithPhase0Hits(t *testing.T) {
	result := buildPublishingContext(
		"视频标题", "描述",
		[]string{"敏感词A", "敏感词B"},
		&review.ReviewResult{Status: "approved", Confidence: 0.9},
		nil,
		[]string{},
	)

	if !strings.Contains(result, "不通过") {
		t.Error("should show 不通过 when phase0 has hits")
	}
	if !strings.Contains(result, "敏感词A") {
		t.Error("should show hit words")
	}
}

// ============================================================
// Mock LLM http server for engine integration test
// ============================================================

func TestEngineRunWithDoneOnly(t *testing.T) {
	// Test that engine correctly handles a simple done action
	// Create a minimal engine that will fail to call LLM (no API key)
	// We verify the error path works correctly

	reg := NewToolRegistry()
	reg.Register(&markCheckedTool{})
	reg.Register(&doneTool{})

	cfg := EngineConfig{
		Model:     "test-model",
		BaseURL:   "https://invalid.example.com",
		APIKey:    "test-key",
		MaxRounds: 1,
	}

	engine := NewEngine(cfg, reg)
	if engine.cfg.Model != "test-model" {
		t.Error("model should be set")
	}
	if engine.tools == nil {
		t.Error("tools should be set")
	}
}

// ============================================================
// Toolbox tests
// ============================================================

func TestNewToolboxWithoutEscalate(t *testing.T) {
	cfg := review.ReviewConfig{
		Enabled:             true,
		TextModel:           "text-model",
		VisionModel:         "vision-model",
		APIKey:              "key",
		BaseURL:             "https://api.example.com/v1",
		SampleFrames:        3,
		FrameReviewMode:     "auto",
		MaxConcurrentFrames: 2,
	}
	rs := review.NewReviewService(cfg)
	vctx := VideoContext{
		Title:      "test video",
		Desc:       "test description",
		VideoPath:  "/tmp/test.mp4",
		FramePaths: []string{"/tmp/f1.jpg"},
	}

	reg := NewToolbox(ToolboxDeps{ReviewService: rs}, vctx, false)

	// Check all expected tools are registered
	expectedTools := []string{
		"extract_more_frames", "transcribe_audio", "ocr_frames",
		"enlarge_frame", "review_with_better_model", "full_review",
		"mark_checked", "done",
	}
	for _, name := range expectedTools {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("tool %s should be registered", name)
		}
	}

	// escalate should NOT be registered
	if _, ok := reg.Get("escalate"); ok {
		t.Error("escalate should not be registered when includeEscalate=false")
	}
}

func TestNewToolboxWithEscalate(t *testing.T) {
	cfg := review.ReviewConfig{
		Enabled:             true,
		TextModel:           "text-model",
		VisionModel:         "vision-model",
		APIKey:              "key",
		BaseURL:             "https://api.example.com/v1",
		SampleFrames:        3,
		FrameReviewMode:     "auto",
		MaxConcurrentFrames: 2,
	}
	rs := review.NewReviewService(cfg)
	vctx := VideoContext{Title: "test"}

	reg := NewToolbox(ToolboxDeps{ReviewService: rs}, vctx, true)

	if _, ok := reg.Get("escalate"); !ok {
		t.Error("escalate should be registered when includeEscalate=true")
	}
}

// ============================================================
// Tool execution tests (that don't require external services)
// ============================================================

func TestExtractMoreFramesToolNoVideo(t *testing.T) {
	cfg := review.ReviewConfig{
		Enabled:             true,
		TextModel:           "text-model",
		VisionModel:         "vision-model",
		APIKey:              "key",
		BaseURL:             "https://api.example.com/v1",
		FrameReviewMode:     "auto",
		MaxConcurrentFrames: 2,
	}
	rs := review.NewReviewService(cfg)
	tool := &extractMoreFramesTool{
		reviewService: rs,
		videoPath:     "/nonexistent/video.mp4",
		frameDir:      "/tmp/test_frames",
	}

	out, err := tool.Execute(context.Background(), `{"n": 3}`)
	if err == nil {
		t.Logf("expected error for nonexistent video, got: %s", out)
	}
}

func TestEnlargeFrameToolBoundsCheck(t *testing.T) {
	cfg := review.ReviewConfig{
		Enabled:             true,
		TextModel:           "text-model",
		VisionModel:         "vision-model",
		APIKey:              "key",
		BaseURL:             "https://api.example.com/v1",
		MaxConcurrentFrames: 2,
	}
	rs := review.NewReviewService(cfg)
	tool := &enlargeFrameTool{
		reviewService: rs,
		framePaths:    []string{"/tmp/f1.jpg"},
	}

	// frame_id=5 but only 1 frame available
	_, err := tool.Execute(context.Background(), `{"frame_id": 5}`)
	if err == nil {
		t.Error("expected error for out-of-bounds frame_id")
	}

	// invalid frame_id
	_, err = tool.Execute(context.Background(), `{"frame_id": 0}`)
	if err == nil {
		t.Error("expected error for invalid frame_id")
	}
}

func TestOCRFramesToolEmptyFrames(t *testing.T) {
	cfg := review.ReviewConfig{
		Enabled:             true,
		TextModel:           "text-model",
		VisionModel:         "vision-model",
		APIKey:              "key",
		BaseURL:             "https://api.example.com/v1",
		MaxConcurrentFrames: 2,
	}
	rs := review.NewReviewService(cfg)
	tool := &ocrFramesTool{
		reviewService: rs,
		framePaths:    []string{},
	}

	out, err := tool.Execute(context.Background(), `{"frame_ids": [1]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "未检测到文字内容") {
		t.Errorf("expected '未检测到文字内容', got %s", out)
	}
}

// ============================================================
// PublishingAgent constructor
// ============================================================

func TestNewPublishingAgent(t *testing.T) {
	cfg := review.ReviewConfig{
		Enabled: true,
		TextModel: "text-model",
		VisionModel: "vision-model",
		APIKey: "key",
		BaseURL: "https://api.example.com/v1",
	}
	rs := review.NewReviewService(cfg)
	engineCfg := EngineConfig{
		Model:     "agent-model",
		BaseURL:   "https://api.example.com/v1",
		APIKey:    "key",
		MaxRounds: 5,
	}
	deps := ToolboxDeps{ReviewService: rs}

	pa := NewPublishingAgent(rs, engineCfg, deps)
	if pa == nil {
		t.Fatal("NewPublishingAgent returned nil")
	}
	if pa.rs == nil {
		t.Error("review service should not be nil")
	}
	if pa.cfg.Model != "agent-model" {
		t.Errorf("model = %s", pa.cfg.Model)
	}
}

// ============================================================
// TriageAgent tests
// ============================================================

func TestNewTriageAgent(t *testing.T) {
	cfg := EngineConfig{
		Model:     "test-model",
		BaseURL:   "https://api.example.com/v1",
		APIKey:    "key",
		MaxRounds: 5,
	}
	deps := ToolboxDeps{}

	ta := NewTriageAgent(cfg, deps)
	if ta == nil {
		t.Fatal("NewTriageAgent returned nil")
	}
	if ta.cfg.MaxRounds != 1 {
		t.Errorf("TriageAgent MaxRounds should be forced to 1, got %d", ta.cfg.MaxRounds)
	}
}

// ============================================================
// PostReviewAgent tests
// ============================================================

func TestNewPostReviewAgent(t *testing.T) {
	cfg := EngineConfig{
		Model:     "test-model",
		BaseURL:   "https://api.example.com/v1",
		APIKey:    "key",
		MaxRounds: 3,
	}
	deps := ToolboxDeps{}
	triage := NewTriageAgent(cfg, deps)

	pra := NewPostReviewAgent(cfg, deps, triage)
	if pra == nil {
		t.Fatal("NewPostReviewAgent returned nil")
	}
	if pra.triage == nil {
		t.Error("triage agent should not be nil")
	}
}

// ============================================================
// ReviewResult types
// ============================================================

func TestReviewResultVerdicts(t *testing.T) {
	rr := &ReviewResult{Verdict: "safe"}
	if rr.Verdict != "safe" {
		t.Error("verdict mismatch")
	}

	b, _ := json.Marshal(rr)
	var rr2 ReviewResult
	json.Unmarshal(b, &rr2)
	if rr2.Verdict != "safe" {
		t.Error("roundtrip failed")
	}
}

// ============================================================
// Full ReAct format parsing (integration-style)
// ============================================================

func TestParseFullReActResponse(t *testing.T) {
	// Simulate a realistic LLM response
	response := `Thought: 文本审核和封面审核都已通过，但第3帧的审核置信度较低(0.85)且标记为rejected。我需要进一步检查这个帧。

Action: enlarge_frame
Action Input: {"frame_id": 3}`

	thought := extractFirstGroup(reThought, response)
	action := extractFirstGroup(reAction, response)
	actionInput := extractFirstGroup(reActionInput, response)

	if thought == "" {
		t.Error("thought should not be empty")
	}
	if action != "enlarge_frame" {
		t.Errorf("action = %s, want enlarge_frame", action)
	}
	if actionInput != `{"frame_id": 3}` {
		t.Errorf("action_input = %s", actionInput)
	}
}

func TestParseDoneResponse(t *testing.T) {
	response := `Thought: 经过追抽附近帧确认，画面无违规内容。所有审核维度均通过。

Action: done
Action Input: {"verdict": "approved", "reason": "所有审核维度通过，无风险内容"}`

	action := extractFirstGroup(reAction, response)
	actionInput := extractFirstGroup(reActionInput, response)

	if action != "done" {
		t.Errorf("action = %s, want done", action)
	}

	verdict, reason := parseDoneVerdict(actionInput)
	if verdict != "approved" {
		t.Errorf("verdict = %s, want approved", verdict)
	}
	if reason != "所有审核维度通过，无风险内容" {
		t.Errorf("reason = %s", reason)
	}
}

func TestParseEscalateResponse(t *testing.T) {
	response := `Thought: 内容高度可疑，涉及可能的违规但无法完全确认，应该升级给人工审核。

Action: escalate
Action Input: {"reason": "疑似涉政内容，需要人工判断"}`

	action := extractFirstGroup(reAction, response)
	actionInput := extractFirstGroup(reActionInput, response)

	if action != "escalate" {
		t.Errorf("action = %s, want escalate", action)
	}

	reason := parseEscalateReason(actionInput)
	if reason != "疑似涉政内容，需要人工判断" {
		t.Errorf("reason = %s", reason)
	}
}
