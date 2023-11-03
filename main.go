// The main file.
// After reading and opening the database, this file kickstarts the bot to start.
// And then waits until the process is killed off by the user.
// It also sets the cronjob for the gofers and announcers to start periodically.

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/hermitpopcorn/decatholac-mango/database"
	"github.com/hermitpopcorn/decatholac-mango/helpers"
	"github.com/hermitpopcorn/decatholac-mango/types"
	"github.com/robfig/cron/v3"
)

type configuration struct {
	Database         string
	Token            string
	Targets          []types.Target
	WebInterfacePort string
	CronInterval     string
}

// Read configuration file
var config configuration

func init() {
	configFile := "config.toml"

	if _, err := os.Stat(configFile); err != nil {
		log.Panicln("Config file not found")
	}

	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Panicln(err.Error())
	}
}

// Prepare database
var db database.Database

func init() {
	databaseFile := config.Database
	if databaseFile == "" {
		databaseFile = "database.db"
	}

	var err error
	db, err = database.OpenSQLiteDatabase(databaseFile)
	if err != nil {
		panic(err.Error())
	}
}

// Initialize bot
var session *discordgo.Session

func init() {
	var err error
	session, err = discordgo.New("Bot " + config.Token)
	if err != nil {
		log.Println(err.Error())
	}
}

func main() {
	fmt.Println(helpers.FormattedNow(), "Press Ctrl+C to exit")

	// Open session
	err := session.Open()
	if err != nil {
		log.Println(err.Error())
		session = nil
	} else {
		defer session.Close()
	}

	// Setup Discord commands
	if session != nil {
		registerCommands()
	}

	// Setup cron
	job := func() {
		fmt.Println(helpers.FormattedNow(), "Fetch process triggered by cronjob")
		startGofers(db, &config.Targets)

		if session != nil {
			fmt.Println(helpers.FormattedNow(), "Global announcement process triggered by cronjob")
			startAnnouncers(db)
		} else {
			fmt.Println(helpers.FormattedNow(), "Global announcement process halted: no Discord session")
		}
	}
	cron := cron.New()
	cron.AddFunc(config.CronInterval, job)
	cron.Start()
	// Start once immediately on startup
	go job()
	fmt.Println(helpers.FormattedNow(), "Running cron", config.CronInterval)

	// Setup web interface
	go startWebInterface()

	// Exit on Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	fmt.Println(helpers.FormattedNow(), "Goodbye...")

	// Remove commands
	if session != nil {
		unregisterCommands()
	}

	// Close database
	db.Close()
}
