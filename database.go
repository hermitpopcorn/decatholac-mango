// The functions in this file handles anything related to the database,
// be it querying for data or saving them.

package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// This error is thrown whenever a guild (Discord server)-related query is requested
// but it requires the guild to have set a feed channel and it has not done that yet.
type NoFeedChannelSetError struct{}

func (e *NoFeedChannelSetError) Error() string {
	return "The feed channel hasn't been set yet."
}

// Opens a local SQLite database.
func openDatabase() (*sql.DB, error) {
	if _, err := os.Stat("database.db"); err != nil {
		os.Create("database.db")
	}

	db, err := sql.Open("sqlite3", "file:database.db")
	if err != nil {
		return db, err
	}

	if err := db.Ping(); err != nil {
		return db, err
	}

	if err := initializeDatabase(db); err != nil {
		return db, err
	}

	return db, nil
}

// Initializes the database.
// This creates the neccessary tables if they don't exist yet.
func initializeDatabase(db *sql.DB) error {
	check := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Chapters'")
	err := check.Scan()
	if err == sql.ErrNoRows {
		_, err := db.Exec(`CREATE TABLE 'Chapters' (
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

	check = db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Servers'")
	err = check.Scan()
	if err == sql.ErrNoRows {
		_, err := db.Exec(`CREATE TABLE 'Servers' (
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

	return nil
}

// Pairs a channel ID to a guild ID (sets the channel as the guild's feed channel).
func setFeedChannel(db *sql.DB, guildId string, channelId string) error {
	stmt, err := db.Prepare("SELECT channelId FROM Servers WHERE guildId = ?")
	if err != nil {
		return err
	}

	check := stmt.QueryRow(guildId)
	var currentChannelId string
	err = check.Scan(&currentChannelId)
	if err == sql.ErrNoRows {
		// Insert new row if none found
		stmt, err = db.Prepare("INSERT INTO Servers (guildId, channelId, lastAnnouncedAt) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		_, err := stmt.Exec(guildId, channelId, time.Now().UTC())
		if err != nil {
			return err
		}
	} else {
		// Do not write to db if it's the same
		if currentChannelId == channelId {
			return nil
		}

		stmt, err = db.Prepare("UPDATE Servers SET channelId = ? WHERE guildId = ?")
		if err != nil {
			return err
		}
		_, err := stmt.Exec(channelId, guildId)
		if err != nil {
			return err
		}
	}

	return nil
}

// Gets the guild's feed channel ID.
func getFeedChannel(db *sql.DB, guildId string) (string, error) {
	stmt, err := db.Prepare("SELECT channelId FROM Servers WHERE guildId = ?")
	if err != nil {
		return "", err
	}

	check := stmt.QueryRow(guildId)
	var currentChannelId string
	err = check.Scan(&currentChannelId)
	if err == sql.ErrNoRows {
		return "", &NoFeedChannelSetError{}
	}

	return currentChannelId, nil
}

// Gets the timestamp of when an announcement happens for a certain guild.
func getLastAnnouncedTime(db *sql.DB, guildId string) (time.Time, error) {
	stmt, err := db.Prepare("SELECT lastAnnouncedAt FROM Servers WHERE guildId = ?")
	if err != nil {
		return time.Time{}, err
	}

	check := stmt.QueryRow(guildId)
	var lastAnnouncedAt time.Time
	err = check.Scan(&lastAnnouncedAt)
	if err == sql.ErrNoRows {
		return time.Time{}, &NoFeedChannelSetError{}
	}

	return lastAnnouncedAt, nil
}

// Sets the timestamp of... see above.
func setLastAnnouncedTime(db *sql.DB, guildId string, lastAnnouncedAt time.Time) error {
	stmt, err := db.Prepare("UPDATE Servers SET lastAnnouncedAt = ? WHERE guildId = ?")
	if err != nil {
		return err
	}

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
func getAnnouncingServerFlag(db *sql.DB, guildId string) (bool, error) {
	stmt, err := db.Prepare("SELECT isAnnouncing FROM Servers WHERE guildId = ?")
	if err != nil {
		return true, err
	}

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
func setAnnouncingServerFlag(db *sql.DB, guildId string, announcing bool) error {
	stmt, err := db.Prepare("UPDATE Servers SET isAnnouncing = ? WHERE guildId = ?")
	if err != nil {
		return err
	}

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
func saveChapters(db *sql.DB, chapters *[]chapter) error {
	for _, chapter := range *chapters {
		// Check if exists; only write if it doesn't
		stmt, err := db.Prepare("SELECT id FROM Chapters WHERE manga = ? AND title = ? AND number = ?")
		if err != nil {
			return err
		}

		check := stmt.QueryRow(chapter.Manga, chapter.Title, chapter.Number)
		err = check.Scan()
		if err == sql.ErrNoRows {
			log.Print("Saving new chapter... [", chapter.Manga, "]: ", chapter.Title)

			// Insert new row
			stmt, err = db.Prepare("INSERT INTO Chapters (manga, title, number, url, date, loggedAt) VALUES (?, ?, ?, ?, ?, ?)")
			if err != nil {
				return err
			}
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
func getUnannouncedChapters(db *sql.DB, guildId string) (*[]chapter, error) {
	lastAnnouncedAt, err := getLastAnnouncedTime(db, guildId)
	if err != nil {
		return nil, err
	}

	var chapters []chapter

	stmt, err := db.Prepare(`
		SELECT manga, title, number, url, date, loggedAt
		FROM Chapters
		WHERE datetime(loggedAt) > datetime(?)
		AND datetime(date) > datetime(?)
		ORDER BY datetime(loggedAt) ASC
	`)
	if err != nil {
		return nil, err
	}

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
		chapters = append(chapters, chapter{
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
func getServers(db *sql.DB) ([]server, error) {
	var servers []server

	rows, err := db.Query("SELECT guildId, channelId, lastAnnouncedAt, isAnnouncing FROM Servers")
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

		servers = append(servers, server{
			Identifier:            identifier,
			FeedChannelIdentifier: feedChannelIdentifier,
			LastAnnouncedAt:       lastAnnouncedAt,
			IsAnnouncing:          isAnnouncing,
		})
	}

	return servers, nil
}
