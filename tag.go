package main

import (
	"os/exec"
	"strconv"
	"sync"

	"fmt"

	"github.com/jaytaylor/html2text"
	"github.com/wtolson/go-taglib"
)

//EpisodeTag is a podcast episode tag
type EpisodeTag struct {
}

var execWg sync.WaitGroup

func completeTags(episode *Episode) {

	logger.Debug.Println("Tag update : " + episode.Podcast.feedPodcast.Title + " - " + episode.feedEpisode.Title + " : " + episode.file())

	tag, err := taglib.Read(episode.file())
	if err != nil {
		logger.Warning.Println("Cannot complete episode tags for "+episode.Podcast.feedPodcast.Title+" - "+episode.feedEpisode.Title, err)
		return
	}
	defer tag.Close()

	var replaceArtist string
	if episode.feedEpisode.Author.Name != "" {
		replaceArtist = episode.feedEpisode.Author.Name
	} else {
		replaceArtist = episode.Podcast.feedPodcast.Title
	}

	//use the podcast title for now
	replaceArtist = episode.Podcast.feedPodcast.Title

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
	completeTag(taglib.Title, episode.feedEpisode.Title+" "+episode.formattedPubDate(dateFormat), tag)
	completeTag(taglib.Genre, "Podcast", tag)

	pubdate, err := episode.feedEpisode.ParsedPubDate()
	if err == nil {
		completeTag(taglib.Year, strconv.Itoa(pubdate.Year()), tag)
	}
	logger.Debug.Println("Tag Write Start for : " + episode.file())
	err = tag.Save()
	//setAlbumArtist(episode.Podcast.feedPodcast.Title, episode.file())

	logger.Debug.Println("Tag Write End for : " + episode.file())
	if err != nil {
		logger.Warning.Println(episode.Podcast.feedPodcast.Title+" - "+episode.feedEpisode.Title+" : Cannot save the modified tags", err)
	}
	logger.Debug.Println("Tag update END : " + episode.Podcast.feedPodcast.Title + " - " + episode.feedEpisode.Title)

}
func completeTag(tagname taglib.TagName, tagvalue string, tag *taglib.File) {
	logger.Debug.Println(fmt.Sprintf("Tag: %s --> %s", tagname.String(), tagvalue))
	tag.SetTag(tagname, tagvalue)
}

func setAlbumArtist(albumartist string, filepath string) {
	cmd := exec.Command("eyeD3", "--text-frame=TPE2:"+albumartist, filepath)
	logger.Debug.Println("Command to be executed : ", cmd.Args)
	err := cmd.Run()
	if err != nil {
		cmd := exec.Command("eyeD3", "--set-text-frame=TPE2:"+albumartist, filepath)
		err = cmd.Run()
		if err != nil {
			logger.Error.Println("Cannot set albumartist tag : ", err)
		}
	}
}
