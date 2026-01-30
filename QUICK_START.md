# Quick Start Guide: New Ralph-Loop Features

## Overview

Three new features were added to the cli-tools repository:

1. **Spec-Kit Autopilot** - Auto-detects conversational pivots
2. **Config File Support** - Layered configuration for ralph-loop
3. **Notification Integration** - OpenClaw webhook notifications

## 🚀 Quick Start: Config Files

### 1. Check Global Defaults

Your global config was created at `~/.config/ralph-loop/config`:

```bash
cat ~/.config/ralph-loop/config
```

**Current defaults:**
- AI_CLI=claude
- IMPL_MODEL=opus
- VAL_MODEL=opus
- CROSS_VALIDATE=1
- MAX_ITERATIONS=20
- NOTIFY_WEBHOOK=http://127.0.0.1:18789/webhook
- NOTIFY_CHANNEL=telegram

### 2. Create Project-Specific Config (Optional)

Override defaults for a specific project:

```bash
cd ~/source/coreentities  # or ~/source/mda or ~/source/bcl
mkdir -p .ralph-loop
cat > .ralph-loop/config << 'EOF'
# CoreEntities-specific config
MAX_ITERATIONS=30
IMPL_MODEL=sonnet
VAL_MODEL=opus
EOF
```

### 3. Use Ralph-Loop with Config

```bash
# Uses global config
ralph-loop.sh

# Uses project config (if exists) + global config
cd ~/source/coreentities
ralph-loop.sh

# Override with CLI flags (always highest priority)
ralph-loop.sh --max-iterations 10 --implementation-model opus
```

**Precedence:** CLI flags > project config > global config > defaults

### 4. Load Additional Config

```bash
# Create high-stakes config for critical work
cat > ~/configs/high-stakes.conf << 'EOF'
MAX_ITERATIONS=50
MAX_INADMISSIBLE=3
CROSS_VALIDATE=1
IMPL_MODEL=opus
VAL_MODEL=opus
EOF

# Use it
ralph-loop.sh --config ~/configs/high-stakes.conf
```

## 🔔 Quick Start: Notifications

### Without OpenClaw (Test Mode)

Notifications work out-of-the-box with default webhook URL, but won't actually deliver anywhere until OpenClaw is installed:

```bash
# Run ralph-loop (notifications sent to default webhook, but nobody listening)
ralph-loop.sh

# Check notification payload format:
cat /tmp/test-notification.sh  # See IMPLEMENTATION_SUMMARY.md
```

### With OpenClaw (Full Integration)

**5-Minute Setup:**

```bash
# 1. Install OpenClaw
npm install -g openclaw@latest

# 2. Onboard daemon
openclaw onboard --install-daemon

# 3. Authenticate with Claude
claude setup-token

# 4. Configure Telegram
# - Message @BotFather on Telegram
# - Create new bot
# - Get bot token
# - Add to ~/.openclaw/openclaw.json:
{
  "agent": {
    "model": "anthropic/claude-sonnet-4-5"
  },
  "channels": {
    "telegram": {
      "token": "<YOUR_BOT_TOKEN>",
      "dmPolicy": "pairing"
    }
  }
}

# 5. Pair your Telegram
openclaw pairing approve telegram <code>

# 6. Get your chat ID
# Message @userinfobot on Telegram
# Add to ~/.config/ralph-loop/config:
NOTIFY_CHAT_ID=123456789

# 7. Test it
ralph-loop.sh --status
# You should get a notification (if ralph-loop is configured to notify on status)
```

**Test Notification:**

```bash
curl -X POST http://127.0.0.1:18789/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Test from ralph-loop",
    "channel": "telegram",
    "to": "123456789"
  }'
```

### Notification Events

Ralph-loop will notify you when:

