package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"github.com/pivotal-golang/bytefmt" 
)

func downloadFromUrl(url string, folder string) (path string, err error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fileName = filepath.Join(folder, fileName)
	fmt.Println("Downloading", url, "to", fileName)
	// TODO: check file existence first with io.IsExist
	output, err := os.Create(fileName)
	if err != nil {
		return fileName, err
	}
	defer output.Close()
	response, err := http.Get(url)
	if err != nil {
		return fileName, err
	}
	defer response.Body.Close()
	n, err := io.Copy(output, response.Body)
	if err != nil {
		return fileName, err
	}
	fmt.Println(bytefmt.ByteSize(uint64(n)), " downloaded for " +url)
	return fileName, err
}
