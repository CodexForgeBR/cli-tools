# Main Loop Decomposition

This document describes the decomposition of the monolithic `main()` function in `ralph-loop.sh` into modular sub-functions in `main-loop.sh`.

## Overview

The original `main()` function was approximately **1,209 lines** long (lines 3917-5125 in ralph-loop.sh). It has been decomposed into **14 focused sub-functions** plus a cleanup handler and orchestrator.

## File Structure

```
bin/lib/ralph-loop/
└── main-loop.sh (1,312 lines, 61KB)
    ├── cleanup()                        - Trap handler for SIGINT/SIGTERM
    ├── main_init()                      - Load configs, parse args, apply defaults, set up models
    ├── main_handle_commands()           - Handle --status, --clean, --cancel (early exits)
    ├── main_display_banner()            - Show startup banner with config summary
    ├── main_find_tasks()                - Find and validate tasks.md
    ├── main_handle_resume()             - Detect interrupted session, load state, show resume info
    ├── main_validate_setup()            - Validate model/AI combinations, tasks hash
    ├── main_fetch_github_issue()        - Fetch issue body if --github-issue provided
    ├── main_tasks_validation()          - Run tasks-vs-plan validation (iteration 1 only)
    ├── main_handle_schedule()           - Wait for --start-at time
    ├── main_run_post_validation_chain() - Cross-validation -> final plan validation -> success/reject
    ├── main_handle_verdict()            - Case statement on verdict (COMPLETE/NEEDS_MORE_WORK/etc.)
    ├── main_exit_success()              - Success banner, notification, cleanup, exit 0
    ├── main_iteration_loop()            - The while loop driving impl + validation iterations
    └── main()                           - Orchestrator calling sub-functions in sequence
```

## Sub-Function Breakdown

### 1. cleanup() (~20 lines)
**Purpose:** Trap handler for graceful shutdown on SIGINT/SIGTERM

**Responsibilities:**
- Catch interrupt signals
- Save interrupted state if mid-iteration
- Display resume instructions
- Exit with standard interrupt code (130)

### 2. main_init() (~30 lines)
**Purpose:** Initialize configuration and parse command-line arguments

**Responsibilities:**
- Load global config from `~/.config/ralph-loop/config`
- Load project config from `.ralph-loop/config`
- Parse command-line arguments
- Apply configuration precedence (CLI > project > global > defaults)
- Set default models for AI CLI
- Configure cross-validation, final plan validation, and tasks validation AIs
- Validate mutually exclusive flags (--original-plan-file vs --github-issue)

### 3. main_handle_commands() (~65 lines)
**Purpose:** Handle command flags that cause early exits

**Responsibilities:**
- Handle `--status` flag (show current session status)
- Handle `--clean` flag (remove state directory)
- Handle `--cancel` flag (cancel active session)
- Exit appropriately after command execution

### 4. main_display_banner() (~14 lines)
**Purpose:** Show startup banner with branding

**Responsibilities:**
- Display ASCII art banner
- Show "RALPH LOOP" title
- Show subtitle: "Dual-Model Validation for Spec-Driven Dev"

### 5. main_find_tasks() (~11 lines)
**Purpose:** Locate and validate tasks.md file

**Responsibilities:**
- Call `find_tasks_file()` to search for tasks.md
- Store path in `$TASKS_FILE`
- Log the discovered path
- Exit if not found

### 6. main_handle_resume() (~148 lines)
**Purpose:** Load and validate saved session state for resume

**Responsibilities:**
- Check for existing state directory
- Load state file if `--resume` or `--resume-force` flag set
- Restore all saved configuration:
  - Tasks file path
  - Original plan file / GitHub issue
  - Learnings settings
  - AI CLI and models (with override support)
  - Max iterations / max inadmissible
- Validate state integrity (detect modified tasks.md)
- Show resume summary banner
- Restore iteration, feedback, and phase variables
- Handle retry state restoration

