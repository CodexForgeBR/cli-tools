package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// whitelistSet is a precomputed lookup table for fast whitelist membership checks.
var whitelistSet map[string]bool

func init() {
	whitelistSet = make(map[string]bool, len(WhitelistedVars))
	for _, v := range WhitelistedVars {
		whitelistSet[v] = true
	}
}

// LoadFile parses a KEY=VALUE config file at the given path.
//
// Lines are processed according to these rules:
//   - Empty lines and lines starting with # are skipped.
//   - Lines without an = sign are skipped.
//   - Leading and trailing whitespace is trimmed from both key and value.
//   - Keys not present in WhitelistedVars are silently ignored.
//
// Returns a map of whitelisted key-value pairs, or an error if the file
// cannot be opened.
func LoadFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first '=' only.
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Enforce whitelist.
		if !whitelistSet[key] {
			continue
		}

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	return result, nil
}

// LoadWithPrecedence assembles a Config by merging sources in order of
// increasing priority:
//
//  1. Built-in defaults
//  2. Global config file (globalPath)
//  3. Project config file (projectPath)
//  4. Explicit config file (explicitPath)
//  5. CLI overrides (cliOverrides map)
//
// Any path that is empty is silently skipped. If a non-empty path cannot be
// loaded, an error is returned.
func LoadWithPrecedence(globalPath, projectPath, explicitPath string, cliOverrides map[string]string) (*Config, error) {
	cfg := NewDefaultConfig()

	// Layer 2: global config file.
	if globalPath != "" {
		m, err := LoadFile(globalPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("global config: %w", err)
			}
			// Missing global config is not an error.
		} else {
			ApplyMapToConfig(cfg, m)
		}
	}

	// Layer 3: project config file.
	if projectPath != "" {
		m, err := LoadFile(projectPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("project config: %w", err)
			}
		} else {
			ApplyMapToConfig(cfg, m)
		}
	}

	// Layer 4: explicit config file (must exist if specified).
	if explicitPath != "" {
		m, err := LoadFile(explicitPath)
		if err != nil {
			return nil, fmt.Errorf("explicit config: %w", err)
		}
		ApplyMapToConfig(cfg, m)
	}

	// Layer 5: CLI overrides (highest priority).
	if len(cliOverrides) > 0 {
		ApplyMapToConfig(cfg, cliOverrides)
	}

	return cfg, nil
}

// ApplyMapToConfig sets fields on cfg from the key-value pairs in m.
// Keys must use the WhitelistedVars naming convention (e.g., "AI_CLI").
// Unknown keys are silently ignored. Integer fields that fail to parse
// are silently ignored (the previous value is preserved).
func ApplyMapToConfig(cfg *Config, m map[string]string) {
	for key, value := range m {
		switch key {
		case "AI_CLI":
			cfg.AIProvider = value
		case "IMPL_MODEL":
			cfg.ImplModel = value
		case "VAL_MODEL":
			cfg.ValModel = value
		case "CROSS_VALIDATE":
			cfg.CrossValidate = parseBool(value)
		case "CROSS_AI":
			cfg.CrossAI = value
		case "CROSS_MODEL":
			cfg.CrossModel = value
		case "FINAL_PLAN_AI":
			cfg.FinalPlanAI = value
		case "FINAL_PLAN_MODEL":
			cfg.FinalPlanModel = value
		case "TASKS_VAL_AI":
			cfg.TasksValAI = value
		case "TASKS_VAL_MODEL":
			cfg.TasksValModel = value
		case "MAX_ITERATIONS":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MaxIterations = v
			}
		case "MAX_INADMISSIBLE":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MaxInadmissible = v
			}
		case "MAX_CLAUDE_RETRY":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MaxClaudeRetry = v
			}
		case "MAX_TURNS":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MaxTurns = v
			}
		case "INACTIVITY_TIMEOUT":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.InactivityTimeout = v
			}
		case "TASKS_FILE":
			cfg.TasksFile = value
		case "ORIGINAL_PLAN_FILE":
			cfg.OriginalPlanFile = value
		case "GITHUB_ISSUE":
			cfg.GithubIssue = value
		case "LEARNINGS_FILE":
			cfg.LearningsFile = value
		case "ENABLE_LEARNINGS":
			cfg.EnableLearnings = parseBool(value)
		case "VERBOSE":
			cfg.Verbose = parseBool(value)
		case "NOTIFY_WEBHOOK":
			cfg.NotifyWebhook = value
		case "NOTIFY_CHANNEL":
			cfg.NotifyChannel = value
		case "NOTIFY_CHAT_ID":
			cfg.NotifyChatID = value
		}
	}
}

// parseBool interprets common boolean representations.
// "true", "1", "yes" (case-insensitive) return true; everything else returns false.
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}
