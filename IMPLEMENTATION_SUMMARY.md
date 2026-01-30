# Implementation Summary: Spec-Kit Autopilot + Ralph-Loop Config + OpenClaw Integration

## Overview

Successfully implemented three major enhancements based on research into GitHub Spec-Kit, OpenClaw (formerly Clawdbot/Moltbot), and ralph-loop workflow automation:

1. **Spec-Kit Autopilot Skill** - Automatic detection of conversational pivots
2. **Ralph-Loop Config File Support** - Layered configuration with notification defaults
3. **OpenClaw Notification Integration** - Webhook-based notifications for ralph-loop events

## What Was Implemented

### 1. Spec-Kit Autopilot Skill

**File Created:** `~/.claude/skills/speckit-autopilot.md`

**Purpose:** Automatically detects conversational pivots (tech stack changes, feature requests, principle updates, etc.) and suggests appropriate Spec-Kit commands to keep specification artifacts synchronized.

**Trigger Patterns:**
- Tech stack pivots: "Let's switch from REST to gRPC"
- Feature additions: "We need OAuth2 authentication"
- Principle changes: "Backward compatibility is more important here"
- Requirement modifications: "API should support JSON and XML"
- Architecture pivots: "Switch to microservices"
- Breaking changes: "Remove XML support"

**How It Works:**
1. Detects pivot in conversation
2. Analyzes which artifacts are affected (constitution, specs, plans, tasks)
3. Suggests appropriate `/speckit.*` commands
4. References project-specific constitution for context

**Project Coverage:**
- CoreEntities (constitution v1.7.0, battle-tested)
- MDA (constitution v1.0.0, ratified)
- BCL (placeholder constitution, needs ratification)

**Command Mapping:**

| Pivot Type | Primary Command | Secondary Commands |
|-----------|----------------|-------------------|
| Tech stack | `/speckit.plan` | `/speckit.tasks` |
| New feature | `/speckit.specify` | `/speckit.plan`, `/speckit.tasks` |
| Principle change | `/speckit.clarify` | Constitution amendment |
| Requirement mod | `/speckit.specify` | `/speckit.plan` |
| Architecture | `/speckit.plan` | `/speckit.specify` |
| Breaking change | `/speckit.specify` | `/speckit.plan`, version bump |

### 2. Ralph-Loop Config File Support

**Files Modified:**
- `~/source/cli-tools/bin/ralph-loop.sh`

**Files Created:**
- `~/.config/ralph-loop/config` (global defaults)

**New Functions Added:**

#### `load_config(config_file_path)`
- Reads shell-sourceable `KEY=VALUE` config files
- Whitelist-based security (prevents arbitrary code execution)
- Stores values with `CONFIG_` prefix to avoid overriding CLI flags
- Supports comments (`#`) and empty lines
- Debug logging for loaded values

**Whitelisted Variables:** 27 total including:
- AI configuration (AI_CLI, IMPL_MODEL, VAL_MODEL)
- Cross-validation (CROSS_VALIDATE, CROSS_AI, CROSS_MODEL)
- Final plan validation (FINAL_PLAN_AI, FINAL_PLAN_MODEL)
- Tasks validation (TASKS_VAL_AI, TASKS_VAL_MODEL)
- Iteration limits (MAX_ITERATIONS, MAX_INADMISSIBLE, MAX_CLAUDE_RETRY, MAX_TURNS)
- Timeouts (INACTIVITY_TIMEOUT)
- File paths (TASKS_FILE, ORIGINAL_PLAN_FILE, LEARNINGS_FILE)
- Features (ENABLE_LEARNINGS, VERBOSE)
- Notifications (NOTIFY_WEBHOOK, NOTIFY_CHANNEL, NOTIFY_CHAT_ID)

#### `apply_config()`
- Called after `parse_args()` but before model setup
- Applies config values ONLY where CLI flags were NOT provided
- Checks `OVERRIDE_*` flags to preserve CLI precedence
- Sets notification defaults: `http://127.0.0.1:18789/webhook`, `telegram`

