package main

import (
	"image"
	"os"
	"path/filepath"

	rss "github.com/jteeuwen/go-pkg-rss"
	"github.com/kennygrant/sanitize"
)

type Podcast struct {
	baseFolder  string
	feedPodcast *rss.Channel
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
	imageName := extractResourceNameFromUrl(podcast.feedPodcast.Image.Url)
	imageName = sanitize.Path(imageName)

	return filepath.Join(podcast.dir(), imageName)
}

func (podcast Podcast) convertedImage() string {
	return filepath.Join(podcast.dir(), "folder.jpg")
}

func (podcast Podcast) downloadImage() {
	if len(podcast.feedPodcast.Image.Url) > 0 {
		if !pathExists(podcast.image()) {
			logger.Info.Println("Cover available for podcast : " + podcast.feedPodcast.Title)
			logger.Debug.Println("Downloading image : " + podcast.feedPodcast.Image.Url)
			_, err, _ := downloadFromUrl(podcast.feedPodcast.Image.Url, podcast.dir(), maxRetryDownload, httpClient, filepath.Base(podcast.image()))
			if err == nil {
				podcast.convertImage()
			} else {
				logger.Error.Println("Podcast image processing failure", err)
			}
		}
	}

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
