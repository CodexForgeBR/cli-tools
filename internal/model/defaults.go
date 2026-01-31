// Package model provides AI-model helpers for the ralph-loop CLI.
//
// It centralises default model names, opposite-AI resolution, and
// validation that a requested model is compatible with the chosen
// AI backend (claude or codex).
package model

// AI backend identifiers used throughout the CLI.
const (
	Claude = "claude"
	Codex  = "codex"
)

// DefaultImplModel returns the default implementation-phase model
// for the given AI backend.
func DefaultImplModel(ai string) string {
	if ai == Claude {
		return "opus"
	}
	return "default"
}

// DefaultValModel returns the default validation-phase model
// for the given AI backend.
func DefaultValModel(ai string) string {
	if ai == Claude {
		return "opus"
	}
	return "default"
}

// OppositeAI returns the counterpart AI backend:
// claude -> codex, codex -> claude.
func OppositeAI(ai string) string {
	if ai == Claude {
		return Codex
	}
	return Claude
}

// DefaultModelForAI returns the general-purpose default model
// for the given AI backend.
func DefaultModelForAI(ai string) string {
	if ai == Claude {
		return "opus"
	}
	return "default"
}
