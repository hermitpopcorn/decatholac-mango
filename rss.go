package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

func parseRss(target target, rssString string) ([]chapter, error) {
	// Parse RSS string
	parser := gofeed.NewParser()
	feed, err := parser.ParseString(rssString)
	if err != nil {
		return nil, err
	}

	// Collect chapters data into an array
	collectData := func(chapterFeedItem gofeed.Item, counter uint64) chapter {
		chapter := chapter{}

		chapter.Title = chapterFeedItem.Title
		if len(chapterFeedItem.GUID) > 0 {
			chapter.Number = chapterFeedItem.GUID
		} else {
			chapter.Number = strconv.FormatUint(counter, 10)
		}

		// If the URL is relative, append the target's base URL
		url := chapterFeedItem.Link
		if strings.HasPrefix(url, "/") && target.BaseUrl != "" {
			url = target.BaseUrl + url
		}
		chapter.Url = url

		// If Date key is specified and it exists, use. If not, just use Now as the chapter's publish date
		if len(chapterFeedItem.Published) > 0 {
			chapter.Date = *chapterFeedItem.PublishedParsed
		} else {
			chapter.Date = time.Now()
		}

		return chapter
	}

	// Loop over the JSON
	chapters := make([]chapter, 0)
	if target.AscendingSource {
		for i := 0; i < len(feed.Items); i++ {
			chapter := collectData(*feed.Items[i], uint64(i+1))
			chapters = append(chapters, chapter)
		}
	} else {
		for i := len(feed.Items) - 1; i >= 0; i-- {
			chapter := collectData(*feed.Items[i], uint64(i+1))
			chapters = append(chapters, chapter)
		}
	}

	return chapters, nil
}
