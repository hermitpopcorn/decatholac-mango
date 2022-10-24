// The main file.
// After reading and opening the database, this file kickstarts the bot to start.
// And then waits until the process is killed off by the user.
// It also sets the cronjob for the gofers and announcers to start periodically.

package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/hermitpopcorn/decatholac-mango/database"
	"github.com/hermitpopcorn/decatholac-mango/types"
	"github.com/robfig/cron/v3"
)

type configuration struct {
	Token   string
	Targets []target
}

type target struct {
	Name            string
	Source          string
	AscendingSource bool // Whether the source lists item A->Z instead of Z->A like normal
	Mode            string
	BaseUrl         string

	// JSON mode
	Keys keys

	// HTML mode
	Tags tags
}

type keys struct {
	Chapters string
	Number   string
	Title    string
	Date     string
	Url      string
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
		log.Panicln("Config file not found.")
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

// Helper functions
// Send a normal message as response
func sendResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

// Send a message that only the user can read as response
func sendEphemeralResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// Update a previous message that was sent as a response
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
			Description: "Set current channel as the feed channel. You must have channel management permissions to do this.",
		},
		{
			Name:        "announce",
			Description: "Print all unannounced feed items.",
		},
		{
			Name:        "fetch",
			Description: "Manually trigger the fetch process for new chapters.",
		},
		{
			Name:        "subscribe",
			Description: "Tell the bot you want to be mentioned whenever a new chapter for a specific manga is announced.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "title",
					Description: "The manga title you'd like to get subscribed to.",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
					MinLength:   func(i int) *int { return &i }(1),
					MaxLength:   255,
				},
			},
		},
		{
			Name:        "unsubscribe",
			Description: "Cancels a subscription.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "title",
					Description: "The manga title you'd like to not be subscibed to.",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
					MinLength:   func(i int) *int { return &i }(1),
					MaxLength:   255,
				},
			},
		},
	}

	// Define command handlers
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// Set a channel as the guild's feed channel (also saves the guild into the database)
		"set-as-feed-channel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member.Permissions&discordgo.PermissionManageChannels == 0 {
				sendEphemeralResponse(s, i, "You do not have the permission to set the feed channel.")
				return
			}

			var err error = nil
			err = db.SetFeedChannel(i.GuildID, i.ChannelID)
			if err != nil {
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when setting the feed channel...")
				return
			}
			sendResponse(s, i, "This channel has been set as the feed channel.")
		},

		// Manually trigger the announcement for the current guild (Discord server)
		"announce": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var isAnnouncing bool
			var err error = nil
			// Check if the bot is working on announcing the chapters in this guild
			isAnnouncing, err = db.GetAnnouncingServerFlag(i.GuildID)
			if err != nil {
				switch err.(type) {
				case *database.NoFeedChannelSetError:
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
			err = db.SetAnnouncingServerFlag(i.GuildID, true)
			if err != nil {
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when setting the server flags...")
				return
			}

			// Get the feed channel ID
			var channelId string
			channelId, err = db.GetFeedChannel(i.GuildID)
			if err != nil {
				var nf *database.NoFeedChannelSetError
				if errors.As(err, &nf) {
					sendEphemeralResponse(s, i, "You have to set the feed channel for this server first.")
					db.SetAnnouncingServerFlag(i.GuildID, false)
					return
				}
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when getting the feed channel...")
				db.SetAnnouncingServerFlag(i.GuildID, false)
				return
			}

			// Fetch all unannounced chapters
			chapters, err := db.GetUnannouncedChapters(i.GuildID)
			if err != nil {
				var nf *database.NoFeedChannelSetError
				if errors.As(err, &nf) {
					sendEphemeralResponse(s, i, "You have to set the feed channel for this server first.")
					db.SetAnnouncingServerFlag(i.GuildID, false)
					return
				}
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when fetching the chapters...")
				db.SetAnnouncingServerFlag(i.GuildID, false)
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
				var server = types.Server{
					Identifier:            i.GuildID,
					FeedChannelIdentifier: channelId,
				}
				botched := false
				var lastLoggedAt time.Time
				for _, chapter := range *chapters {
					_, err = announceChapter(s, &server, &chapter)
					if err != nil {
						log.Print(server.Identifier, ": ", err.Error())
						updateResponse(s, i.Interaction, "Something went wrong when announcing a chapter...")
						db.SetAnnouncingServerFlag(i.GuildID, false)
						botched = true
						break
					}

					_, err = mentionSubscribers(db, s, &server, &chapter)
					if err != nil {
						log.Print(server.Identifier, ": ", err.Error())
					}

					lastLoggedAt = chapter.LoggedAt
				}

				err = db.SetLastAnnouncedTime(i.GuildID, lastLoggedAt)
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
			err = db.SetAnnouncingServerFlag(i.GuildID, false)
			if err != nil {
				log.Print(err.Error())
				sendEphemeralResponse(s, i, "Something went wrong when clearing the server flag...")
				return
			}
		},

		// Manually trigger the gofers
		"fetch": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if currentlyFetchingTargets {
				sendEphemeralResponse(s, i, "The fetch process is currently in progress.")
				return
			}

			go startGofers(db, &config.Targets)
			sendEphemeralResponse(s, i, "Started the fetch process.")
		},

		// Add a user and a specified manga title to the subscribe list
		"subscribe": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			title := i.ApplicationCommandData().Options[0].StringValue()
			err := db.SaveSubscription(i.Member.User.ID, i.GuildID, title)
			if err != nil {
				switch err.(type) {
				case *database.TitleDoesNotExistError:
					sendEphemeralResponse(s, i, "That title does not exist.")
					return
				default:
					log.Print(err.Error())
					sendEphemeralResponse(s, i, "Something went wrong when trying to subscribe you...")
					return
				}
			}

			sendEphemeralResponse(s, i, "You are now subscribed to ["+title+"].")
		},

		// Add a user and a specified manga title to the subscribe list
		"unsubscribe": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			title := i.ApplicationCommandData().Options[0].StringValue()
			err := db.RemoveSubscription(i.Member.User.ID, i.GuildID, title)
			if err != nil {
				switch err.(type) {
				case *database.NoSubscriptionFoundError:
					sendEphemeralResponse(s, i, "You are not subscribed to that title.")
					return
				default:
					log.Print(err.Error())
					sendEphemeralResponse(s, i, "Something went wrong when trying to subscribe you...")
					return
				}
			}

			sendEphemeralResponse(s, i, "You are no longer subscribed to ["+title+"].")
		},
	}

	// Match the commands and the handlers
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	})

	// Registers the commands
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
		startGofers(db, &config.Targets)

		log.Print("Global announcement process triggered by cronjob.")
		startAnnouncers(db)
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