### 7. main_validate_setup() (~76 lines)
**Purpose:** Validate configuration and initialize state

**Responsibilities:**
- Validate model/AI combinations are compatible
- Parse scheduled start time (if provided)
- Count initial checked/unchecked tasks
- Exit early if all tasks already completed (unless resuming mid-phase)
- Initialize state directory (if new session)
- Initialize learnings file
- Log summary of session start
- Set script start timestamp

### 8. main_fetch_github_issue() (~75 lines)
**Purpose:** Fetch GitHub issue body as original plan

**Responsibilities:**
- Check if `--github-issue` flag provided
- Extract issue number from URL or number format
- Verify cached plan matches requested issue
- Fetch issue content via GitHub CLI (`gh issue view`)
- Extract issue number, title, and body
- Save to `$STATE_DIR/github-issue-plan.md`
- Set `$ORIGINAL_PLAN_FILE` to cached plan

### 9. main_tasks_validation() (~174 lines)
**Purpose:** Validate tasks.md implements the original plan

**Responsibilities:**
- Determine if validation should run (fresh start or resuming tasks_validation phase)
- Run `run_tasks_validation()` to invoke AI validator
- Check for template violations (fast fail)
- Parse `RALPH_TASKS_VALIDATION` JSON from output
- Extract verdict (VALID/INVALID)
- Programmatically override VALID if contradictions/missing requirements/scope narrowing detected
- Display feedback and exit if INVALID
- Clean up state directory on failure
- Send tasks_invalid notification

### 10. main_handle_schedule() (~32 lines)
**Purpose:** Wait until scheduled start time (if --start-at provided)

**Responsibilities:**
- Check if `$SCHEDULE_TARGET_EPOCH` is set
- Handle resume during waiting phase
- Wait until scheduled time (or skip if already passed)
- Log progress during wait

### 11. main_run_post_validation_chain() (~136 lines)
**Purpose:** Run cross-validation and final plan validation (eliminates duplication)

**Responsibilities:**
- Run cross-validation with alternate AI (if enabled and available)
- Parse `RALPH_CROSS_VALIDATION` JSON
- Return REJECTED if cross-validation fails
- Run final plan validation (if original plan provided)
- Parse `RALPH_FINAL_PLAN_VALIDATION` JSON
- Return NOT_IMPLEMENTED if plan not fully implemented
- Return CONFIRMED if all validations pass
- Return formatted result string: `VERDICT:feedback`

### 12. main_exit_success() (~44 lines)
**Purpose:** Display success banner, send notification, clean up, exit 0 (eliminates duplication)

**Responsibilities:**
- Calculate iteration and total elapsed time
- Log success message
- Save COMPLETE state
- Display success banner (with/without cross-validation message)
- Clean up state directory
- Send completion notification
- Exit with `$EXIT_SUCCESS`

### 13. main_iteration_loop() (~165 lines)
**Purpose:** Main iteration loop driving implementation and validation cycles

**Responsibilities:**
- Loop from 0 to `$MAX_ITERATIONS`
- Handle phase-aware resumption (implementation/validation/cross_validation)
- Run implementation phase (unless skipping due to resume)
- Extract and append learnings from implementation
- Run validation phase
- Call `main_handle_verdict()` to process validation result
- Handle max iterations reached (display banner, send notification, exit)

### 14. main_handle_verdict() (~246 lines)
**Purpose:** Parse validation verdict and take appropriate action

**Responsibilities:**
- Parse `RALPH_VALIDATION` JSON from validation output
- Extract verdict (COMPLETE/NEEDS_MORE_WORK/ESCALATE/INADMISSIBLE/BLOCKED)
- Handle COMPLETE verdict:
  - Double-check unchecked task count
  - Run post-validation chain (cross-validation + final plan validation)
  - Call `main_exit_success()` if all validations pass
  - Continue loop with feedback if rejected
  - Handle blocked tasks scenario
