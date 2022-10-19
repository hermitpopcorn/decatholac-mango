package main

import (
	"testing"
	"time"
)

func TestJsonParser(t *testing.T) {
	// Prepare a pre-set JSON
	testJson := `{"comic": {"episodes": [{"id": 16255,"volume": "Karte.106","sort_volume": 113,"page_count": 0,"title": "僕らは負けている","publish_start": "2022-10-11T10:00:00.000+09:00","publish_end": "2022-11-22T10:00:00.000+09:00","member_publish_start": "2022-10-11T10:00:00.000+09:00","member_publish_end": "2022-11-22T10:00:00.000+09:00","status": "public","page_url": "/comics/yabai/113","ogp_url": "/episode_ogps/original/missing.png","list_image_url": "https://mangacross.jp/images/episode/MP2afeZAmh091rWkMR1Fxhf6CoP3t8pdXob7mayRAo0/episode_thumbnail/thumb_single.png?1665121954","list_image_double_url": "https://mangacross.jp/images/episode/MP2afeZAmh091rWkMR1Fxhf6CoP3t8pdXob7mayRAo0/episode_thumbnail/thumb_double.png?1665121954","episode_next_date": "2022-10-25T00:00:00.000+09:00","next_date_customize_text": "","is_unlimited_comic": false},{"id": 16180,"volume": "Karte.105","sort_volume": 112,"page_count": 0,"title": "僕は負けたくない","publish_start": "2022-09-27T10:00:00.000+09:00","publish_end": "2022-11-08T10:00:00.000+09:00","member_publish_start": "2022-09-27T10:00:00.000+09:00","member_publish_end": "2022-11-08T10:00:00.000+09:00","status": "public","page_url": "/comics/yabai/112","ogp_url": "/episode_ogps/original/missing.png","list_image_url": "https://mangacross.jp/images/episode/2QetLttutv2RvbOB_smJ6RWE2A6G94i6q_-IE6kKF-g/episode_thumbnail/thumb_single.png?1663826886","list_image_double_url": "https://mangacross.jp/images/episode/2QetLttutv2RvbOB_smJ6RWE2A6G94i6q_-IE6kKF-g/episode_thumbnail/thumb_double.png?1663826886","episode_next_date": "2022-10-11T00:00:00.000+09:00","next_date_customize_text": "","is_unlimited_comic": false}]}}`
	testTarget := target{
		Name:            "Bokuyaba",
		Mode:            "json",
		BaseUrl:         "https://mangacross.jp",
		AscendingSource: false,
		Keys: keys{
			Chapters: "comic.episodes",
			Number:   "volume",
			Title:    "title",
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
		Manga:  "Bokuyaba",
		Number: "Karte.105",
		Title:  "僕は負けたくない",
		Date:   firstDate,
		Url:    "https://mangacross.jp/comics/yabai/112",
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
		Manga:  "Bokuyaba",
		Number: "Karte.106",
		Title:  "僕らは負けている",
		Date:   secondDate,
		Url:    "https://mangacross.jp/comics/yabai/113",
	}
	if parsed[1].Manga != secondChapter.Manga ||
		parsed[1].Title != secondChapter.Title ||
		parsed[1].Number != secondChapter.Number ||
		parsed[1].Url != secondChapter.Url ||
		parsed[1].Date.Unix() != secondChapter.Date.Unix() {
		t.Error("Different second element", parsed[1], secondChapter)
	}
}
