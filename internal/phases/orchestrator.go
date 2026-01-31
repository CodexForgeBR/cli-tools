package phases

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
	"github.com/CodexForgeBR/cli-tools/internal/banner"
	"github.com/CodexForgeBR/cli-tools/internal/config"
	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
	ghissue "github.com/CodexForgeBR/cli-tools/internal/github"
	"github.com/CodexForgeBR/cli-tools/internal/learnings"
	"github.com/CodexForgeBR/cli-tools/internal/logging"
	"github.com/CodexForgeBR/cli-tools/internal/notification"
	"github.com/CodexForgeBR/cli-tools/internal/prompt"
	"github.com/CodexForgeBR/cli-tools/internal/schedule"
	"github.com/CodexForgeBR/cli-tools/internal/state"
	"github.com/CodexForgeBR/cli-tools/internal/tasks"
)

// CommandChecker is a function type that checks tool availability.
// It takes a list of tool names and returns a map of tool name to availability.
type CommandChecker func(tools ...string) map[string]bool

// Orchestrator runs the 10-phase state machine.
type Orchestrator struct {
	Config          *config.Config
	StateDir        string
	ImplRunner      ai.AIRunner
	ValRunner       ai.AIRunner
	CrossRunner     ai.AIRunner
	FinalPlanRunner ai.AIRunner
	TasksValRunner  ai.AIRunner
	CommandChecker  CommandChecker
	session         *state.SessionState
	startTime       time.Time
}

// NewOrchestrator creates a new orchestrator with the given config.
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		Config:   cfg,
		StateDir: ".ralph-loop",
	}
}

// Run executes the 10-phase orchestration loop and returns an exit code.
func (o *Orchestrator) Run(ctx context.Context) int {
	o.startTime = time.Now()

	// Phase 1: Init
	if code := o.phaseInit(); code >= 0 {
		return code
	}

	// Phase 2: Command checks
	if code := o.phaseCommandChecks(); code >= 0 {
		return code
	}

	// Phase 3: Banner
	o.phaseBanner()

	// Phase 4: Find tasks
	if code := o.phaseFindTasks(); code >= 0 {
		return code
	}

	// Phase 5: Resume check
	if code := o.phaseResumeCheck(); code >= 0 {
		return code
	}

	// Phase 6: Validate setup
	if code := o.phaseValidateSetup(); code >= 0 {
		return code
	}

	// Phase 7: Fetch issue
	o.phaseFetchIssue()

	// Phase 8: Tasks validation
	if code := o.phaseTasksValidation(ctx); code >= 0 {
		return code
	}

	// Phase 9: Schedule wait
	if code := o.phaseScheduleWait(ctx); code >= 0 {
		return code
	}

	// Phase 10: Iteration loop
	return o.phaseIterationLoop(ctx)
}

func (o *Orchestrator) phaseInit() int {
	logging.Phase("Initializing session")

	if err := state.InitStateDir(o.StateDir); err != nil {
		logging.Error(fmt.Sprintf("Failed to init state dir: %v", err))
		return exitcode.Error
	}

	// Check if we're resuming an existing session
	// This happens early to avoid creating a new session when resuming
	if o.Config.Resume || o.Config.ResumeForce {
		// Resume logic is handled in phaseResumeCheck
		// For now, just skip creating a new session
		return -1
	}

	// Create new session
	sessionID := fmt.Sprintf("ralph-%s", time.Now().Format("20060102-150405"))
	o.session = &state.SessionState{
		SchemaVersion:   2,
		SessionID:       sessionID,
		StartedAt:       time.Now().Format(time.RFC3339),
		LastUpdated:     time.Now().Format(time.RFC3339),
		Iteration:       0,
		Status:          state.StatusInProgress,
		Phase:           state.PhaseImplementation,
		AICli:           o.Config.AIProvider,
		ImplModel:       o.Config.ImplModel,
		ValModel:        o.Config.ValModel,
		MaxIterations:   o.Config.MaxIterations,
		MaxInadmissible: o.Config.MaxInadmissible,
		Learnings: state.LearningsState{
			Enabled: boolToInt(o.Config.EnableLearnings),
			File:    o.Config.LearningsFile,
		},
		CrossValidation: state.CrossValState{
			Enabled: boolToInt(o.Config.CrossValidate),
			AI:      o.Config.CrossAI,
			Model:   o.Config.CrossModel,
		},
	}

	return -1 // continue
}