- Handle NEEDS_MORE_WORK: extract feedback, continue loop
- Handle ESCALATE: display banner, send notification, exit
- Handle INADMISSIBLE: increment counter, check threshold, escalate or continue
- Handle BLOCKED: check if all remaining tasks blocked, exit or continue
- Handle unknown verdicts: log warning, continue loop
- Display iteration elapsed time

### 15. main() (~44 lines)
**Purpose:** Orchestrator that calls all sub-functions in sequence

**Responsibilities:**
- Set up trap handlers (SIGINT/SIGTERM)
- Initialize local variables (iteration, feedback, resuming)
- Call sub-functions in 10 phases:
  1. Initialize configuration
  2. Handle command flags
  3. Display banner
  4. Find tasks file
  5. Handle resume logic
  6. Validate setup
  7. Fetch GitHub issue
  8. Run tasks validation
  9. Handle scheduled start
  10. Run iteration loop

## Key Improvements

### 1. Separation of Concerns
Each function has a single, well-defined responsibility. This makes the code easier to:
- Understand (each function does one thing)
- Test (can test individual functions)
- Modify (changes are localized)
- Debug (easier to trace execution flow)

### 2. Elimination of Duplication
Two key areas of duplication were eliminated:

**Cross-validation + Final Plan Validation Chain:**
- Previously duplicated in 2 places (resume cross_validation, normal COMPLETE verdict)
- Now consolidated in `main_run_post_validation_chain()`
- Saves ~120 lines of duplicate code

**Success Exit Path:**
- Previously duplicated in 3 places (with/without cross-validation, alternate AI unavailable)
- Now consolidated in `main_exit_success()`
- Saves ~60 lines of duplicate code

### 3. Improved Error Handling
- Consistent error handling patterns across all functions
- Clear separation between recoverable errors (continue loop) and fatal errors (exit)
- Proper cleanup on all exit paths

### 4. Better State Management
- State variables passed by reference (nameref) to sub-functions
- Clear ownership of variables (local vs global)
- Predictable state transitions between phases

### 5. Enhanced Maintainability
- Logical progression of phases in `main()` orchestrator
- Self-documenting function names
- Comments explain "why" not "what"
- Easier to add new phases or modify existing ones

## Migration Path

To integrate this decomposition into `ralph-loop.sh`:

1. **Source the library:**
   ```bash
   # Add near top of ralph-loop.sh, after initial setup
   source "$(dirname "$0")/lib/ralph-loop/main-loop.sh"
   ```

2. **Replace the monolithic main() function:**
   - Remove lines 3917-5125 (original `main()` and `main "$@"` call)
   - The sourced `main-loop.sh` provides both `main()` and the execution guard

3. **Test thoroughly:**
   - Test all phases: init, resume, validation, iteration loop
   - Test all verdicts: COMPLETE, NEEDS_MORE_WORK, ESCALATE, INADMISSIBLE, BLOCKED
   - Test edge cases: interrupted sessions, scheduled starts, GitHub issues

## Function Dependencies

