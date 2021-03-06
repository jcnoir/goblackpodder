package main

import (
	"image"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	rss "github.com/jteeuwen/go-pkg-rss"
	"github.com/kennygrant/sanitize"
)

// Podcast is a poscast
type Podcast struct {
	baseFolder  string
	feedPodcast *rss.Channel
	wg          *sync.WaitGroup
}

func (podcast Podcast) dir() (path string) {
	podcastFolder := filepath.Join(podcast.baseFolder, podcast.feedPodcast.Title)
	podcastFolder = sanitize.Path(podcastFolder)
	return podcastFolder
}

func (podcast Podcast) mkdir() error {
	return os.MkdirAll(podcast.dir(), 0777)
}

func (podcast Podcast) image() string {
	imageName := extractResourceNameFromURL(podcast.feedPodcast.Image.Url)
	imageName = sanitize.Path(imageName)

	return filepath.Join(podcast.dir(), imageName)
}

func (podcast Podcast) convertedImage() string {
	return filepath.Join(podcast.dir(), "folder.jpg")
}

func (podcast Podcast) downloadImage() {
	var err error
	if len(podcast.feedPodcast.Image.Url) > 0 {
		if !pathExists(podcast.image()) {
			logger.Info.Println("Cover available for podcast : " + podcast.feedPodcast.Title)
			logger.Debug.Println("Downloading image : " + podcast.feedPodcast.Image.Url)
			_, _, err := downloadFromURL(podcast.feedPodcast.Image.Url, podcast.dir(), maxRetryDownload, httpClient, filepath.Base(podcast.image()))
			if err == nil {
				err = podcast.convertImage()
				if err != nil {
					logger.Error.Println("Podcast image conversion failure", err)
				}
			} else {
				logger.Error.Println("Podcast image processing failure", err)
			}
		}
	}

	if !pathExists(podcast.convertedImage()) {
		logger.Warning.Println("Podcast image has not been retrieved properly, using default image.")
		err = useDefaultImage(podcast)
		if err != nil {
			logger.Error.Println("Default podcast image has not been retrieved properly.", err)
		}
	}

}

func useDefaultImage(podcast Podcast) error {
	return copyFile(filepath.Join(podcast.baseFolder, "folder.jpg"), podcast.convertedImage())
}

func (podcast Podcast) convertImage() error {
	var err error
	var inputImage image.Image

	if !pathExists(podcast.convertedImage()) {
		inputImage, err = ImageRead(podcast.image())
		if err == nil {
			err = Formatjpg(inputImage, podcast.convertedImage())
		}
	}
	return err
}

//NewPodcast makes a new podcast
func NewPodcast(baseFolder string, feedPodcast *rss.Channel) *Podcast {
	var wg sync.WaitGroup
	p := new(Podcast)
	p.baseFolder = baseFolder
	p.feedPodcast = feedPodcast
	p.wg = &wg
	return p
}

func (podcast Podcast) fetchNewEpisodes(newitems []*rss.Item) {
	podcast.mkdir()
	podcast.downloadImage()

	episodeCounter := 0

	for _, item := range newitems {
		episode := NewEpisode(item, &podcast)
		selectedEnclosure := episode.enclosure
		if selectedEnclosure != nil {
			if len(episode.feedEpisode.Enclosures) > 0 {
				episodeCounter++
				podcast.wg.Add(1)
				episodeTasks <- episode
				if episodeCounter >= maxEpisodes {
					break
				}
			}
		} else {
			logger.Debug.Println("No audio found for episode " + podcast.feedPodcast.Title + " - " + item.Title)
		}
	}
	logger.Debug.Println("Wait for all episodes to be processed : " + podcast.feedPodcast.Title)
	podcast.wg.Wait()
	podcast.removeOldEpisodes()
}

func process(episode *Episode) {
	defer episode.Podcast.wg.Done()
	selectedEnclosure := episode.enclosure
	if !pathExists(episode.file()) {
		logger.Info.Println("New episode available : " + episode.Podcast.feedPodcast.Title + " | " + episode.feedEpisode.Title)
		file, newEpisode, err := downloadFromURL(selectedEnclosure.Url, episode.Podcast.dir(), maxRetryDownload, httpClient, filepath.Base(episode.file()))
		if err != nil {
			logger.Error.Println("Episode download failure : "+selectedEnclosure.Url, err)
		} else {
			if newEpisode {
				logger.Info.Println("New episode downloaded : " + episode.Podcast.feedPodcast.Title + " | " + episode.feedEpisode.Title)
				if strings.Contains(episode.enclosure.Type, "ogg") {
					logger.Warning.Println("Fixing tag has been disabled for ogg (file corruption)")

				} else {
					completeTags(episode)
				}
				newEpisodes <- file
			}
		}
	}
	if retagExisting {
		completeTags(episode)
	}
}

//removeOldEpisodes remove old podcast epipsode files
func (podcast Podcast) removeOldEpisodes() {
	if keptEpisodes > 0 {
		var episodeFiles []os.FileInfo
		files, _ := ioutil.ReadDir(podcast.dir())
		for _, f := range files {
			if strings.HasPrefix(f.Name(), EpisodePrefix) {
				episodeFiles = append(episodeFiles, f)
			}
		}
		sort.Sort(ByModDate(episodeFiles))
		for i, f := range episodeFiles {
			if i >= keptEpisodes {
				filePath := filepath.Join(podcast.dir(), f.Name())
				logger.Info.Println("Remove old episode : " + filePath + " (Keep only " + strconv.Itoa(keptEpisodes) + " episodes)")
				os.Remove(filePath)
			}
		}
	}
}

//ByModDate sorts by modification date
type ByModDate []os.FileInfo

func (a ByModDate) Len() int           { return len(a) }
func (a ByModDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByModDate) Less(i, j int) bool { return a[i].ModTime().Unix() > a[j].ModTime().Unix() }
