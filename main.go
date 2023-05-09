// The main file.
// After reading and opening the database, this file kickstarts the bot to start.
// And then waits until the process is killed off by the user.
// It also sets the cronjob for the gofers and announcers to start periodically.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/hermitpopcorn/decatholac-mango/database"
	"github.com/hermitpopcorn/decatholac-mango/helpers"
	"github.com/robfig/cron/v3"
)

type configuration struct {
	Token   string
	Targets []target

	WebInterfacePort string

	CronInterval string
}

type target struct {
	Name            string
	Source          string
	AscendingSource bool // Whether the source lists item A->Z instead of Z->A like normal
	Mode            string
	BaseUrl         string
	RequestHeaders  map[string]string

	// JSON mode
	Keys keys

	// HTML mode
	Tags tags
}

type keys struct {
	Chapters   string
	Number     string
	Title      string
	Date       string
	DateFormat string
	Url        string
}

type tags struct {
	ChaptersTag     string
	NumberTag       string
	NumberAttribute string
	TitleTag        string
	TitleAttribute  string
	DateTag         string
	DateAttribute   string
	DateFormat      string
	UrlTag          string
	UrlAttribute    string
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
	var err error
	db, err = database.OpenSQLiteDatabase("database.db")
	if err != nil {
		log.Panicln(err.Error())
	}
}

// Initialize bot
var session *discordgo.Session

func init() {
	var err error
	session, err = discordgo.New("Bot " + config.Token)
	if err != nil {
		log.Panicln(err.Error())
	}
}

func main() {
	// Open session
	err := session.Open()
	if err != nil {
		log.Panicln(err.Error())
	}
	defer session.Close()

	// Setup Discord commands
	var commands = registerCommands()

	// Setup cron
	job := func() {
		fmt.Println(helpers.FormattedNow(), "Fetch process triggered by cronjob")
		startGofers(db, &config.Targets)

		fmt.Println(helpers.FormattedNow(), "Global announcement process triggered by cronjob")
		startAnnouncers(db)
	}
	cron := cron.New()
	cron.AddFunc(config.CronInterval, job)
	cron.Start()
	// Start once immediately on startup
	go job()
	fmt.Println(helpers.FormattedNow(), "Running cron", config.CronInterval)

	// Setup web interface
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			html, err := os.ReadFile("web_interface.html")
			if err != nil {
				log.Panicln(err.Error())
			}
			w.Write(html)
		})

		http.HandleFunc("/fetch", func(w http.ResponseWriter, req *http.Request) {
			if currentlyFetchingTargets {
				w.Write([]byte("Fetching currently in progress."))
				return
			}

			go startGofers(db, &config.Targets)
			w.Write([]byte("Fetch process started."))
		})

		http.HandleFunc("/announce", func(w http.ResponseWriter, req *http.Request) {
			go startAnnouncers(db)
			w.Write([]byte("Announcement process started."))
		})

		port := config.WebInterfacePort
		if port == "" {
			port = ":8080"
		} else {
			port = ":" + config.WebInterfacePort
		}

		err := http.ListenAndServe(port, nil)
		if err != nil {
			log.Println(err.Error())
		}
	}()

	// Exit on Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	fmt.Println(helpers.FormattedNow(), "Press Ctrl+C to exit")
	<-stop

	fmt.Println(helpers.FormattedNow(), "Goodbye...")

	// Remove commands
	unregisterCommands(commands)

	// Close database
	db.Close()
}
