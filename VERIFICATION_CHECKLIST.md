# Verification Checklist: Spec-Kit Autopilot + Ralph-Loop Config + Notifications

## ✅ Completed Implementation

### Files Created
- [x] `~/.claude/skills/speckit-autopilot.md` (6.8KB)
- [x] `~/.config/ralph-loop/config` (comprehensive example)
- [x] `/Users/bccs/source/cli-tools/IMPLEMENTATION_SUMMARY.md`
- [x] `/Users/bccs/source/cli-tools/VERIFICATION_CHECKLIST.md` (this file)

### Files Modified
- [x] `~/source/cli-tools/bin/ralph-loop.sh`
  - [x] Added 3 notification variables
  - [x] Added `load_config()` function
  - [x] Added `apply_config()` function
  - [x] Added `send_notification()` function
  - [x] Updated `cleanup()` handler
  - [x] Updated `main()` to load configs
  - [x] Added 4 new CLI flags
  - [x] Updated `usage()` documentation
  - [x] Added notifications at 7 exit points

### Code Quality
- [x] Bash syntax verified (`bash -n` passed)
- [x] Notification payload tested (valid JSON)
- [x] Help text displays correctly
- [x] Config file format validated

## 🧪 Testing Checklist

### 1. Spec-Kit Autopilot Skill

**Test in CoreEntities:**
```bash
cd ~/source/coreentities
claude

# Test tech stack pivot:
User: "Let's switch from Entity Framework to Dapper"

Expected: Autopilot suggests /speckit.plan
```

**Test in MDA:**
```bash
cd ~/source/mda
claude

# Test feature addition:
User: "We need to add OAuth2 authentication"

Expected: Autopilot suggests /speckit.specify → /speckit.plan → /speckit.tasks
```

**Test in BCL:**
```bash
cd ~/source/bcl
claude

# Test principle change:
User: "We should enforce 100% code coverage for all new code"

Expected: Autopilot suggests /speckit.clarify (constitution amendment)
```

- [ ] Tech stack pivot detected
- [ ] Feature addition detected
- [ ] Principle change detected
- [ ] Suggested commands are appropriate
- [ ] References project constitution

### 2. Config File Loading

**Test global config:**
```bash
# Verify global config exists and is readable
cat ~/.config/ralph-loop/config | head -20

# Verify it contains expected defaults
grep "AI_CLI=claude" ~/.config/ralph-loop/config
grep "NOTIFY_WEBHOOK=" ~/.config/ralph-loop/config
```

- [x] Global config file exists
- [x] Contains all 27 whitelisted variables
- [x] Has comprehensive comments
- [x] Sets notification defaults

**Test project config override:**
```bash
# Create test project
mkdir -p /tmp/test-ralph/.ralph-loop
cat > /tmp/test-ralph/.ralph-loop/config << 'EOF'
AI_CLI=codex
MAX_ITERATIONS=5
NOTIFY_WEBHOOK=http://example.com/test
EOF

cd /tmp/test-ralph
echo "- [ ] Test task" > tasks.md

# Run with --help to verify config loads
ralph-loop.sh --help >/dev/null 2>&1 && echo "Config loaded successfully"
```

- [ ] Project config created
- [ ] Script loads without errors
- [ ] Project config overrides global config
- [ ] CLI flags still override project config

**Test CLI flag precedence:**
```bash
# With project config setting AI_CLI=codex, verify CLI flag wins:
ralph-loop.sh --ai claude --status
# Should use claude, not codex from project config
```

- [ ] CLI flags override project config
- [ ] CLI flags override global config
- [ ] Precedence order correct

**Test --config flag:**
```bash
# Create additional config
cat > /tmp/custom.conf << 'EOF'
MAX_ITERATIONS=100
VERBOSE=--verbose
EOF

# Load it with --config
ralph-loop.sh --config /tmp/custom.conf --help >/dev/null 2>&1
echo "Additional config loaded"
```

- [ ] --config flag loads file
- [ ] Additional config applies correctly
- [ ] Error handling works for missing files

### 3. Notification Function

**Test payload generation:**
```bash
# Already verified in /tmp/test-notification.sh
/tmp/test-notification.sh
```

