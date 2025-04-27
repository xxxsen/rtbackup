package utils

import (
	"os"
	"path"
)

func IsFileExists(dir string, ps []string) bool {
	for _, p := range ps {
		if _, err := os.Stat(path.Join(dir, p)); err == nil {
			return true
		}
	}
	return false
}
