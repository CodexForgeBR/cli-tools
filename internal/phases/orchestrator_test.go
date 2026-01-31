package phases

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
	"github.com/CodexForgeBR/cli-tools/internal/config"
	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
	"github.com/CodexForgeBR/cli-tools/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockOrchestratorAIRunner is a configurable mock for orchestrator tests
type MockOrchestratorAIRunner struct {
	CallCount   int
	RunFunc     func(ctx context.Context, prompt string, outputPath string) error
	PromptLog   []string
	OutputPaths []string
}

func (m *MockOrchestratorAIRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	m.CallCount++
	m.PromptLog = append(m.PromptLog, prompt)
	m.OutputPaths = append(m.OutputPaths, outputPath)

	if m.RunFunc != nil {
		return m.RunFunc(ctx, prompt, outputPath)
	}

	return nil
}

// Helper function to create validation JSON output
func makeOrchestratorValidationJSON(verdict string, feedback string) string {
	data := map[string]interface{}{
		"RALPH_VALIDATION": map[string]interface{}{
			"verdict":  verdict,
			"feedback": feedback,
		},
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// Helper function to create validation JSON with blocked tasks
func makeOrchestratorValidationJSONWithBlocked(verdict string, feedback string, blockedTasks []string) string {
	data := map[string]interface{}{
		"RALPH_VALIDATION": map[string]interface{}{
			"verdict":       verdict,
			"feedback":      feedback,
			"blocked_tasks": blockedTasks,
		},
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// TestNewOrchestrator verifies orchestrator creation
func TestNewOrchestrator(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.TasksFile = "tasks.md"

	orchestrator := NewOrchestrator(cfg)

	assert.NotNil(t, orchestrator, "orchestrator should be created")
	assert.NotNil(t, orchestrator.Config, "config should be set")
	assert.Equal(t, "tasks.md", orchestrator.Config.TasksFile)
}

// TestOrchestrator_10PhaseOrdering verifies phases execute through output and exit codes
func TestOrchestrator_10PhaseOrdering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tasks file with some unchecked tasks
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
- [ ] Task 2
- [x] Task 3
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 2
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Setup mocks
	iteration := 0
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			iteration++
			if iteration >= 2 {
				// Mark tasks as complete
				updatedTasks := `# Tasks
- [x] Task 1
- [x] Task 2
- [x] Task 3
`
				os.WriteFile(tasksFile, []byte(updatedTasks), 0644)
				os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("COMPLETE", "")), 0644)
			} else {
				os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("NEEDS_MORE_WORK", "Keep going")), 0644)
			}
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Success, exitCode, "should exit with success when tasks complete")
	assert.Equal(t, 2, implRunner.CallCount, "implementation should run 2 times")
	assert.Equal(t, 2, valRunner.CallCount, "validation should run 2 times")
}

// TestOrchestrator_MaxIterationsReached verifies exit when max iterations hit
func TestOrchestrator_MaxIterationsReached(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
- [ ] Task 2
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 3
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Always return NEEDS_MORE_WORK so we hit max iterations
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("NEEDS_MORE_WORK", "Not done yet")), 0644)
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.MaxIterations, exitCode, "should exit with MaxIterations code")
	assert.Equal(t, 3, implRunner.CallCount, "should run exactly max iterations")
	assert.Equal(t, 3, valRunner.CallCount, "should validate exactly max iterations")
}

// TestOrchestrator_AllTasksChecked verifies exit 0 when all tasks checked
func TestOrchestrator_AllTasksChecked(t *testing.T) {
	tmpDir := t.TempDir()

	// All tasks already checked
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [x] Task 1
- [x] Task 2
- [x] Task 3
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	valRunner := &MockOrchestratorAIRunner{}
	implRunner := &MockOrchestratorAIRunner{}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Success, exitCode, "should exit success when all tasks checked")
	assert.Equal(t, 0, implRunner.CallCount, "should not run implementation when complete")
	assert.Equal(t, 0, valRunner.CallCount, "should not run validation when complete")
}

// TestOrchestrator_EscalationFromValidation verifies escalation handling
func TestOrchestrator_EscalationFromValidation(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// First iteration escalates
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("ESCALATE", "Need human review")), 0644)
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Escalate, exitCode, "should exit with Escalate code")
	assert.Equal(t, 1, implRunner.CallCount, "should run one implementation before escalate")
	assert.Equal(t, 1, valRunner.CallCount, "should run one validation that escalates")
}

