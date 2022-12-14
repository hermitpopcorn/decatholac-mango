// This is the parser for JSON mode.

package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
)

func parseJson(target *target, jsonString *string) ([]types.Chapter, error) {
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
	collectData := func(chapterJson map[string]any) types.Chapter {
		chapter := types.Chapter{}

		chapter.Manga = target.Name

		keys := strings.Split(target.Keys.Title, "+")
		if len(keys) == 1 {
			chapter.Title = chapterJson[target.Keys.Title].(string)
		} else if len(keys) > 1 {
			var titleComponents []string
			for _, k := range keys {
				titleComponents = append(titleComponents, chapterJson[k].(string))
			}
			chapter.Title = strings.Join(titleComponents, " ")
		}

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
