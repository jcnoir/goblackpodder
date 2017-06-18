package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"code.cloudfoundry.org/bytefmt"
	"github.com/kennygrant/sanitize"
	"github.com/smira/go-ftp-protocol/protocol"
	"regexp"
)

func downloadFromURL(url string, folder string, maxretry int, httpClient *http.Client, fileName string) (path string, newEpisode bool, err error) {

	for i := 1; i <= maxretry; i++ {
		path, newEpisode, err = download(url, folder, httpClient, fileName)
		if err == nil {
			break
		} else {
			logger.Warning.Println("Download failure at attempt "+strconv.Itoa(i)+"/"+strconv.Itoa(maxretry)+" for url "+url, err)
		}
	}
	return path, newEpisode, err
}

func downloadFromURLWithoutName(url string, folder string, maxretry int, httpClient *http.Client) (path string, newEpisode bool, err error) {
	fileName := extractResourceNameFromURL(url)
	return downloadFromURL(url, folder, maxretry, httpClient, fileName)
}

func extractResourceNameFromURL(uri string) string {
	var urlPath string
	parsedURL, err := url.Parse(uri)
	if err != nil {
		logger.Warning.Println("Cannot extract path from url : "+uri+" : ", err)
		urlPath = uri
	} else {
		urlPath = parsedURL.Path
	}
	tokens := strings.Split(urlPath, "/")
	resource := tokens[len(tokens)-1]
	return resource
}
/**
Replace // with / in urls - except in http(s)://
 */
func cleanUrl(url string) (cleanUrl string){
	re := regexp.MustCompile("([^:])(\\/\\/)")
	cleanUrl = re.ReplaceAllString(url, "$1/")
	return cleanUrl
}

func download(referenceUri string, folder string, httpClient *http.Client, fileName string) (path string, newEpisode bool, err error) {
	fileName = filepath.Join(folder, fileName)
	fileName = sanitize.Path(fileName)
	uri := cleanUrl(referenceUri)
	logger.Debug.Println("Local resource path : " + fileName)
	tmpFilename := fileName + ".part"
	resourceName := filepath.Base(folder) + " - " + filepath.Base(fileName)
	defer removeTempFile(tmpFilename)
	var resp* http.Response


	if !pathExists(fileName) {
		logger.Debug.Println("New resource available : " + resourceName)
		// TODO: check file existence first with io.IsExist
		output, err := os.Create(tmpFilename)
		if err != nil {
			return fileName, newEpisode, err

		}
		defer output.Close()

		if strings.HasPrefix(uri, "ftp") {
			logger.Debug.Println("FTP download detected")
			transport := &http.Transport{}
			transport.RegisterProtocol("ftp", &protocol.FTPRoundTripper{})

			client := &http.Client{Transport: transport}
			response, err := client.Get(uri)
			if err != nil {
				return fileName, newEpisode, err

			}
			resp = response
		} else {

			req, err := http.NewRequest("GET", uri, nil)
			if err != nil {
				return fileName, newEpisode, err

			}
			req.Close = true
			response, err := httpClient.Do(req)
			if err != nil {
				return fileName, newEpisode, err

			}
			resp = response
			defer response.Body.Close()
		}

		n, err := io.Copy(output, resp.Body)
		if err != nil {
			return fileName, newEpisode, err

		}
		logger.Debug.Println("Resource downloaded : " + resourceName + " (" + bytefmt.ByteSize(uint64(n)) + ")")

		os.Rename(tmpFilename, fileName)
		newEpisode = true

	} else {
		logger.Debug.Println("No download since the file exists", fileName)
		newEpisode = false
	}

	return fileName, newEpisode, err
}

func removeTempFile(tmpFilename string) {
	if pathExists(tmpFilename) {
		os.Remove(tmpFilename)
	}
}
