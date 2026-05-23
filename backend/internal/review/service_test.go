package review

import (
	"testing"
)

func TestClassify(t *testing.T) {
	svc := NewReviewService(ReviewConfig{
		Enabled:               true,
		ConfidenceThreshold:   0.70,
		ManualReviewThreshold: 0.50,
		MaxRetries:            3,
	})

	tests := []struct {
		name     string
		result   *ReviewResult
		expected string
	}{
		// Nil result
		{"nil result", nil, "manual_review"},

		// High confidence zone (>= 0.70): trust AI
		{"high confidence approved", &ReviewResult{Status: "approved", Confidence: 0.95}, "approved"},
		{"high confidence rejected", &ReviewResult{Status: "rejected", Confidence: 0.85}, "rejected"},
		{"boundary high approved", &ReviewResult{Status: "approved", Confidence: 0.70}, "approved"},
		{"boundary high rejected", &ReviewResult{Status: "rejected", Confidence: 0.70}, "rejected"},

		// Gray zone (0.50 <= conf < 0.70): always manual_review
		{"gray zone approved", &ReviewResult{Status: "approved", Confidence: 0.62}, "manual_review"},
		{"gray zone rejected", &ReviewResult{Status: "rejected", Confidence: 0.55}, "manual_review"},
		{"gray boundary low", &ReviewResult{Status: "approved", Confidence: 0.50}, "manual_review"},

		// Low confidence (< 0.50): reject if AI says rejected, else manual
		{"low confidence rejected", &ReviewResult{Status: "rejected", Confidence: 0.30}, "rejected"},
		{"low confidence approved", &ReviewResult{Status: "approved", Confidence: 0.20}, "manual_review"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.Classify(tt.result)
			if got != tt.expected {
				t.Errorf("Classify() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain json", `{"status":"approved","confidence":0.9}`, `{"status":"approved","confidence":0.9}`},
		{"json with markdown", "```json\n{\"status\":\"approved\"}\n```", `{"status":"approved"}`},
		{"json with surrounding text", "审核结果：{\"status\":\"rejected\"}，请处理", `{"status":"rejected"}`},
		{"only braces", `{"status":"approved"}`, `{"status":"approved"}`},
		{"no braces", "plain text", "plain text"},
		{"nested braces", `{"a":{"b":"c"}}`, `{"a":{"b":"c"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReviewConfig_FrameReviewEnabled(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"off", false},
		{"on", true},
		{"auto", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			cfg := ReviewConfig{FrameReviewMode: tt.mode}
			if got := cfg.FrameReviewEnabled(); got != tt.want {
				t.Errorf("FrameReviewEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReviewService_IsEnabled(t *testing.T) {
	t.Run("disabled when false", func(t *testing.T) {
		svc := NewReviewService(ReviewConfig{Enabled: false, APIKey: "sk-xxx"})
		if svc.IsEnabled() {
			t.Error("should be disabled when Enabled=false")
		}
	})

	t.Run("disabled when no API key", func(t *testing.T) {
		svc := NewReviewService(ReviewConfig{Enabled: true, APIKey: ""})
		if svc.IsEnabled() {
			t.Error("should be disabled when APIKey is empty")
		}
	})

	t.Run("enabled with key", func(t *testing.T) {
		svc := NewReviewService(ReviewConfig{Enabled: true, APIKey: "sk-xxx"})
		if !svc.IsEnabled() {
			t.Error("should be enabled when Enabled=true and APIKey is set")
		}
	})
}