func (o *Orchestrator) phaseCommandChecks() int {
	logging.Phase("Checking required commands")
	// Check availability of primary AI tool
	checker := o.CommandChecker
	if checker == nil {
		checker = ai.CheckAvailability
	}
	avail := checker(o.Config.AIProvider)
	if !avail[o.Config.AIProvider] {
		logging.Error(fmt.Sprintf("Required tool not found: %s", o.Config.AIProvider))
		return exitcode.Error
	}
	return -1
}

func (o *Orchestrator) phaseBanner() {
	if o.session == nil {
		// Session not yet loaded (e.g., during resume). Banner will be
		// printed after phaseResumeCheck restores the session.
		return
	}
	banner.PrintStartupBanner(
		o.session.SessionID,
		o.Config.AIProvider,
		o.Config.ImplModel,
		o.Config.TasksFile,
	)
}

func (o *Orchestrator) phaseFindTasks() int {
	// Skip if resuming — the resumed session already has the tasks file
	if o.session == nil {
		return -1
	}

	logging.Phase("Finding tasks file")

	tasksFile := o.Config.TasksFile
	if tasksFile == "" {
		discovered, err := tasks.DiscoverTasksFile("")
		if err != nil {
			logging.Error(fmt.Sprintf("No tasks file found: %v", err))
			return exitcode.Error
		}
		tasksFile = discovered
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(tasksFile)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to resolve path: %v", err))
		return exitcode.Error
	}

	o.Config.TasksFile = absPath
	o.session.TasksFile = absPath

	// Compute hash
	hash, err := tasks.HashFile(absPath)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to hash tasks file: %v", err))
		return exitcode.Error
	}
	o.session.TasksFileHash = hash

	// Check unchecked count
	unchecked, err := tasks.CountUnchecked(absPath)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to count tasks: %v", err))
		return exitcode.Error
	}
	if unchecked == 0 {
		logging.Success("All tasks already checked!")
		return exitcode.Success
	}

	logging.Info(fmt.Sprintf("Found %d unchecked tasks in %s", unchecked, absPath))
	return -1
}

