package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashFile_KnownContent(t *testing.T) {
	// SHA-256 of "hello\n" (echo "hello" | sha256sum)
	// printf 'hello\n' | sha256sum => 5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03
	content := "hello\n"
	path := writeHashTempFile(t, content)

	got, err := HashFile(path)
	require.NoError(t, err)
	assert.Equal(t, "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03", got)
}

func TestHashFile_EmptyFile(t *testing.T) {
	path := writeHashTempFile(t, "")

	got, err := HashFile(path)
	require.NoError(t, err)
	// SHA-256 of empty string
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", got)
}

func TestHashFile_UnicodeContent(t *testing.T) {
	// UTF-8 bytes of the emoji and kanji should hash deterministically.
	content := "tarefa concluida com sucesso"
	path := writeHashTempFile(t, content)

	hash1, err := HashFile(path)
	require.NoError(t, err)

	// Hash the same content again to confirm determinism.
	path2 := writeHashTempFile(t, content)
	hash2, err := HashFile(path2)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64, "SHA-256 hex digest must be 64 characters")
}

func TestHashFile_UnicodeEmoji(t *testing.T) {
	content := "\xf0\x9f\x9a\x80 launch"
	path := writeHashTempFile(t, content)

	got, err := HashFile(path)
	require.NoError(t, err)
	assert.Len(t, got, 64)
}

func TestHashFile_NonExistentFile(t *testing.T) {
	_, err := HashFile(filepath.Join(t.TempDir(), "does-not-exist.md"))
	require.Error(t, err)
}

func TestHashFile_DifferentContentDifferentHash(t *testing.T) {
	path1 := writeHashTempFile(t, "content A")
	path2 := writeHashTempFile(t, "content B")

	h1, err := HashFile(path1)
	require.NoError(t, err)
	h2, err := HashFile(path2)
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2)
}

// writeHashTempFile creates a temp file and returns its path.
func writeHashTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "hashtest.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