#### `send_notification(event, message, exit_code)`
- Sends JSON webhook POST to configured URL
- Fire-and-forget with short timeouts (5s connect, 10s total)
- Never blocks the loop (all errors suppressed)
- Includes: session_id, event, exit_code, message, iteration, elapsed time, project name

**Config Precedence (Highest Wins):**
```
CLI flags > project config (.ralph-loop/config) > global config (~/.config/ralph-loop/config) > script defaults
```

**New CLI Flags:** 4 total
- `--notify-webhook <URL>` - Webhook URL (default: http://127.0.0.1:18789/webhook)
- `--notify-channel <NAME>` - Channel for routing (default: telegram)
- `--notify-chat-id <ID>` - Recipient identifier
- `--config <PATH>` - Load additional config file (highest priority after CLI)

**Notification Events:** 7 exit points instrumented

| Exit Code | Event | Message Example |
|-----------|-------|----------------|
| 0 | `completed` | "All tasks completed in 12 iterations (45m 23s)" |
| 2 | `max_iterations` | "Exhausted 20 iterations with 3 tasks remaining (1h 15m)" |
| 3 | `escalate` | "Needs human escalation: Conflicting requirements" |
| 4 | `blocked` | "All 5 remaining tasks blocked - human intervention required" |
| 5 | `tasks_invalid` | "Tasks don't properly implement the plan" |
| 6 | `inadmissible` | "Repeated inadmissible practices (7 violations) - needs redesign" |
| 130 | `interrupted` | "Ralph loop interrupted by user (iteration 8)" |

**Config File Format:**
```bash
# Comments supported
AI_CLI=claude
IMPL_MODEL=opus
VAL_MODEL=opus
MAX_ITERATIONS=30
NOTIFY_WEBHOOK=http://127.0.0.1:18789/webhook
NOTIFY_CHANNEL=telegram
NOTIFY_CHAT_ID=123456789
```

**Global Config Defaults:**
- AI_CLI=claude
- IMPL_MODEL=opus
- VAL_MODEL=opus
- CROSS_VALIDATE=1
- MAX_ITERATIONS=20
- MAX_INADMISSIBLE=5
- MAX_CLAUDE_RETRY=10
- MAX_TURNS=100
- INACTIVITY_TIMEOUT=1800
- ENABLE_LEARNINGS=1
- NOTIFY_WEBHOOK=http://127.0.0.1:18789/webhook
- NOTIFY_CHANNEL=telegram

**Updated Help Text:**
- New "Configuration Files" section with precedence explanation
- New "Notifications" section with event types and OpenClaw setup
- Documentation for all 4 new flags

### 3. OpenClaw Notification Integration

**Purpose:** Enable remote notifications when ralph-loop completes, fails, or needs escalation.

**Architecture:**
- Ralph-loop POSTs JSON to webhook endpoint
- OpenClaw daemon receives webhook at `http://127.0.0.1:18789/webhook`
- OpenClaw routes to configured channel (Telegram, Discord, Slack, Signal, WhatsApp)
- Notifications arrive on user's device

**Notification Payload:**
```json
{
    "session_id": "20260130-091523-abc123",
    "event": "completed",
    "exit_code": 0,
    "message": "All tasks completed in 12 iterations (45m 23s)",
    "iteration": 12,
    "max_iterations": 20,
    "elapsed": "45m 23s",
    "project": "coreentities",
    "channel": "telegram",
    "to": "123456789"
}
```

**Security:**
- Gateway binds to loopback only (127.0.0.1)
- Webhook endpoint not exposed to external network
- Fire-and-forget prevents webhook failures from blocking loop
- Short timeouts prevent hanging

**Setup Instructions (Included in Config Comments):**
```bash
# 1. Install OpenClaw
npm install -g openclaw@latest

# 2. Onboard daemon
openclaw onboard --install-daemon

# 3. Authenticate with Claude
claude setup-token

# 4. Pair Telegram
openclaw pairing approve telegram <code>

# 5. Get chat ID
# Message @userinfobot on Telegram

# 6. Configure ralph-loop
# Add NOTIFY_CHAT_ID to ~/.config/ralph-loop/config
```

**Minimal OpenClaw Config (Notification-Only):**
```json
{
  "agent": {
    "model": "anthropic/claude-sonnet-4-5"
  },
  "channels": {
    "telegram": {
      "token": "<BOT_TOKEN>",
      "dmPolicy": "pairing"
    }
  }
}
```

## Files Created/Modified

### Created
1. `~/.claude/skills/speckit-autopilot.md` (6.8KB)
   - Global skill for all projects
   - Auto-detects conversational pivots
   - Suggests appropriate speckit commands

2. `~/.config/ralph-loop/config` (Full example config)
   - Global defaults for all projects
   - Extensive comments and examples
   - Notification configuration

3. `/Users/bccs/source/cli-tools/IMPLEMENTATION_SUMMARY.md` (this file)

### Modified
1. `~/source/cli-tools/bin/ralph-loop.sh`
   - Added 3 new variables (NOTIFY_WEBHOOK, NOTIFY_CHANNEL, NOTIFY_CHAT_ID)
   - Added `load_config()` function (52 lines)
   - Added `apply_config()` function (43 lines)
   - Added `send_notification()` function (38 lines)
   - Updated `cleanup()` handler to send interruption notification
   - Updated `main()` to load configs before parse_args
   - Updated `parse_args()` with 4 new CLI flags
   - Updated `usage()` with configuration and notification documentation
   - Added notification calls at 7 exit points
   - Total additions: ~200 lines

## Testing Performed

### 1. Config File Loading
```bash
# Verified global config file created
cat ~/.config/ralph-loop/config | head -30

# Verified help text updated
ralph-loop.sh --help | grep -A 5 "Configuration Files:"
ralph-loop.sh --help | grep -A 3 "notify-webhook"
```

### 2. Skill Creation
```bash
# Verified skill file created with correct size
ls -lh ~/.claude/skills/speckit-autopilot.md
# Output: -rw-r--r--@ 1 bccs  staff   6.8K
```

### 3. Script Syntax
```bash
# Verified no bash syntax errors
bash -n ~/source/cli-tools/bin/ralph-loop.sh
# (No output = success)
```

## What We Did NOT Implement

Per the plan, we explicitly did NOT:

1. **Replace ralph-loop.sh with OpenClaw** - OpenClaw cannot replicate adversarial multi-phase validation, plan drift detection, or inadmissible practice detection
2. **Install coding skills in OpenClaw** - Notification-only mode, no write access to repos
3. **Fix BCL constitution** - Not selected by user, can be done later
4. **Give OpenClaw write access** - Read-only notification relay for security
5. **Use `source` for config** - Whitelist-based parsing prevents code injection

## Verification Steps

### 1. Test Spec-Kit Autopilot Skill

Open Claude Code in any project with Spec-Kit installed:

```bash
cd ~/source/coreentities  # or ~/source/mda or ~/source/bcl
claude

# Then mention a tech stack change:
"Let's switch from Entity Framework to Dapper for better performance"

# Autopilot should detect the pivot and suggest:
# /speckit.plan - Update data access layer to use Dapper
```

### 2. Test Config File Loading

Create a test project config:

```bash
mkdir -p /tmp/test-ralph/.ralph-loop
cat > /tmp/test-ralph/.ralph-loop/config << 'EOF'
# Test project config
AI_CLI=codex
MAX_ITERATIONS=5
NOTIFY_WEBHOOK=http://example.com/test
EOF

cd /tmp/test-ralph
ralph-loop.sh --status  # Should load config without errors
```

### 3. Test Notification Webhook

After OpenClaw is running:

```bash
# Send test notification
curl -X POST http://127.0.0.1:18789/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Test notification from ralph-loop",
    "channel": "telegram",
    "event": "test"
  }'

# Check Telegram for notification
```

### 4. End-to-End Test

```bash
# Create minimal test project
mkdir -p /tmp/test-ralph
cd /tmp/test-ralph
echo "- [ ] Test task" > tasks.md

# Run ralph-loop (will fail quickly due to no plan, but tests notification)
ralph-loop.sh

# Check for notification on configured channel
```

## Benefits

### Spec-Kit Autopilot
- **No manual artifact management** - Auto-detects when specs, plans, tasks need updates
- **Prevents drift** - Catches when conversation diverges from specification
- **Constitution coherence** - Flags principle violations early
- **Reduces rework** - Specification changes caught before implementation

### Config File Support
- **Eliminates repetitive flags** - Set defaults once, use everywhere
- **Per-project customization** - Override global defaults per repo
- **Sensible notification defaults** - Pre-configured for local OpenClaw
- **Better DX** - Simpler invocations, less typing

### OpenClaw Notifications
- **Remote monitoring** - Check ralph-loop status from phone
- **Overnight automation** - Get notified when scheduled runs complete
- **Proactive alerts** - Know immediately when escalation needed
- **Multi-channel** - Telegram, Discord, Slack, Signal, WhatsApp

## Key Insights from Research

### GitHub Spec-Kit
- **Spec-Driven Development** - Specifications as executable artifacts
- **Multi-agent compatible** - Works with Claude Code, Copilot, Gemini CLI, 8+ others
- **Community extensions** - Autopilot skill from Discussion #991
- **Constitution-based** - Immutable project principles prevent drift

### OpenClaw (Evolution)
- **November 2025**: Launched as "Clawdbot" (Claude pun)
- **January 2026**: Renamed to "Moltbot" after Anthropic trademark C&D
- **January 30, 2026**: Renamed to "OpenClaw" (trademark-cleared)
- **100K+ GitHub stars** in weeks
- **Security incidents**: 900+ exposed instances with leaked API keys
- **3 months old**: Still experimental, unstable

### OpenClaw vs Ralph-Loop
OpenClaw is a **personal AI assistant** (messaging-based orchestration), NOT a **coding automation tool**. Ralph-loop's adversarial validation, plan drift detection, and inadmissible practice detection cannot be replicated.

OpenClaw complements ralph-loop as a notification/monitoring layer, but cannot replace it.

## Usage Examples

### Example 1: Default Configuration
```bash
# Uses global config (~/.config/ralph-loop/config)
ralph-loop.sh
```

### Example 2: Project-Specific Config
```bash
# Create project config
cat > .ralph-loop/config << 'EOF'
MAX_ITERATIONS=50
IMPL_MODEL=sonnet
VAL_MODEL=sonnet
CROSS_VALIDATE=0
EOF

# Run (uses project config + global defaults)
ralph-loop.sh
```

### Example 3: CLI Override
```bash
# CLI flags always win
ralph-loop.sh --max-iterations 10 --implementation-model opus
```

### Example 4: Notifications with Telegram
```bash
# Configure in ~/.config/ralph-loop/config:
# NOTIFY_WEBHOOK=http://127.0.0.1:18789/webhook
# NOTIFY_CHANNEL=telegram
# NOTIFY_CHAT_ID=123456789

# Run ralph-loop
ralph-loop.sh

# Receive notification on Telegram when complete
```

### Example 5: Additional Config File
```bash
# Load high-stakes config for critical projects
ralph-loop.sh --config ~/configs/high-stakes.conf
```

## Next Steps (Optional)

1. **Ratify BCL Constitution** - BCL still has placeholder template, needs project-specific principles
2. **Install OpenClaw** - Set up local daemon for notifications
3. **Pair Telegram** - Get chat ID and test notifications
4. **Create project configs** - Add `.ralph-loop/config` to coreentities, mda, bcl
5. **Test autopilot skill** - Trigger conversational pivots in each project

## References

- [GitHub Spec-Kit Repository](https://github.com/github/spec-kit)
- [Spec-Kit Blog Post](https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/)
- [Spec-Kit Autopilot Discussion](https://github.com/github/spec-kit/discussions/991)
- [OpenClaw Wikipedia](https://en.wikipedia.org/wiki/Moltbot)
- [OpenClaw Evolution Article](https://www.surfercloud.com/blog/from-clawbot-to-openclaw-the-evolution-of-a-personal-ai-giant/)
- [OpenClaw Claude Code Integration Issue](https://github.com/moltbot/moltbot/issues/2555)

---

**Implementation Date:** January 30, 2026
**Total Lines Added:** ~450 (including skill + config + script modifications)
**Files Created:** 3
**Files Modified:** 1
