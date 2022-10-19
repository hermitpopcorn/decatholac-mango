package main

import (
	"testing"
	"time"
)

func TestJsonParser(t *testing.T) {
	// Prepare a pre-set JSON
	testJson := `{"comic": {"episodes": [{"id": 16255,"volume": "Chapter 106","sort_volume": 113,"page_count": 0,"title": "Dat Boi","publish_start": "2022-10-11T10:00:00.000+09:00","publish_end": "2022-11-22T10:00:00.000+09:00","member_publish_start": "2022-10-11T10:00:00.000+09:00","member_publish_end": "2022-11-22T10:00:00.000+09:00","status": "public","page_url": "/comics/json/113"},{"id": 16180,"volume": "Chapter 105","sort_volume": 112,"page_count": 0,"title": "Here comes","publish_start": "2022-09-27T10:00:00.000+09:00","publish_end": "2022-11-08T10:00:00.000+09:00","member_publish_start": "2022-09-27T10:00:00.000+09:00","member_publish_end": "2022-11-08T10:00:00.000+09:00","status": "public","page_url": "/comics/json/112"}]}}`
	testTarget := target{
		Name:            "JSON Test Manga",
		Mode:            "json",
		BaseUrl:         "https://mangacross.jp",
		AscendingSource: false,
		Keys: keys{
			Chapters: "comic.episodes",
			Number:   "volume",
			Title:    "volume+title",
			Date:     "publish_start",
			Url:      "page_url",
		},
	}

	// Parse
	parsed, err := parseJson(&testTarget, &testJson)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare array length
	if len(parsed) != 2 {
		t.Error("Size mismatch: expected 2, found", len(parsed))
	}

	// Check if the first element is correct
	firstDate, err := time.Parse(time.RFC3339, "2022-09-27T10:00:00.000+09:00")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	firstChapter := chapter{
		Manga:  "JSON Test Manga",
		Number: "Chapter 105",
		Title:  "Chapter 105 Here comes",
		Date:   firstDate,
		Url:    "https://mangacross.jp/comics/json/112",
	}
	if parsed[0].Manga != firstChapter.Manga ||
		parsed[0].Title != firstChapter.Title ||
		parsed[0].Number != firstChapter.Number ||
		parsed[0].Url != firstChapter.Url ||
		parsed[0].Date.Unix() != firstChapter.Date.Unix() {
		t.Error("Different first element", parsed[0], firstChapter)
	}

	// Check if the second element is correct
	secondDate, err := time.Parse(time.RFC3339, "2022-10-11T10:00:00.000+09:00")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	secondChapter := chapter{
		Manga:  "JSON Test Manga",
		Number: "Chapter 106",
		Title:  "Chapter 106 Dat Boi",
		Date:   secondDate,
		Url:    "https://mangacross.jp/comics/json/113",
	}
	if parsed[1].Manga != secondChapter.Manga ||
		parsed[1].Title != secondChapter.Title ||
		parsed[1].Number != secondChapter.Number ||
		parsed[1].Url != secondChapter.Url ||
		parsed[1].Date.Unix() != secondChapter.Date.Unix() {
		t.Error("Different second element", parsed[1], secondChapter)
	}
}
