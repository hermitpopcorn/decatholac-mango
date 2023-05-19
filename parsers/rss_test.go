package parsers

import (
	"testing"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
)

func TestRssParser(t *testing.T) {
	// Prepare a pre-set JSON
	testRss := `<?xml version="1.0"?>
	<rss version="2.0" xmlns:giga="https://gigaviewer.com">
		<channel>
			<title>RSS Test Publishing</title>
			<pubDate>Fri, 23 Sep 2022 03:00:00 +0000</pubDate>
			<link>https://comic-rss.com/title/11111</link>
			<description>Lorem ipsum</description>
			<docs>http://blogs.law.harvard.edu/tech/rss</docs>
			<item>
				<title>Part 24: The Omega</title>
				<link>https://comic-rss.com/episode/00024</link>
				<guid isPermalink="false">00024</guid>
				<pubDate>Fri, 23 Sep 2022 03:00:00 +0000</pubDate>
				<enclosure url="https://cdn-img.comic-rss.com/public/episode-thumbnail/123" length="0" type="image/jpeg" />
				<author>Noowee</author>
			</item>
			<item>
				<title>Part 23: The Alpha</title>
				<link>https://comic-rss.com/episode/00023</link>
				<guid isPermalink="false">00023</guid>
				<pubDate>Fri, 16 Sep 2022 03:00:00 +0000</pubDate>
				<enclosure url="https://cdn-img.comic-rss.com/public/episode-thumbnail/321" length="0" type="image/jpeg" />
				<author>Noowee</author>
			</item>
		</channel>
	</rss>`
	testTarget := types.Target{
		Name:            "RSS Test Publishing",
		Mode:            "rss",
		AscendingSource: false,
	}

	// Parse
	parsed, err := ParseRss(&testTarget, &testRss)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare array length
	if len(parsed) != 2 {
		t.Error("Size mismatch: expected 2, found", len(parsed))
	}

	// Check if the first element is correct
	firstDate, err := time.Parse(time.RFC1123Z, "Fri, 16 Sep 2022 03:00:00 +0000")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	firstChapter := types.Chapter{
		Manga:  "RSS Test Publishing",
		Number: "00023",
		Title:  "Part 23: The Alpha",
		Date:   firstDate,
		Url:    "https://comic-rss.com/episode/00023",
	}
	if parsed[0].Manga != firstChapter.Manga ||
		parsed[0].Title != firstChapter.Title ||
		parsed[0].Number != firstChapter.Number ||
		parsed[0].Url != firstChapter.Url ||
		parsed[0].Date.Unix() != firstChapter.Date.Unix() {
		t.Error("Different first element", parsed[0], firstChapter)
	}

	// Check if the second element is correct
	secondDate, err := time.Parse(time.RFC1123Z, "Fri, 23 Sep 2022 03:00:00 +0000")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	secondChapter := types.Chapter{
		Manga:  "RSS Test Publishing",
		Number: "00024",
		Title:  "Part 24: The Omega",
		Date:   secondDate,
		Url:    "https://comic-rss.com/episode/00024",
	}
	if parsed[1].Manga != secondChapter.Manga ||
		parsed[1].Title != secondChapter.Title ||
		parsed[1].Number != secondChapter.Number ||
		parsed[1].Url != secondChapter.Url ||
		parsed[1].Date.Unix() != secondChapter.Date.Unix() {
		t.Error("Different second element", parsed[1], secondChapter)
	}
}
