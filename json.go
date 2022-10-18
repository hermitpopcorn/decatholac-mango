package main

import (
	"encoding/json"
	"strings"
	"time"
)

func parseJson(target target, jsonString string) ([]chapter, error) {
	unmarshalled := make(map[string]any)
	json.Unmarshal([]byte(jsonString), &unmarshalled)

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

	chapters := make([]chapter, 0)
	for _, chapterJson := range chaptersJson {
		chapter := chapter{}

		chapter.Title = chapterJson.(map[string]any)[target.Keys.Title].(string)
		chapter.Number = chapterJson.(map[string]any)[target.Keys.Number].(string)

		url := chapterJson.(map[string]any)[target.Keys.Url].(string)
		if strings.HasPrefix(url, "/") && target.BaseUrl != "" {
			url = target.BaseUrl + url
		}
		chapter.Url = url

		if target.Keys.Date != "" {
			date, err := time.Parse(time.RFC3339, chapterJson.(map[string]any)[target.Keys.Date].(string))
			if err != nil {
				chapter.Date = time.Now()
			} else {
				chapter.Date = date
			}
		} else {
			chapter.Date = time.Now()
		}

		chapters = append(chapters, chapter)
	}

	return chapters, nil
}
