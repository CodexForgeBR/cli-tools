package tasks

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// HashFile returns the lowercase hexadecimal SHA-256 digest of the entire
// contents of filePath.
func HashFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
