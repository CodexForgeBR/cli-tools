package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAvailability_SingleTool(t *testing.T) {
	t.Run("returns true for installed tool", func(t *testing.T) {
		// Use 'ls' which is available on all Unix systems
		result := CheckAvailability("ls")

		require.NotNil(t, result)
		require.Contains(t, result, "ls")
		assert.True(t, result["ls"], "ls should be available")
	})

	t.Run("returns false for missing tool", func(t *testing.T) {
		// Use a tool name that definitely doesn't exist
		result := CheckAvailability("this-tool-definitely-does-not-exist-12345")

		require.NotNil(t, result)
		require.Contains(t, result, "this-tool-definitely-does-not-exist-12345")
		assert.False(t, result["this-tool-definitely-does-not-exist-12345"],
			"nonexistent tool should not be available")
	})

	t.Run("checks common system tools", func(t *testing.T) {
		testCases := []struct {
			tool      string
			shouldExist bool
		}{
			{"ls", true},
			{"cat", true},
			{"echo", true},
			{"nonexistent-tool-xyz", false},
		}

		for _, tc := range testCases {
			t.Run(tc.tool, func(t *testing.T) {
				result := CheckAvailability(tc.tool)
				require.Contains(t, result, tc.tool)

				if tc.shouldExist {
					assert.True(t, result[tc.tool], "%s should be available", tc.tool)
				} else {
					assert.False(t, result[tc.tool], "%s should not be available", tc.tool)
				}
			})
		}
	})
}

func TestCheckAvailability_MultipleTools(t *testing.T) {
	t.Run("checks multiple tools at once", func(t *testing.T) {
		tools := []string{"ls", "cat", "echo"}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)
		assert.Len(t, result, 3, "should return results for all tools")

		for _, tool := range tools {
			require.Contains(t, result, tool, "result should include %s", tool)
			assert.True(t, result[tool], "%s should be available", tool)
		}
	})

	t.Run("checks mix of installed and missing tools", func(t *testing.T) {
		tools := []string{
			"ls",                                    // exists
			"cat",                                   // exists
			"nonexistent-tool-abc",                 // doesn't exist
			"another-missing-tool-xyz",             // doesn't exist
		}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)
		assert.Len(t, result, 4, "should return results for all tools")

		assert.True(t, result["ls"], "ls should be available")
		assert.True(t, result["cat"], "cat should be available")
		assert.False(t, result["nonexistent-tool-abc"], "nonexistent tool should not be available")
		assert.False(t, result["another-missing-tool-xyz"], "another nonexistent tool should not be available")
	})

	t.Run("handles empty tool list", func(t *testing.T) {
		result := CheckAvailability()

		require.NotNil(t, result)
		assert.Empty(t, result, "empty input should return empty map")
	})

	t.Run("handles duplicate tool names", func(t *testing.T) {
		tools := []string{"ls", "ls", "cat", "cat"}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)

		// Should handle duplicates gracefully (exact behavior depends on implementation)
		// At minimum, should include the tools
		assert.Contains(t, result, "ls")
		assert.Contains(t, result, "cat")
	})
}

func TestCheckAvailability_AITools(t *testing.T) {
	t.Run("checks claude availability", func(t *testing.T) {
		result := CheckAvailability("claude")

		require.NotNil(t, result)
		require.Contains(t, result, "claude")

		// Don't assert true/false since it depends on the environment
		// Just verify we get a boolean result
		available := result["claude"]
		assert.IsType(t, false, available, "should return a boolean")
	})

	t.Run("checks codex availability", func(t *testing.T) {
		result := CheckAvailability("codex")

		require.NotNil(t, result)
		require.Contains(t, result, "codex")

		available := result["codex"]
		assert.IsType(t, false, available, "should return a boolean")
	})

	t.Run("checks multiple AI tools", func(t *testing.T) {
		tools := []string{"claude", "codex", "coderabbit"}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)
		assert.Len(t, result, 3, "should return results for all tools")

		for _, tool := range tools {
			require.Contains(t, result, tool)
			assert.IsType(t, false, result[tool], "should return boolean for %s", tool)
		}
	})
}

func TestCheckAvailability_EdgeCases(t *testing.T) {
	t.Run("handles tools with special characters", func(t *testing.T) {
		tools := []string{
			"tool-with-dashes",
			"tool_with_underscores",
			"tool.with.dots",
		}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)

		for _, tool := range tools {
			require.Contains(t, result, tool)
			assert.IsType(t, false, result[tool], "should return boolean for %s", tool)
		}
	})

	t.Run("handles tools with paths", func(t *testing.T) {
		// Some implementations might support checking full paths
		tools := []string{
			"/bin/ls",
			"/usr/bin/cat",
		}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)

		for _, tool := range tools {
			require.Contains(t, result, tool)
			assert.IsType(t, false, result[tool], "should return boolean for %s", tool)
		}
	})

	t.Run("handles empty string tool name", func(t *testing.T) {
		result := CheckAvailability("")

		require.NotNil(t, result)
		// Behavior for empty string depends on implementation
		// Just verify it doesn't panic and returns a map
	})

	t.Run("checks many tools at once", func(t *testing.T) {
		tools := make([]string, 20)
		for i := 0; i < 20; i++ {
			if i%2 == 0 {
				tools[i] = "ls" // exists
			} else {
				tools[i] = "nonexistent-tool-" + string(rune('a'+i))
			}
		}

		result := CheckAvailability(tools...)

		require.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result), 1, "should return results")

		// Verify all tools are in results
		for _, tool := range tools {
			assert.Contains(t, result, tool)
		}
	})
}

func TestCheckAvailability_ReturnType(t *testing.T) {
	t.Run("returns map with string keys and bool values", func(t *testing.T) {
		result := CheckAvailability("ls", "cat")

		require.NotNil(t, result)
		require.IsType(t, map[string]bool{}, result)

		for tool, available := range result {
			assert.IsType(t, "", tool, "keys should be strings")
			assert.IsType(t, false, available, "values should be bools")
		}
	})

	t.Run("returns non-nil map even for empty input", func(t *testing.T) {
		result := CheckAvailability()

		assert.NotNil(t, result, "should return non-nil map")
		assert.IsType(t, map[string]bool{}, result)
	})
}

func TestCheckAvailability_CommonDevelopmentTools(t *testing.T) {
	t.Run("checks git availability", func(t *testing.T) {
		result := CheckAvailability("git")
		require.NotNil(t, result)
		require.Contains(t, result, "git")
		// git is usually available on development machines
	})

	t.Run("checks make availability", func(t *testing.T) {
		result := CheckAvailability("make")
		require.NotNil(t, result)
		require.Contains(t, result, "make")
	})

	t.Run("checks go availability", func(t *testing.T) {
		result := CheckAvailability("go")
		require.NotNil(t, result)
		require.Contains(t, result, "go")
	})

	t.Run("checks multiple development tools", func(t *testing.T) {
		tools := []string{"git", "make", "go", "docker"}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)
		assert.Len(t, result, len(tools))

		for _, tool := range tools {
			require.Contains(t, result, tool)
			assert.IsType(t, false, result[tool])
		}
	})
}

func TestCheckAvailability_CaseInsensitivity(t *testing.T) {
	t.Run("handles tool names with different cases", func(t *testing.T) {
		// Tool names are typically case-sensitive on Unix
		tools := []string{"ls", "LS", "Ls"}
		result := CheckAvailability(tools...)

		require.NotNil(t, result)

		for _, tool := range tools {
			require.Contains(t, result, tool)
			// On Unix, only "ls" (lowercase) should exist
		}
	})
}
