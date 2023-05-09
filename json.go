// This is the parser for JSON mode.

package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
)

func traverse(data map[string]any, key string) any {
	unpack := data
	traverse := strings.Split(key, ".")
	for index, key := range traverse {
		if index < len(traverse)-1 {
			unpack = unpack[key].(map[string]any)
		} else {
			return unpack[key]
		}
	}

	return nil
}

func parseJson(target *target, jsonString *string) ([]types.Chapter, error) {
	// Unpack the entire JSON
	unmarshalled := make(map[string]any)
	json.Unmarshal([]byte(*jsonString), &unmarshalled)

	// Delve for the array of objects marked by targets.Keys.Chapters key
	chaptersJson := traverse(unmarshalled, target.Keys.Chapters).([]any)

	// Collect chapters data into an array
	collectData := func(chapterJson map[string]any) types.Chapter {
		chapter := types.Chapter{}

		chapter.Manga = target.Name

		keys := strings.Split(target.Keys.Title, "+")
		if len(keys) == 1 {
			chapter.Title = traverse(chapterJson, target.Keys.Title).(string)
		} else if len(keys) > 1 {
			var titleComponents []string
			for _, k := range keys {
				titleComponent := traverse(chapterJson, k).(string)
				if titleComponent != "" {
					titleComponents = append(titleComponents, titleComponent)
				}
			}
			chapter.Title = strings.Join(titleComponents, " ")
		}

		chapter.Number = traverse(chapterJson, target.Keys.Number).(string)

		// If the URL is relative, append the target's base URL
		url := traverse(chapterJson, target.Keys.Url).(string)
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
	chapters := make([]types.Chapter, 0)
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