- [x] JSON payload structure correct
- [x] All fields populated
- [x] Message escaping works
- [x] Duration formatting works
- [x] Project name extracted correctly

**Test with local OpenClaw (if installed):**
```bash
# Check if OpenClaw is running
curl -s http://127.0.0.1:18789/health 2>/dev/null && echo "OpenClaw running"

# Send test notification
curl -X POST http://127.0.0.1:18789/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Test from ralph-loop verification",
    "channel": "telegram",
    "event": "test",
    "project": "cli-tools"
  }'

# Check Telegram for message
```

- [ ] OpenClaw daemon running
- [ ] Webhook endpoint responds
- [ ] Notification received on Telegram
- [ ] Message content correct

**Test fire-and-forget behavior:**
```bash
# Test with invalid webhook (should not block)
export NOTIFY_WEBHOOK=http://invalid.example.com:9999/webhook
time ralph-loop.sh --status 2>&1 | grep -i "notification"

# Should complete quickly (not hang waiting for webhook)
```

- [ ] Invalid webhook doesn't block script
- [ ] Timeout protection works (5s + 10s = 15s max)
- [ ] No error output to user

### 4. Exit Point Notifications

**Test completed notification:**
```bash
# Create simple test project
mkdir -p /tmp/test-complete
cd /tmp/test-complete
echo "- [x] Done" > tasks.md

# Run (should exit 0 immediately since task is checked)
ralph-loop.sh 2>&1 | tee /tmp/ralph-test.log

# Check for notification call
grep -i "notification" /tmp/ralph-test.log || echo "Check OpenClaw/logs"
```

- [ ] EXIT_SUCCESS (0) sends "completed" notification
- [ ] Message includes iteration count
- [ ] Message includes elapsed time

**Test max iterations notification:**
```bash
# Create test with unchecked tasks and low iteration limit
mkdir -p /tmp/test-max-iter
cd /tmp/test-max-iter
echo "- [ ] Task 1" > tasks.md
echo "- [ ] Task 2" >> tasks.md

# Run with max-iterations=1
ralph-loop.sh --max-iterations 1 2>&1
# Should exit 2 after 1 iteration
```

- [ ] EXIT_MAX_ITERATIONS (2) sends "max_iterations" notification
- [ ] Message includes remaining task count
- [ ] Message includes total elapsed time

**Test interrupted notification:**
```bash
# Run and interrupt with Ctrl+C after a few seconds
# (Manual test - start ralph-loop and hit Ctrl+C)
ralph-loop.sh &
PID=$!
sleep 3
kill -INT $PID
wait $PID
# Should exit 130 with "interrupted" notification
```

- [ ] SIGINT (Ctrl+C) sends "interrupted" notification
- [ ] State saved before exit
- [ ] Exit code 130 correct

**Test other exit codes:**
(These require specific conditions to trigger - defer to integration testing)

- [ ] EXIT_ESCALATE (3) sends "escalate" notification
- [ ] EXIT_BLOCKED (4) sends "blocked" notification
- [ ] EXIT_TASKS_INVALID (5) sends "tasks_invalid" notification
- [ ] EXIT_INADMISSIBLE (6) sends "inadmissible" notification

### 5. Help Text and Documentation

**Verify help sections:**
```bash
ralph-loop.sh --help | grep -A 10 "Configuration Files:"
ralph-loop.sh --help | grep -A 15 "Notifications:"
ralph-loop.sh --help | grep "notify-webhook"
ralph-loop.sh --help | grep "notify-channel"
ralph-loop.sh --help | grep "notify-chat-id"
ralph-loop.sh --help | grep "config PATH"
```

- [x] "Configuration Files:" section present
- [x] "Notifications:" section present
- [x] Precedence explanation clear
- [x] All 4 new flags documented
- [x] OpenClaw setup instructions included
- [x] Notification events listed

### 6. Edge Cases and Error Handling

**Test missing config file:**
```bash
# Global config missing (should not error)
mv ~/.config/ralph-loop/config ~/.config/ralph-loop/config.bak
ralph-loop.sh --help >/dev/null 2>&1 && echo "No error on missing config"
mv ~/.config/ralph-loop/config.bak ~/.config/ralph-loop/config
```

