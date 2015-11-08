package main

import (
	"strconv"

	"github.com/jaytaylor/html2text"
	rss "github.com/jteeuwen/go-pkg-rss"
	"github.com/wtolson/go-taglib"
)

func completeTags(episodeFile string, episode *rss.Item, podcast *rss.Channel) {

	logger.Info.Println("Tag update : " + podcast.Title + " - " + episode.Title)

	tag, err := taglib.Read(episodeFile)
	if err != nil {
		logger.Warning.Println("Cannot complete episode tags for "+podcast.Title+" - "+episode.Title, err)
		return
	}
	defer tag.Close()

	completeTag(taglib.Artist, podcast.Title, tag)
	completeTag(taglib.Album, podcast.Title, tag)

	plaintextDescription, err := html2text.FromString(episode.Description)
	if err == nil {
		episode.Description = plaintextDescription
	}
	if len(episode.Description) > maxCommentSize+5 {
		episode.Description = episode.Description[:maxCommentSize] + " ..."
	}

	completeTag(taglib.Comments, episode.Description, tag)
	completeTag(taglib.Title, episode.Title, tag)
	completeTag(taglib.Genre, "Podcast", tag)

	pubdate, err := episode.ParsedPubDate()
	if err == nil {
		completeTag(taglib.Year, strconv.Itoa(pubdate.Year()), tag)
	}

	if err := tag.Save(); err != nil {
		logger.Warning.Println(podcast.Title+" - "+episode.Title+" : Cannot save the modified tags", err)
	}
}
func completeTag(tagname taglib.TagName, tagvalue string, tag *taglib.File) {
	logger.Info.Println(tagname.String() + " --> " + tagvalue)
	tag.SetTag(tagname, tagvalue)
}
