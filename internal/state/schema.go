package state

// SessionState represents the persisted state of a ralph-loop session.
// Written to .ralph-loop/current-state.json.
type SessionState struct {
	SchemaVersion       int            `json:"schema_version"`
	SessionID           string         `json:"session_id"`
	StartedAt           string         `json:"started_at"`
	LastUpdated         string         `json:"last_updated"`
	Iteration           int            `json:"iteration"`
	Status              string         `json:"status"`
	Phase               string         `json:"phase"`
	Verdict             string         `json:"verdict"`
	TasksFile           string         `json:"tasks_file"`
	TasksFileHash       string         `json:"tasks_file_hash"`
	AICli               string         `json:"ai_cli"`
	ImplModel           string         `json:"implementation_model"`
	ValModel            string         `json:"validation_model"`
	MaxIterations       int            `json:"max_iterations"`
	MaxInadmissible     int            `json:"max_inadmissible"`
	OriginalPlanFile    *string        `json:"original_plan_file"`
	GithubIssue         *string        `json:"github_issue"`
	Learnings           LearningsState `json:"learnings"`
	CrossValidation     CrossValState  `json:"cross_validation"`
	FinalPlanValidation PlanValState   `json:"final_plan_validation"`
	TasksValidation     TasksValState  `json:"tasks_validation"`
	Schedule            ScheduleState  `json:"schedule"`
	RetryState          RetryState     `json:"retry_state"`
	InadmissibleCount   int            `json:"inadmissible_count"`
	LastFeedback        string         `json:"last_feedback"`
}

type LearningsState struct {
	Enabled int    `json:"enabled"`
	File    string `json:"file"`
}

type CrossValState struct {
	Enabled   int    `json:"enabled"`
	AI        string `json:"ai"`
	Model     string `json:"model"`
	Available bool   `json:"available"`
}

type PlanValState struct {
	AI        string `json:"ai"`
	Model     string `json:"model"`
	Available bool   `json:"available"`
}

type TasksValState struct {
	AI        string `json:"ai"`
	Model     string `json:"model"`
	Available bool   `json:"available"`
}

type ScheduleState struct {
	Enabled     bool   `json:"enabled"`
	TargetEpoch int64  `json:"target_epoch"`
	TargetHuman string `json:"target_human"`
}

type RetryState struct {
	Attempt int `json:"attempt"`
	Delay   int `json:"delay"`
}

// Status constants
const (
	StatusInProgress  = "IN_PROGRESS"
	StatusInterrupted = "INTERRUPTED"
	StatusComplete    = "COMPLETE"
	StatusCancelled   = "CANCELLED"
)

// Phase constants
const (
	PhaseImplementation      = "implementation"
	PhaseValidation          = "validation"
	PhaseCrossValidation     = "cross_validation"
	PhaseFinalPlanValidation = "final_plan_validation"
	PhaseWaitingForSchedule  = "waiting_for_schedule"
)
