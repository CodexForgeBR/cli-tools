package notification

import (
	"context"
	"os/exec"
	"time"
)

// SendNotification sends a notification via openclaw CLI.
// Fire-and-forget: never blocks loop, silent on failure.
// No-op when chatID is empty.
func SendNotification(webhook, channel, chatID, message string) {
	if chatID == "" {
		return
	}

	// 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "openclaw", "message", "send",
		"--webhook", webhook,
		"--channel", channel,
		"--chat-id", chatID,
		"--message", message,
	)

	// Fire and forget - ignore errors
	_ = cmd.Run()
}
