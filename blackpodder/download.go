package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kennygrant/sanitize"
	"github.com/pivotal-golang/bytefmt"
)

func downloadFromUrl(url string, folder string, maxretry int, httpClient *http.Client) (path string, err error, newEpisode bool) {

	for i := 1; i <= maxretry; i++ {
		path, err, newEpisode = download(url, folder, httpClient)
		if err == nil {
			break
		} else {
			logger.Warning.Println("Download failure at attempt "+strconv.Itoa(i)+" for url "+url, err)
		}
	}
	return path, err, newEpisode
}

func download(uri string, folder string, httpClient *http.Client) (path string, err error, newEpisode bool) {
	var urlPath string
	parsedUrl, err := url.Parse(uri)
	if err != nil {
		logger.Warning.Println("Cannot extract path from url : "+uri+" : ", err)
		urlPath = uri
	} else {
		urlPath = parsedUrl.Path
	}
	tokens := strings.Split(urlPath, "/")
	fileName := tokens[len(tokens)-1]
	fileName = filepath.Join(folder, fileName)
	fileName = sanitize.Path(fileName)

	tmpFilename := fileName + ".part"
	resourceName := filepath.Base(folder) + " - " + filepath.Base(fileName)
	defer removeTempFile(tmpFilename)

	if !pathExists(fileName) {
		logger.Info.Println("New resource available : " + resourceName)
		// TODO: check file existence first with io.IsExist
		output, err := os.Create(tmpFilename)
		if err != nil {
			return fileName, err, newEpisode

		}
		defer output.Close()
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return fileName, err, newEpisode

		}
		req.Close = true
		response, err := httpClient.Do(req)
		if err != nil {
			return fileName, err, newEpisode

		}
		defer response.Body.Close()
		n, err := io.Copy(output, response.Body)
		if err != nil {
			return fileName, err, newEpisode

		}
		logger.Info.Println("Resource downloaded : " + resourceName + " (" + bytefmt.ByteSize(uint64(n)) + ")")
		os.Rename(tmpFilename, fileName)
		newEpisode = true

	} else {
		logger.Debug.Println("No download since the file exists", fileName)
		newEpisode = false
	}

	return fileName, err, newEpisode
}

func removeTempFile(tmpFilename string) {
	if pathExists(tmpFilename) {
		os.Remove(tmpFilename)
	}
}
