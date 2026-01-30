# Ralph-Loop Library

This directory contains the modular library files for the `ralph-loop.sh` script.

## Structure

```
lib/ralph-loop/
├── README.md                    - This file
├── DECOMPOSITION.md             - Detailed decomposition documentation
├── verify-decomposition.sh      - Verification script for main-loop.sh
├── main-loop.sh                 - Decomposed main() orchestrator and sub-functions
├── globals.sh                   - Global variables and constants
├── config.sh                    - Configuration loading and management
├── logging.sh                   - Logging utilities
├── notifications.sh             - Notification system integration
├── cli.sh                       - Command-line argument parsing
├── models.sh                    - AI model configuration and validation
├── scheduling.sh                - Scheduled execution support
├── tasks.sh                     - Task file operations
├── state.sh                     - Session state management
├── json-parsing.sh              - JSON extraction utilities
├── ai-runners.sh                - AI CLI execution wrappers
├── phases.sh                    - Phase execution (impl/val/cross-val/plan-val)
└── prompts/                     - AI prompt templates
    ├── implementation.txt
    ├── validation.txt
    ├── cross-validation.txt
    ├── final-plan-validation.txt
    └── tasks-validation.txt
```

## Main Loop Decomposition

The **main-loop.sh** file contains the decomposed `main()` function, split into 15 focused sub-functions:

### Function Categories

#### 1. Initialization & Setup (Lines 31-161)
- `cleanup()` - Trap handler for SIGINT/SIGTERM
- `main_init()` - Load configs, parse args, apply defaults, set up models
- `main_handle_commands()` - Handle --status, --clean, --cancel (early exits)
- `main_display_banner()` - Show startup banner
- `main_find_tasks()` - Find and validate tasks.md

#### 2. Session Management (Lines 172-396)
- `main_handle_resume()` - Detect interrupted session, load state, show resume info
- `main_validate_setup()` - Validate model/AI combinations, tasks hash

#### 3. Pre-Loop Phases (Lines 397-676)
- `main_fetch_github_issue()` - Fetch issue body if --github-issue provided
- `main_tasks_validation()` - Run tasks-vs-plan validation (iteration 1 only)
- `main_handle_schedule()` - Wait for --start-at time

#### 4. Validation Chain (Lines 677-856)
- `main_run_post_validation_chain()` - Cross-validation -> final plan validation -> success/reject (eliminates duplication)
- `main_exit_success()` - Success banner, notification, cleanup, exit 0 (eliminates duplication)

#### 5. Iteration Loop (Lines 857-1267)
- `main_iteration_loop()` - The while loop driving impl + validation iterations
- `main_handle_verdict()` - Case statement on verdict (COMPLETE/NEEDS_MORE_WORK/etc.)

#### 6. Orchestrator (Lines 1268-1312)
- `main()` - Orchestrator calling sub-functions in sequence

### Orchestrator Flow

The `main()` function orchestrates 10 sequential phases:

1. **Initialize configuration** - Load configs, parse args, set models
2. **Handle command flags** - Process --status, --clean, --cancel
3. **Display banner** - Show startup banner
4. **Find tasks file** - Locate and validate tasks.md
5. **Handle resume logic** - Load state if resuming interrupted session
6. **Validate setup** - Validate models, count tasks, initialize state
7. **Fetch GitHub issue** - Retrieve issue body as original plan (if specified)
8. **Run tasks validation** - Validate tasks.md implements plan (if plan provided)
9. **Handle scheduled start** - Wait until --start-at time (if specified)
10. **Run iteration loop** - Execute implementation + validation cycles

## Key Improvements

### 1. Elimination of Code Duplication

**Before:**
- Cross-validation + final plan validation logic: **duplicated in 2 places** (~120 lines × 2 = 240 lines)
- Success exit path: **duplicated in 3 places** (~40 lines × 3 = 120 lines)