| Event | Trigger | Example Message |
|-------|---------|----------------|
| `completed` | All tasks done | "All tasks completed in 12 iterations (45m 23s)" |
| `max_iterations` | Hit iteration limit | "Exhausted 20 iterations with 3 tasks remaining" |
| `escalate` | Human needed | "Needs human escalation: Conflicting requirements" |
| `blocked` | Tasks blocked | "All 5 tasks blocked - human intervention required" |
| `inadmissible` | Bad practices | "Repeated inadmissible practices (7 violations)" |
| `tasks_invalid` | Plan mismatch | "Tasks don't implement the plan" |
| `interrupted` | Ctrl+C | "Ralph loop interrupted by user (iteration 8)" |

### Disable Notifications

```bash
# Option 1: Clear webhook in config
# Edit ~/.config/ralph-loop/config:
NOTIFY_WEBHOOK=

# Option 2: Override with CLI
ralph-loop.sh --notify-webhook ""

# Option 3: Comment out in config
# NOTIFY_WEBHOOK=http://127.0.0.1:18789/webhook
```

## 🤖 Quick Start: Spec-Kit Autopilot

### How It Works

The autopilot skill **automatically detects** when your conversation pivots away from the current specification and suggests appropriate Spec-Kit commands.

**No setup required** - the skill is global (`~/.claude/skills/speckit-autopilot.md`) and works in all projects with Spec-Kit installed.

### Test It

**1. Open Claude Code in any project:**

```bash
cd ~/source/coreentities  # Must have .specify/ directory
claude
```

**2. Mention a tech stack change:**

```
User: "Let's switch from Entity Framework to Dapper for better performance"
```

**3. Autopilot should respond:**

```
Claude: I've detected a tech stack pivot (ORM replacement). I suggest:
  /speckit.plan - Update data access layer to use Dapper

This will update the technical approach in your plan file.
Shall I proceed?
```

### Trigger Patterns

| You Say | Autopilot Detects | Suggests |
|---------|------------------|----------|
| "Use Redis instead of in-memory cache" | Tech stack pivot | `/speckit.plan` |
| "Add OAuth2 authentication" | New feature | `/speckit.specify` → `/speckit.plan` → `/speckit.tasks` |
| "Enforce 100% code coverage" | Principle change | `/speckit.clarify` (constitution amendment) |
| "API should support XML too" | Requirement mod | `/speckit.specify` → `/speckit.plan` |
| "Switch to microservices" | Architecture pivot | `/speckit.plan` → `/speckit.specify` |
| "Remove XML support" | Breaking change | `/speckit.specify` + version bump |

### When It Activates

The autopilot watches for:
- Technology choices: "switch to X", "use Y instead"
- Feature requests: "add", "implement", "we need"
- Principle shifts: "should prioritize X over Y"
- Requirement changes: "modify", "update", "change"
- Architecture decisions: "refactor to", "redesign as"

### What It Does NOT Do

- ❌ Automatically invoke commands (requires your approval)
- ❌ Modify files directly
- ❌ Validate technical feasibility
- ❌ Implement code changes

It only **suggests** commands - you still control what happens.

### Disable Autopilot (If Needed)

```bash
# Temporarily rename the skill
mv ~/.claude/skills/speckit-autopilot.md ~/.claude/skills/speckit-autopilot.md.disabled

# Re-enable later
mv ~/.claude/skills/speckit-autopilot.md.disabled ~/.claude/skills/speckit-autopilot.md
```

## 📚 Common Workflows

### Workflow 1: Overnight Ralph-Loop with Notifications

```bash
# 1. Configure notifications in ~/.config/ralph-loop/config
NOTIFY_WEBHOOK=http://127.0.0.1:18789/webhook
NOTIFY_CHANNEL=telegram
NOTIFY_CHAT_ID=123456789

# 2. Schedule ralph-loop to start at 10 PM
ralph-loop.sh --start-at "22:00"

# 3. Go to bed

# 4. Wake up to Telegram notification with results
```

### Workflow 2: High-Stakes Project with Conservative Settings

