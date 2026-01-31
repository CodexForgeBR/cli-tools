# Tasks: Convert ralph-loop.sh to Go CLI Binary

**Input**: Design documents from `/specs/001-ralph-loop-go-cli/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-interface.md, quickstart.md

**Tests**: TDD approach — all tests written first (Red phase), then implementation (Green phase), then refactor. Each user story includes tests before implementation.

**Organization**: Tasks grouped by user story. TDD order within each story: tests → implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

- **Binary entry point**: `cmd/ralph-loop/`
- **Internal packages**: `internal/<package>/`
- **Test files**: `internal/<package>/<file>_test.go` (colocated with source)
- **Golden test data**: `testdata/`
- **CI/CD configs**: `.github/workflows/`, `.goreleaser.yml`, `.golangci.yml`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize Go module, project structure, CI/CD configuration, and build tooling

- [ ] T001 Initialize Go module with `go mod init github.com/CodexForgeBR/cli-tools` and add dependencies (cobra, color, testify) to go.mod
- [ ] T002 Create directory structure: cmd/ralph-loop/, internal/ (all 17 package dirs), testdata/ (output/, config/, state/, tasks/ subdirs), .github/workflows/
- [ ] T003 [P] Create Makefile with targets: test, lint, build, all, clean per quickstart.md
- [ ] T004 [P] Create .golangci.yml with linters: govet, errcheck, staticcheck, gosimple, ineffassign, unused, misspell, gofmt, goimports
- [ ] T005 [P] Create .goreleaser.yml for darwin/arm64, darwin/amd64, linux/amd64 builds with ldflags version embedding and homebrew tap auto-update to CodexForgeBR/homebrew-tap
- [ ] T006 [P] Create .github/workflows/ci.yml with 3 parallel jobs (test, lint, build) on PR to main, path-filtered to **.go, go.mod, go.sum, workflow files
- [ ] T007 [P] Create .github/workflows/release.yml triggered on v* tag push, using GoReleaser action
- [ ] T008 Update .gitignore to add: dist/, ralph-loop, *.exe, coverage.out
- [ ] T009 [P] Create cmd/ralph-loop/main.go with version vars (version, commit, date) for ldflags injection, minimal cobra root command wiring, and os.Exit with exit code

**Checkpoint**: Go module compiles (`go build ./cmd/ralph-loop/`), CI config present, empty binary runs

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Leaf packages that all user stories depend on. TDD: write tests first, then implement.

**CRITICAL**: No user story work can begin until this phase is complete.

### Tests for Foundational Packages

- [ ] T010 [P] Write tests for exit code constants in internal/exitcode/codes_test.go — verify all 8 codes (0-6, 130) have correct integer values and names
- [ ] T011 [P] Write tests for logging in internal/logging/logger_test.go — test format_duration (seconds-only, minutes+seconds, hours+minutes+seconds, zero), log level prefixes, color output
- [ ] T012 [P] Write tests for config struct defaults in internal/config/config_test.go — verify all 24 whitelisted variables have correct defaults, Config struct initialization
- [ ] T013 [P] Write tests for config loader in internal/config/loader_test.go — test KEY=VALUE parsing (basic, comments, whitespace, empty lines), whitelist enforcement (unknown keys skipped), precedence chain (defaults < global < project < --config < CLI flags)
- [ ] T014 [P] Write tests for tasks discovery in internal/tasks/discovery_test.go — test auto-detection of ./tasks.md, ./specs/*/tasks.md, and --tasks-file flag
- [ ] T015 [P] Write tests for tasks counter in internal/tasks/counter_test.go — test unchecked counting (- [ ] patterns), checked counting (- [x] and - [X] patterns), mixed content
- [ ] T016 [P] Write tests for tasks hasher in internal/tasks/hasher_test.go — test SHA-256 hash computation: known file → known hash, empty file, unicode content
- [ ] T017 [P] Write tests for tasks compliance in internal/tasks/compliance_test.go — test forbidden pattern detection (git push, gh pr create), clean files pass, violation files fail
- [ ] T018 [P] Write tests for JSON extractor in internal/parser/json_extractor_test.go — test markdown code block extraction, bracket-matching fallback, nested objects, escaped quotes, missing key returns nil
- [ ] T019 [P] Write tests for model defaults in internal/model/defaults_test.go — test claude defaults (impl=opus, val=opus), codex defaults, auto-opposite selection for cross-validation
- [ ] T020 [P] Write tests for model validator in internal/model/validator_test.go — test model/AI compatibility (claude models for claude, GPT models for codex), invalid combinations rejected
- [ ] T021 [P] Write golden test fixture files in testdata/ — create sample Claude stream-json output, Codex JSONL output, validation verdict samples, cross-validation samples, config files, state JSON files, tasks.md files

### Implementation for Foundational Packages

- [ ] T022 [P] Implement exit code constants in internal/exitcode/codes.go — define Success(0), Error(1), MaxIterations(2), Escalate(3), Blocked(4), TasksInvalid(5), Inadmissible(6), Interrupted(130)
- [ ] T023 [P] Implement logging in internal/logging/logger.go — INFO/SUCCESS/WARN/ERROR/PHASE/DEBUG log functions with fatih/color, format_duration helper
- [ ] T024 [P] Implement config struct in internal/config/config.go — Config struct with all fields from data-model.md, NewDefaultConfig() constructor, 24 whitelisted variable names as constants
- [ ] T025 [P] Implement config loader in internal/config/loader.go — LoadFile() for KEY=VALUE parsing with whitelist, LoadWithPrecedence() applying defaults < global < project < explicit < CLI
- [ ] T026 [P] Implement tasks discovery in internal/tasks/discovery.go — DiscoverTasksFile() searching ./tasks.md, ./specs/*/tasks.md (branch pattern match), --tasks-file override
- [ ] T027 [P] Implement tasks counter in internal/tasks/counter.go — CountUnchecked() and CountChecked() using regex patterns matching shell version
- [ ] T028 [P] Implement tasks hasher in internal/tasks/hasher.go — HashFile() returning SHA-256 hex digest using crypto/sha256
- [ ] T029 [P] Implement tasks compliance in internal/tasks/compliance.go — CheckCompliance() scanning for forbidden patterns (git push, PR creation)
- [ ] T030 [P] Implement JSON extractor in internal/parser/json_extractor.go — ExtractJSON() with markdown code block extraction and bracket-matching fallback, string boundary tracking
- [ ] T031 [P] Implement model defaults in internal/model/defaults.go — DefaultImplModel(), DefaultValModel() per AI provider, OppositeAI() for cross-validation selection
- [ ] T032 [P] Implement model validator in internal/model/validator.go — ValidateModelAI() checking compatibility between model names and AI providers

**Checkpoint**: All foundational packages pass tests with `go test -race ./internal/exitcode/... ./internal/logging/... ./internal/config/... ./internal/tasks/... ./internal/parser/json_extractor_test.go ./internal/model/...`

---

## Phase 3: User Story 1 — Run Implementation Loop from Binary (Priority: P1)

**Goal**: Full implementation-validation loop with AI subprocess invocation, output parsing, verdict processing, state persistence, and correct exit codes.

**Independent Test**: Run `ralph-loop --tasks-file ./tasks.md` and verify implementation → validation → verdict loop executes correctly with proper state files and exit codes.

### Tests for User Story 1

- [ ] T033 [P] [US1] Write tests for Claude stream-json parser in internal/parser/stream_json_test.go — test type:assistant content blocks (text + tool_use), type:result fallback, malformed lines skipped
- [ ] T034 [P] [US1] Write tests for Codex JSONL parser in internal/parser/codex_jsonl_test.go — test item.completed events, agent_message/assistant_message types, function_call formatting
- [ ] T035 [P] [US1] Write tests for validation parser in internal/parser/validation_test.go — test RALPH_VALIDATION verdict/feedback/remaining/blocked_count/blocked_tasks extraction, missing fields graceful handling
- [ ] T036 [P] [US1] Write tests for state schema in internal/state/schema_test.go — test SessionState JSON marshal/unmarshal round-trip, schema v2 field names, base64 encoding/decoding of feedback, nested objects
- [ ] T037 [P] [US1] Write tests for state manager in internal/state/manager_test.go — test save_state writes valid JSON, load_state restores all fields, validate_state checks file existence and hash, init_state_dir creates .ralph-loop/
- [ ] T038 [P] [US1] Write tests for learnings extractor in internal/learnings/extractor_test.go — test RALPH_LEARNINGS regex extraction from AI output, empty learnings, bare dash handling
- [ ] T039 [P] [US1] Write tests for learnings manager in internal/learnings/manager_test.go — test init creates markdown template, append formats with iteration number and timestamp, empty → no append
- [ ] T040 [P] [US1] Write tests for prompt builder in internal/prompt/builder_test.go — test BuildImplFirstPrompt (includes inadmissible rules, evidence rules, playwright rules), BuildImplContinuePrompt (includes feedback), BuildValidationPrompt (includes impl_output reference), learnings section injected when present/omitted when empty
- [ ] T041 [P] [US1] Write tests for prompt templates in internal/prompt/templates_test.go — test all template files load via go:embed, variable substitution with {{VARIABLE}} markers, templates are non-empty
- [ ] T042 [P] [US1] Write tests for AIRunner interface in internal/ai/runner_test.go — test interface contract defines Run() method with context, prompt, output path
- [ ] T043 [P] [US1] Write tests for Claude runner in internal/ai/claude_test.go — test command construction (--dangerously-skip-permissions, --model, --print, --max-turns, --verbose, --output-format stream-json)
- [ ] T044 [P] [US1] Write tests for Codex runner in internal/ai/codex_test.go — test command construction (codex exec --json --output-last-message, --dangerously-bypass-approvals-and-sandbox)
- [ ] T045 [P] [US1] Write tests for process monitor in internal/ai/monitor_test.go — test inactivity timeout kills process, hard cap kills process, result detection triggers grace period, zombie detection
- [ ] T046 [P] [US1] Write tests for retry logic in internal/ai/retry_test.go — test exponential backoff (5s, 10s, 20s, 40s...), max retries exceeded → error, state callback on each retry, context cancellation during sleep, resume from saved attempt/delay
- [ ] T047 [P] [US1] Write tests for AI availability in internal/ai/availability_test.go — test exec.LookPath for installed/missing tools
- [ ] T048 [P] [US1] Write tests for signal handler in internal/signal/handler_test.go — test SIGINT saves state with INTERRUPTED status, context cancellation propagates
- [ ] T049 [P] [US1] Write tests for banner display in internal/banner/display_test.go — test startup banner, completion banner (iteration count + duration), escalation banner, blocked banner
- [ ] T050 [P] [US1] Write tests for verdict state machine in internal/phases/verdict_test.go — test all verdict transitions: COMPLETE+0 unchecked→exit 0, COMPLETE+doable unchecked→NEEDS_MORE_WORK override, COMPLETE+all blocked→exit 4, NEEDS_MORE_WORK→feedback+continue, ESCALATE→exit 3, INADMISSIBLE under/over threshold, BLOCKED partial/full, unknown verdict→fallback
- [ ] T051 [P] [US1] Write tests for implementation phase in internal/phases/implementation_test.go — test prompt generated, AI runner called, output saved, learnings extracted
- [ ] T052 [P] [US1] Write tests for validation phase in internal/phases/validation_test.go — test prompt generated, AI runner called, JSON extracted from output
- [ ] T053 [P] [US1] Write tests for post-validation chain in internal/phases/post_validation_chain_test.go — test cross-val→final-plan→success/reject flow
- [ ] T054 [US1] Write tests for orchestrator in internal/phases/orchestrator_test.go — test 10-phase ordering (init→commands→banner→find tasks→resume→validate setup→fetch issue→tasks validation→schedule→iteration loop), max iterations→exit 2, all tasks checked→exit 0

### Implementation for User Story 1

- [ ] T055 [P] [US1] Implement Claude stream-json parser in internal/parser/stream_json.go — parse type:assistant content blocks, type:result fallback, skip malformed lines
- [ ] T056 [P] [US1] Implement Codex JSONL parser in internal/parser/codex_jsonl.go — parse item.completed events, extract text from agent_message/assistant_message
- [ ] T057 [P] [US1] Implement validation parser in internal/parser/validation.go — extract RALPH_VALIDATION fields (verdict, feedback, remaining, blocked_count, blocked_tasks)
- [ ] T058 [P] [US1] Implement state schema in internal/state/schema.go — SessionState struct with JSON tags matching schema v2, nested structs (LearningsState, CrossValState, PlanValState, TasksValState, ScheduleState, RetryState)
- [ ] T059 [P] [US1] Implement state manager in internal/state/manager.go — SaveState() with MarshalIndent 4-space, LoadState(), ValidateState(), InitStateDir()
- [ ] T060 [P] [US1] Implement learnings extractor in internal/learnings/extractor.go — ExtractLearnings() using regex for RALPH_LEARNINGS blocks
- [ ] T061 [P] [US1] Implement learnings manager in internal/learnings/manager.go — InitLearnings(), AppendLearnings() with iteration number and timestamp, ReadLearnings()
- [ ] T062 [US1] Extract prompt template text files from shell version into internal/prompt/templates/ — copy verbatim text from bin/lib/ralph-loop/prompts/*.sh into 11+ .txt files, replacing bash variable references with {{VARIABLE}} markers
- [ ] T063 [US1] Implement prompt templates embedding in internal/prompt/templates.go — go:embed directives for all .txt files in templates/ directory
- [ ] T064 [US1] Implement prompt builder in internal/prompt/builder.go — BuildImplFirstPrompt(), BuildImplContinuePrompt(), BuildValidationPrompt() using strings.NewReplacer for variable substitution, composing shared sections
- [ ] T065 [P] [US1] Implement AIRunner interface in internal/ai/runner.go — define AIRunner interface with Run(ctx, prompt, outputPath) method
- [ ] T066 [P] [US1] Implement Claude runner in internal/ai/claude.go — ClaudeRunner implementing AIRunner, constructing exec.CommandContext with correct flags
- [ ] T067 [P] [US1] Implement Codex runner in internal/ai/codex.go — CodexRunner implementing AIRunner, constructing exec.CommandContext with correct flags
- [ ] T068 [US1] Implement process monitor in internal/ai/monitor.go — goroutine with 2s ticker polling file size, inactivity detection, hard cap (7200s), result detection with 2s grace period, zombie detection
- [ ] T069 [US1] Implement retry logic in internal/ai/retry.go — exponential backoff (5s base, doubling), max retry check, state persistence callback, context-aware sleep
- [ ] T070 [P] [US1] Implement AI availability checker in internal/ai/availability.go — CheckAvailability() using exec.LookPath for claude, codex, gh, openclaw
- [ ] T071 [US1] Implement signal handler in internal/signal/handler.go — register SIGINT/SIGTERM, save state callback, context cancellation, exit 130
- [ ] T072 [P] [US1] Implement banner display in internal/banner/display.go — startup banner, completion banner, escalation banner, blocked banner with color output
- [ ] T073 [US1] Implement verdict state machine in internal/phases/verdict.go — ProcessVerdict() handling all 5 primary verdicts with COMPLETE override logic, inadmissible counting, blocked partial/full distinction
- [ ] T074 [US1] Implement implementation phase in internal/phases/implementation.go — generate prompt, invoke AI runner, save output to iteration dir, extract learnings
- [ ] T075 [US1] Implement validation phase in internal/phases/validation.go — generate validation prompt, invoke AI runner, extract RALPH_VALIDATION JSON from output
- [ ] T076 [US1] Implement post-validation chain in internal/phases/post_validation_chain.go — orchestrate cross-val→final-plan→success/reject flow
- [ ] T077 [US1] Implement orchestrator in internal/phases/orchestrator.go — 10-phase state machine: init, command checks, banner, find tasks, resume check, validate setup, fetch issue, tasks validation, schedule wait, iteration loop
- [ ] T078 [US1] Wire orchestrator into cmd/ralph-loop/main.go — connect cobra root command to orchestrator, pass config, handle exit codes

**Checkpoint**: Core loop functional — `ralph-loop --tasks-file ./tasks.md` executes implementation-validation loop with mock AI or real AI, produces correct state files, and exits with appropriate code. `go test -race ./...` passes.

---

## Phase 4: User Story 2 — Configure and Customize Behavior via CLI Flags (Priority: P1)

**Goal**: All 32 CLI flags parsed correctly with help text matching shell version character-for-character.

**Independent Test**: Run `ralph-loop --help` and diff against `ralph-loop.sh --help`. Verify all flag combinations configure behavior correctly.

### Tests for User Story 2

- [ ] T079 [P] [US2] Write tests for all 32 CLI flags in internal/cli/flags_test.go — one test per flag: --ai (claude|codex|invalid), --verbose/-v, --max-iterations (int, default 20), --max-inadmissible (int, default 5), --max-claude-retry (int, default 10), --max-turns (int, default 100), --inactivity-timeout (int, default 1800), --implementation-model, --validation-model, --tasks-file, --original-plan-file (must exist), --github-issue, --learnings-file, --no-learnings, --no-cross-validate, --cross-model, --cross-validation-ai, --final-plan-validation-ai, --final-plan-validation-model, --tasks-validation-ai, --tasks-validation-model, --start-at/--at alias, --notify-webhook, --notify-channel, --notify-chat-id, --config (must exist), --resume, --resume-force (implies --resume), --clean, --status, --cancel, --help/-h, override detection via Changed()
- [ ] T080 [P] [US2] Write tests for mutual exclusion in internal/cli/flags_test.go — test --original-plan-file + --github-issue → error exit 1
- [ ] T081 [P] [US2] Write tests for help text in internal/cli/usage_test.go — capture help output and compare character-for-character against shell version's help text

### Implementation for User Story 2

- [ ] T082 [US2] Implement all 32 flag definitions in internal/cli/flags.go — cobra flag bindings with correct types, defaults, aliases (--at for --start-at, -v for --verbose, -h for --help), mutual exclusion validation, override detection via cmd.Flags().Changed()
- [ ] T083 [US2] Implement help text in internal/cli/usage.go — custom cobra help template matching shell version's --help output character-for-character
- [ ] T084 [US2] Implement model setup logic in internal/model/setup.go — SetupCrossValidation() (auto-opposite AI/model), SetupFinalPlanValidation() (defaults to cross-val settings), SetupTasksValidation() (defaults to impl settings), model/AI compatibility validation

**Checkpoint**: `ralph-loop --help` output matches shell version. All 32 flags parse correctly. Invalid combinations rejected with exit 1.

---

## Phase 5: User Story 3 — Load Configuration from Files (Priority: P2)

**Goal**: Configuration loaded from global/project/explicit files with correct precedence, whitelisted variables only.

**Independent Test**: Create config files at expected paths, run binary without flags, verify config values applied correctly.

### Tests for User Story 3

- [ ] T085 [P] [US3] Write tests for full config precedence integration in internal/config/loader_test.go — test end-to-end: create global config, project config, explicit config, set CLI flags, verify final resolved Config has correct values from highest-precedence source for each field

### Implementation for User Story 3

- [ ] T086 [US3] Integrate config loading into orchestrator init phase in internal/phases/orchestrator.go — load global config (~/.config/ralph-loop/config), project config (.ralph-loop/config), explicit config (--config), apply CLI flag overrides, validate final config

**Checkpoint**: Config files at all 3 locations loaded with correct precedence. Unknown keys silently ignored. CLI flags override all.

---

## Phase 6: User Story 4 — Resume Interrupted Sessions (Priority: P2)

**Goal**: Session state saved on interruption, resumed from exact phase on --resume, hash validation on tasks.md.

**Independent Test**: Start loop, send SIGINT, run with --resume, verify it continues from interrupted phase.

### Tests for User Story 4

- [ ] T087 [P] [US4] Write tests for state resume in internal/state/resume_test.go — test phase-aware resume (cross_validation skips impl+val, validation skips impl, implementation restarts full iteration, waiting_for_schedule checks if time passed), retry state resume (attempt > 1), tasks hash changed → error, tasks hash changed + --resume-force → ok, CLI flag overrides on resume
- [ ] T088 [P] [US4] Write tests for --status flag in internal/phases/orchestrator_test.go — test --status exits after showing status display
- [ ] T089 [P] [US4] Write tests for --clean flag in internal/phases/orchestrator_test.go — test --clean deletes state directory and starts fresh
- [ ] T090 [P] [US4] Write tests for --cancel flag in internal/phases/orchestrator_test.go — test --cancel marks session as cancelled and exits

### Implementation for User Story 4

- [ ] T091 [US4] Implement state resume logic in internal/state/resume.go — ResumeFromState() with phase-aware continuation, hash comparison, --resume-force bypass, CLI flag override application, retry state restoration
- [ ] T092 [US4] Integrate resume into orchestrator in internal/phases/orchestrator.go — check for existing state on startup, prompt/auto-resume based on flags, implement --status (display + exit), --clean (delete + fresh start), --cancel (mark cancelled + exit)

**Checkpoint**: SIGINT → state saved → --resume continues from correct phase. --status, --clean, --cancel work correctly.

---

## Phase 7: User Story 5 — Cross-Validate and Plan-Validate Results (Priority: P2)

**Goal**: Cross-validation with opposite AI, tasks validation pre-implementation, final plan validation post-cross-val.

**Independent Test**: Run with --original-plan-file, verify tasks validation → impl loop → cross-val → final plan validation chain.

### Tests for User Story 5

- [ ] T093 [P] [US5] Write tests for cross-validation parser in internal/parser/cross_validation_test.go — test RALPH_CROSS_VALIDATION agreement/verdict/feedback extraction
- [ ] T094 [P] [US5] Write tests for tasks validation parser in internal/parser/tasks_validation_test.go — test RALPH_TASKS_VALIDATION verdict/feedback extraction
- [ ] T095 [P] [US5] Write tests for final plan parser in internal/parser/final_plan_test.go — test RALPH_FINAL_PLAN_VALIDATION verdict/feedback extraction
- [ ] T096 [P] [US5] Write tests for cross-validation prompt in internal/prompt/builder_test.go — test BuildCrossValidationPrompt includes all three inputs (tasks, impl output, validation output)
- [ ] T097 [P] [US5] Write tests for tasks validation prompt in internal/prompt/builder_test.go — test BuildTasksValidationPrompt includes spec and tasks references
- [ ] T098 [P] [US5] Write tests for final plan prompt in internal/prompt/builder_test.go — test BuildFinalPlanPrompt includes spec, tasks, plan references
- [ ] T099 [P] [US5] Write tests for cross-validation phase in internal/phases/cross_validation_test.go — test uses opposite AI, prompt includes all three inputs, CONFIRMED and REJECTED handling
- [ ] T100 [P] [US5] Write tests for tasks validation phase in internal/phases/tasks_validation_test.go — test runs pre-implementation, handles VALID (proceed) and INVALID (exit 5)
- [ ] T101 [P] [US5] Write tests for final plan validation phase in internal/phases/final_plan_validation_test.go — test runs after cross-val, handles CONFIRMED (exit 0) and NOT_IMPLEMENTED (feedback + continue)
- [ ] T102 [P] [US5] Write tests for GitHub issue fetching in internal/github/issue_test.go — test URL parsing (owner/repo/number), number-only input, gh command construction, cached issue usage

### Implementation for User Story 5

- [ ] T103 [P] [US5] Implement cross-validation parser in internal/parser/cross_validation.go — extract RALPH_CROSS_VALIDATION fields
- [ ] T104 [P] [US5] Implement tasks validation parser in internal/parser/tasks_validation.go — extract RALPH_TASKS_VALIDATION fields
- [ ] T105 [P] [US5] Implement final plan parser in internal/parser/final_plan.go — extract RALPH_FINAL_PLAN_VALIDATION fields
- [ ] T106 [US5] Add cross-validation and plan-validation prompt builders to internal/prompt/builder.go — BuildCrossValidationPrompt(), BuildTasksValidationPrompt(), BuildFinalPlanPrompt()
- [ ] T107 [US5] Extract cross-validation and tasks-validation and final-plan prompt template .txt files from shell version into internal/prompt/templates/ — copy verbatim text from cross-validation.prompt.sh, tasks-validation.prompt.sh, final-plan.prompt.sh
- [ ] T108 [US5] Implement cross-validation phase in internal/phases/cross_validation.go — use opposite AI, generate prompt, invoke runner, parse verdict, handle CONFIRMED/REJECTED
- [ ] T109 [US5] Implement tasks validation phase in internal/phases/tasks_validation.go — pre-implementation check, generate prompt, invoke runner, parse verdict, exit 5 on INVALID
- [ ] T110 [US5] Implement final plan validation phase in internal/phases/final_plan_validation.go — post-cross-val check, generate prompt, invoke runner, parse verdict, handle CONFIRMED/NOT_IMPLEMENTED
- [ ] T111 [US5] Implement GitHub issue fetching in internal/github/issue.go — ParseIssueRef() for URL/number, FetchIssue() via gh CLI subprocess, CacheIssue() for reuse

**Checkpoint**: Cross-validation runs with opposite AI after COMPLETE. Tasks validation runs pre-implementation when plan file provided. Final plan validation runs after cross-val confirms. All verdict paths work correctly.

---

## Phase 8: User Story 6 — Install via Package Manager (Priority: P3)

**Goal**: Binary compiles cross-platform, releases via GitHub Releases, installs via Homebrew tap.

**Independent Test**: `brew tap codexforgebr/tap && brew install ralph-loop && ralph-loop --version` outputs correct version string.

### Implementation for User Story 6

- [ ] T112 [US6] Verify cross-compilation succeeds for all 3 platforms by running: GOOS=darwin GOARCH=arm64, GOOS=darwin GOARCH=amd64, GOOS=linux GOARCH=amd64 go build ./cmd/ralph-loop/
- [ ] T113 [US6] Verify .goreleaser.yml produces correct archives by running: goreleaser check
- [ ] T114 [US6] Verify version output format in cmd/ralph-loop/main.go — `ralph-loop version vX.Y.Z (commit: abc1234, built: 2026-01-30T12:00:00Z)` matches contract

**Checkpoint**: Binary compiles for all 3 platforms. GoReleaser config validates. Version output matches contract format.

---

## Phase 9: User Story 7 — Send Notifications on Loop Events (Priority: P3)

**Goal**: Fire-and-forget notifications via OpenClaw CLI for all 7 event types.

**Independent Test**: Configure --notify-chat-id, verify correct notification message for each event type.

### Tests for User Story 7

- [ ] T115 [P] [US7] Write tests for notification events in internal/notification/events_test.go — test message formatting for all 7 events: completed, max_iterations, escalate, blocked, tasks_invalid, inadmissible, interrupted
- [ ] T116 [P] [US7] Write tests for notification sender in internal/notification/sender_test.go — test openclaw command construction, 10s timeout, silent skip when chat ID empty, fire-and-forget behavior

### Implementation for User Story 7

- [ ] T117 [P] [US7] Implement notification events in internal/notification/events.go — FormatEvent() for all 7 types with project name, session ID, iteration, exit code
- [ ] T118 [US7] Implement notification sender in internal/notification/sender.go — SendNotification() via openclaw CLI subprocess, 10s timeout, silent on failure, no-op when chat ID empty
- [ ] T119 [US7] Integrate notifications into orchestrator exit paths in internal/phases/orchestrator.go — call SendNotification() before exit for each exit code condition

**Checkpoint**: Notifications sent for all 7 events. Silent when chat ID empty. Fire-and-forget (never blocks loop).

---

## Phase 10: User Story 8 — Schedule Start Time (Priority: P3)

**Goal**: Parse datetime formats and wait with countdown until scheduled time.

**Independent Test**: Run with --start-at "HH:MM" (future time), verify countdown displayed and loop starts at scheduled time.

### Tests for User Story 8

- [ ] T120 [P] [US8] Write tests for schedule parser in internal/schedule/parser_test.go — test YYYY-MM-DD format, HH:MM format (today if future, tomorrow if past), "YYYY-MM-DD HH:MM" format, YYYY-MM-DDTHH:MM ISO format, invalid format → error, past-time detection
- [ ] T121 [P] [US8] Write tests for schedule waiter in internal/schedule/waiter_test.go — test countdown with context cancellation, immediate return for past times

### Implementation for User Story 8

- [ ] T122 [P] [US8] Implement schedule parser in internal/schedule/parser.go — ParseSchedule() supporting 4 datetime formats, detecting past times
- [ ] T123 [US8] Implement schedule waiter in internal/schedule/waiter.go — WaitUntil() with countdown display, context cancellation support
- [ ] T124 [US8] Integrate scheduling into orchestrator in internal/phases/orchestrator.go — parse --start-at, wait before iteration loop, save schedule state

**Checkpoint**: --start-at with all 4 formats parses correctly. Countdown displays and respects cancellation.

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Refactoring, lint cleanup, parity verification, final validation

- [ ] T125 Run golangci-lint and fix all lint issues across all packages
- [ ] T126 Verify help text parity — diff `ralph-loop --help` output against `ralph-loop.sh --help` output and fix any discrepancies
- [ ] T127 Verify state file parity — create a state file with the shell version, load it with the Go binary, save it back, and diff the result
- [ ] T128 Verify prompt text parity — diff each .txt template against the corresponding shell prompt file and fix any discrepancies
- [ ] T129 Verify exit code parity — test each exit condition (0-6, 130) produces the same code as the shell version
- [ ] T130 Run full test suite with race detection: go test -v -race -coverprofile=coverage.out ./...
- [ ] T131 Verify cross-compilation: go build for all 3 platforms (darwin/arm64, darwin/amd64, linux/amd64)
- [ ] T132 Run goreleaser check to validate release configuration

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Foundational — core loop, must complete first
- **US2 (Phase 4)**: Depends on Foundational — can partially parallel with US1 (flag definitions are independent)
- **US3 (Phase 5)**: Depends on US2 (needs flag definitions) and Foundational (needs config loader)
- **US4 (Phase 6)**: Depends on US1 (needs state management and orchestrator)
- **US5 (Phase 7)**: Depends on US1 (needs core loop) and US2 (needs flag definitions for plan file flags)
- **US6 (Phase 8)**: Depends on US1+US2 (needs compilable binary with flags)
- **US7 (Phase 9)**: Depends on US1 (needs orchestrator exit paths)
- **US8 (Phase 10)**: Depends on US1 (needs orchestrator) and US2 (needs --start-at flag)
- **Polish (Phase 11)**: Depends on all user stories

### User Story Dependencies

- **US1 (P1 - Core Loop)**: After Foundational — no story dependencies
- **US2 (P1 - CLI Flags)**: After Foundational — can partially parallel with US1
- **US3 (P2 - Config Files)**: After US2
- **US4 (P2 - Resume)**: After US1
- **US5 (P2 - Cross/Plan Val)**: After US1 + US2
- **US6 (P3 - Distribution)**: After US1 + US2
- **US7 (P3 - Notifications)**: After US1
- **US8 (P3 - Scheduling)**: After US1 + US2

### Within Each User Story (TDD)

1. Tests MUST be written first and FAIL before implementation
2. Implementation in dependency order (parsers → managers → phases → orchestrator)
3. Verify tests pass after implementation
4. Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tests (T010-T021) can run in parallel
- All Foundational implementations (T022-T032) can run in parallel
- US1 and US2 tests can partially overlap (different packages)
- US4, US5, US7, US8 can start in parallel after US1 completes
- Within each story, all tests marked [P] can run in parallel
- Within each story, all implementations marked [P] can run in parallel

---

## Parallel Example: User Story 1

```text
# Launch all US1 tests together (all [P] — different files):
T033: stream_json_test.go
T034: codex_jsonl_test.go
T035: validation_test.go
T036: schema_test.go
T037: manager_test.go
T038: extractor_test.go (learnings)
T039: manager_test.go (learnings)
T040: builder_test.go
T041: templates_test.go
T042: runner_test.go
T043: claude_test.go
T044: codex_test.go
T045: monitor_test.go
T046: retry_test.go
T047: availability_test.go
T048: handler_test.go
T049: display_test.go
T050: verdict_test.go
T051: implementation_test.go
T052: validation_test.go (phases)
T053: post_validation_chain_test.go