**After:**
- `main_run_post_validation_chain()` - **single implementation** (136 lines)
- `main_exit_success()` - **single implementation** (44 lines)

**Total savings:** ~180 lines of duplicate code eliminated

### 2. Improved Separation of Concerns

Each function has a single, well-defined responsibility:
- Easier to understand (focused logic)
- Easier to test (isolated functions)
- Easier to modify (localized changes)
- Easier to debug (clear execution flow)

### 3. Better Error Handling

- Consistent error handling patterns
- Clear separation between recoverable errors (continue loop) and fatal errors (exit)
- Proper cleanup on all exit paths via trap handlers

### 4. Enhanced State Management

- State variables passed by reference (nameref) to sub-functions
- Clear ownership of variables (local vs global)
- Predictable state transitions between phases

## Usage

### Sourcing the Library

To use the decomposed main loop in `ralph-loop.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Source all library files
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/lib/ralph-loop"

source "$LIB_DIR/globals.sh"
source "$LIB_DIR/config.sh"
source "$LIB_DIR/logging.sh"
source "$LIB_DIR/notifications.sh"
source "$LIB_DIR/cli.sh"
source "$LIB_DIR/models.sh"
source "$LIB_DIR/scheduling.sh"
source "$LIB_DIR/tasks.sh"
source "$LIB_DIR/state.sh"
source "$LIB_DIR/json-parsing.sh"
source "$LIB_DIR/ai-runners.sh"
source "$LIB_DIR/phases.sh"
source "$LIB_DIR/main-loop.sh"

# main() is now available from main-loop.sh
main "$@"
```

### Verification

Run the verification script to check structure:

```bash
./lib/ralph-loop/verify-decomposition.sh
```

Expected output:
```
Verifying main-loop.sh structure...

✓ File exists: /path/to/main-loop.sh

✓ Function defined: cleanup
✓ Function defined: main_init
✓ Function defined: main_handle_commands
...
✓ Function defined: main

----------------------------------------
Total functions expected: 15
Total functions found: 15
Total functions missing: 0

✓ All functions present!

File statistics:
  Lines: 1312
  Size: 61K

✓ Bash syntax valid!

✓ Decomposition verified successfully!
```

## Testing

See **DECOMPOSITION.md** for complete testing checklist covering:
- Session lifecycle (start, resume, cancel)
- All command flags
- All validation verdicts
- Signal handling (SIGINT, SIGTERM)
- Edge cases (modified tasks.md, scheduled starts, GitHub issues)

## Dependencies

### External Commands
- `python3` - JSON parsing
- `gh` - GitHub CLI (for --github-issue)
- `jq` - JSON processing (for GitHub issues)

### Internal Functions
See dependency tree in **DECOMPOSITION.md** for complete function call graph.

## Future Enhancements

1. **Further decomposition:**
   - Split `main_handle_verdict()` into per-verdict handlers
   - Extract resume logic from `main_iteration_loop()`

2. **Unit testing:**
   - Test each sub-function independently
   - Mock dependencies (file I/O, external commands)

3. **Parallel execution:**
   - Parallelize independent phases (e.g., fetch GitHub issue while validating models)

4. **Plugin architecture:**
   - Sub-functions provide natural extension points
   - New validation phases can be added without modifying existing code

## Documentation

- **DECOMPOSITION.md** - Comprehensive decomposition documentation with:
  - Sub-function breakdown (purpose, responsibilities, line numbers)
  - Key improvements analysis
  - Function dependency tree
  - Testing checklist
  - Migration path

- **verify-decomposition.sh** - Automated verification script

## Contributing

When modifying main-loop.sh:

1. Maintain the established function naming convention (`main_*`)
2. Keep functions focused (single responsibility principle)
3. Update DECOMPOSITION.md if adding/removing functions
4. Run `verify-decomposition.sh` to ensure structure integrity
5. Update the testing checklist if adding new functionality

## License

Same as parent project (ralph-loop.sh).
