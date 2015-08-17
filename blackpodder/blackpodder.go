// test.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	rss "black/go-pkg-rss-custom"
)

var (
	logger       Logger
	wg           sync.WaitGroup
	targetFolder string
)

func main() {
	logger = NewLogger()
	targetFolder = "/tmp/test-podcasts"
	os.MkdirAll(targetFolder, 0777)
	logger.Info.Println("Blackpodder starts")
	feed("http://lhspodcast.info/category/podcast-ogg/feed")
	feed("http://feed.ubuntupodcast.org/ogg")
	feed("http://feeds.feedburner.com/systemau-ogg")
	feed("http://hanselminutes.com/subscribearchives")
	feed("http://feeds.feedburner.com/PuppetLabsPodcast")

	logger.Debug.Println("Before wait")
	wg.Wait()
	logger.Debug.Println("After wait")
	logger.Info.Println("Blackpodder stops")
}

func feed(url string) {
	wg.Add(1)
	go PollFeed(url, 5, charsetReader, &wg)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {

	fmt.Printf("%d new item(s) in %s\n", len(newitems), feed.Url)
	var slice []*rss.Item = newitems[0:1]
	for i, item := range slice {
		fmt.Println(strconv.Itoa(i)+":Title :", item.Title)
		fmt.Println(", Date :", item.PubDate)
		pubtime, _ := parseTime(item.PubDate)
		fmt.Println(", Date :", pubtime)
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
	logger.Debug.Println("Download image : " + ch.Image.Url)
	if len(ch.Image.Url) > 0 {
		imagepath := downloadFromUrl(ch.Image.Url, targetFolder)
		convertImage(imagepath, filepath.Join(targetFolder, "folder.jpg"))
	}
	defer wg.Done()

}

func process(selectedEnclosure *rss.Enclosure) {
	downloadFromUrl(selectedEnclosure.Url, targetFolder)
	logger.Info.Println("selected enclosure : " + selectedEnclosure.Url)
	defer wg.Done()
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
		panic(err)
	}
	err = Formatjpg(inputImage, outputFile)
	if err != nil {
		panic(err)
	}
}
