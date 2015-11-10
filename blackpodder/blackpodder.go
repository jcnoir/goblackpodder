// test.go
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	rss "github.com/jteeuwen/go-pkg-rss"
	cobra "github.com/spf13/cobra"
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
	feedTasks        chan string
	newEpisodes      chan string
	rootCmd          *cobra.Command
	httpClient       *http.Client
	maxCommentSize   int
	retagExisting    bool
)

type episodeTask struct {
	selectedEnclosure *rss.Enclosure
	folder            string
	item              *rss.Item
	channel           *rss.Channel
}

type Episode struct {
	feedEpisode *rss.Item
	Podcast     Podcast
}

func fetchPodcasts() {

	targetFolder = viper.GetString("directory")
	feedsPath = viper.GetString("feeds")
	maxEpisodes = viper.GetInt("episodes")
	verbose = viper.GetBool("verbose")
	maxEpisodeRunner = viper.GetInt("maxEpisodeRunner")
	maxFeedRunner = viper.GetInt("maxFeedRunner")
	maxImageRunner = viper.GetInt("maxImageRunner")
	maxRetryDownload = viper.GetInt("maxRetryDownload")
	maxCommentSize = viper.GetInt("maxCommentSize")
	retagExisting = viper.GetBool("retagExisting")

	logger = NewLogger(verbose)
	logger.Info.Println("Podcast Update ...")

	if verbose {
		viper.Debug()
		rootCmd.DebugFlags()
	}

	err := os.MkdirAll(targetFolder, 0777)
	if err != nil {
		logger.Error.Panic("Cannot create the target folder : "+targetFolder+" : ", err)
	}

	episodeTasks = make(chan episodeTask)
	feedTasks = make(chan string)
	newEpisodes = make(chan string, 1000)

	httpClient = &http.Client{}

	for i := 0; i < maxFeedRunner; i++ {
		feedWg.Add(1)
		go func() {
			for f := range feedTasks {
				downloadFeed(f)
			}
			feedWg.Done()
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
	close(episodeTasks)
	wg.Wait()
	close(newEpisodes)
	processNewEpisodes()
	logger.Info.Println("Podcast Update Completed")

}

func processNewEpisodes() {

	if len(newEpisodes) > 0 {

		filename := filepath.Join(targetFolder, "last-episodes.m3u")
		file, err := os.Create(filename)
		if err != nil {
			logger.Error.Println("Cannot write the new episode file ", err)
			return
		}
		defer file.Close()

		logger.Debug.Println("Write the new episode file ", filename)

		for newEpisode := range newEpisodes {
			logger.Debug.Println("new episode added to playlist", newEpisode)
			io.WriteString(file, newEpisode+"\n")
		}
		logger.Debug.Println("Last episode playlist written")
	}
}

func main() {

	rootCmd = &cobra.Command{
		Use:   "blackpodder",
		Short: "Blackpodder is a podcast fetcher",
		Long:  `Blackpodder is a podcast fetcher written in GO`,
		Run: func(cmd *cobra.Command, args []string) {
			fetchPodcasts()
		},
	}
	readConfig()
	rootCmd.Execute()

}

func downloadFeed(url string) {
	logger.Debug.Println("Downloading feed ", url)
	PollFeed(url, 5, charsetReader)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {

	logger.Debug.Println(strconv.Itoa(len(newitems)) + " available episodes for " + ch.Title)

	podcast := Podcast{targetFolder, ch}
	podcast.mkdir()
	podcast.downloadImage()

	episodeCounter := 0

	for _, item := range newitems {
		selectedEnclosure := selectEnclosure(item)
		if selectedEnclosure != nil {
			if len(item.Enclosures) > 0 {
				episodeCounter += 1
				episodeTasks <- episodeTask{selectedEnclosure, podcast.dir(), item, ch}
				if episodeCounter >= maxEpisodes {
					break
				}
			}
		} else {
			logger.Debug.Println("No audio found for episode " + ch.Title + " - " + item.Title)
		}
	}
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
}

func process(selectedEnclosure *rss.Enclosure, folder string, item *rss.Item, channel *rss.Channel) {
	logger.Debug.Println("Downloading episode ", selectedEnclosure.Url)

	episodeTime, converr := item.ParsedPubDate()
	var episodeTimeStr string
	if converr != nil {
		episodeTimeStr = item.PubDate
	} else {
		episodeTimeStr = episodeTime.Format("060102")
	}
	fileName := "BLP_" + episodeTimeStr + "_"

	file, err, newEpisode := downloadFromUrl(selectedEnclosure.Url, folder, maxRetryDownload, httpClient, fileName)

	if err != nil {
		logger.Error.Println("Episode download failure : "+selectedEnclosure.Url, err)
	} else {

		if newEpisode || retagExisting {
			completeTags(file, item, channel)
		}
		if newEpisode {
			newEpisodes <- file
		}
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

	addProperty("feeds", "f", filepath.Join(configFolder, "feeds.dev"), "Feed file path")
	addProperty("directory", "d", "/tmp/test-podcasts", "Podcast folder path")
	addProperty("episodes", "e", 3, "Max episodes to download")
	addProperty("verbose", "v", false, "Enable verbose mode")
	addProperty("maxFeedRunner", "g", 5, "Max runners to fetch feeds")
	addProperty("maxImageRunner", "i", 3, "Max runners to fetch images")
	addProperty("maxEpisodeRunner", "j", 10, "Max runners to fetch episodes")
	addProperty("maxRetryDownload", "k", 3, "Max http retries")
	addProperty("maxCommentSize", "l", 500, "Max comment length")
	addProperty("retagExisting", "r", false, "Retag existing episodes")

	err := viper.ReadInConfig()
	if err != nil {
		logger.Error.Println("Fatal error config file: %s \n", err)
	}

}
func addProperty(name string, short string, defaultValue interface{}, description string) {

	if typeValue, ok := defaultValue.(int); ok {
		rootCmd.Flags().IntP(name, short, typeValue, description)
	} else if typeValue, ok := defaultValue.(string); ok {
		rootCmd.Flags().StringP(name, short, typeValue, description)
	} else if typeValue, ok := defaultValue.(bool); ok {
		rootCmd.Flags().BoolP(name, short, typeValue, description)
	} else {
		fmt.Println("Unknwown Property type will be ignored ", name)
		return
	}
	viper.SetDefault(name, defaultValue)
	viper.BindPFlag(name, rootCmd.Flags().Lookup(name))

}
