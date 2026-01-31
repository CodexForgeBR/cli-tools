package exitcode_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
)

func TestExitCodeValues(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"Success", exitcode.Success, 0},
		{"Error", exitcode.Error, 1},
		{"MaxIterations", exitcode.MaxIterations, 2},
		{"Escalate", exitcode.Escalate, 3},
		{"Blocked", exitcode.Blocked, 4},
		{"TasksInvalid", exitcode.TasksInvalid, 5},
		{"Inadmissible", exitcode.Inadmissible, 6},
		{"Interrupted", exitcode.Interrupted, 130},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code)
		})
	}
}

func TestExitCodeNames(t *testing.T) {
	tests := []struct {
		code         int
		expectedName string
	}{
		{exitcode.Success, "Success"},
		{exitcode.Error, "Error"},
		{exitcode.MaxIterations, "MaxIterations"},
		{exitcode.Escalate, "Escalate"},
		{exitcode.Blocked, "Blocked"},
		{exitcode.TasksInvalid, "TasksInvalid"},
		{exitcode.Inadmissible, "Inadmissible"},
		{exitcode.Interrupted, "Interrupted"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, exitcode.Name(tt.code))
		})
	}
}

func TestExitCodeNameUnknown(t *testing.T) {
	assert.Equal(t, "unknown", exitcode.Name(99))
	assert.Equal(t, "unknown", exitcode.Name(-1))
	assert.Equal(t, "unknown", exitcode.Name(7))
}

func TestAllEightCodesAreDefined(t *testing.T) {
	// Verify all 8 codes are distinct values.
	codes := []int{
		exitcode.Success,
		exitcode.Error,
		exitcode.MaxIterations,
		exitcode.Escalate,
		exitcode.Blocked,
		exitcode.TasksInvalid,
		exitcode.Inadmissible,
		exitcode.Interrupted,
	}
	assert.Len(t, codes, 8, "expected exactly 8 exit codes")

	seen := make(map[int]bool)
	for _, c := range codes {
		assert.False(t, seen[c], "duplicate exit code value: %d", c)
		seen[c] = true
	}
}
