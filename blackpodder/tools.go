package main

import (
	"os"
)

func pathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
