package main

import (
	"path/filepath"
	"strings"

	rss "github.com/jteeuwen/go-pkg-rss"
	"github.com/kennygrant/sanitize"
)

const EPISODE_PREFIX string = "blp-"

type Episode struct {
	feedEpisode *rss.Item
	Podcast     *Podcast
	enclosure   *rss.Enclosure
}

func (e Episode) selectEnclosure() *rss.Enclosure {
	var selectedEnclosure *rss.Enclosure

	if len(e.feedEpisode.Enclosures) > 0 {
		for _, enclosure := range e.feedEpisode.Enclosures {
			if strings.Contains(enclosure.Type, "audio") && (selectedEnclosure == nil || enclosure.Length > selectedEnclosure.Length) {
				selectedEnclosure = enclosure
			}
		}
	}
	return selectedEnclosure
}

func (e Episode) pubDate() string {
	return e.formattedPubDate("060102")
}

func (e Episode) formattedPubDate(format string) string {
	var episodeTimeStr string
	episodeTime, converr := e.feedEpisode.ParsedPubDate()
	if converr != nil {
		episodeTimeStr = e.feedEpisode.PubDate
	} else {
		episodeTimeStr = episodeTime.Format(format)
	}
	return episodeTimeStr
}

func (e Episode) file() string {

	fileNamePrefix := EPISODE_PREFIX + e.pubDate() + "-"
	return filepath.Join(e.Podcast.dir(), sanitize.Path(fileNamePrefix+extractResourceNameFromUrl(e.enclosure.Url)))
}

func (e Episode) String() string {
	return e.Podcast.feedPodcast.Title + " | " + e.feedEpisode.Title
}

func NewEpisode(feedEpisode *rss.Item, Podcast *Podcast) *Episode {
	e := new(Episode)
	e.feedEpisode = feedEpisode
	e.Podcast = Podcast
	e.enclosure = e.selectEnclosure()
	return e
}