- [ ] Missing global config doesn't error
- [ ] Missing project config doesn't error
- [ ] Script uses defaults

**Test malformed config:**
```bash
# Create config with invalid syntax
cat > /tmp/bad.conf << 'EOF'
INVALID SYNTAX HERE
AI_CLI=claude
=broken=
EOF

ralph-loop.sh --config /tmp/bad.conf --help 2>&1 | grep -i error
# Should skip invalid lines, load valid ones
```

- [ ] Invalid lines skipped
- [ ] Valid lines loaded
- [ ] No fatal errors

**Test unknown config variables:**
```bash
# Create config with unknown variables
cat > /tmp/unknown.conf << 'EOF'
UNKNOWN_VAR=value
MALICIOUS_CODE='$(rm -rf /)'
AI_CLI=claude
EOF

ralph-loop.sh --config /tmp/unknown.conf --help >/dev/null 2>&1
# Should skip unknown vars (whitelist security)
```

- [ ] Unknown variables ignored
- [ ] Malicious code not executed
- [ ] Whitelist security works

**Test notification with special characters:**
```bash
# Message with quotes, newlines, special chars
# (Tested in /tmp/test-notification.sh - message escaping works)
```

- [x] Double quotes escaped
- [x] Newlines handled
- [x] JSON valid

## 🚀 Optional: OpenClaw Installation

**Prerequisites:**
- [ ] Node.js >= 22 installed
- [ ] Telegram account
- [ ] BotFather bot token
- [ ] Claude Pro/Max subscription

**Installation steps:**
```bash
# 1. Install OpenClaw
npm install -g openclaw@latest

# 2. Onboard daemon
openclaw onboard --install-daemon

# 3. Authenticate with Claude
claude setup-token

# 4. Configure Telegram
# Create bot via @BotFather on Telegram
# Add token to ~/.openclaw/openclaw.json

# 5. Pair device
openclaw pairing approve telegram <code>

# 6. Get chat ID
# Message @userinfobot on Telegram

# 7. Update ralph-loop config
# Add NOTIFY_CHAT_ID to ~/.config/ralph-loop/config
```

- [ ] OpenClaw installed
- [ ] Daemon running
- [ ] Claude authenticated
- [ ] Telegram paired
- [ ] Chat ID obtained
- [ ] Config updated

**Minimal OpenClaw config:**
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

- [ ] Config file created
- [ ] Daemon starts successfully
- [ ] Webhook endpoint responds

## 📋 Summary

### What Works Now
- [x] Spec-Kit Autopilot skill created (6.8KB)
- [x] Global config file with all defaults
- [x] Config loading with precedence
- [x] Config application preserves CLI overrides
- [x] Notification function with JSON payload
- [x] 7 exit points instrumented
- [x] Help text updated
- [x] Bash syntax validated
- [x] Notification payload tested

### What Needs Testing
- [ ] Spec-Kit autopilot in live Claude Code sessions
- [ ] Project config override behavior
- [ ] CLI flag precedence
- [ ] OpenClaw integration (if installed)
- [ ] All 7 notification events
- [ ] Edge cases and error handling

### What's Optional
- [ ] OpenClaw installation and setup
- [ ] Telegram bot configuration
- [ ] End-to-end notification testing

## 🎯 Acceptance Criteria

**Minimum for "Done":**
1. All files created ✅
2. All code modifications complete ✅
3. Bash syntax valid ✅
4. Help text complete ✅
5. Config file examples provided ✅
6. Notification function works ✅

**Recommended for "Production Ready":**
1. Spec-Kit autopilot tested in all 3 projects
2. Config precedence verified with real examples
3. At least one notification event tested end-to-end
4. Edge cases handled gracefully

**Ideal for "Full Integration":**
1. OpenClaw installed and configured
2. Telegram notifications working
3. All 7 exit events verified
4. Documentation updated in README

---

**Current Status:** Implementation Complete ✅
**Testing Status:** Partially Tested (syntax, payload generation)
**Deployment Status:** Ready for User Testing