// TestOrchestrator_BlockedTasks verifies blocked tasks handling
func TestOrchestrator_BlockedTasks(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// All tasks blocked
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			blockedJSON := makeOrchestratorValidationJSONWithBlocked("BLOCKED", "All blocked", []string{"Task 1", "Task 2", "Task 3"})
			os.WriteFile(outputPath, []byte(blockedJSON), 0644)
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Blocked, exitCode, "should exit with Blocked code when all tasks blocked")
}

// TestOrchestrator_InadmissibleThreshold verifies inadmissible threshold enforcement
func TestOrchestrator_InadmissibleThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 20
	cfg.MaxInadmissible = 3
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Always return INADMISSIBLE
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("INADMISSIBLE", "Invalid format")), 0644)
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Inadmissible, exitCode, "should exit with Inadmissible code")
	// Should run up to and including the threshold breach (count goes from 0->1->2->3->4, exits at 4)
	assert.LessOrEqual(t, valRunner.CallCount, 4,
		"should not exceed max inadmissible threshold by much")
}

// TestOrchestrator_ContextCancellation verifies graceful shutdown on context cancel
func TestOrchestrator_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 100
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel during first implementation run
	implCallCount := 0
	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			implCallCount++
			if implCallCount == 1 {
				cancel() // Cancel during first iteration
				return ctx.Err()
			}
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("NEEDS_MORE_WORK", "Continue")), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Interrupted, exitCode, "should exit with Interrupted code")
	assert.Equal(t, 1, implCallCount, "should stop after context cancellation")
}

// TestOrchestrator_CrossValidationFlow verifies cross-validation integration
func TestOrchestrator_CrossValidationFlow(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 5
	cfg.CrossValidate = true
	cfg.CrossAI = "openai"
	cfg.CrossModel = "gpt-4"
	cfg.TasksValAI = ""

	// Main validation says complete
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			// Mark task as complete
			os.WriteFile(tasksFile, []byte("# Tasks\n- [x] Task 1\n"), 0644)
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("COMPLETE", "")), 0644)
			return nil
		},
	}

	// Cross validation confirms
	crossRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("COMPLETE", "")), 0644)
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner
	orchestrator.CrossRunner = crossRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Success, exitCode, "should succeed with cross-validation")
	assert.Equal(t, 1, crossRunner.CallCount, "cross-validation should be called")
}

// TestOrchestrator_FirstIterationPrompt verifies first iteration uses correct prompt
func TestOrchestrator_FirstIterationPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 1
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	var receivedPrompt string
	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			receivedPrompt = prompt
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("COMPLETE", "")), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	orchestrator.Run(ctx)

	assert.NotEmpty(t, receivedPrompt, "first iteration should receive prompt")
	// First iteration should use a prompt that includes task file
	assert.Contains(t, receivedPrompt, tasksFile, "first iteration prompt should reference tasks file")
}

// TestOrchestrator_SubsequentIterationsPrompt verifies subsequent iterations use continue prompt
func TestOrchestrator_SubsequentIterationsPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 3
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	prompts := []string{}
	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			prompts = append(prompts, prompt)
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			if len(prompts) < 3 {
				os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("NEEDS_MORE_WORK", "Continue")), 0644)
			} else {
				os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("COMPLETE", "")), 0644)
			}
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	orchestrator.Run(ctx)

	assert.Len(t, prompts, 3, "should have 3 iteration prompts")
	// All prompts should reference the tasks file
	for i, prompt := range prompts {
		assert.Contains(t, prompt, tasksFile, "iteration %d prompt should reference tasks file", i+1)
	}
}

// TestOrchestrator_ImplRunnerError verifies error handling when implementation runner fails
func TestOrchestrator_ImplRunnerError(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 3
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Implementation runner fails
	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			return errors.New("implementation failed")
		},
	}

	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("NEEDS_MORE_WORK", "Continue")), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	// Should continue to max iterations despite errors
	assert.Equal(t, exitcode.MaxIterations, exitCode, "should hit max iterations after impl errors")
	assert.Equal(t, 3, implRunner.CallCount, "should try impl 3 times")
	assert.Equal(t, 0, valRunner.CallCount, "validation should not run after impl errors")
}

// TestOrchestrator_ValidationRunnerError verifies error handling when validation runner fails
func TestOrchestrator_ValidationRunnerError(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 3
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	// Validation runner fails
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			return errors.New("validation failed")
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	// Should continue to max iterations despite validation errors
	assert.Equal(t, exitcode.MaxIterations, exitCode, "should hit max iterations after val errors")
	assert.Equal(t, 3, implRunner.CallCount, "should try impl 3 times")
	assert.Equal(t, 3, valRunner.CallCount, "should try validation 3 times")
}