func (o *Orchestrator) phaseResumeCheck() int {
	// Handle --status flag: show session status and exit
	if o.Config.Status {
		if existing, err := state.LoadState(o.StateDir); err == nil {
			banner.PrintStatusBanner(banner.StatusInfo{
				SessionID:         existing.SessionID,
				Status:            existing.Status,
				Phase:             existing.Phase,
				Verdict:           existing.Verdict,
				Iteration:         existing.Iteration,
				MaxIterations:     existing.MaxIterations,
				InadmissibleCount: existing.InadmissibleCount,
				MaxInadmissible:   existing.MaxInadmissible,
				StartedAt:         existing.StartedAt,
				LastUpdated:       existing.LastUpdated,
				AICli:             existing.AICli,
				ImplModel:         existing.ImplModel,
				ValModel:          existing.ValModel,
				CrossValEnabled:   existing.CrossValidation.Enabled == 1,
				CrossAI:           existing.CrossValidation.AI,
				CrossModel:        existing.CrossValidation.Model,
				RetryAttempt:      existing.RetryState.Attempt,
				RetryDelay:        existing.RetryState.Delay,
				LastFeedback:      existing.LastFeedback,
			})
		} else {
			logging.Info("No active session found.")
		}
		return exitcode.Success
	}

	// Handle --clean flag: remove state directory and start fresh
	if o.Config.Clean {
		logging.Info("Cleaning state directory...")
		if err := os.RemoveAll(o.StateDir); err != nil {
			logging.Warn(fmt.Sprintf("Failed to remove state directory: %v", err))
		}
		if err := state.InitStateDir(o.StateDir); err != nil {
			logging.Warn(fmt.Sprintf("Failed to re-init state dir after clean: %v", err))
		}
	}

	// Handle --cancel flag: mark session as cancelled and exit
	if o.Config.Cancel {
		if existing, err := state.LoadState(o.StateDir); err == nil {
			existing.Status = state.StatusCancelled
			if err := state.SaveState(existing, o.StateDir); err != nil {
				logging.Warn(fmt.Sprintf("Failed to save cancelled state: %v", err))
			}
			logging.Info("Session cancelled.")
		}
		return exitcode.Success
	}

	// Handle --resume and --resume-force flags
	if o.Config.Resume || o.Config.ResumeForce {
		existing, err := state.LoadState(o.StateDir)
		if err != nil {
			logging.Error(fmt.Sprintf("Cannot resume: %v", err))
			return exitcode.Error
		}

		// Resume from existing state
		err = state.ResumeFromState(existing, o.Config.TasksFile, o.Config.ResumeForce)
		if err != nil {
			logging.Error(fmt.Sprintf("Resume failed: %v", err))
			return exitcode.Error
		}

		// Replace the session with the resumed one
		o.session = existing

		// Restore config from saved state so the orchestrator uses the same
		// settings as the original session. CLI flag overrides are applied by
		// the precedence chain in main.go before Run() is called, so any
		// explicit overrides take effect on top of these restored values.
		o.Config.AIProvider = existing.AICli
		o.Config.ImplModel = existing.ImplModel
		o.Config.ValModel = existing.ValModel
		o.Config.MaxIterations = existing.MaxIterations
		o.Config.MaxInadmissible = existing.MaxInadmissible
		o.Config.TasksFile = existing.TasksFile
		o.Config.EnableLearnings = existing.Learnings.Enabled == 1
		o.Config.LearningsFile = existing.Learnings.File
		o.Config.CrossValidate = existing.CrossValidation.Enabled == 1
		o.Config.CrossAI = existing.CrossValidation.AI
		o.Config.CrossModel = existing.CrossValidation.Model

		logging.Info(fmt.Sprintf("Resuming session %s from iteration %d, phase %s",
			existing.SessionID, existing.Iteration, existing.Phase))

		// Skip the rest of init - we already have a session
		return -1
	}

	return -1
}

func (o *Orchestrator) phaseValidateSetup() int {
	logging.Phase("Validating setup")

	// Check compliance
	violations, err := tasks.CheckCompliance(o.session.TasksFile)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to check compliance: %v", err))
		return exitcode.Error
	}
	if len(violations) > 0 {
		for _, v := range violations {
			logging.Warn(fmt.Sprintf("Compliance violation: %s", v))
		}
	}

	// Initialize learnings if enabled
	if o.Config.EnableLearnings {
		learningsPath := o.Config.LearningsFile
		if !filepath.IsAbs(learningsPath) {
			learningsPath = filepath.Join(o.StateDir, filepath.Base(learningsPath))
		}
		o.Config.LearningsFile = learningsPath
		o.session.Learnings.File = learningsPath

		if _, err := os.Stat(learningsPath); os.IsNotExist(err) {
			if err := learnings.InitLearnings(learningsPath); err != nil {
				logging.Warn(fmt.Sprintf("Failed to init learnings file: %v", err))
			}
		}
	}

	return -1
}

func (o *Orchestrator) phaseFetchIssue() {
	if o.Config.GithubIssue == "" {
		return
	}

	logging.Phase("Fetching GitHub issue")

	owner, repo, number, err := ghissue.ParseIssueRef(o.Config.GithubIssue)
	if err != nil {
		logging.Warn(fmt.Sprintf("Failed to parse issue ref: %v", err))
		return
	}

	content, err := ghissue.FetchIssue(owner, repo, number)
	if err != nil {
		logging.Warn(fmt.Sprintf("Failed to fetch issue: %v", err))
		return
	}

	// Cache issue content in state dir
	if err := ghissue.CacheIssue(o.StateDir, content); err != nil {
		logging.Warn(fmt.Sprintf("Failed to cache issue: %v", err))
		return
	}

	issueRef := o.Config.GithubIssue
	o.session.GithubIssue = &issueRef
	if owner != "" {
		logging.Info(fmt.Sprintf("Fetched and cached issue %s/%s#%d", owner, repo, number))
	} else {
		logging.Info(fmt.Sprintf("Fetched and cached issue #%d", number))
	}
}

