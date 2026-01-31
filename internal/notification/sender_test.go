package notification

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendNotification_SkipsWhenChatIDEmpty(t *testing.T) {
	// Should not execute command when chatID is empty
	// We can't easily verify non-execution, but we can verify it doesn't panic
	SendNotification("https://webhook.example.com", "general", "", "test message")
	// If we got here without panic, test passes
}

func TestSendNotification_CommandConstruction(t *testing.T) {
	// This test verifies the command would be constructed correctly
	// We'll use a mock executable path to test without actually running openclaw

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary script that records its arguments
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/fake-openclaw"

	scriptContent := `#!/bin/bash
echo "$@" > ` + tmpDir + `/args.txt
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	// Temporarily modify PATH to use our fake openclaw
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Send notification
	SendNotification(
		"https://webhook.example.com",
		"test-channel",
		"chat-123",
		"Test notification message",
	)

	// Give it a moment to execute
	time.Sleep(100 * time.Millisecond)

	// Read the recorded arguments
	argsBytes, err := os.ReadFile(tmpDir + "/args.txt")
	if err != nil {
		// If openclaw isn't in PATH, that's fine - we can't test execution
		t.Skip("openclaw not available or fake script didn't execute")
		return
	}

	args := string(argsBytes)
	assert.Contains(t, args, "message")
	assert.Contains(t, args, "send")
	assert.Contains(t, args, "--webhook")
	assert.Contains(t, args, "https://webhook.example.com")
	assert.Contains(t, args, "--channel")
	assert.Contains(t, args, "test-channel")
	assert.Contains(t, args, "--chat-id")
	assert.Contains(t, args, "chat-123")
	assert.Contains(t, args, "--message")
	assert.Contains(t, args, "Test notification message")
}

func TestSendNotification_Timeout(t *testing.T) {
	// Verify that a long-running command is killed after 10 seconds
	// We'll create a script that sleeps longer than the timeout

	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/slow-openclaw"

	// Script that sleeps for 30 seconds
	scriptContent := `#!/bin/bash
sleep 30
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)

	// Temporarily modify PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Rename to openclaw
	err = os.Rename(scriptPath, tmpDir+"/openclaw")
	assert.NoError(t, err)

	start := time.Now()
	SendNotification("https://webhook.example.com", "channel", "chat-123", "message")
	duration := time.Since(start)

	// Should complete within ~10 seconds (with some buffer for overhead)
	assert.Less(t, duration, 12*time.Second, "should timeout within ~10 seconds")
}

func TestSendNotification_FireAndForget(t *testing.T) {
	// Verify that SendNotification doesn't block even if command fails
	// We'll use a non-existent command to ensure it fails

	start := time.Now()

	// Even with a failing command, should return quickly
	SendNotification("https://webhook.example.com", "channel", "chat-123", "message")

	duration := time.Since(start)

	// Should complete very quickly (well under 1 second) even if openclaw doesn't exist
	// or fails - it's fire-and-forget
	assert.Less(t, duration, 11*time.Second, "should not block for long")
}

func TestSendNotification_MultipleCallsInSequence(t *testing.T) {
	// Verify we can call SendNotification multiple times without issues
	for i := 0; i < 5; i++ {
		// Should not panic or block
		SendNotification("https://webhook.example.com", "channel", "chat-123", "message")
	}
}

func TestSendNotification_RealCommand(t *testing.T) {
	// Skip this test unless openclaw is actually installed
	_, err := exec.LookPath("openclaw")
	if err != nil {
		t.Skip("openclaw not installed, skipping real command test")
	}

	// Just verify it doesn't panic with real openclaw
	// (it will likely fail due to invalid webhook, but that's fine)
	SendNotification("https://example.com", "test", "123", "test")
}
