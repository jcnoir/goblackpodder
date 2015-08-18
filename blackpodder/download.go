package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func downloadFromUrl(url string, folder string) (path string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fileName = filepath.Join(folder, fileName)
	fmt.Println("Downloading", url, "to", fileName)
	// TODO: check file existence first with io.IsExist
	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer output.Close()
	response, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer response.Body.Close()
	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(n, "bytes downloaded.")
	return fileName
}
