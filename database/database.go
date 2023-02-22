// The functions in this package handles anything related to the database,
// be it querying for data or saving them.

package database

import (
	"time"

	"github.com/hermitpopcorn/decatholac-mango/types"
	_ "modernc.org/sqlite"
)

// This error is thrown whenever a guild (Discord server)-related query is requested
// but it requires the guild to have set a feed channel and it has not done that yet.
type NoFeedChannelSetError struct{}

func (e *NoFeedChannelSetError) Error() string {
	return "The feed channel hasn't been set yet."
}

// This error is thrown whenever a user requests a subscription for a title,
// but that title does not currently exist in the database.
type TitleDoesNotExistError struct{}

func (e *TitleDoesNotExistError) Error() string {
	return "The specified title does not exist in the database yet."
}

// This error is thrown whenever a user requests removal of their subscription for a title,
// but that subscription does not exist in the first place.
type NoSubscriptionFoundError struct{}

func (e *NoSubscriptionFoundError) Error() string {
	return "The user is not subscribed to such title."
}

type Database interface {
	GetServers() ([]types.Server, error)
	GetFeedChannel(guildId string) (string, error)
	SetFeedChannel(guildId string, channelId string) error
	GetLastAnnouncedTime(guildId string) (time.Time, error)
	SetLastAnnouncedTime(guildId string, lastAnnouncedAt time.Time) error
	CheckMangaExistence(title string) (bool, error)
	SaveChapters(chapters *[]types.Chapter) error
	GetUnannouncedChapters(guildId string) (*[]types.Chapter, error)
	GetAnnouncingServerFlag(guildId string) (bool, error)
	SetAnnouncingServerFlag(guildId string, announcing bool) error
	GetSubscribers(guildId string, title string) ([]string, error)
	SaveSubscription(userId string, guildId string, title string) error
	RemoveSubscription(userId string, guildId string, title string) error
	Close() error
}
