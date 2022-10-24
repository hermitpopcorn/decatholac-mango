// This is the parser for HTML mode.

package main

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hermitpopcorn/decatholac-mango/types"
)

// Gets the text inside a DOM node.
// OR, if the "attribute" parameter is specified, gets the value for that attribute.
func getNodeText(node *goquery.Selection, tag string, attribute string) string {
	var selectNode *goquery.Selection

	// If tag is specified, find. Otherwise just use the current selection
	if tag != "" {
		selectNode = node.Find(tag)
	} else {
		selectNode = node
	}
	if selectNode.Length() < 1 {
		return ""
	}
	// If attribute is specified, use the value of that variable. Otherwise just use the text inside the node
	if attribute != "" {
		val, exists := selectNode.First().Attr(attribute)
		if exists {
			return val
		} else {
			return ""
		}
	} else {
		return selectNode.First().Text()
	}
}

// Does the entire HTML parsing thing.
func parseHtml(target *target, htmlString *string) ([]types.Chapter, error) {
	reader := strings.NewReader(*htmlString)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	var chapters = make([]types.Chapter, 0)

	// Get the chapters' list container nodes
	chapterNodes := doc.Find(target.Tags.ChaptersTag)
	if chapterNodes.Length() < 1 {
		return chapters, nil
	}

	// Loop over the chapter nodes.
	chapterNodes.Each(func(i int, node *goquery.Selection) {
		var chapter types.Chapter
		chapter.Manga = target.Name

		// Get title
		title := getNodeText(node, target.Tags.TitleTag, target.Tags.TitleAttribute)
		if len(title) < 1 {
			return
		}
		chapter.Title = title

		// Get number
		number := getNodeText(node, target.Tags.NumberTag, target.Tags.NumberAttribute)
		if len(number) < 1 {
			return
		}
		chapter.Number = number

		// Get URL
		url := getNodeText(node, target.Tags.UrlTag, target.Tags.UrlAttribute)
		if len(url) < 1 {
			return
		}
		chapter.Url = url

		// Get publish date
		chapter.Date = time.Now()
		if target.Tags.DateTag != "" {
			date := getNodeText(node, target.Tags.DateTag, target.Tags.DateAttribute)
			if len(date) > 0 {
				parsedDate, parseErr := time.Parse(target.Tags.DateFormat, date)
				if parseErr == nil {
					chapter.Date = parsedDate
				}
			}
		}

		chapters = append(chapters, chapter)
	})

	// Reverse the array if the souce is in descending order
	if !target.AscendingSource {
		for i, j := 0, len(chapters)-1; i < j; i, j = i+1, j-1 {
			chapters[i], chapters[j] = chapters[j], chapters[i]
		}
	}

	return chapters, nil
}