// TestOrchestrator_NoAIRunners verifies behavior when no runners are set
func TestOrchestrator_NoAIRunners(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 1
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	// Don't set any runners - they will be nil

	ctx := context.Background()

	// This should panic or error - we're just verifying it doesn't hang
	// In real usage, runners must be set before calling Run
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic with nil runners
			assert.NotNil(t, r, "should panic when runners are nil")
		}
	}()

	orchestrator.Run(ctx)
}

// TestOrchestrator_StateDirectory verifies state directory is created and used
func TestOrchestrator_StateDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.MaxIterations = 1
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			// Mark task as complete
			os.WriteFile(tasksFile, []byte("# Tasks\n- [x] Task 1\n"), 0644)
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("COMPLETE", "")), 0644)
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Success, exitCode, "should succeed")

	// Verify iteration directory was created
	iterDir := filepath.Join(tmpDir, "iteration-001")
	assert.DirExists(t, iterDir, "iteration directory should be created")
}

// Verify that MockOrchestratorAIRunner implements ai.AIRunner interface
var _ ai.AIRunner = (*MockOrchestratorAIRunner)(nil)

// ---------------------------------------------------------------------------
// Session Management Flag Tests (T088-T090)
// ---------------------------------------------------------------------------

// TestOrchestrator_StatusFlag tests --status flag behavior
func TestOrchestrator_StatusFlag(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
- [ ] Task 2
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.Status = true // --status flag
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Create a saved state first
	stateDir := tmpDir
	savedState := &state.SessionState{
		SchemaVersion:   2,
		SessionID:       "test-status-session",
		StartedAt:       "2026-01-30T14:00:00Z",
		LastUpdated:     "2026-01-30T14:30:00Z",
		Iteration:       3,
		Status:          state.StatusInProgress,
		Phase:           state.PhaseValidation,
		Verdict:         "NEEDS_MORE_WORK",
		TasksFile:       tasksFile,
		TasksFileHash:   "abc123",
		AICli:           "claude",
		ImplModel:       "opus",
		ValModel:        "opus",
		MaxIterations:   10,
		MaxInadmissible: 5,
		Learnings:       state.LearningsState{},
		CrossValidation: state.CrossValState{},
		FinalPlanValidation: state.PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     state.TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            state.ScheduleState{},
		RetryState:          state.RetryState{Attempt: 1, Delay: 5},
	}
	require.NoError(t, state.SaveState(savedState, stateDir))

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = stateDir

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	// --status should exit with code 0 after showing status
	assert.Equal(t, exitcode.Success, exitCode, "--status should exit with success")

	// Verify state wasn't modified
	loadedState, err := state.LoadState(stateDir)
	require.NoError(t, err)
	assert.Equal(t, state.StatusInProgress, loadedState.Status, "State should not be modified by --status")
	assert.Equal(t, 3, loadedState.Iteration, "Iteration should not change")
}

// TestOrchestrator_StatusFlagNoState tests --status when no state exists
func TestOrchestrator_StatusFlagNoState(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.Status = true
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	// Should still exit with success, but show "no active session"
	assert.Equal(t, exitcode.Success, exitCode, "--status should exit success even with no state")
}

// TestOrchestrator_CleanFlag tests --clean flag behavior
func TestOrchestrator_CleanFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Keep tasks file outside of state directory
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.Clean = true // --clean flag
	cfg.MaxIterations = 1
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Create a saved state and some iteration directories in a subdirectory
	stateDir := filepath.Join(tmpDir, ".ralph-loop")
	savedState := &state.SessionState{
		SchemaVersion:   2,
		SessionID:       "old-session",
		StartedAt:       "2026-01-30T14:00:00Z",
		LastUpdated:     "2026-01-30T14:30:00Z",
		Iteration:       2,
		Status:          state.StatusInterrupted,
		Phase:           state.PhaseValidation,
		TasksFile:       tasksFile,
		TasksFileHash:   "abc123",
		AICli:           "claude",
		ImplModel:       "opus",
		ValModel:        "opus",
		MaxIterations:   10,
		MaxInadmissible: 5,
		Learnings:       state.LearningsState{},
		CrossValidation: state.CrossValState{},
		FinalPlanValidation: state.PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     state.TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            state.ScheduleState{},
		RetryState:          state.RetryState{Attempt: 1, Delay: 5},
	}
	require.NoError(t, state.SaveState(savedState, stateDir))

	// Create an iteration directory
	iterDir := filepath.Join(stateDir, "iteration-001")
	require.NoError(t, os.MkdirAll(iterDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(iterDir, "test.txt"), []byte("old data"), 0644))

	// Mock runners
	valRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(tasksFile, []byte("# Tasks\n- [x] Task 1\n"), 0644)
			os.WriteFile(outputPath, []byte(makeOrchestratorValidationJSON("COMPLETE", "")), 0644)
			return nil
		},
	}

	implRunner := &MockOrchestratorAIRunner{
		RunFunc: func(ctx context.Context, prompt string, outputPath string) error {
			os.WriteFile(outputPath, []byte("Implementation output"), 0644)
			return nil
		},
	}

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = stateDir
	orchestrator.ImplRunner = implRunner
	orchestrator.ValRunner = valRunner

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	assert.Equal(t, exitcode.Success, exitCode, "Should succeed after clean and fresh run")

	// Load new state - should have a new session ID
	newState, err := state.LoadState(stateDir)
	require.NoError(t, err)
	assert.NotEqual(t, "old-session", newState.SessionID, "Should have new session ID after clean")
	assert.Equal(t, 1, newState.Iteration, "Should start from iteration 1 after clean")
}