```
main()
├── main_init()
│   ├── load_config()
│   ├── parse_args()
│   ├── apply_config()
│   ├── set_default_models_for_ai()
│   ├── set_cross_validation_ai()
│   ├── set_final_plan_validation_ai()
│   └── set_tasks_validation_ai()
├── main_handle_commands()
│   ├── show_status()
│   ├── python3 (JSON parsing)
│   └── send_notification()
├── main_display_banner()
├── main_find_tasks()
│   └── find_tasks_file()
├── main_handle_resume()
│   ├── check_existing_state()
│   ├── load_state()
│   ├── set_default_models_for_ai()
│   ├── validate_state()
│   ├── show_resume_summary()
│   └── python3 (JSON parsing)
├── main_validate_setup()
│   ├── validate_models_for_ai()
│   ├── parse_schedule_time()
│   ├── count_unchecked_tasks()
│   ├── count_checked_tasks()
│   ├── init_state_dir()
│   ├── init_learnings_file()
│   ├── log_summary()
│   └── get_timestamp()
├── main_fetch_github_issue()
│   ├── gh (GitHub CLI)
│   └── jq (JSON parsing)
├── main_tasks_validation()
│   ├── save_state()
│   ├── run_tasks_validation()
│   ├── extract_json_from_file()
│   ├── python3 (JSON parsing)
│   ├── send_notification()
│   └── rm (cleanup)
├── main_handle_schedule()
│   └── wait_until_scheduled_time()
└── main_iteration_loop()
    ├── get_timestamp()
    ├── save_state()
    ├── run_implementation()
    ├── extract_learnings()
    ├── append_learnings()
    ├── run_validation()
    ├── main_handle_verdict()
    │   ├── extract_json_from_file()
    │   ├── parse_verdict()
    │   ├── count_unchecked_tasks()
    │   ├── parse_blocked_count()
    │   ├── parse_blocked_tasks()
    │   ├── parse_feedback()
    │   ├── main_run_post_validation_chain()
    │   │   ├── run_cross_validation()
    │   │   ├── extract_json_from_file()
    │   │   ├── python3 (JSON parsing)
    │   │   ├── run_final_plan_validation()
    │   │   └── save_state()
    │   └── main_exit_success()
    │       ├── get_timestamp()
    │       ├── format_duration()
    │       ├── save_state()
    │       ├── log_summary()
    │       ├── send_notification()
    │       └── rm (cleanup)
    ├── format_duration()
    ├── log_summary()
    ├── send_notification()
    └── rm (cleanup)
```

## Testing Checklist

- [ ] Fresh session start (no state)
- [ ] Resume interrupted implementation phase
- [ ] Resume interrupted validation phase
- [ ] Resume interrupted cross_validation phase
- [ ] Resume interrupted tasks_validation phase
- [ ] Resume with modified tasks.md (should require --resume-force)
- [ ] --status flag
- [ ] --clean flag
- [ ] --cancel flag
- [ ] --github-issue flag
- [ ] --original-plan-file flag
- [ ] --start-at scheduled time
- [ ] COMPLETE verdict (all tasks done)
- [ ] COMPLETE verdict with cross-validation REJECTED
- [ ] COMPLETE verdict with final plan validation NOT_IMPLEMENTED
- [ ] NEEDS_MORE_WORK verdict
- [ ] ESCALATE verdict
- [ ] INADMISSIBLE verdict (under threshold)
- [ ] INADMISSIBLE verdict (exceed threshold)
- [ ] BLOCKED verdict (some doable tasks remain)
- [ ] BLOCKED verdict (all tasks blocked)
- [ ] Max iterations reached
- [ ] SIGINT during implementation (Ctrl+C)
- [ ] SIGTERM during validation (kill)

## Future Enhancements

1. **Further decomposition opportunities:**
   - `main_handle_verdict()` could be split into per-verdict handlers
   - `main_iteration_loop()` could extract resume logic to separate function

2. **Unit testing:**
   - Each sub-function can now be tested independently
   - Mock dependencies (file I/O, external commands) for isolated tests

3. **Parallel execution:**
   - Some phases could be parallelized (e.g., fetch GitHub issue while validating models)

4. **Plugin architecture:**
   - Sub-functions provide natural extension points
   - New validation phases could be added without modifying existing code

## Summary

This decomposition transforms a 1,209-line monolithic function into 15 focused sub-functions, improving:
- **Readability:** Each function has a clear purpose
- **Maintainability:** Changes are localized to specific functions
- **Testability:** Functions can be tested independently
- **Reusability:** Common patterns extracted to shared functions
- **Extensibility:** New phases can be added easily

The orchestrator (`main()`) provides a clear, linear flow through 10 phases, making the overall execution logic easy to understand at a glance.
