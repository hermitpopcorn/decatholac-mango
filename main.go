package main

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type configuration struct {
	Token   string
	Targets map[string]target
}

type target struct {
	Name            string
	Source          string
	AscendingSource bool // Whether the source lists item A->Z instead of Z->A like normal
	Mode            string
	BaseUrl         string

	// JSON mode
	Keys keys
}

type keys struct {
	Chapters string
	Number   string
	Title    string
	Date     string
	Url      string
}

type chapter struct {
	Manga    string
	Number   string
	Title    string
	Date     time.Time
	Url      string
	LoggedAt time.Time
}

type server struct {
	Identifier            string
	FeedChannelIdentifier string
	LastAnnouncedAt       time.Time
	IsAnnouncing          bool
}

// Read configuration file
var config configuration

func init() {
	configFile := "config.toml"

	if _, err := os.Stat(configFile); err != nil {
		log.Panicln("Config file not found.")
	}

	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Panicln(err.Error())
	}
}

// Prepare database
var db *sql.DB

func init() {
	var err error
	db, err = openDatabase()
	if err != nil {
		log.Panicln(err.Error())
	}
}

// Helper functions
func sendResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

func sendEphemeralResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func updateResponse(s *discordgo.Session, i *discordgo.Interaction, message string) {
	s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content: &message,
	})
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

	// Register commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "set-as-feed-channel",
			Description: "Set current channel as the feed channel.",
		},
		{
			Name:        "announce",
			Description: "Print all unannounced feed items.",
		},
		{
			Name:        "fetch",
			Description: "Manually trigger the fetch process for new chapters.",
		},
	}

	// Define command handlers
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"set-as-feed-channel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var err error = nil
			err = setFeedChannel(db, i.GuildID, i.ChannelID)
			if err != nil {
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when setting the feed channel...")
				return
			}
			sendResponse(s, i, "This channel has been set as the feed channel.")
		},
		"announce": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var isAnnouncing bool
			var err error = nil
			// Check if the bot is working on announcing the chapters in this server
			isAnnouncing, err = getAnnouncingServerFlag(db, i.GuildID)
			if err != nil {
				switch err.(type) {
				case *NoFeedChannelSetError:
					sendEphemeralResponse(s, i, "You have to set the feed channel for this server first.")
					return
				default:
					log.Print(err.Error())
					sendEphemeralResponse(s, i, "Something went wrong when checking the server flags...")
					return
				}
			}

			if isAnnouncing {
				sendEphemeralResponse(s, i, "The bot is working, so hold on.")
				return
			}

			// Set the "is announcing" flag to true
			err = setAnnouncingServerFlag(db, i.GuildID, true)
			if err != nil {
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when setting the server flags...")
				return
			}

			// Get the feed channel ID
			var channelId string
			channelId, err = getFeedChannel(db, i.GuildID)
			if err != nil {
				var nf *NoFeedChannelSetError
				if errors.As(err, &nf) {
					sendEphemeralResponse(s, i, "You have to set the feed channel for this server first.")
					setAnnouncingServerFlag(db, i.GuildID, false)
					return
				}
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when getting the feed channel...")
				setAnnouncingServerFlag(db, i.GuildID, false)
				return
			}

			// Fetch all unnanounced chapters
			chapters, err := getUnannouncedChapters(db, i.GuildID)
			if err != nil {
				var nf *NoFeedChannelSetError
				if errors.As(err, &nf) {
					sendEphemeralResponse(s, i, "You have to set the feed channel for this server first.")
					setAnnouncingServerFlag(db, i.GuildID, false)
					return
				}
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when fetching the chapters...")
				setAnnouncingServerFlag(db, i.GuildID, false)
				return
			}

			if len(*chapters) > 0 {
				// Say that chapters are found
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Chapters found. Announcing...",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

				// Send all the chapters
				botched := false
				var lastLoggedAt time.Time
				for _, chapter := range *chapters {
					_, err = announceChapter(s, channelId, &chapter)
					if err != nil {
						log.Print(err.Error())
						updateResponse(s, i.Interaction, "Something went wrong when announcing a chapter...")
						setAnnouncingServerFlag(db, i.GuildID, false)
						botched = true
						break
					}

					lastLoggedAt = chapter.LoggedAt
				}

				err = setLastAnnouncedTime(db, i.GuildID, lastLoggedAt)
				if err != nil {
					log.Print(err.Error())
					sendEphemeralResponse(s, i, "Something went wrong when setting the last announcement timestamp...")
				}

				if !botched {
					updateResponse(s, i.Interaction, "Announcing finished.")
				}
			} else {
				sendEphemeralResponse(s, i, "There are no new chapters to announce.")
			}

			// Clear the "is announcing" flag back to false
			err = setAnnouncingServerFlag(db, i.GuildID, false)
			if err != nil {
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when clearing the server flag...")
				return
			}
		},
		"fetch": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if currentlyFetchingTargets {
				sendEphemeralResponse(s, i, "The fetch process is currently in progress.")
				return
			}

			go startGofers()
			sendEphemeralResponse(s, i, "Started the fetch process.")
		},
	}

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	})

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, command := range commands {
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, "", command)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", command.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer session.Close()

	// Setup cron
	cron := cron.New()
	cron.AddFunc("@every 6h", func() {
		log.Print("Fetch process triggered by cronjob.")
		startGofers()

		log.Print("Global announcement process triggered by cronjob.")
		startAnnouncers()
	})
	cron.Start()

	// Exit on Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Print("Press Ctrl+C to exit")
	<-stop

	log.Println("Goodbye...")

	// Remove commands
	for _, v := range registeredCommands {
		err := session.ApplicationCommandDelete(session.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	// Close database
	db.Close()
}
