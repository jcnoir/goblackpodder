package main

import (
	"strconv"

	"github.com/jaytaylor/html2text"
	"github.com/wtolson/go-taglib"
)

func completeTags(episode *Episode) {

	logger.Info.Println("Tag update : " + episode.Podcast.feedPodcast.Title + " - " + episode.feedEpisode.Title + " : " + episode.file())

	tag, err := taglib.Read(episode.file())
	if err != nil {
		logger.Warning.Println("Cannot complete episode tags for "+episode.Podcast.feedPodcast.Title+" - "+episode.feedEpisode.Title, err)
		return
	}
	defer tag.Close()

	var replaceArtist string
	if episode.Podcast.feedPodcast.Author.Name != "" {
		replaceArtist = episode.Podcast.feedPodcast.Author.Name
	} else {
		replaceArtist = episode.Podcast.feedPodcast.Title
	}
	completeTag(taglib.Artist, replaceArtist, tag)
	completeTag(taglib.Album, episode.Podcast.feedPodcast.Title, tag)

	plaintextDescription, err := html2text.FromString(episode.feedEpisode.Description)
	if err == nil {
		episode.feedEpisode.Description = plaintextDescription
	}
	if len(episode.feedEpisode.Description) > maxCommentSize+5 {
		episode.feedEpisode.Description = episode.feedEpisode.Description[:maxCommentSize] + " ..."
	}

	completeTag(taglib.Comments, episode.feedEpisode.Description, tag)
	completeTag(taglib.Title, episode.feedEpisode.Title, tag)
	completeTag(taglib.Genre, "Podcast", tag)

	pubdate, err := episode.feedEpisode.ParsedPubDate()
	if err == nil {
		completeTag(taglib.Year, strconv.Itoa(pubdate.Year()), tag)
	}
	logger.Debug.Println("Tag Write Start for : " + episode.file())
	err = tag.Save()
	logger.Debug.Println("Tag Write End for : " + episode.file())
	if err != nil {
		logger.Warning.Println(episode.Podcast.feedPodcast.Title+" - "+episode.feedEpisode.Title+" : Cannot save the modified tags", err)
	}
	logger.Info.Println("Tag update END : " + episode.Podcast.feedPodcast.Title + " - " + episode.feedEpisode.Title)

}
func completeTag(tagname taglib.TagName, tagvalue string, tag *taglib.File) {
	logger.Info.Println(tagname.String() + " --> " + tagvalue)
	tag.SetTag(tagname, tagvalue)
}