```bash
# 1. Create project config
cd ~/source/coreentities
cat > .ralph-loop/config << 'EOF'
MAX_ITERATIONS=50
MAX_INADMISSIBLE=3
CROSS_VALIDATE=1
IMPL_MODEL=opus
VAL_MODEL=opus
CROSS_MODEL=opus
EOF

# 2. Run with original plan validation
ralph-loop.sh --original-plan-file specs/feature/plan.md

# 3. Get notified on completion or escalation
```

### Workflow 3: Fast Iteration with Single AI

```bash
# 1. Create speed config
cat > ~/configs/fast.conf << 'EOF'
MAX_ITERATIONS=100
CROSS_VALIDATE=0
IMPL_MODEL=sonnet
VAL_MODEL=haiku
ENABLE_LEARNINGS=0
EOF

# 2. Run
ralph-loop.sh --config ~/configs/fast.conf

# 3. Still get notifications on completion
```

### Workflow 4: Spec-Kit + Ralph-Loop Integration

```bash
# 1. Start in Claude Code
cd ~/source/mda
claude

# 2. Discuss feature with user
User: "Let's add webhook support for event notifications"

# 3. Autopilot suggests spec-kit commands
Claude: I suggest /speckit.specify to define webhook requirements

# 4. Use spec-kit to create plan and tasks
/speckit.specify
/speckit.plan
/speckit.tasks

# 5. Switch to ralph-loop for implementation
# Exit Claude Code, then:
ralph-loop.sh --original-plan-file specs/webhooks/plan.md

# 6. Get notified when done
```

## 🔧 Troubleshooting

### Config not loading

```bash
# Check config file syntax
cat ~/.config/ralph-loop/config | grep -v "^#" | grep -v "^$"

# Run with verbose to see loading
ralph-loop.sh -v --status 2>&1 | grep -i config
```

### Notifications not arriving

```bash
# 1. Check OpenClaw is running
curl -s http://127.0.0.1:18789/health

# 2. Test webhook directly
curl -X POST http://127.0.0.1:18789/webhook \
  -H "Content-Type: application/json" \
  -d '{"message": "test", "channel": "telegram"}'

# 3. Check OpenClaw logs
openclaw logs --follow

# 4. Verify chat ID is correct
# Message @userinfobot on Telegram to confirm
```

### Autopilot not detecting pivots

```bash
# 1. Check skill file exists
ls -lh ~/.claude/skills/speckit-autopilot.md

# 2. Verify project has .specify/ directory
ls -la .specify/

# 3. Be more explicit in your request
# Instead of: "Maybe use Dapper?"
# Say: "Let's switch from Entity Framework to Dapper"
```

### CLI flags not overriding config

```bash
# Flags ALWAYS win. If not working, check for typos:
ralph-loop.sh --max-iterations 10  # Correct
ralph-loop.sh --max-iteration 10   # Wrong (singular)

# Check parsed values
ralph-loop.sh --status
```

## 📖 Next Steps

1. **Read full documentation:**
   - `IMPLEMENTATION_SUMMARY.md` - Complete feature description
   - `VERIFICATION_CHECKLIST.md` - Testing guide
   - `ralph-loop.sh --help` - All CLI options

2. **Try the features:**
   - Create project configs for your repos
   - Test autopilot in Claude Code sessions
   - Set up OpenClaw for notifications (optional)

3. **Customize for your workflow:**
   - Adjust global defaults in `~/.config/ralph-loop/config`
   - Create specialty configs (fast, safe, overnight, etc.)
   - Set up notification chat IDs

4. **Provide feedback:**
   - What works well?
   - What's confusing?
   - What's missing?

---

**Need Help?**
- Check `ralph-loop.sh --help`
- Read `IMPLEMENTATION_SUMMARY.md`
- Review `VERIFICATION_CHECKLIST.md`
- Test with small examples first
