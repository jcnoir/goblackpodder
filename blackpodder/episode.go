package main

import (
	"path/filepath"
	"strings"

	rss "github.com/jteeuwen/go-pkg-rss"
	"github.com/kennygrant/sanitize"
)

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
	var episodeTimeStr string
	episodeTime, converr := e.feedEpisode.ParsedPubDate()
	if converr != nil {
		episodeTimeStr = e.feedEpisode.PubDate
	} else {
		episodeTimeStr = episodeTime.Format("060102")
	}
	return episodeTimeStr
}

func (e Episode) file() string {

	fileNamePrefix := "BLP_" + e.pubDate() + "_"
	return filepath.Join(e.Podcast.dir(), sanitize.Path(fileNamePrefix+extractResourceNameFromUrl(e.enclosure.Url)))
}

func NewEpisode(feedEpisode *rss.Item, Podcast *Podcast) *Episode {
	e := new(Episode)
	e.feedEpisode = feedEpisode
	e.Podcast = Podcast
	e.enclosure = e.selectEnclosure()
	return e
}
