package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
)

type configuration struct {
	Token   string
	Targets map[string][]target
}

type target struct {
	Name    string
	Source  string
	Mode    string
	BaseUrl string

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
	Number string
	Title  string
	Date   time.Time
	Url    string
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
	}

	// Define command handlers
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"set-as-feed-channel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			setFeedChannel(db, i.GuildID, i.ChannelID)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "This channel has been set as the feed channel.",
				},
			})
		},
		"announce": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			channelId, err := getFeedChannel(db, i.GuildID)
			if err != nil {
				log.Panic(err.Error())
			}

			if len(channelId) < 1 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "The feed channel is yet to be set.",
					},
				})
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "To be implemented. Sorry!",
				},
			})
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
