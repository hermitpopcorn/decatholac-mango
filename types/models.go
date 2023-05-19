package types

import "time"

type Chapter struct {
	Manga    string
	Number   string
	Title    string
	Date     time.Time
	Url      string
	LoggedAt time.Time
}

type Server struct {
	Identifier            string
	FeedChannelIdentifier string
	LastAnnouncedAt       time.Time
	IsAnnouncing          bool
}
