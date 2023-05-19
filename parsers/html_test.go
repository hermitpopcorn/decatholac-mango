package parsers

import (
	"testing"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
)

func TestHtmlParser(t *testing.T) {
	// Prepare a pre-set JSON
	testHtml := `
		<div class="eplister" id="chapterlist">
			<ul class="clstyle">
				<li data-num="51">
					<div class="chbox">
						<div class="eph-num">
							<a href="https://test.com/chapter/51">
								<span class="chapternum">Chapter 51</span>
								<span class="chapterdate">June 3, 2022</span>
							</a>
						</div>
					</div>
				</li>
				<li data-num="50">
					<div class="chbox">
						<div class="eph-num">
							<a href="https://test.com/chapter/50">
								<span class="chapternum">Chapter 50</span>
								<span class="chapterdate">May 15, 2022</span>
							</a>
						</div>
					</div>
				</li>
			</ul>
		</div>
	`
	testTarget := types.Target{
		Name:            "HTML Test Manga",
		Mode:            "html",
		BaseUrl:         "https://test.com",
		AscendingSource: false,
		Tags: types.Tags{
			ChaptersTag:     "div#chapterlist li",
			NumberTag:       "",
			NumberAttribute: "data-num",
			TitleTag:        "div div a span.chapternum",
			DateTag:         "div div a span.chapterdate",
			DateFormat:      "January 2, 2006",
			UrlTag:          "div div a",
			UrlAttribute:    "href",
		},
	}

	// Parse
	parsed, err := ParseHtml(&testTarget, &testHtml)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare array length
	if len(parsed) != 2 {
		t.Error("Size mismatch: expected 2, found", len(parsed))
	}

	// Check if the first element is correct
	firstDate, err := time.Parse(testTarget.Tags.DateFormat, "May 15, 2022")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	firstChapter := types.Chapter{
		Manga:  "HTML Test Manga",
		Number: "50",
		Title:  "Chapter 50",
		Date:   firstDate,
		Url:    "https://test.com/chapter/50",
	}
	if parsed[0].Manga != firstChapter.Manga ||
		parsed[0].Title != firstChapter.Title ||
		parsed[0].Number != firstChapter.Number ||
		parsed[0].Url != firstChapter.Url ||
		parsed[0].Date.Unix() != firstChapter.Date.Unix() {
		t.Error("Different first element", parsed[0], firstChapter)
	}

	// Check if the second element is correct
	secondDate, err := time.Parse(testTarget.Tags.DateFormat, "June 3, 2022")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	secondChapter := types.Chapter{
		Manga:  "HTML Test Manga",
		Number: "51",
		Title:  "Chapter 51",
		Date:   secondDate,
		Url:    "https://test.com/chapter/51",
	}
	if parsed[1].Manga != secondChapter.Manga ||
		parsed[1].Title != secondChapter.Title ||
		parsed[1].Number != secondChapter.Number ||
		parsed[1].Url != secondChapter.Url ||
		parsed[1].Date.Unix() != secondChapter.Date.Unix() {
		t.Error("Different second element", parsed[1], secondChapter)
	}
}
