package main

import (
	"database/sql"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

func initializeDatabase(db *sql.DB) error {
	check := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Chapters'")
	err := check.Scan()
	if err == sql.ErrNoRows {
		_, err := db.Exec(`CREATE TABLE 'Chapters' (
			'id'		INTEGER,
			'manga'		VARCHAR(255) NOT NULL,
			'title'		VARCHAR(255) NOT NULL,
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
			PRIMARY KEY('id' AUTOINCREMENT)
		)`)
		if err != nil {
			return err
		}
	}

	return nil
}

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

func getFeedChannel(db *sql.DB, guildId string) (string, error) {
	stmt, err := db.Prepare("SELECT channelId FROM Servers WHERE guildId = ?")
	if err != nil {
		return "", err
	}

	check := stmt.QueryRow(guildId)
	var currentChannelId string
	err = check.Scan(&currentChannelId)
	if err == sql.ErrNoRows {
		return "", nil
	}

	return currentChannelId, nil
}
