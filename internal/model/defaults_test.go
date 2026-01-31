package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultImplModel(t *testing.T) {
	assert.Equal(t, "opus", DefaultImplModel(Claude), "claude impl default should be opus")
	assert.Equal(t, "default", DefaultImplModel(Codex), "codex impl default should be default")
}

func TestDefaultValModel(t *testing.T) {
	assert.Equal(t, "opus", DefaultValModel(Claude), "claude val default should be opus")
	assert.Equal(t, "default", DefaultValModel(Codex), "codex val default should be default")
}

func TestOppositeAI(t *testing.T) {
	assert.Equal(t, Codex, OppositeAI(Claude), "opposite of claude is codex")
	assert.Equal(t, Claude, OppositeAI(Codex), "opposite of codex is claude")
}

func TestDefaultModelForAI(t *testing.T) {
	assert.Equal(t, "opus", DefaultModelForAI(Claude))
	assert.Equal(t, "default", DefaultModelForAI(Codex))
}

func TestAutoOppositeModelSelection(t *testing.T) {
	// When switching AI backends, the default model for the opposite
	// AI should be used automatically.
	opposite := OppositeAI(Claude)
	assert.Equal(t, "default", DefaultModelForAI(opposite),
		"switching from claude should yield codex default model")

	opposite = OppositeAI(Codex)
	assert.Equal(t, "opus", DefaultModelForAI(opposite),
		"switching from codex should yield claude default model")
}
