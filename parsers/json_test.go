package parsers

import (
	"testing"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
)

func TestJsonParser(t *testing.T) {
	// Prepare a pre-set JSON
	testJson := `
	{
		"comic": {
			"episodes": [
				{
					"id": 16255,
					"volume": "Chapter 106",
					"sort_volume": 113,
					"page_count": 0,
					"title": "Dat Boi",
					"publish_start": "2022-10-11T10:00:00.000+09:00",
					"publish_end": "2022-11-22T10:00:00.000+09:00",
					"member_publish_start": "2022-10-11T10:00:00.000+09:00",
					"member_publish_end": "2022-11-22T10:00:00.000+09:00",
					"status": "public",
					"page_url": "/comics/json/113"
				},
				{
					"id": 16180,
					"volume": "Chapter 105",
					"sort_volume": 112,
					"page_count": 0,
					"title": "Here comes",
					"publish_start": "2022-09-27T10:00:00.000+09:00",
					"publish_end": "2022-11-08T10:00:00.000+09:00",
					"member_publish_start": "2022-09-27T10:00:00.000+09:00",
					"member_publish_end": "2022-11-08T10:00:00.000+09:00",
					"status": "public",
					"page_url": "/comics/json/112"
				}
			]
		}
	}`
	testTarget := types.Target{
		Name:            "JSON Test Manga",
		Mode:            "json",
		BaseUrl:         "https://mangacross.jp",
		AscendingSource: false,
		Keys: types.Keys{
			Chapters: "comic.episodes",
			Number:   "volume",
			Title:    "volume+title",
			Date:     "publish_start",
			Url:      "page_url",
		},
	}

	// Parse
	parsed, err := ParseJson(&testTarget, &testJson)
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
	firstChapter := types.Chapter{
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
	secondChapter := types.Chapter{
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

func TestJsonParserWithSkipKeys(t *testing.T) {
	// Prepare a pre-set JSON
	testJson := `
	{
		"data": {
			"episodes": [
				{
					"readable": true,
					"episode": {
					"id": 8,
					"numbering_title": "Chapter 8",
					"sub_title": "The End",
					"read_start_at": 1681959600000,
					"viewer_path": "/viewer/stories/82"
					}
				},
				{ "readable": false, "message": "Chapter 3~7 is now GONE" },
				{
					"readable": true,
					"episode": {
					"id": 96352,
					"numbering_title": "Chapter 2",
					"sub_title": "The Revolution",
					"read_start_at": 1622689200000,
					"viewer_path": "/viewer/stories/2"
					}
				},
				{
					"readable": true,
					"episode": {
					"id": 95786,
					"numbering_title": "Chapter 1",
					"sub_title": "The Pilot",
					"read_start_at": 1622084400000,
					"viewer_path": "/viewer/stories/1"
					}
				}
			]
		}
	}`
	testTarget := types.Target{
		Name:            "JSON Test Manga",
		Mode:            "json",
		BaseUrl:         "https://mangacross.jp",
		AscendingSource: false,
		Keys: types.Keys{
			Chapters:   "data.episodes",
			Number:     "episode.numbering_title+episode.sub_title",
			Title:      "episode.numbering_title+episode.sub_title",
			Date:       "episode.read_start_at",
			DateFormat: "unix",
			Url:        "episode.viewer_path",
			Skip: map[string]any{
				"readable": false,
			},
		},
	}

	// Parse
	parsed, err := ParseJson(&testTarget, &testJson)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare array length
	if len(parsed) != 3 {
		t.Error("Size mismatch: expected 3, found", len(parsed))
	}

	// Check if the second element is correct
	intTimestamp := int64(1622689200000)
	secondDate := time.Unix(intTimestamp/1000, (intTimestamp%1000)*int64(time.Millisecond))
	secondChapter := types.Chapter{
		Manga:  "JSON Test Manga",
		Number: "Chapter 2 The Revolution",
		Title:  "Chapter 2 The Revolution",
		Date:   secondDate,
		Url:    "https://mangacross.jp/viewer/stories/2",
	}
	if parsed[1].Manga != secondChapter.Manga ||
		parsed[1].Title != secondChapter.Title ||
		parsed[1].Number != secondChapter.Number ||
		parsed[1].Url != secondChapter.Url ||
		parsed[1].Date.Unix() != secondChapter.Date.Unix() {
		t.Error("Different second element", parsed[1], secondChapter)
	}

	// Check if the last element is correct
	// (should skip the "unreadable" entry)
	intTimestamp = int64(1681959600000)
	thirdDate := time.Unix(intTimestamp/1000, (intTimestamp%1000)*int64(time.Millisecond))
	thirdChapter := types.Chapter{
		Manga:  "JSON Test Manga",
		Number: "Chapter 8 The End",
		Title:  "Chapter 8 The End",
		Date:   thirdDate,
		Url:    "https://mangacross.jp/viewer/stories/82",
	}
	if parsed[2].Manga != thirdChapter.Manga ||
		parsed[2].Title != thirdChapter.Title ||
		parsed[2].Number != thirdChapter.Number ||
		parsed[2].Url != thirdChapter.Url ||
		parsed[2].Date.Unix() != thirdChapter.Date.Unix() {
		t.Error("Different last element", parsed[2], thirdChapter)
	}
}
