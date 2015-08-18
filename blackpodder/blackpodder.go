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
)

func main() {
	logger = NewLogger()
	targetFolder = "/tmp/test-podcasts"
	user, _ := user.Current()
	feedsPath = filepath.Join(user.HomeDir, ".blackpod", "feeds.dev")
	os.MkdirAll(targetFolder, 0777)
	logger.Info.Println("Blackpodder starts")
	feeds := parseFeeds(feedsPath)
	for _, feed := range feeds {
		downloadFeed(feed)
	}

	logger.Debug.Println("Before wait")
	wg.Wait()
	logger.Debug.Println("After wait")
	logger.Info.Println("Blackpodder stops")
}

func downloadFeed(url string) {
	wg.Add(1)
	go PollFeed(url, 5, charsetReader, &wg)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {

	fmt.Printf("%d new item(s) in %s\n", len(newitems), feed.Url)
	var slice []*rss.Item = newitems[0:5]
	for i, item := range slice {
		fmt.Println(strconv.Itoa(i)+":Title :", item.Title)
		fmt.Println(", Date :", item.PubDate)
		pubtime, _ := parseTime(item.PubDate)
		fmt.Println(", Date :", pubtime)
		fmt.Println("Found enclosures : ", len(item.Enclosures))
		if len(item.Enclosures) > 0 {
			selectedEnclosure := selectEnclosure(item)
			wg.Add(1)
			go process(selectedEnclosure)

		}

	}
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
	fmt.Printf("%d new channel(s) in %s\n", len(newchannels), feed.Url)
	wg.Add(1)
	go processImage(newchannels[0])
}

func processImage(ch *rss.Channel) {
	defer wg.Done()
	logger.Debug.Println("Download image : " + ch.Image.Url)
	if len(ch.Image.Url) > 0 {
		imagepath := downloadFromUrl(ch.Image.Url, targetFolder)

		//		convertImage(imagepath, filepath.Join(targetFolder, "folder.jpg"))
		convertImage(imagepath, filepath.Join(targetFolder, filepath.Base(imagepath)+".jpg"))

	}

}

func process(selectedEnclosure *rss.Enclosure) {
	defer wg.Done()
	downloadFromUrl(selectedEnclosure.Url, targetFolder)
	logger.Info.Println("selected enclosure : " + selectedEnclosure.Url)
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

func convertImage(inputFile string, outputFile string) {
	inputImage, err := ImageRead(inputFile)
	if err != nil {
		fmt.Println(err)
	}
	err = Formatjpg(inputImage, outputFile)
	if err != nil {
		fmt.Println(err)
	}
}

func parseFeeds(filePath string) []string {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
	}
	lines := strings.Split(string(content), "\n")
	return lines
}