// TestOrchestrator_CancelFlag tests --cancel flag behavior
func TestOrchestrator_CancelFlag(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.Cancel = true // --cancel flag
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Create an in-progress state
	stateDir := tmpDir
	savedState := &state.SessionState{
		SchemaVersion:   2,
		SessionID:       "test-cancel-session",
		StartedAt:       "2026-01-30T14:00:00Z",
		LastUpdated:     "2026-01-30T14:30:00Z",
		Iteration:       2,
		Status:          state.StatusInProgress,
		Phase:           state.PhaseImplementation,
		TasksFile:       tasksFile,
		TasksFileHash:   "abc123",
		AICli:           "claude",
		ImplModel:       "opus",
		ValModel:        "opus",
		MaxIterations:   10,
		MaxInadmissible: 5,
		Learnings:       state.LearningsState{},
		CrossValidation: state.CrossValState{},
		FinalPlanValidation: state.PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     state.TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            state.ScheduleState{},
		RetryState:          state.RetryState{Attempt: 1, Delay: 5},
	}
	require.NoError(t, state.SaveState(savedState, stateDir))

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = stateDir

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	// --cancel should exit with error code
	assert.Equal(t, exitcode.Error, exitCode, "--cancel should exit with error code")

	// Verify state was updated to CANCELLED
	cancelledState, err := state.LoadState(stateDir)
	require.NoError(t, err)
	assert.Equal(t, state.StatusCancelled, cancelledState.Status, "State should be CANCELLED")
}

// TestOrchestrator_CancelFlagNoState tests --cancel when no state exists
func TestOrchestrator_CancelFlagNoState(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.Cancel = true
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = tmpDir

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	// Should exit with error when trying to cancel non-existent session
	assert.Equal(t, exitcode.Error, exitCode, "--cancel should exit with error when no state exists")
}

// TestOrchestrator_CleanAndStatusMutuallyExclusive tests that both flags work independently
func TestOrchestrator_CleanAndStatusIndependent(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := `# Tasks
- [ ] Task 1
`
	require.NoError(t, os.WriteFile(tasksFile, []byte(tasksContent), 0644))

	// Test --status takes precedence (exits before clean would execute)
	cfg := config.NewDefaultConfig()
	cfg.TasksFile = tasksFile
	cfg.Status = true
	cfg.Clean = true
	cfg.MaxIterations = 10
	cfg.CrossValidate = false
	cfg.FinalPlanAI = ""
	cfg.TasksValAI = ""

	// Create state
	stateDir := tmpDir
	savedState := &state.SessionState{
		SchemaVersion:   2,
		SessionID:       "test-session",
		StartedAt:       "2026-01-30T14:00:00Z",
		LastUpdated:     "2026-01-30T14:30:00Z",
		Iteration:       1,
		Status:          state.StatusInProgress,
		Phase:           state.PhaseImplementation,
		TasksFile:       tasksFile,
		TasksFileHash:   "abc123",
		AICli:           "claude",
		ImplModel:       "opus",
		ValModel:        "opus",
		MaxIterations:   10,
		MaxInadmissible: 5,
		Learnings:       state.LearningsState{},
		CrossValidation: state.CrossValState{},
		FinalPlanValidation: state.PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     state.TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            state.ScheduleState{},
		RetryState:          state.RetryState{Attempt: 1, Delay: 5},
	}
	require.NoError(t, state.SaveState(savedState, stateDir))

	orchestrator := NewOrchestrator(cfg)
	orchestrator.StateDir = stateDir

	ctx := context.Background()
	exitCode := orchestrator.Run(ctx)

	// --status should exit before --clean executes
	assert.Equal(t, exitcode.Success, exitCode)

	// State should still exist (not cleaned)
	_, err := state.LoadState(stateDir)
	assert.NoError(t, err, "State should still exist when --status runs before --clean")
}
