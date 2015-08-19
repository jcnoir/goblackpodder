package main

import (
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
	tmpFilename := fileName + ".part"
	resourceName := filepath.Base(folder) + " - " + filepath.Base(fileName)
	defer removeTempFile(tmpFilename)

	if !pathExists(fileName) {
		logger.Info.Println("New resource available : " + resourceName)
		// TODO: check file existence first with io.IsExist
		output, err := os.Create(tmpFilename)
		if err != nil {
			return fileName, err

		}
		defer output.Close()
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fileName, err

		}
		req.Header.Add("Accept-Encoding", "identity")
		req.Close = true
		response, err := client.Do(req)
		if err != nil {
			return fileName, err

		}
		defer response.Body.Close()
		n, err := io.Copy(output, response.Body)
		if err != nil {
			return fileName, err

		}
		logger.Info.Println("Resource downloaded : " + resourceName + "(" + bytefmt.ByteSize(uint64(n)) + ")")
		os.Rename(tmpFilename, fileName)

	} else {
		logger.Debug.Println("No download since the file exists", fileName)
	}

	return fileName, err
}

func removeTempFile(tmpFilename string) {
	if pathExists(tmpFilename) {
		os.Remove(tmpFilename)
	}
}