func (o *Orchestrator) phaseTasksValidation(ctx context.Context) int {
	if o.Config.OriginalPlanFile == "" && o.Config.GithubIssue == "" {
		return -1
	}

	if o.TasksValRunner == nil {
		logging.Warn("Tasks validation runner not configured, skipping")
		return -1
	}

	logging.Phase("Validating tasks against plan")

	specFile := o.Config.OriginalPlanFile
	if specFile == "" {
		// Use cached issue as spec
		specFile = filepath.Join(o.StateDir, "github-issue.md")
	}

	result := RunTasksValidation(ctx, TasksValidationConfig{
		Runner:    o.TasksValRunner,
		SpecFile:  specFile,
		TasksFile: o.session.TasksFile,
	})

	switch result.Action {
	case "success":
		logging.Success("Tasks validation passed")
		return -1
	case "exit":
		logging.Error(fmt.Sprintf("Tasks validation failed: %s", result.Feedback))
		o.notify(notification.EventTasksInvalid, exitcode.TasksInvalid)
		return exitcode.TasksInvalid
	default:
		return -1
	}
}

func (o *Orchestrator) phaseScheduleWait(ctx context.Context) int {
	if o.Config.StartAt == "" {
		return -1
	}

	logging.Phase("Waiting for scheduled start time")

	target, err := schedule.ParseSchedule(o.Config.StartAt)
	if err != nil {
		logging.Error(fmt.Sprintf("Invalid schedule: %v", err))
		return exitcode.Error
	}

	// Save schedule state
	o.session.Schedule = state.ScheduleState{
		Enabled:     true,
		TargetEpoch: target.Unix(),
		TargetHuman: target.Format("2006-01-02 15:04:05"),
	}
	o.session.Phase = state.PhaseWaitingForSchedule
	if err := state.SaveState(o.session, o.StateDir); err != nil {
		logging.Warn(fmt.Sprintf("Failed to save schedule state: %v", err))
	}

	if err := schedule.WaitUntil(ctx, target); err != nil {
		if ctx.Err() != nil {
			banner.PrintInterruptedBanner(o.session.Iteration, o.session.Phase)
			o.notify(notification.EventInterrupted, exitcode.Interrupted)
			if saveErr := state.SaveState(o.session, o.StateDir); saveErr != nil {
				logging.Warn(fmt.Sprintf("Failed to save interrupted state: %v", saveErr))
			}
			return exitcode.Interrupted
		}
		logging.Error(fmt.Sprintf("Schedule wait failed: %v", err))
		return exitcode.Error
	}

	logging.Success("Schedule wait complete, starting iteration loop")
	return -1
}

