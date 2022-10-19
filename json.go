package main

import (
	"encoding/json"
	"strings"
	"time"
)

func parseJson(target *target, jsonString *string) ([]chapter, error) {
	// Unpack the entire JSON
	unmarshalled := make(map[string]any)
	json.Unmarshal([]byte(*jsonString), &unmarshalled)

	// Delve for the array of objects marked by targets.Keys.Chapters key
	chaptersJson := make([]any, 0)
	unpack := unmarshalled
	traverse := strings.Split(target.Keys.Chapters, ".")
	for index, key := range traverse {
		if index < len(traverse)-1 {
			unpack = unpack[key].(map[string]any)
		} else {
			chaptersJson = unpack[key].([]any)
		}
	}

	// Collect chapters data into an array
	collectData := func(chapterJson map[string]any) chapter {
		chapter := chapter{}

		chapter.Manga = target.Name
		chapter.Title = chapterJson[target.Keys.Title].(string)
		chapter.Number = chapterJson[target.Keys.Number].(string)

		// If the URL is relative, append the target's base URL
		url := chapterJson[target.Keys.Url].(string)
		if strings.HasPrefix(url, "/") && target.BaseUrl != "" {
			url = target.BaseUrl + url
		}
		chapter.Url = url

		// If Date key is specified and it exists, use. If not, just use Now as the chapter's publish date
		if target.Keys.Date != "" {
			date, err := time.Parse(time.RFC3339, chapterJson[target.Keys.Date].(string))
			if err != nil {
				chapter.Date = time.Now()
			} else {
				chapter.Date = date
			}
		} else {
			chapter.Date = time.Now()
		}

		return chapter
	}

	// Loop over the JSON
	chapters := make([]chapter, 0)
	if target.AscendingSource {
		for i := 0; i < len(chaptersJson); i++ {
			chapter := collectData(chaptersJson[i].(map[string]any))
			chapters = append(chapters, chapter)
		}
	} else {
		for i := len(chaptersJson) - 1; i >= 0; i-- {
			chapter := collectData(chaptersJson[i].(map[string]any))
			chapters = append(chapters, chapter)
		}
	}

	return chapters, nil
}
