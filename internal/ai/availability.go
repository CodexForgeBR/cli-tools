package ai

import "os/exec"

// CheckAvailability checks if the given tools are available in PATH.
// Returns a map of tool name to availability status.
func CheckAvailability(tools ...string) map[string]bool {
	result := make(map[string]bool, len(tools))
	for _, tool := range tools {
		_, err := exec.LookPath(tool)
		result[tool] = err == nil
	}
	return result
}