func (o *Orchestrator) phaseIterationLoop(ctx context.Context) int {
	logging.Phase("Starting iteration loop")

	for o.session.Iteration < o.session.MaxIterations {
		o.session.Iteration++
		o.session.LastUpdated = time.Now().Format(time.RFC3339)

		logging.Info(fmt.Sprintf("=== Iteration %d/%d ===", o.session.Iteration, o.session.MaxIterations))

		// Check for context cancellation
		if ctx.Err() != nil {
			banner.PrintInterruptedBanner(o.session.Iteration, o.session.Phase)
			o.notify(notification.EventInterrupted, exitcode.Interrupted)
			if err := state.SaveState(o.session, o.StateDir); err != nil {
				logging.Warn(fmt.Sprintf("Failed to save interrupted state: %v", err))
			}
			return exitcode.Interrupted
		}

		// Save state before implementation
		o.session.Phase = state.PhaseImplementation
		if err := state.SaveState(o.session, o.StateDir); err != nil {
			logging.Warn(fmt.Sprintf("Failed to save implementation state: %v", err))
		}

		// Run implementation
		isFirst := o.session.Iteration == 1 && o.session.LastFeedback == ""
		feedback := ""
		if o.session.LastFeedback != "" {
			decoded, err := base64.StdEncoding.DecodeString(o.session.LastFeedback)
			if err == nil {
				feedback = string(decoded)
			} else {
				feedback = o.session.LastFeedback
			}
		}

		// Build prompts
		learningsText := learnings.ReadLearnings(o.Config.LearningsFile)
		var implPrompt string
		if isFirst {
			implPrompt = prompt.BuildImplFirstPrompt(o.session.TasksFile, learningsText)
		} else {
			implPrompt = prompt.BuildImplContinuePrompt(o.session.TasksFile, feedback, learningsText)
		}

		// Create iteration directory
		iterDir := filepath.Join(o.StateDir, fmt.Sprintf("iteration-%03d", o.session.Iteration))
		if err := os.MkdirAll(iterDir, 0755); err != nil {
			logging.Warn(fmt.Sprintf("Failed to create iteration dir: %v", err))
		}

		// Run implementation phase
		logging.Phase("Implementation phase")
		implOutputPath := filepath.Join(iterDir, "implementation-output.txt")
		implConfig := ImplementationConfig{
			Runner:           o.ImplRunner,
			Iteration:        o.session.Iteration,
			OutputPath:       implOutputPath,
			FirstPrompt:      implPrompt,
			ContinuePrompt:   implPrompt, // For consistency
			ExtractLearnings: o.Config.EnableLearnings,
		}

		implResult, implErr := RunImplementationPhaseWithLearnings(ctx, implConfig)
		if implErr != nil {
			logging.Error(fmt.Sprintf("Implementation failed: %v", implErr))
			// Check for context cancellation
			if ctx.Err() != nil {
				return exitcode.Interrupted
			}
			continue
		}

		// Append learnings if any
		if implResult.Learnings != "" && o.Config.EnableLearnings {
			if err := learnings.AppendLearnings(o.Config.LearningsFile, o.session.Iteration, implResult.Learnings); err != nil {
				logging.Warn(fmt.Sprintf("Failed to append learnings: %v", err))
			}
		}

		// Run validation
		o.session.Phase = state.PhaseValidation
		if err := state.SaveState(o.session, o.StateDir); err != nil {
			logging.Warn(fmt.Sprintf("Failed to save validation state: %v", err))
		}

		logging.Phase("Validation phase")
		valPrompt := prompt.BuildValidationPrompt(o.session.TasksFile, implOutputPath)
		valOutputPath := filepath.Join(iterDir, "validation-output.txt")
		valConfig := ValidationConfig{
			Runner:     o.ValRunner,
			OutputPath: valOutputPath,
			Prompt:     valPrompt,
		}

		valResult, valErr := RunValidationPhaseWithResult(ctx, valConfig)
		if valErr != nil {
			logging.Error(fmt.Sprintf("Validation failed: %v", valErr))
			// Check for context cancellation
			if ctx.Err() != nil {
				return exitcode.Interrupted
			}
			continue
		}

		// Get current task counts
		unchecked, _ := tasks.CountUnchecked(o.session.TasksFile)

		// Process verdict
		o.session.Verdict = valResult.Verdict
		verdictResult := ProcessVerdict(VerdictInput{
			Verdict:           valResult.Verdict,
			Feedback:          valResult.Feedback,
			Remaining:         unchecked,
			BlockedCount:      len(valResult.BlockedTasks),
			BlockedTasks:      valResult.BlockedTasks,
			InadmissibleCount: o.session.InadmissibleCount,
			MaxInadmissible:   o.session.MaxInadmissible,
		})

		o.session.InadmissibleCount = verdictResult.NewInadmissibleCount

		if verdictResult.Action == "exit" {
			duration := int(time.Since(o.startTime).Seconds())
			switch verdictResult.ExitCode {
			case exitcode.Success:
				// Compute specFile for post-validation chain
				specFile := o.Config.OriginalPlanFile
				if specFile == "" && o.Config.GithubIssue != "" {
					specFile = filepath.Join(o.StateDir, "github-issue.md")
				}

				// Run post-validation chain
				postResult := RunPostValidationChain(ctx, PostValidationConfig{
					CrossValRunner:   o.CrossRunner,
					FinalPlanRunner:  o.FinalPlanRunner,
					CrossValEnabled:  o.Config.CrossValidate && o.CrossRunner != nil,
					FinalPlanEnabled: o.FinalPlanRunner != nil,
					TasksFile:        o.session.TasksFile,
					ImplOutputFile:   implOutputPath,
					ValOutputFile:    valOutputPath,
					SpecFile:         specFile,
					PlanFile:         o.Config.OriginalPlanFile,
				})

				if postResult.Action == "continue" {
					// Cross-val or final-plan rejected, continue loop
					o.session.LastFeedback = base64.StdEncoding.EncodeToString([]byte(postResult.Feedback))
					continue
				}

				o.session.Status = state.StatusComplete
				if err := state.SaveState(o.session, o.StateDir); err != nil {
					logging.Warn(fmt.Sprintf("Failed to save complete state: %v", err))
				}
				banner.PrintCompletionBanner(o.session.Iteration, duration)
				o.notify(notification.EventCompleted, exitcode.Success)
				return exitcode.Success

			case exitcode.Escalate:
				banner.PrintEscalationBanner(verdictResult.Feedback)
				o.notify(notification.EventEscalate, exitcode.Escalate)
				if err := state.SaveState(o.session, o.StateDir); err != nil {
					logging.Warn(fmt.Sprintf("Failed to save escalate state: %v", err))
				}
				return exitcode.Escalate

			case exitcode.Blocked:
				banner.PrintBlockedBanner(valResult.BlockedTasks)
				o.notify(notification.EventBlocked, exitcode.Blocked)
				if err := state.SaveState(o.session, o.StateDir); err != nil {
					logging.Warn(fmt.Sprintf("Failed to save blocked state: %v", err))
				}
				return exitcode.Blocked

			case exitcode.Inadmissible:
				banner.PrintInadmissibleBanner(o.session.InadmissibleCount, o.session.MaxInadmissible)
				o.notify(notification.EventInadmissible, exitcode.Inadmissible)
				if err := state.SaveState(o.session, o.StateDir); err != nil {
					logging.Warn(fmt.Sprintf("Failed to save inadmissible state: %v", err))
				}
				return exitcode.Inadmissible

			default:
				if err := state.SaveState(o.session, o.StateDir); err != nil {
					logging.Warn(fmt.Sprintf("Failed to save state: %v", err))
				}
				return verdictResult.ExitCode
			}
		}

		// Continue: store feedback
		o.session.LastFeedback = base64.StdEncoding.EncodeToString([]byte(verdictResult.Feedback))
		if err := state.SaveState(o.session, o.StateDir); err != nil {
			logging.Warn(fmt.Sprintf("Failed to save feedback state: %v", err))
		}
	}

	// Max iterations reached
	banner.PrintMaxIterationsBanner(o.session.Iteration, o.session.MaxIterations)
	o.notify(notification.EventMaxIterations, exitcode.MaxIterations)
	if err := state.SaveState(o.session, o.StateDir); err != nil {
		logging.Warn(fmt.Sprintf("Failed to save max iterations state: %v", err))
	}
	return exitcode.MaxIterations
}

// notify sends a fire-and-forget notification for the given event.
func (o *Orchestrator) notify(event string, code int) {
	projectName := filepath.Base(filepath.Dir(o.session.TasksFile))
	if projectName == "." || projectName == "" {
		projectName = "ralph-loop"
	}
	msg := notification.FormatEvent(event, projectName, o.session.SessionID, o.session.Iteration, code)
	notification.SendNotification(o.Config.NotifyWebhook, o.Config.NotifyChannel, o.Config.NotifyChatID, msg)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
