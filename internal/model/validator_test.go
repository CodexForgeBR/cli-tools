package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- ValidateModelAI ----------

func TestValidateModelAI_EmptyModelAlwaysOK(t *testing.T) {
	assert.NoError(t, ValidateModelAI(Claude, "", "impl-model"))
	assert.NoError(t, ValidateModelAI(Codex, "", "impl-model"))
}

func TestValidateModelAI_ClaudeWithClaudeModels(t *testing.T) {
	for _, m := range []string{"opus", "sonnet", "haiku", "claude-3-opus"} {
		assert.NoError(t, ValidateModelAI(Claude, m, "impl-model"),
			"claude + %q should be ok", m)
	}
}

func TestValidateModelAI_CodexWithCodexModels(t *testing.T) {
	for _, m := range []string{"default", "o1", "o3-mini", "gpt-4", "gpt4o", "chatgpt-4o", "text-davinci-003", "ft:gpt-3.5-turbo"} {
		assert.NoError(t, ValidateModelAI(Codex, m, "impl-model"),
			"codex + %q should be ok", m)
	}
}

func TestValidateModelAI_ClaudeWithDefault_Error(t *testing.T) {
	err := ValidateModelAI(Claude, "default", "val-model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default")
	assert.Contains(t, err.Error(), "claude")
}

func TestValidateModelAI_CodexWithClaudeModel_Error(t *testing.T) {
	for _, m := range []string{"opus", "sonnet", "haiku", "claude-3-opus"} {
		err := ValidateModelAI(Codex, m, "impl-model")
		require.Error(t, err, "codex + %q should error", m)
		assert.Contains(t, err.Error(), "claude")
	}
}

func TestValidateModelAI_ClaudeWithCodexModel_Error(t *testing.T) {
	for _, m := range []string{"gpt-4", "o1", "chatgpt-4o", "text-davinci-003"} {
		err := ValidateModelAI(Claude, m, "impl-model")
		require.Error(t, err, "claude + %q should error", m)
		assert.Contains(t, err.Error(), "codex")
	}
}

func TestValidateModelAI_UnknownModelAccepted(t *testing.T) {
	// Models that don't match any known pattern are accepted for both.
	assert.NoError(t, ValidateModelAI(Claude, "my-custom-model", "impl-model"))
	assert.NoError(t, ValidateModelAI(Codex, "my-custom-model", "impl-model"))
}

// ---------- IsClaudeModelHint ----------

func TestIsClaudeModelHint(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"opus", true},
		{"sonnet", true},
		{"haiku", true},
		{"claude-3-opus", true},
		{"claude-3.5-sonnet", true},
		{"OPUS", true},   // case insensitive
		{"Sonnet", true}, // case insensitive
		{"default", false},
		{"gpt-4", false},
		{"o1", false},
		{"my-model", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, IsClaudeModelHint(tt.model),
			"IsClaudeModelHint(%q)", tt.model)
	}
}

// ---------- IsCodexModelHint ----------

func TestIsCodexModelHint(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"default", true},
		{"o1", true},
		{"o3-mini", true},
		{"gpt-4", true},
		{"gpt4o", true},
		{"chatgpt-4o", true},
		{"text-davinci-003", true},
		{"ft:gpt-3.5-turbo", true},
		{"GPT-4", true},   // case insensitive
		{"DEFAULT", true}, // case insensitive
		{"opus", false},
		{"sonnet", false},
		{"haiku", false},
		{"claude-3-opus", false},
		{"my-model", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, IsCodexModelHint(tt.model),
			"IsCodexModelHint(%q)", tt.model)
	}
}
