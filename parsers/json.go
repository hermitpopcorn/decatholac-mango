// This is the parser for JSON mode.

package parsers

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
)

func parseComponent(json map[string]any, key string) string {
	keys := strings.Split(key, "+")

	var titleComponents []string
	if len(keys) == 1 {
		titleComponents = append(titleComponents, traverse(json, key).(string))
	} else if len(keys) > 1 {
		for _, k := range keys {
			titleComponent := traverse(json, k).(string)
			if titleComponent != "" {
				titleComponents = append(titleComponents, titleComponent)
			}
		}
	}

	return strings.Join(titleComponents, " ")
}

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

		// If the URL is relative, append the target's base URL
		url := traverse(chapterJson, target.Keys.Url).(string)
		if strings.HasPrefix(url, "/") && target.BaseUrl != "" {
			url = target.BaseUrl + url
		}
		chapter.Url = url

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
					err = errors.New("unable to parse timestamp: " + strconv.FormatFloat(traverse(chapterJson, target.Keys.Date).(float64), 'f', 2, 64))
				}
			} else if dateFormat == "RFC3339" {
				date, err = time.Parse(time.RFC3339, traverse(chapterJson, target.Keys.Date).(string))
			} else {
				err = errors.New("DateFormat is invalid: " + dateFormat)
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
	if target.AscendingSource {
		for i := 0; i < len(chaptersJson); i++ {
			chapter, skip := collectData(chaptersJson[i].(map[string]any))
			if !skip {
				chapters = append(chapters, chapter)
			}
		}
	} else {
		for i := len(chaptersJson) - 1; i >= 0; i-- {
			chapter, skip := collectData(chaptersJson[i].(map[string]any))
			if !skip {
				chapters = append(chapters, chapter)
			}
		}
	}

	return chapters, nil
}
