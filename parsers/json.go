// This is the parser for JSON mode.

package parsers

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
)

// Traverses to one or more values and then concatenates them.
func parseComponent(data map[string]any, key string) string {
	keys := strings.Split(key, "+")

	var components []string
	for _, k := range keys {
		component := traverse(data, k).(string)
		if component != "" {
			components = append(components, component)
		}
	}

	return strings.Join(components, " ")
}

// Traverse the given data map using dot notation and returns the value.
func traverse(data map[string]any, key string) any {
	traverse := strings.Split(key, ".")
	for index, key := range traverse {
		if index < len(traverse)-1 {
			data = data[key].(map[string]any)
		} else {
			return data[key]
		}
	}

	return nil
}

// Parses the given JSON string using the target information and returns an array of Chapters.
func ParseJson(target *types.Target, jsonString *string) ([]types.Chapter, error) {
	// Unpack the entire JSON
	unmarshalled := make(map[string]any)
	json.Unmarshal([]byte(*jsonString), &unmarshalled)

	// Delve for the array of objects marked by targets.Keys.Chapters key
	chaptersJson := traverse(unmarshalled, target.Keys.Chapters).([]any)

	// Collect chapters data into an array
	collectData := func(chapterJson map[string]any) (types.Chapter, bool) {
		chapter := types.Chapter{}

		// Check for skip
		for key, value := range target.Keys.Skip {
			valueInJson := traverse(chapterJson, key)
			if valueInJson == value {
				return chapter, true
			}
		}

		// Get chapter data
		chapter.Manga = target.Name
		chapter.Title = parseComponent(chapterJson, target.Keys.Title)
		chapter.Number = parseComponent(chapterJson, target.Keys.Number)
		url := parseComponent(chapterJson, target.Keys.Url)
		chapter.Url = makeFullUrl(url, target.BaseUrl)

		// If Date key is specified and it exists, use. If not, just use Now as the chapter's publish date
		if target.Keys.Date != "" {
			dateFormat := target.Keys.DateFormat
			if dateFormat == "" {
				dateFormat = "RFC3339"
			}

			var date time.Time
			var err error
			if dateFormat == "unix" {
				timestamp, ok := traverse(chapterJson, target.Keys.Date).(float64)
				if ok {
					intTimestamp := int64(timestamp)
					date = time.Unix(intTimestamp/1000, (intTimestamp%1000)*int64(time.Millisecond))
				} else {
					err = errors.New("unable to parse timestamp")
				}
			} else if dateFormat == "RFC3339" {
				dateString, ok := traverse(chapterJson, target.Keys.Date).(string)
				if ok {
					date, err = time.Parse(time.RFC3339, dateString)
				} else {
					err = errors.New("unable to parse RFC3339 date")
				}
			} else {
				err = errors.New("dateFormat is invalid: " + dateFormat)
			}

			if err != nil {
				chapter.Date = time.Now()
			} else {
				chapter.Date = date
			}
		} else {
			chapter.Date = time.Now()
		}

		return chapter, false
	}

	// Loop over the JSON
	chapters := make([]types.Chapter, 0)
	for i := 0; i < len(chaptersJson); i++ {
		index := i
		if !target.AscendingSource {
			index = len(chaptersJson) - 1 - i
		}

		chapter, skip := collectData(chaptersJson[index].(map[string]any))
		if !skip {
			chapters = append(chapters, chapter)
		}
	}

	return chapters, nil
}
