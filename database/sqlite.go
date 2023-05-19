// The functions in this package handles anything related to the database,
// be it querying for data or saving them.

package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/helpers"
	"github.com/hermitpopcorn/decatholac-mango/types"
	_ "modernc.org/sqlite"
)

type SQLiteDatabase struct {
	connection *sql.DB
}

// Opens a local SQLite database.
func OpenSQLiteDatabase(file string) (*SQLiteDatabase, error) {
	var db SQLiteDatabase

	if file == "" {
		panic("No database file specified.")
	}

	if _, err := os.Stat(file); err != nil {
		os.Create(file)
	}

	connection, err := sql.Open("sqlite", "file:"+file)
	if err != nil {
		return &db, err
	}

	if err := connection.Ping(); err != nil {
		return &db, err
	}

	db.connection = connection

	if err := db.InitializeDatabase(); err != nil {
		return &db, err
	}

	// Prevent lock-up by "wrapping mutex around every DB access"
	// https://github.com/mattn/go-sqlite3/issues/274#issuecomment-191597862
	db.connection.SetMaxOpenConns(1)

	return &db, nil
}

func (db *SQLiteDatabase) Close() error {
	return db.connection.Close()
}

// Initializes the database.
// This creates the neccessary tables if they don't exist yet.
func (db *SQLiteDatabase) InitializeDatabase() error {
	check := db.connection.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Chapters'")
	err := check.Scan()
	if err == sql.ErrNoRows {
		_, err := db.connection.Exec(`CREATE TABLE 'Chapters' (
			'id'		INTEGER,
			'manga'		VARCHAR(255) NOT NULL,
			'title'		VARCHAR(255) NOT NULL,
			'number'	VARCHAR(255) NOT NULL,
			'url'		VARCHAR(255) NOT NULL,
			'date'		DATETIME,
			'loggedAt'	DATETIME NOT NULL,
			PRIMARY KEY('id' AUTOINCREMENT)
		)`)
		if err != nil {
			return err
		}
	}

	check = db.connection.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Servers'")
	err = check.Scan()
	if err == sql.ErrNoRows {
		_, err := db.connection.Exec(`CREATE TABLE 'Servers' (
			'id'				INTEGER,
			'guildId'			VARCHAR(255) NOT NULL,
			'channelId'			VARCHAR(255),
			'lastAnnouncedAt'	DATETIME,
			'isAnnouncing'		INTEGER DEFAULT 0,
			PRIMARY KEY('id' AUTOINCREMENT)
		)`)
		if err != nil {
			return err
		}
	}

	check = db.connection.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Subscriptions'")
	err = check.Scan()
	if err == sql.ErrNoRows {
		_, err := db.connection.Exec(`CREATE TABLE 'Subscriptions' (
			'id'				INTEGER,
			'guildId'			VARCHAR(255) NOT NULL,
			'userId'			VARCHAR(255) NOT NULL,
			'title'				VARCHAR(255) NOT NULL,
			PRIMARY KEY('id' AUTOINCREMENT)
		)`)
		if err != nil {
			return err
		}
	}

	return nil
}

