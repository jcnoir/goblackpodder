package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pivotal-golang/bytefmt"
)

func downloadFromUrl(url string, folder string, maxretry int) (path string, err error) {

	for i := 1; i <= maxretry; i++ {
		path, err = download(url, folder)
		if err == nil {
			break
		} else {
			logger.Warning.Println("Download failure at attempt "+strconv.Itoa(i)+" for url "+url, err)
		}
	}
	return path, err
}

func download(url string, folder string) (path string, err error) {
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
		logger.Info.Println("Resource downloaded : " + resourceName + " (" + bytefmt.ByteSize(uint64(n)) + ")")
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
