// test.go
package main

import (
	"black/go-taglib"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	rss "github.com/jteeuwen/go-pkg-rss"
	viper "github.com/spf13/viper"
)

var (
	logger       Logger
	wg           sync.WaitGroup
	targetFolder string
	feedsPath    string
	maxEpisodes  int
	verbose      bool
)

func main() {
	readConfig()
	logger = NewLogger(verbose)
	err := os.MkdirAll(targetFolder, 0777)
	if err != nil {
		logger.Error.Panic("Cannot create the target folder : "+targetFolder+" : ", err)
	}
	logger.Info.Println("Podcast Update ...")
	feeds, err := parseFeeds(feedsPath)
	if err == nil {
		for _, feed := range feeds {
			downloadFeed(feed)
		}
	} else {
		logger.Error.Println("Cannot parse feed file : ", err)
	}

	wg.Wait()
	logger.Info.Println("Podcast Update Completed")
}

func downloadFeed(url string) {
	wg.Add(1)
	go PollFeed(url, 5, charsetReader, &wg)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {

	logger.Debug.Println(strconv.Itoa(len(newitems)) + " available episodes for " + ch.Title)

	podcastFolder := filepath.Join(targetFolder, ch.Title)
	os.MkdirAll(podcastFolder, 0777)

	wg.Add(1)
	go processImage(ch, podcastFolder)

	episodeCounter := 0

	for _, item := range newitems {
		selectedEnclosure := selectEnclosure(item)
		if selectedEnclosure != nil {
			if len(item.Enclosures) > 0 {
				episodeCounter += 1
				wg.Add(1)
				go process(selectedEnclosure, podcastFolder, item, ch)
				if episodeCounter >= maxEpisodes {
					break
				}
			}
		} else {
			logger.Warning.Println("No audio found for episode " + ch.Title + " - " + item.Title)
		}
	}
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
}

func processImage(ch *rss.Channel, folder string) {
	defer wg.Done()
	logger.Debug.Println("Download image : " + ch.Image.Url)
	if len(ch.Image.Url) > 0 {
		imagepath, err := downloadFromUrl(ch.Image.Url, folder)
		if err == nil {
			convertImage(imagepath, filepath.Join(folder, "folder.jpg"))
		} else {
			logger.Error.Println("Podcast image processing failure", err)
		}
	}
}

func process(selectedEnclosure *rss.Enclosure, folder string, item *rss.Item, channel *rss.Channel) {
	defer wg.Done()
	file, err := downloadFromUrl(selectedEnclosure.Url, folder)
	if err != nil {
		logger.Error.Println("Episode download failure : "+selectedEnclosure.Url, err)
	} else {
		completeTags(file, item, channel)
	}
}

func selectEnclosure(item *rss.Item) *rss.Enclosure {
	var selectedEnclosure *rss.Enclosure

	if len(item.Enclosures) > 0 {
		for _, enclosure := range item.Enclosures {
			if strings.Contains(enclosure.Type, "audio") && (selectedEnclosure == nil || enclosure.Length > selectedEnclosure.Length) {
				selectedEnclosure = enclosure
			}
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
	} else {
		logger.Error.Println("Cannot convert podcast image : "+inputFile, err)
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

func readConfig() {

	user, _ := user.Current()
	configFolder := filepath.Join(user.HomeDir, ".blackpod")

	viper.SetConfigName("config")
	viper.AddConfigPath(configFolder)

	viper.SetDefault("feeds", filepath.Join(configFolder, "feeds.dev"))
	viper.SetDefault("directory", "/tmp/test-podcasts")
	viper.SetDefault("episodes", 1)
	viper.SetDefault("verbose", false)

	err := viper.ReadInConfig()
	if err != nil {
		logger.Error.Println("Fatal error config file: %s \n", err)
	}
	if verbose {
		viper.Debug()
	}
	targetFolder = viper.GetString("directory")
	feedsPath = viper.GetString("feeds")
	maxEpisodes = viper.GetInt("episodes")
	verbose = viper.GetBool("verbose")
}

func completeTags(episodeFile string, episode *rss.Item, podcast *rss.Channel) {
	tag, err := taglib.Read(episodeFile)
	modified := 0
	if err != nil {
		logger.Warning.Println("Cannot complete episode tags for " + podcast.Title + " - " + episode.Title)
		return
	}

	if tag.Artist() == "" {
		logger.Info.Println(podcast.Title + " - " + episode.Title + " : Add missing artist tag --> " + podcast.Title)
		tag.SetArtist(podcast.Title)
		modified += 1
	}
	if tag.Album() == "" {
		logger.Info.Println(podcast.Title + " - " + episode.Title + " : Add missing album tag --> " + podcast.Title)
		tag.SetAlbum(podcast.Title)
		modified += 1
	}
	if tag.Comment() == "" {
		logger.Info.Println(podcast.Title + " - " + episode.Title + " : Add missing comment tag --> " + episode.Description)
		tag.SetComment(episode.Description)
		modified += 1
	}
	if tag.Title() == "" {
		logger.Info.Println(podcast.Title + " - " + episode.Title + " : Add missing title tag --> " + episode.Title)
		tag.SetTitle(episode.Title)
		modified += 1
	}
	if tag.Genre() == "" {
		logger.Info.Println(podcast.Title + " - " + episode.Title + " : Add missing genre tag --> " + "Podcast")
		tag.SetGenre("Podcast")
		modified += 1
	}
	if tag.Year() == 0 {
		pubdate, err := episode.ParsedPubDate()
		if err == nil {
			logger.Info.Println(podcast.Title + " - " + episode.Title + " : Add missing year tag --> " + strconv.Itoa(pubdate.Year()))
			tag.SetYear(pubdate.Year())
			modified += 1
		}
	}
	if modified > 0 {
		if !tag.Save() {
			logger.Warning.Println(podcast.Title + " - " + episode.Title + " : Cannot save the modified tags")
		}
		defer tag.Close()
	}

}
