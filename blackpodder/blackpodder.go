// test.go
package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/wtolson/go-taglib"

	rss "github.com/jteeuwen/go-pkg-rss"
	viper "github.com/spf13/viper"
)

var (
	logger           Logger
	wg               sync.WaitGroup
	feedWg           sync.WaitGroup
	targetFolder     string
	feedsPath        string
	maxEpisodes      int
	verbose          bool
	maxFeedRunner    int
	maxImageRunner   int
	maxEpisodeRunner int
	maxRetryDownload int
	episodeTasks     chan episodeTask
	imageTasks       chan imageTask
	feedTasks        chan string
)

type episodeTask struct {
	selectedEnclosure *rss.Enclosure
	folder            string
	item              *rss.Item
	channel           *rss.Channel
}

type imageTask struct {
	ch     *rss.Channel
	folder string
}

func main() {
	readConfig()
	logger = NewLogger(verbose)
	err := os.MkdirAll(targetFolder, 0777)
	if err != nil {
		logger.Error.Panic("Cannot create the target folder : "+targetFolder+" : ", err)
	}
	logger.Info.Println("Podcast Update ...")

	episodeTasks = make(chan episodeTask)
	imageTasks = make(chan imageTask)
	feedTasks = make(chan string)

	for i := 0; i < maxFeedRunner; i++ {
		feedWg.Add(1)
		go func() {
			for f := range feedTasks {
				downloadFeed(f)
			}
			feedWg.Done()
		}()
	}

	for i := 0; i < maxImageRunner; i++ {
		wg.Add(1)
		go func() {
			for imageTask := range imageTasks {
				processImage(imageTask.ch, imageTask.folder)
			}
			wg.Done()
		}()
	}

	// spawn four worker goroutines
	for i := 0; i < maxEpisodeRunner; i++ {
		wg.Add(1)
		go func() {
			for episodeTask := range episodeTasks {
				process(episodeTask.selectedEnclosure, episodeTask.folder, episodeTask.item, episodeTask.channel)
			}
			wg.Done()
		}()
	}

	feeds, err := parseFeeds(feedsPath)
	if err == nil {
		for _, feed := range feeds {
			feedTasks <- feed
		}
	} else {
		logger.Error.Println("Cannot parse feed file : ", err)
	}
	close(feedTasks)
	feedWg.Wait()
	close(imageTasks)
	close(episodeTasks)
	wg.Wait()
	logger.Info.Println("Podcast Update Completed")
}

func downloadFeed(url string) {
	PollFeed(url, 5, charsetReader)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {

	logger.Debug.Println(strconv.Itoa(len(newitems)) + " available episodes for " + ch.Title)

	podcastFolder := filepath.Join(targetFolder, ch.Title)
	os.MkdirAll(podcastFolder, 0777)

	imageTasks <- imageTask{ch, podcastFolder}

	// generate some tasks
	episodeCounter := 0

	for _, item := range newitems {
		selectedEnclosure := selectEnclosure(item)
		if selectedEnclosure != nil {
			if len(item.Enclosures) > 0 {
				episodeCounter += 1
				episodeTasks <- episodeTask{selectedEnclosure, podcastFolder, item, ch}
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
	logger.Debug.Println("Download image : " + ch.Image.Url)
	if len(ch.Image.Url) > 0 {
		imagepath, err := downloadFromUrl(ch.Image.Url, folder, maxRetryDownload)
		if err == nil {
			convertImage(imagepath, filepath.Join(folder, "folder.jpg"))
		} else {
			logger.Error.Println("Podcast image processing failure", err)
		}
	}
}

func process(selectedEnclosure *rss.Enclosure, folder string, item *rss.Item, channel *rss.Channel) {
	file, err := downloadFromUrl(selectedEnclosure.Url, folder, maxRetryDownload)
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
		lines = lines[:len(lines)-1]
		logger.Info.Println(strconv.Itoa(len(lines)) + " Podcasts found in the configuration")
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
	viper.SetDefault("maxFeedRunner", 5)
	viper.SetDefault("maxImageRunner", 2)
	viper.SetDefault("maxEpisodeRunner", 10)
	viper.SetDefault("maxRetryDownload", 3)

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
	maxEpisodeRunner = viper.GetInt("maxEpisodeRunner")
	maxFeedRunner = viper.GetInt("maxFeedRunner")
	maxImageRunner = viper.GetInt("maxImageRunner")
	maxRetryDownload = viper.GetInt("maxRetryDownload")

}

func completeTags(episodeFile string, episode *rss.Item, podcast *rss.Channel) {
	tag, err := taglib.Read(episodeFile)
	modified := 0
	if err != nil {
		logger.Warning.Println("Cannot complete episode tags for " + podcast.Title + " - " + episode.Title, err)
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
		if len(episode.Description) > 500 {
			episode.Description = episode.Description[:500] + " ..."
		}
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
		if err := tag.Save(); err != nil {
			logger.Warning.Println(podcast.Title + " - " + episode.Title + " : Cannot save the modified tags", err)
		}
		defer tag.Close()
	}

}
