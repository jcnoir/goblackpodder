// test.go
package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"time"

	rss "github.com/jteeuwen/go-pkg-rss"

	xmlx "github.com/jteeuwen/go-pkg-xmlx"
)

func PollFeed(uri string, timeout int, cr xmlx.CharsetFunc) {
	feed := rss.New(timeout, true, chanHandler, itemHandler)
	if err := feed.Fetch(uri, cr); err != nil {
		logger.Info.Println(os.Stderr, "[e] %s: %s", uri, err)
		return
	}
	//<-time.After(time.Duration(feed.SecondsTillUpdate() * 1e9))
}

func charsetReader(charset string, r io.Reader) (io.Reader, error) {
	if charset == "ISO-8859-1" || charset == "iso-8859-1" {
		return r, nil
	}
	return nil, errors.New("Unsupported character set encoding: " + charset)
}

func parseTime(formatted string) (time.Time, error) {
	var layouts = [...]string{
		"Mon, _2 Jan 2006 15:04:05 MST",
		"Mon, _2 Jan 2006 15:04:05 -0700",
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		"Mon, 2, Jan 2006 15:4",
		"02 Jan 2006 15:04:05 MST",
	}
	var t time.Time
	var err error
	formatted = strings.TrimSpace(formatted)
	for _, layout := range layouts {
		t, err = time.Parse(layout, formatted)
		if !t.IsZero() {
			break
		}
	}
	return t, err
}
