// This is the parser for RSS mode.

package parsers

import (
	"strconv"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
	"github.com/mmcdole/gofeed"
)

func ParseRss(target *types.Target, rssString *string) ([]types.Chapter, error) {
	// Parse RSS string
	parser := gofeed.NewParser()
	feed, err := parser.ParseString(*rssString)
	if err != nil {
		return nil, err
	}

	// Collect chapters data into an array
	collectData := func(chapterFeedItem gofeed.Item, counter uint64) types.Chapter {
		chapter := types.Chapter{}

		chapter.Manga = target.Name
		chapter.Title = chapterFeedItem.Title
		if len(chapterFeedItem.GUID) > 0 {
			chapter.Number = chapterFeedItem.GUID
		} else {
			chapter.Number = strconv.FormatUint(counter, 10)
		}

		// If the URL is relative, append the target's base URL
		url := chapterFeedItem.Link
		chapter.Url = makeFullUrl(url, target.BaseUrl)

		// If Date key is specified and it exists, use. If not, just use Now as the chapter's publish date
		if len(chapterFeedItem.Published) > 0 {
			chapter.Date = *chapterFeedItem.PublishedParsed
		} else {
			chapter.Date = time.Now()
		}

		return chapter
	}

	// Loop over the feed items
	chapters := make([]types.Chapter, 0)
	for i := 0; i < len(feed.Items); i++ {
		index := i
		if !target.AscendingSource {
			index = len(feed.Items) - 1 - i
		}
		chapter := collectData(*feed.Items[index], uint64(index+1))
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}
