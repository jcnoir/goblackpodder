// test.go
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	rss "github.com/jteeuwen/go-pkg-rss"
)

var (
	logger       Logger
	wg           sync.WaitGroup
	targetFolder string
	feedsPath    string
	maxEpisodes  int
)

func main() {
	logger = NewLogger()
	targetFolder = "/tmp/test-podcasts"
	maxEpisodes = 10
	user, _ := user.Current()
	feedsPath = filepath.Join(user.HomeDir, ".blackpod", "feeds.dev")
	os.MkdirAll(targetFolder, 0777)
	logger.Info.Println("Blackpodder starts")
	feeds, err := parseFeeds(feedsPath)
	if err == nil {
		for _, feed := range feeds {
			downloadFeed(feed)
		}
	}

	wg.Wait()
	logger.Info.Println("Blackpodder stops")
}

func downloadFeed(url string) {
	wg.Add(1)
	go PollFeed(url, 5, charsetReader, &wg)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {

	fmt.Printf("%d new item(s) in %s\n", len(newitems), feed.Url)

	podcastFolder := filepath.Join(targetFolder, ch.Title)
	os.MkdirAll(podcastFolder, 0777)

	wg.Add(1)
	go processImage(ch, podcastFolder)

	var slice []*rss.Item = newitems[0:maxEpisodes]
	for i, item := range slice {
		fmt.Println(strconv.Itoa(i)+":Title :", item.Title)
		fmt.Println("Found enclosures : ", len(item.Enclosures))
		if len(item.Enclosures) > 0 {
			selectedEnclosure := selectEnclosure(item)
			wg.Add(1)
			go process(selectedEnclosure, podcastFolder)
		}
	}
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
	fmt.Printf("%d new channel(s) in %s\n", len(newchannels), feed.Url)
}

func processImage(ch *rss.Channel, folder string) {
	defer wg.Done()
	logger.Debug.Println("Download image : " + ch.Image.Url)
	if len(ch.Image.Url) > 0 {
		imagepath, err := downloadFromUrl(ch.Image.Url, folder)
		if err == nil {
			convertImage(imagepath, filepath.Join(folder, "folder.jpg"))
		} else {
			logger.Error.Println("Podacast image processing failure", err)
		}
	}
}

func process(selectedEnclosure *rss.Enclosure, folder string) {
	defer wg.Done()
	filepath, err := downloadFromUrl(selectedEnclosure.Url, folder)
	if err == nil {
		logger.Info.Println("Episode downloaded", filepath)
	} else {
		logger.Error.Println("Episode download failure : "+selectedEnclosure.Url, err)
	}
}

func selectEnclosure(item *rss.Item) *rss.Enclosure {
	selectedEnclosure := item.Enclosures[0]
	for _, enclosure := range item.Enclosures {
		if enclosure.Length > selectedEnclosure.Length {
			selectedEnclosure = enclosure
		}
	}
	return selectedEnclosure
}

func convertImage(inputFile string, outputFile string) error {
	inputImage, err := ImageRead(inputFile)
	if err == nil {
		if !pathExists(outputFile) {
			err = Formatjpg(inputImage, outputFile)
		} else {
			logger.Debug.Println("Skipping the image conversion since it already exists", outputFile)
		}
	}
	return err

}

func parseFeeds(filePath string) ([]string, error) {
	var lines []string
	content, err := ioutil.ReadFile(filePath)
	if err == nil {
		lines = strings.Split(string(content), "\n")
	}
	return lines, err

}
