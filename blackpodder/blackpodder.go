// test.go
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
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
	feedWg           sync.WaitGroup
	targetFolder     string
	feedsPath        string
	maxEpisodes      int
	keptEpisodes     int
	verbose          bool
	maxFeedRunner    int
	maxImageRunner   int
	maxEpisodeRunner int
	maxRetryDownload int
	episodeTasks     chan *Episode
	feedTasks        chan string
	newEpisodes      chan string
	rootCmd          *cobra.Command
	httpClient       *http.Client
	maxCommentSize   int
	retagExisting    bool
	dateFormat       string
)

type episodeTask struct {
	selectedEnclosure *rss.Enclosure
	folder            string
	item              *rss.Item
	channel           *rss.Channel
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
	dateFormat = viper.GetString("dateFormat")
	keptEpisodes = int(math.Max(float64(viper.GetInt("keptEpisodes")), float64(maxEpisodes)))

	logger = NewLogger(verbose)
	logger.Info.Println("Podcast Update")

	if verbose {
		viper.Debug()
		rootCmd.DebugFlags()
	}

	err := os.MkdirAll(targetFolder, 0777)
	if err != nil {
		logger.Error.Panic("Cannot create the target folder : "+targetFolder+" : ", err)
	}

	episodeTasks = make(chan *Episode)
	feedTasks = make(chan string)
	newEpisodes = make(chan string, 1000)

	httpClient = &http.Client{}

	for i := 0; i < maxFeedRunner; i++ {
		feedWg.Add(1)
		go func() {
			defer feedWg.Done()
			for f := range feedTasks {
				downloadFeed(f)
			}
		}()
	}

	for i := 0; i < maxEpisodeRunner; i++ {
		go func() {
			for episodeTask := range episodeTasks {
				process(episodeTask)
			}
		}()
	}

	feeds, err := parseFeeds(feedsPath)
	logger.Debug.Println("Feeds : ", feeds)
	if err == nil {
		for _, feed := range feeds {
			feedTasks <- feed
		}
	} else {
		logger.Error.Println("Cannot parse feed file : ", err)
	}
	close(feedTasks)
	logger.Debug.Println("Wait for all feeds to be processed ...")
	feedWg.Wait()
	close(episodeTasks)
	close(newEpisodes)
	processNewEpisodes()
	logger.Info.Println("Podcasts Updated")
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
			if pathExists(newEpisode) {
				logger.Debug.Println("new episode added to playlist", newEpisode)
				io.WriteString(file, newEpisode+"\n")
			} else {
				logger.Error.Println("Non existing new episode path : " + newEpisode)
			}
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
	logger.Debug.Println("Channel : ", ch)
	if ch.Title == "" {
		if ch.Author.Name != "" {
			ch.Title = ch.Author.Name
		} else if ch.Description != "" {
			ch.Title = ch.Description
		} else {
			ch.Title = extractResourceNameFromURL(feed.Url)
		}

		logger.Warning.Println("Missing podcast title in the feed, this replacement will be used : " + ch.Title)

	}

	podcast := NewPodcast(targetFolder, ch)
	podcast.fetchNewEpisodes(newitems)
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
}

func parseFeeds(filePath string) ([]string, error) {
	var lines []string
	var filteredLines []string
	content, err := ioutil.ReadFile(filePath)
	if err == nil {
		lines = strings.Split(string(content), "\n")
		lines = lines[:len(lines)-1]

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "#") {
				filteredLines = append(filteredLines, line)
			}
		}
		logger.Info.Println(strconv.Itoa(len(filteredLines)) + " Podcasts found in the configuration")
	}
	return filteredLines, err

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
	addProperty("dateFormat", "m", "020106", "Date format to be used in tags based on this reference date : Mon Jan _2 15:04:05 2006")
	addProperty("keptEpisodes", "n", 3, "Number of episodes to keep (0 or -1 means no old episode remval)")

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
