package ai

import (
	"context"
	"os"
	"strings"
	"time"
)

// MonitorConfig configures process monitoring behavior.
type MonitorConfig struct {
	InactivityTimeout int           // seconds before killing inactive process
	HardCap           int           // absolute max seconds (default 7200)
	OutputPath        string        // file to monitor for size changes
	TickInterval      time.Duration // interval between checks (default 2s, configurable for testing)
}

// MonitorProcess monitors an AI process by watching its output file.
// It cancels the context if:
// - No output for InactivityTimeout seconds
// - Total runtime exceeds HardCap seconds
// - A result marker (RALPH_STATUS or RALPH_VALIDATION) is detected, after a 2s grace period
func MonitorProcess(ctx context.Context, cancel context.CancelFunc, cfg MonitorConfig) {
	if cfg.HardCap == 0 {
		cfg.HardCap = 7200
	}
	if cfg.TickInterval == 0 {
		cfg.TickInterval = 2 * time.Second
	}

	ticker := time.NewTicker(cfg.TickInterval)
	defer ticker.Stop()

	startTime := time.Now()
	lastSize := int64(0)
	lastChange := time.Now()
	resultDetected := false
	var resultTime time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(startTime)

			// Hard cap check
			if elapsed.Seconds() >= float64(cfg.HardCap) {
				cancel()
				return
			}

			// Check file size
			info, err := os.Stat(cfg.OutputPath)
			if err != nil {
				// File doesn't exist yet, continue waiting
				continue
			}

			currentSize := info.Size()
			if currentSize != lastSize {
				lastSize = currentSize
				lastChange = time.Now()

				// Check for result markers
				if !resultDetected {
					data, err := os.ReadFile(cfg.OutputPath)
					if err == nil {
						content := string(data)
						if strings.Contains(content, "RALPH_STATUS") || strings.Contains(content, "RALPH_VALIDATION") {
							resultDetected = true
							resultTime = time.Now()
						}
					}
				}
			}

			// Result detected - grace period
			if resultDetected && time.Since(resultTime) > 2*time.Second {
				cancel()
				return
			}

			// Inactivity check
			if cfg.InactivityTimeout > 0 && time.Since(lastChange).Seconds() >= float64(cfg.InactivityTimeout) {
				cancel()
				return
			}
		}
	}
}
