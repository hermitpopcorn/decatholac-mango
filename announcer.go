// This file handles the announcing of new chapters to guilds.

package main

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hermitpopcorn/decatholac-mango/database"
	"github.com/hermitpopcorn/decatholac-mango/types"
)

// Announce a single chapter to a certain guild's feed channel.
func announceChapter(session *discordgo.Session, server *types.Server, chapter *types.Chapter) (*discordgo.Message, error) {
	message, err := session.ChannelMessageSendEmbed(server.FeedChannelIdentifier, &discordgo.MessageEmbed{
		Type:      discordgo.EmbedTypeLink,
		URL:       chapter.Url,
		Title:     "[" + chapter.Manga + "] " + chapter.Title,
		Timestamp: chapter.Date.In(time.FixedZone("JST", 9*60*60)).Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}

	return message, nil
}

// Mention subscribers for announced chapter.
func mentionSubscribers(db database.Database, session *discordgo.Session, server *types.Server, chapter *types.Chapter) (*discordgo.Message, error) {
	userIds, err := db.GetSubscribers(server.Identifier, chapter.Manga)
	if err != nil {
		return nil, err
	}

	// Collect mention string
	var mentions []string
	for _, userId := range userIds {
		user, err := session.User(userId)
		if err != nil {
			continue
		}

		mentions = append(mentions, user.Mention())
	}

	// Send all mention strings in a single message
	// TODO: Split the message if it's too long?
	if len(mentions) > 0 {
		message, err := session.ChannelMessageSend(server.FeedChannelIdentifier, strings.Join(mentions, " "))
		if err != nil {
			return nil, err
		}

		return message, nil
	}

	return nil, nil
}

// The "mother" announcer process.
// This gets the list of all registered guilds and their unannounced chapters.
// If found, it sends the new chapters to the guilds' feed channels,
// and then logs the last announcement time of each guild.
func startAnnouncers(db database.Database) error {
	// Get the list of servers
	servers, err := db.GetServers()
	if err != nil {
		return err
	}

	// Iterate through servers
	var waiter sync.WaitGroup
	for _, s := range servers {
		waiter.Add(1)

		// Run a parallel process for each server
		go func(server types.Server) {
			log.Print("Starting announcement process for server ", server.Identifier)

			var err error = nil
			// Check if the bot is working on announcing the chapters in this server
			var isAnnouncing bool
			isAnnouncing, err = db.GetAnnouncingServerFlag(server.Identifier)
			if err != nil {
				log.Print(server.Identifier, ": ", err.Error())
				waiter.Done()
				return
			}

			// Cancel if is announcing
			if isAnnouncing {
				waiter.Done()
				return
			}

			// Set the "is announcing" flag to true
			err = db.SetAnnouncingServerFlag(server.Identifier, true)
			if err != nil {
				log.Print(server.Identifier, ": ", err.Error())
				waiter.Done()
				return
			}

			// Fetch all unannounced chapters
			chapters, err := db.GetUnannouncedChapters(server.Identifier)
			if err != nil {
				log.Print(server.Identifier, ": ", err.Error())
				db.SetAnnouncingServerFlag(server.Identifier, false)
				waiter.Done()
				return
			}

			if len(*chapters) > 0 {
				// Send all the chapters
				log.Print("Announcing new chapters for server ", server.Identifier, "...")
				announced := false
				var lastLoggedAt time.Time
				// Loop for each chapter
				for _, chapter := range *chapters {
					_, err = announceChapter(session, &server, &chapter)
					if err != nil {
						log.Print(server.Identifier, ": ", err.Error())
						break
					}

					mentionSubscribers(db, session, &server, &chapter)
					if err != nil {
						log.Print(server.Identifier, ": ", err.Error())
					}

					lastLoggedAt = chapter.LoggedAt
					announced = true
				}

				if announced {
					err = db.SetLastAnnouncedTime(server.Identifier, lastLoggedAt)
					if err != nil {
						log.Print(server.Identifier, ": ", err.Error())
					}
				}
			} else {
				log.Print("No new chapters for server ", server.Identifier)
			}

			// Clear the "is announcing" flag back to false
			err = db.SetAnnouncingServerFlag(server.Identifier, false)
			if err != nil {
				log.Print(server.Identifier, ": ", err.Error())
			}

			log.Print("Announcement process finished for server ", server.Identifier)
			waiter.Done()
		}(s)
	}

	waiter.Wait()

	log.Print("Global announcement process finished")
	return nil
}