// Pairs a channel ID to a guild ID (sets the channel as the guild's feed channel).
func (db *SQLiteDatabase) SetFeedChannel(guildId string, channelId string) error {
	stmt, err := db.connection.Prepare("SELECT channelId FROM Servers WHERE guildId = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	check := stmt.QueryRow(guildId)
	var currentChannelId string
	err = check.Scan(&currentChannelId)
	if err == sql.ErrNoRows {
		// Insert new row if none found
		stmt, err = db.connection.Prepare("INSERT INTO Servers (guildId, channelId, lastAnnouncedAt) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err := stmt.Exec(guildId, channelId, time.Now().Add((time.Hour*24*7)*-1).UTC())
		if err != nil {
			return err
		}
	} else {
		// Do not write to db if it's the same
		if currentChannelId == channelId {
			return nil
		}

		stmt, err = db.connection.Prepare("UPDATE Servers SET channelId = ? WHERE guildId = ?")
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err := stmt.Exec(channelId, guildId)
		if err != nil {
			return err
		}
	}

	return nil
}

// Gets the guild's feed channel ID.
func (db *SQLiteDatabase) GetFeedChannel(guildId string) (string, error) {
	stmt, err := db.connection.Prepare("SELECT channelId FROM Servers WHERE guildId = ?")
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	check := stmt.QueryRow(guildId)
	var currentChannelId string
	err = check.Scan(&currentChannelId)
	if err == sql.ErrNoRows {
		return "", &NoFeedChannelSetError{}
	}

	return currentChannelId, nil
}

// Gets the timestamp of when an announcement happens for a certain guild.
func (db *SQLiteDatabase) GetLastAnnouncedTime(guildId string) (time.Time, error) {
	stmt, err := db.connection.Prepare("SELECT lastAnnouncedAt FROM Servers WHERE guildId = ?")
	if err != nil {
		return time.Time{}, err
	}
	defer stmt.Close()

	check := stmt.QueryRow(guildId)
	var lastAnnouncedAt time.Time
	err = check.Scan(&lastAnnouncedAt)
	if err == sql.ErrNoRows {
		return time.Time{}, &NoFeedChannelSetError{}
	}

	return lastAnnouncedAt, nil
}

// Sets the timestamp of... see above.
func (db *SQLiteDatabase) SetLastAnnouncedTime(guildId string, lastAnnouncedAt time.Time) error {
	stmt, err := db.connection.Prepare("UPDATE Servers SET lastAnnouncedAt = ? WHERE guildId = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	exec, err := stmt.Exec(lastAnnouncedAt.UTC(), guildId)
	if err != nil {
		return err
	}

	affected, err := exec.RowsAffected()
	if err != nil {
		return err
	}
	if affected < 1 {
		return &NoFeedChannelSetError{}
	}

	return nil
}

// Gets the status for the announcing server flag of a certain guild.
func (db *SQLiteDatabase) GetAnnouncingServerFlag(guildId string) (bool, error) {
	stmt, err := db.connection.Prepare("SELECT isAnnouncing FROM Servers WHERE guildId = ?")
	if err != nil {
		return true, err
	}
	defer stmt.Close()

	check := stmt.QueryRow(guildId)
	var isAnnouncing int
	err = check.Scan(&isAnnouncing)
	if err == sql.ErrNoRows {
		return true, &NoFeedChannelSetError{}
	}

	if isAnnouncing == 0 {
		return false, nil
	} else if isAnnouncing >= 1 {
		return true, nil
	}

	return true, nil
}

// Sets the... see above.
func (db *SQLiteDatabase) SetAnnouncingServerFlag(guildId string, announcing bool) error {
	stmt, err := db.connection.Prepare("UPDATE Servers SET isAnnouncing = ? WHERE guildId = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	var boolint int
	if announcing {
		boolint = 1
	} else {
		boolint = 0
	}

	exec, err := stmt.Exec(boolint, guildId)
	if err != nil {
		return err
	}

	affected, err := exec.RowsAffected()
	if err != nil {
		return err
	}
	if affected < 1 {
		return &NoFeedChannelSetError{}
	}

	return nil
}

// Saves an array of chapters to the database.
func (db *SQLiteDatabase) SaveChapters(chapters *[]types.Chapter) error {
	for _, chapter := range *chapters {
		// Check if exists; only write if it doesn't
		stmt, err := db.connection.Prepare("SELECT id FROM Chapters WHERE manga = ? AND title = ? AND number = ?")
		if err != nil {
			return err
		}
		defer stmt.Close()

		check := stmt.QueryRow(chapter.Manga, chapter.Title, chapter.Number)
		err = check.Scan()
		if err == sql.ErrNoRows {
			fmt.Println(helpers.FormattedNow(), "Saving new chapter... ["+chapter.Manga+"]:", chapter.Title)

			// Insert new row
			stmt, err = db.connection.Prepare("INSERT INTO Chapters (manga, title, number, url, date, loggedAt) VALUES (?, ?, ?, ?, ?, ?)")
			if err != nil {
				return err
			}
			defer stmt.Close()

			_, err := stmt.Exec(chapter.Manga, chapter.Title, chapter.Number, chapter.Url, chapter.Date.UTC(), time.Now().UTC())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Get unannounced chapters for a specific guild.
// How a chapter is "unannounced" is determined by:
// (1) the guild's lastAnnouncedAt; (2) the chapter's loggedAt; and (3) the chapter's publish date.
// If a chapter is logged into the database AFTER a guild's last announcement timestamp,
// it means the chapter is new and thus needs to be announced...
// UNLESS that chapter was released BEFORE the guild's last announcement time,
// which means the chapter is actually old, but was just logged into the database recently.
func (db *SQLiteDatabase) GetUnannouncedChapters(guildId string) (*[]types.Chapter, error) {
	lastAnnouncedAt, err := db.GetLastAnnouncedTime(guildId)
	if err != nil {
		return nil, err
	}

	var chapters []types.Chapter

	stmt, err := db.connection.Prepare(`
		SELECT manga, title, number, url, date, loggedAt
		FROM Chapters
		WHERE loggedAt > ?
		AND date > ?
		ORDER BY date ASC
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(lastAnnouncedAt, lastAnnouncedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var manga string
		var title string
		var number string
		var url string
		var date time.Time
		var loggedAt time.Time
		err = rows.Scan(&manga, &title, &number, &url, &date, &loggedAt)
		if err != nil {
			return nil, err
		}
		chapters = append(chapters, types.Chapter{
			Manga:    manga,
			Title:    title,
			Number:   number,
			Url:      url,
			Date:     date,
			LoggedAt: loggedAt,
		})
	}

	return &chapters, nil
}

// Gets all the guilds saved in the database.
// Guilds are saved into the database whenever it sets a channel as its feed channel.
// (see setFeedChannel() function)
func (db *SQLiteDatabase) GetServers() ([]types.Server, error) {
	var servers []types.Server

	rows, err := db.connection.Query("SELECT guildId, channelId, lastAnnouncedAt, isAnnouncing FROM Servers")
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var identifier string
		var feedChannelIdentifier string
		var lastAnnouncedAt time.Time
		var isAnnouncingInt int
		var isAnnouncing bool
		err = rows.Scan(&identifier, &feedChannelIdentifier, &lastAnnouncedAt, &isAnnouncingInt)
		if err != nil {
			return nil, err
		}

		if isAnnouncingInt == 0 {
			isAnnouncing = false
		}
		if isAnnouncingInt == 1 {
			isAnnouncing = true
		}

		servers = append(servers, types.Server{
			Identifier:            identifier,
			FeedChannelIdentifier: feedChannelIdentifier,
			LastAnnouncedAt:       lastAnnouncedAt,
			IsAnnouncing:          isAnnouncing,
		})
	}

	return servers, nil
}

// Checks if a manga title exists in the Chapters table.
func (db *SQLiteDatabase) CheckMangaExistence(title string) (bool, error) {
	stmt, err := db.connection.Prepare("SELECT manga FROM Chapters WHERE manga = ? LIMIT 1")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	check := stmt.QueryRow(title)
	err = check.Scan()
	if err == sql.ErrNoRows {
		return false, nil
	} else {
		return true, nil
	}
}

// Saves a subscription entry.
func (db *SQLiteDatabase) SaveSubscription(userId string, guildId string, title string) error {
	titleExists, err := db.CheckMangaExistence(title)
	if err != nil {
		return err
	}

	if !titleExists {
		return &TitleDoesNotExistError{}
	}

	stmt, err := db.connection.Prepare("SELECT id FROM Subscriptions WHERE userId = ? AND guildId = ? AND title = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	subscriptionExists := stmt.QueryRow(userId, guildId, title)
	err = subscriptionExists.Scan()
	if err == sql.ErrNoRows {
		// Insert new row if none found
		stmt, err = db.connection.Prepare("INSERT INTO Subscriptions (userId, guildId, title) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err := stmt.Exec(userId, guildId, title)
		if err != nil {
			return err
		}
	}

	return nil
}

// Removes a subscription entry.
func (db *SQLiteDatabase) RemoveSubscription(userId string, guildId string, title string) error {
	stmt, err := db.connection.Prepare("SELECT id FROM Subscriptions WHERE userId = ? AND guildId = ? AND title = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	subscriptionExists := stmt.QueryRow(userId, guildId, title)
	err = subscriptionExists.Scan()
	if err != sql.ErrNoRows {
		// Remove if found
		stmt, err = db.connection.Prepare("DELETE FROM Subscriptions WHERE userId = ? AND guildId = ? AND title = ?")
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err := stmt.Exec(userId, guildId, title)
		if err != nil {
			return err
		}
	} else {
		return &NoSubscriptionFoundError{}
	}

	return nil
}

// Get the list of user IDs that are subscribed to a certain title in a certain guild.
func (db *SQLiteDatabase) GetSubscribers(guildId string, title string) ([]string, error) {
	var userIds []string

	stmt, err := db.connection.Prepare("SELECT userId FROM Subscriptions WHERE guildId = ? AND title = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(guildId, title)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userId string
		err = rows.Scan(&userId)
		if err != nil {
			return nil, err
		}
		userIds = append(userIds, userId)
	}

	return userIds, nil
}
