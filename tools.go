package main

import (
	"io"
	"os"
)

func pathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func copyFile(source string, target string) error {
	from, err := os.Open(source)
	if err == nil {
		defer from.Close()
		to, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE, 0666)
		if err == nil {
			defer to.Close()
			_, err = io.Copy(to, from)
		}
	}
	return err
}