# Then launch parallelizable implementations:
T055+T056+T057+T058+T059+T060+T061 (all [P] — different files)
T065+T066+T067+T070+T072 (all [P] — different files)

# Then sequential implementations (depend on previous):
T062→T063→T064 (templates extraction → embedding → builder)
T068→T069 (monitor → retry, retry depends on monitor)
T073→T074→T075→T076→T077→T078 (verdict → phases → orchestrator → main wiring)
```

---

## Implementation Strategy

### MVP First (User Stories 1+2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (Core Loop)
4. Complete Phase 4: User Story 2 (CLI Flags)
5. **STOP and VALIDATE**: Binary runs with correct flags, loop executes, state persists
6. Deploy dev binary for internal testing

### Incremental Delivery

1. Setup + Foundational → Go module compiles
2. + US1 + US2 → Core loop with all flags (MVP)
3. + US3 → Config file loading
4. + US4 → Resume from interruption
5. + US5 → Cross-validation + plan validation
6. + US6 → Distribution via Homebrew
7. + US7 → Notifications
8. + US8 → Scheduled starts
9. Polish → Parity verification, lint, final release

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- TDD strict order: write failing tests → implement → verify tests pass
- Prompt templates extracted verbatim from shell — no rewriting
- State schema v2 compatibility with shell version is a hard requirement
- Exit codes must match exactly (same code for same condition)
- Help text must match character-for-character
