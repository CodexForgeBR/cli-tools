package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupCrossValidation_BothEmpty(t *testing.T) {
	ai, model := SetupCrossValidation("claude", "", "")
	assert.Equal(t, "codex", ai, "Should use opposite AI when CrossAI is empty")
	assert.Equal(t, "default", model, "Should use default model for codex")
}

func TestSetupCrossValidation_AIEmptyModelSpecified(t *testing.T) {
	ai, model := SetupCrossValidation("claude", "", "custom-model")
	assert.Equal(t, "codex", ai, "Should use opposite AI when CrossAI is empty")
	assert.Equal(t, "custom-model", model, "Should preserve specified model")
}

func TestSetupCrossValidation_AISpecifiedModelEmpty(t *testing.T) {
	ai, model := SetupCrossValidation("claude", "codex", "")
	assert.Equal(t, "codex", ai, "Should preserve specified AI")
	assert.Equal(t, "default", model, "Should use default model for codex")
}

func TestSetupCrossValidation_BothSpecified(t *testing.T) {
	ai, model := SetupCrossValidation("claude", "codex", "custom-model")
	assert.Equal(t, "codex", ai, "Should preserve specified AI")
	assert.Equal(t, "custom-model", model, "Should preserve specified model")
}

func TestSetupCrossValidation_CodexToClaudeOpposite(t *testing.T) {
	ai, model := SetupCrossValidation("codex", "", "")
	assert.Equal(t, "claude", ai, "Should use opposite AI (claude) when primary is codex")
	assert.Equal(t, "opus", model, "Should use default model for claude")
}

func TestSetupFinalPlanValidation_BothEmpty(t *testing.T) {
	ai, model := SetupFinalPlanValidation("codex", "default", "", "")
	assert.Equal(t, "codex", ai, "Should use cross-validation AI when FinalPlanAI is empty")
	assert.Equal(t, "default", model, "Should use cross-validation model when FinalPlanModel is empty")
}

func TestSetupFinalPlanValidation_AIEmptyModelSpecified(t *testing.T) {
	ai, model := SetupFinalPlanValidation("codex", "default", "", "custom-model")
	assert.Equal(t, "codex", ai, "Should use cross-validation AI when FinalPlanAI is empty")
	assert.Equal(t, "custom-model", model, "Should preserve specified model")
}

func TestSetupFinalPlanValidation_AISpecifiedModelEmpty(t *testing.T) {
	ai, model := SetupFinalPlanValidation("codex", "default", "claude", "")
	assert.Equal(t, "claude", ai, "Should preserve specified AI")
	assert.Equal(t, "default", model, "Should use cross-validation model when FinalPlanModel is empty")
}

func TestSetupFinalPlanValidation_BothSpecified(t *testing.T) {
	ai, model := SetupFinalPlanValidation("codex", "default", "claude", "opus")
	assert.Equal(t, "claude", ai, "Should preserve specified AI")
	assert.Equal(t, "opus", model, "Should preserve specified model")
}

func TestSetupTasksValidation_BothEmpty(t *testing.T) {
	ai, model := SetupTasksValidation("claude", "opus", "", "")
	assert.Equal(t, "claude", ai, "Should use implementation AI when TasksValAI is empty")
	assert.Equal(t, "opus", model, "Should use implementation model when TasksValModel is empty")
}

func TestSetupTasksValidation_AIEmptyModelSpecified(t *testing.T) {
	ai, model := SetupTasksValidation("claude", "opus", "", "sonnet")
	assert.Equal(t, "claude", ai, "Should use implementation AI when TasksValAI is empty")
	assert.Equal(t, "sonnet", model, "Should preserve specified model")
}

func TestSetupTasksValidation_AISpecifiedModelEmpty(t *testing.T) {
	ai, model := SetupTasksValidation("claude", "opus", "codex", "")
	assert.Equal(t, "codex", ai, "Should preserve specified AI")
	assert.Equal(t, "opus", model, "Should use implementation model when TasksValModel is empty")
}

func TestSetupTasksValidation_BothSpecified(t *testing.T) {
	ai, model := SetupTasksValidation("claude", "opus", "codex", "default")
	assert.Equal(t, "codex", ai, "Should preserve specified AI")
	assert.Equal(t, "default", model, "Should preserve specified model")
}

func TestSetupTasksValidation_WithCodexImpl(t *testing.T) {
	ai, model := SetupTasksValidation("codex", "default", "", "")
	assert.Equal(t, "codex", ai, "Should use implementation AI (codex)")
	assert.Equal(t, "default", model, "Should use implementation model (default)")
}
