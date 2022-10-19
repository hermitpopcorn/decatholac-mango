package main

import (
	"testing"
	"time"
)

func TestRssParser(t *testing.T) {
	// Prepare a pre-set JSON
	testRss := `<?xml version="1.0"?>
	<rss version="2.0" xmlns:giga="https://gigaviewer.com">
		<channel>
			<title>ゼノン編集部（少年を飼う）</title>
			<pubDate>Fri, 23 Sep 2022 03:00:00 +0000</pubDate>
			<link>https://comic-zenon.com/episode/13933686331689929651</link>
			<description>森川藍は都内で働くバリキャリOL。残業や深夜帰宅は当たり前、結婚ラッシュどころか終電にすら乗れない日々を送っている。そんな彼女が拾ったのは、とびきり綺麗な16歳の男の子・凪沙だった。猫のようにマイペースな凪沙との出会いが、藍の日常を少しずつ変えていく…。年の差10歳以上、孤独なふたりの奇妙な同居生活。</description>
			<docs>http://blogs.law.harvard.edu/tech/rss</docs>
			<item>
				<title>第24話① 決意</title>
				<link>https://comic-zenon.com/episode/316112896807373046</link>
				<guid isPermalink="false">zenon:episode:316112896807373046</guid>
				<pubDate>Fri, 23 Sep 2022 03:00:00 +0000</pubDate>            <description>少年を飼う</description>
				<enclosure url="https://cdn-img.comic-zenon.com/public/episode-thumbnail/316112896807373046-c2afd9f963ee8ea95449e8a0299acd1f" length="0" type="image/jpeg" />
				<author>青井ぬゐ</author>
			</item>
			<item>
				<title>第23話②  つなぐ</title>
				<link>https://comic-zenon.com/episode/316112896807373040</link>
				<guid isPermalink="false">zenon:episode:316112896807373040</guid>
				<pubDate>Fri, 16 Sep 2022 03:00:00 +0000</pubDate>                <giga:freeTermStartDate>Fri, 23 Sep 2022 03:00:00 +0000</giga:freeTermStartDate>            <description>少年を飼う</description>
				<enclosure url="https://cdn-img.comic-zenon.com/public/episode-thumbnail/316112896807373040-76c56eac4de078c8dbce48954765b47f" length="0" type="image/jpeg" />
				<author>青井ぬゐ</author>
			</item>
		</channel>
	</rss>`
	testTarget := target{
		Name:            "Shounen wo Kau",
		Mode:            "rss",
		AscendingSource: false,
	}

	// Parse
	parsed, err := parseRss(testTarget, testRss)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare array length
	if len(parsed) != 2 {
		t.Error("Size mismatch: expected 2, found", len(parsed))
	}

	// Check if the first element is correct
	firstDate, err := time.Parse(time.RFC1123Z, "Fri, 16 Sep 2022 03:00:00 +0000")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	firstChapter := chapter{
		Number: "zenon:episode:316112896807373040",
		Title:  "第23話②  つなぐ",
		Date:   firstDate,
		Url:    "https://comic-zenon.com/episode/316112896807373040",
	}
	if parsed[0].Number != firstChapter.Number ||
		parsed[0].Title != firstChapter.Title ||
		parsed[0].Url != firstChapter.Url ||
		parsed[0].Date.Unix() != firstChapter.Date.Unix() {
		t.Error("Different first element", parsed[0], firstChapter)
	}

	// Check if the second element is correct
	secondDate, err := time.Parse(time.RFC1123Z, "Fri, 23 Sep 2022 03:00:00 +0000")
	if err != nil {
		t.Error("The test itself failed (time parsing)")
	}
	secondChapter := chapter{
		Number: "zenon:episode:316112896807373046",
		Title:  "第24話① 決意",
		Date:   secondDate,
		Url:    "https://comic-zenon.com/episode/316112896807373046",
	}
	if parsed[1].Number != secondChapter.Number ||
		parsed[1].Title != secondChapter.Title ||
		parsed[1].Url != secondChapter.Url ||
		parsed[1].Date.Unix() != secondChapter.Date.Unix() {
		t.Error("Different second element", parsed[1], secondChapter)
	}
}
