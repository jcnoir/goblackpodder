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
	tmpFilename := fileName + ".part"
	defer removeTempFile(tmpFilename)

	if ! pathExists(fileName) {
		fmt.Println(url, " --> ", fileName)
		// TODO: check file existence first with io.IsExist
		output, err := os.Create(tmpFilename)
		if err == nil {
			defer output.Close()
			client := &http.Client{}
			req, err := http.NewRequest("GET", url, nil)
			if err == nil {
				req.Header.Add("Accept-Encoding", "identity")
				req.Close = true
				response, err := client.Do(req)
				if err == nil {
					defer response.Body.Close()
					n, err := io.Copy(output, response.Body)
					if err == nil {
						fmt.Println(bytefmt.ByteSize(uint64(n)), " downloaded for "+url)
						os.Rename(tmpFilename, fileName)
					}
				}
			}
		}

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
