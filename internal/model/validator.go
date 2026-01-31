package model

import (
	"fmt"
	"regexp"
	"strings"
)

// codexModelRe matches OpenAI-family model prefixes: o1, o3, gpt-*, etc.
var codexModelRe = regexp.MustCompile(`^(o[0-9]|gpt|chatgpt|text|ft|gpt4)`)

// claudeModelHints are lower-cased prefixes that strongly indicate a
// Claude-compatible model.
var claudeModelHints = []string{"opus", "sonnet", "haiku", "claude-"}

// ValidateModelAI checks whether model is compatible with the chosen
// AI backend. label is a human-readable name for the flag being
// validated (e.g. "impl-model", "val-model") used in error messages.
//
// Rules:
//   - Empty model is always allowed (the caller will apply defaults).
//   - "default" is only valid for codex.
//   - Claude-style hints (opus, sonnet, haiku, claude-*) are invalid
//     with codex.
//   - Codex-style hints (default, o[0-9]*, gpt*, chatgpt*, text*,
//     ft*, gpt4*) are invalid with claude.
//   - Anything else is accepted without opinion.
func ValidateModelAI(ai, model, label string) error {
	if model == "" {
		return nil
	}

	lower := strings.ToLower(model)

	// "default" is codex-only.
	if lower == "default" {
		if ai == Claude {
			return fmt.Errorf("%s %q is not compatible with ai=%s (\"default\" is a codex model)", label, model, ai)
		}
		return nil
	}

	// Check cross-AI mismatches.
	if ai == Codex && IsClaudeModelHint(model) {
		return fmt.Errorf("%s %q looks like a claude model but ai=%s", label, model, ai)
	}

	if ai == Claude && IsCodexModelHint(model) {
		return fmt.Errorf("%s %q looks like a codex/openai model but ai=%s", label, model, ai)
	}

	return nil
}

// IsClaudeModelHint returns true when model appears to target a Claude
// backend (opus, sonnet, haiku, or claude-* prefix).
func IsClaudeModelHint(model string) bool {
	lower := strings.ToLower(model)
	for _, hint := range claudeModelHints {
		if strings.HasPrefix(lower, hint) {
			return true
		}
	}
	return false
}

// IsCodexModelHint returns true when model appears to target an
// OpenAI / Codex backend (default, o1, o3, gpt-*, chatgpt-*, etc.).
func IsCodexModelHint(model string) bool {
	lower := strings.ToLower(model)
	if lower == "default" {
		return true
	}
	return codexModelRe.MatchString(lower)
}
