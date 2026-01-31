package state

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSessionStateJSONRoundTrip tests that SessionState can be marshaled to JSON
// and unmarshaled back without losing data
func TestSessionStateJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		state SessionState
	}{
		{
			name: "complete state with all fields",
			state: SessionState{
				SchemaVersion:    2,
				SessionID:        "ralph-20260130-143000",
				StartedAt:        "2026-01-30T14:30:00Z",
				LastUpdated:      "2026-01-30T14:35:00Z",
				Iteration:        3,
				Status:           "IN_PROGRESS",
				Phase:            "validation",
				Verdict:          "NEEDS_MORE_WORK",
				TasksFile:        "/tmp/test/tasks.md",
				TasksFileHash:    "abc123def456",
				AICli:            "claude",
				ImplModel:        "opus",
				ValModel:         "opus",
				MaxIterations:    20,
				MaxInadmissible:  5,
				OriginalPlanFile: stringPtr("/tmp/test/plan.md"),
				GithubIssue:      stringPtr("https://github.com/owner/repo/issues/123"),
				Learnings: LearningsState{
					Enabled: 1,
					File:    "/tmp/test/.ralph-loop/learnings.md",
				},
				CrossValidation: CrossValState{
					Enabled:   1,
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				FinalPlanValidation: PlanValState{
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				TasksValidation: TasksValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				Schedule: ScheduleState{
					Enabled:     true,
					TargetEpoch: 1706623800,
					TargetHuman: "2026-01-30T16:30:00Z",
				},
				RetryState: RetryState{
					Attempt: 1,
					Delay:   5,
				},
				InadmissibleCount: 2,
				LastFeedback:      "Task implementation incomplete",
			},
		},
		{
			name: "state with null optional fields",
			state: SessionState{
				SchemaVersion:    2,
				SessionID:        "ralph-20260130-150000",
				StartedAt:        "2026-01-30T15:00:00Z",
				LastUpdated:      "2026-01-30T15:05:00Z",
				Iteration:        1,
				Status:           "PENDING",
				Phase:            "implementation",
				Verdict:          "",
				TasksFile:        "/tmp/test/tasks.md",
				TasksFileHash:    "xyz789",
				AICli:            "claude",
				ImplModel:        "opus",
				ValModel:         "opus",
				MaxIterations:    20,
				MaxInadmissible:  5,
				OriginalPlanFile: nil,
				GithubIssue:      nil,
				Learnings: LearningsState{
					Enabled: 0,
					File:    "",
				},
				CrossValidation: CrossValState{
					Enabled:   0,
					AI:        "",
					Model:     "",
					Available: false,
				},
				FinalPlanValidation: PlanValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				TasksValidation: TasksValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				Schedule: ScheduleState{
					Enabled:     false,
					TargetEpoch: 0,
					TargetHuman: "",
				},
				RetryState: RetryState{
					Attempt: 1,
					Delay:   5,
				},
				InadmissibleCount: 0,
				LastFeedback:      "",
			},
		},
		{
			name: "empty state with minimal fields",
			state: SessionState{
				SchemaVersion:       2,
				SessionID:           "",
				StartedAt:           "",
				LastUpdated:         "",
				Iteration:           0,
				Status:              "",
				Phase:               "",
				Verdict:             "",
				TasksFile:           "",
				TasksFileHash:       "",
				AICli:               "",
				ImplModel:           "",
				ValModel:            "",
				MaxIterations:       0,
				MaxInadmissible:     0,
				Learnings:           LearningsState{},
				CrossValidation:     CrossValState{},
				FinalPlanValidation: PlanValState{},
				TasksValidation:     TasksValState{},
				Schedule:            ScheduleState{},
				RetryState:          RetryState{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tt.state)
			require.NoError(t, err, "Marshal should not fail")

			// Unmarshal back to struct
			var restored SessionState
			err = json.Unmarshal(jsonData, &restored)
			require.NoError(t, err, "Unmarshal should not fail")

			// Compare the two structs
			assert.Equal(t, tt.state, restored, "Round-trip should preserve all fields")
		})
	}
}

// TestSchemaV2FieldNames validates that the JSON field names match the exact contract
// specified in the sample-state.json schema
func TestSchemaV2FieldNames(t *testing.T) {
	state := SessionState{
		SchemaVersion:    2,
		SessionID:        "ralph-20260130-143000",
		StartedAt:        "2026-01-30T14:30:00Z",
		LastUpdated:      "2026-01-30T14:35:00Z",
		Iteration:        3,
		Status:           "IN_PROGRESS",
		Phase:            "validation",
		Verdict:          "NEEDS_MORE_WORK",
		TasksFile:        "/tmp/test/tasks.md",
		TasksFileHash:    "abc123def456",
		AICli:            "claude",
		ImplModel:        "opus",
		ValModel:         "opus",
		MaxIterations:    20,
		MaxInadmissible:  5,
		OriginalPlanFile: nil,
		GithubIssue:      nil,
		Learnings: LearningsState{
			Enabled: 1,
			File:    "/tmp/test/.ralph-loop/learnings.md",
		},
		CrossValidation: CrossValState{
			Enabled:   1,
			AI:        "codex",
			Model:     "default",
			Available: true,
		},
		FinalPlanValidation: PlanValState{
			AI:        "codex",
			Model:     "default",
			Available: true,
		},
		TasksValidation: TasksValState{
			AI:        "claude",
			Model:     "opus",
			Available: true,
		},
		Schedule: ScheduleState{
			Enabled:     false,
			TargetEpoch: 0,
			TargetHuman: "",
		},
		RetryState: RetryState{
			Attempt: 1,
			Delay:   5,
		},
		InadmissibleCount: 0,
		LastFeedback:      "",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(state)
	require.NoError(t, err)

	// Unmarshal to generic map to check field names
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	// Verify all expected top-level field names exist
	expectedFields := []string{
		"schema_version",
		"session_id",
		"started_at",
		"last_updated",
		"iteration",
		"status",
		"phase",
		"verdict",
		"tasks_file",
		"tasks_file_hash",
		"ai_cli",
		"implementation_model",
		"validation_model",
		"max_iterations",
		"max_inadmissible",
		"original_plan_file",
		"github_issue",
		"learnings",
		"cross_validation",
		"final_plan_validation",
		"tasks_validation",
		"schedule",
		"retry_state",
		"inadmissible_count",
		"last_feedback",
	}

	for _, field := range expectedFields {
		assert.Contains(t, jsonMap, field, "JSON should contain field: %s", field)
	}

	// Verify nested object field names
	learnings, ok := jsonMap["learnings"].(map[string]interface{})
	require.True(t, ok, "learnings should be an object")
	assert.Contains(t, learnings, "enabled")
	assert.Contains(t, learnings, "file")

	crossVal, ok := jsonMap["cross_validation"].(map[string]interface{})
	require.True(t, ok, "cross_validation should be an object")
	assert.Contains(t, crossVal, "enabled")
	assert.Contains(t, crossVal, "ai")
	assert.Contains(t, crossVal, "model")
	assert.Contains(t, crossVal, "available")

	planVal, ok := jsonMap["final_plan_validation"].(map[string]interface{})
	require.True(t, ok, "final_plan_validation should be an object")
	assert.Contains(t, planVal, "ai")
	assert.Contains(t, planVal, "model")
	assert.Contains(t, planVal, "available")

	tasksVal, ok := jsonMap["tasks_validation"].(map[string]interface{})
	require.True(t, ok, "tasks_validation should be an object")
	assert.Contains(t, tasksVal, "ai")
	assert.Contains(t, tasksVal, "model")
	assert.Contains(t, tasksVal, "available")

	schedule, ok := jsonMap["schedule"].(map[string]interface{})
	require.True(t, ok, "schedule should be an object")
	assert.Contains(t, schedule, "enabled")
	assert.Contains(t, schedule, "target_epoch")
	assert.Contains(t, schedule, "target_human")

	retryState, ok := jsonMap["retry_state"].(map[string]interface{})
	require.True(t, ok, "retry_state should be an object")
	assert.Contains(t, retryState, "attempt")
	assert.Contains(t, retryState, "delay")
}

// TestBase64EncodingDecoding tests that the LastFeedback field can handle
// base64-encoded content (if that's how feedback is stored)
func TestBase64EncodingDecoding(t *testing.T) {
	tests := []struct {
		name     string
		feedback string
	}{
		{
			name:     "plain text feedback",
			feedback: "Task implementation incomplete",
		},
		{
			name:     "base64 encoded feedback",
			feedback: base64.StdEncoding.EncodeToString([]byte("This is encoded feedback with special chars: 你好, émoji 🎉")),
		},
		{
			name:     "empty feedback",
			feedback: "",
		},
		{
			name:     "multi-line feedback",
			feedback: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "feedback with json characters",
			feedback: `{"error": "syntax error", "line": 42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := SessionState{
				SchemaVersion: 2,
				SessionID:     "test-session",
				LastFeedback:  tt.feedback,
			}

			// Marshal to JSON
			jsonData, err := json.Marshal(state)
			require.NoError(t, err)

			// Unmarshal back
			var restored SessionState
			err = json.Unmarshal(jsonData, &restored)
			require.NoError(t, err)

			// Feedback should be preserved exactly
			assert.Equal(t, tt.feedback, restored.LastFeedback)
		})
	}
}

// TestNestedObjectsMarshaling tests that all nested structs marshal correctly
func TestNestedObjectsMarshaling(t *testing.T) {
	t.Run("LearningsState", func(t *testing.T) {
		learning := LearningsState{
			Enabled: 1,
			File:    "/path/to/learnings.md",
		}

		jsonData, err := json.Marshal(learning)
		require.NoError(t, err)

		var restored LearningsState
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, learning, restored)

		// Verify JSON field names
		var jsonMap map[string]interface{}
		err = json.Unmarshal(jsonData, &jsonMap)
		require.NoError(t, err)
		assert.Contains(t, jsonMap, "enabled")
		assert.Contains(t, jsonMap, "file")
		assert.Equal(t, float64(1), jsonMap["enabled"]) // JSON numbers are float64
		assert.Equal(t, "/path/to/learnings.md", jsonMap["file"])
	})

	t.Run("CrossValState", func(t *testing.T) {
		crossVal := CrossValState{
			Enabled:   1,
			AI:        "codex",
			Model:     "default",
			Available: true,
		}

		jsonData, err := json.Marshal(crossVal)
		require.NoError(t, err)

		var restored CrossValState
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, crossVal, restored)
	})

	t.Run("PlanValState", func(t *testing.T) {
		planVal := PlanValState{
			AI:        "claude",
			Model:     "opus",
			Available: false,
		}

		jsonData, err := json.Marshal(planVal)
		require.NoError(t, err)

		var restored PlanValState
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, planVal, restored)
	})

	t.Run("TasksValState", func(t *testing.T) {
		tasksVal := TasksValState{
			AI:        "claude",
			Model:     "sonnet",
			Available: true,
		}

		jsonData, err := json.Marshal(tasksVal)
		require.NoError(t, err)

		var restored TasksValState
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, tasksVal, restored)
	})

	t.Run("ScheduleState", func(t *testing.T) {
		schedule := ScheduleState{
			Enabled:     true,
			TargetEpoch: 1706623800,
			TargetHuman: "2026-01-30T16:30:00Z",
		}

		jsonData, err := json.Marshal(schedule)
		require.NoError(t, err)

		var restored ScheduleState
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, schedule, restored)

		// Verify JSON field names
		var jsonMap map[string]interface{}
		err = json.Unmarshal(jsonData, &jsonMap)
		require.NoError(t, err)
		assert.Contains(t, jsonMap, "enabled")
		assert.Contains(t, jsonMap, "target_epoch")
		assert.Contains(t, jsonMap, "target_human")
	})

	t.Run("RetryState", func(t *testing.T) {
		retry := RetryState{
			Attempt: 3,
			Delay:   10,
		}

		jsonData, err := json.Marshal(retry)
		require.NoError(t, err)

		var restored RetryState
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, retry, restored)

		// Verify JSON field names
		var jsonMap map[string]interface{}
		err = json.Unmarshal(jsonData, &jsonMap)
		require.NoError(t, err)
		assert.Contains(t, jsonMap, "attempt")
		assert.Contains(t, jsonMap, "delay")
		assert.Equal(t, float64(3), jsonMap["attempt"])
		assert.Equal(t, float64(10), jsonMap["delay"])
	})
}

// TestNullValuesForOptionalFields verifies that optional pointer fields
// serialize as null in JSON when not set
func TestNullValuesForOptionalFields(t *testing.T) {
	state := SessionState{
		SchemaVersion:    2,
		SessionID:        "test-session",
		OriginalPlanFile: nil,
		GithubIssue:      nil,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(state)
	require.NoError(t, err)

	// Unmarshal to map to check null values
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	// These fields should be null in JSON
	assert.Nil(t, jsonMap["original_plan_file"], "original_plan_file should be null")
	assert.Nil(t, jsonMap["github_issue"], "github_issue should be null")

	// Verify round-trip preserves nil
	var restored SessionState
	err = json.Unmarshal(jsonData, &restored)
	require.NoError(t, err)

	assert.Nil(t, restored.OriginalPlanFile)
	assert.Nil(t, restored.GithubIssue)
}

// TestNonNullOptionalFields verifies that optional pointer fields serialize
// correctly when they have values
func TestNonNullOptionalFields(t *testing.T) {
	planFile := "/tmp/plan.md"
	issueURL := "https://github.com/owner/repo/issues/42"

	state := SessionState{
		SchemaVersion:    2,
		SessionID:        "test-session",
		OriginalPlanFile: &planFile,
		GithubIssue:      &issueURL,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(state)
	require.NoError(t, err)

	// Unmarshal to map to check values
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/plan.md", jsonMap["original_plan_file"])
	assert.Equal(t, "https://github.com/owner/repo/issues/42", jsonMap["github_issue"])

	// Verify round-trip preserves values
	var restored SessionState
	err = json.Unmarshal(jsonData, &restored)
	require.NoError(t, err)

	require.NotNil(t, restored.OriginalPlanFile)
	assert.Equal(t, planFile, *restored.OriginalPlanFile)

	require.NotNil(t, restored.GithubIssue)
	assert.Equal(t, issueURL, *restored.GithubIssue)
}

// TestUnmarshalFromSampleJSON tests unmarshaling from the actual sample JSON
// schema to ensure compatibility
func TestUnmarshalFromSampleJSON(t *testing.T) {
	sampleJSON := `{
    "schema_version": 2,
    "session_id": "ralph-20260130-143000",
    "started_at": "2026-01-30T14:30:00Z",
    "last_updated": "2026-01-30T14:35:00Z",
    "iteration": 3,
    "status": "IN_PROGRESS",
    "phase": "validation",
    "verdict": "NEEDS_MORE_WORK",
    "tasks_file": "/tmp/test/tasks.md",
    "tasks_file_hash": "abc123def456",
    "ai_cli": "claude",
    "implementation_model": "opus",
    "validation_model": "opus",
    "max_iterations": 20,
    "max_inadmissible": 5,
    "original_plan_file": null,
    "github_issue": null,
    "learnings": {
        "enabled": 1,
        "file": "/tmp/test/.ralph-loop/learnings.md"
    },
    "cross_validation": {
        "enabled": 1,
        "ai": "codex",
        "model": "default",
        "available": true
    },
    "final_plan_validation": {
        "ai": "codex",
        "model": "default",
        "available": true
    },
    "tasks_validation": {
        "ai": "claude",
        "model": "opus",
        "available": true
    },
    "schedule": {
        "enabled": false,
        "target_epoch": 0,
        "target_human": ""
    },
    "retry_state": {
        "attempt": 1,
        "delay": 5
    },
    "inadmissible_count": 0,
    "last_feedback": ""
}`

	var state SessionState
	err := json.Unmarshal([]byte(sampleJSON), &state)
	require.NoError(t, err, "Should unmarshal sample JSON")

	// Verify key fields
	assert.Equal(t, 2, state.SchemaVersion)
	assert.Equal(t, "ralph-20260130-143000", state.SessionID)
	assert.Equal(t, "2026-01-30T14:30:00Z", state.StartedAt)
	assert.Equal(t, "2026-01-30T14:35:00Z", state.LastUpdated)
	assert.Equal(t, 3, state.Iteration)
	assert.Equal(t, "IN_PROGRESS", state.Status)
	assert.Equal(t, "validation", state.Phase)
	assert.Equal(t, "NEEDS_MORE_WORK", state.Verdict)
	assert.Equal(t, "/tmp/test/tasks.md", state.TasksFile)
	assert.Equal(t, "abc123def456", state.TasksFileHash)
	assert.Equal(t, "claude", state.AICli)
	assert.Equal(t, "opus", state.ImplModel)
	assert.Equal(t, "opus", state.ValModel)
	assert.Equal(t, 20, state.MaxIterations)
	assert.Equal(t, 5, state.MaxInadmissible)
	assert.Nil(t, state.OriginalPlanFile)
	assert.Nil(t, state.GithubIssue)
	assert.Equal(t, 0, state.InadmissibleCount)
	assert.Equal(t, "", state.LastFeedback)

	// Verify nested objects
	assert.Equal(t, 1, state.Learnings.Enabled)
	assert.Equal(t, "/tmp/test/.ralph-loop/learnings.md", state.Learnings.File)

	assert.Equal(t, 1, state.CrossValidation.Enabled)
	assert.Equal(t, "codex", state.CrossValidation.AI)
	assert.Equal(t, "default", state.CrossValidation.Model)
	assert.True(t, state.CrossValidation.Available)

	assert.Equal(t, "codex", state.FinalPlanValidation.AI)
	assert.Equal(t, "default", state.FinalPlanValidation.Model)
	assert.True(t, state.FinalPlanValidation.Available)

	assert.Equal(t, "claude", state.TasksValidation.AI)
	assert.Equal(t, "opus", state.TasksValidation.Model)
	assert.True(t, state.TasksValidation.Available)

	assert.False(t, state.Schedule.Enabled)
	assert.Equal(t, int64(0), state.Schedule.TargetEpoch)
	assert.Equal(t, "", state.Schedule.TargetHuman)

	assert.Equal(t, 1, state.RetryState.Attempt)
	assert.Equal(t, 5, state.RetryState.Delay)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
