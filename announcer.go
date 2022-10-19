package main

import (
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

func announceChapter(session *discordgo.Session, channelId string, chapter *chapter) (*discordgo.Message, error) {
	message, err := session.ChannelMessageSendEmbed(channelId, &discordgo.MessageEmbed{
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

func startAnnouncers() error {
	// Get the list of servers
	servers, err := getServers(db)
	if err != nil {
		return err
	}

	// Iterate through servers
	var waiter sync.WaitGroup
	for _, s := range servers {
		waiter.Add(1)

		go func(server server) {
			log.Print("Starting announcement process for server ", server.Identifier, ".")

			var err error = nil
			// Check if the bot is working on announcing the chapters in this server
			var isAnnouncing bool
			isAnnouncing, err = getAnnouncingServerFlag(db, server.Identifier)
			if err != nil {
				log.Print(server.Identifier, err.Error())
				waiter.Done()
				return
			}

			// Cancel if is announcing
			if isAnnouncing {
				waiter.Done()
				return
			}

			// Set the "is announcing" flag to true
			err = setAnnouncingServerFlag(db, server.Identifier, true)
			if err != nil {
				log.Print(server.Identifier, err.Error())
				waiter.Done()
				return
			}

			// Fetch all unnanounced chapters
			chapters, err := getUnannouncedChapters(db, server.Identifier)
			if err != nil {
				log.Print(server.Identifier, err.Error())
				setAnnouncingServerFlag(db, server.Identifier, false)
				waiter.Done()
				return
			}

			if len(*chapters) > 0 {
				// Send all the chapters
				log.Print("Announcing new chapters for server ", server.Identifier, "...")
				inserted := false
				var lastLoggedAt time.Time
				for _, chapter := range *chapters {
					_, err = announceChapter(session, server.FeedChannelIdentifier, &chapter)
					if err != nil {
						break
					}

					lastLoggedAt = chapter.LoggedAt
					inserted = true
				}

				if inserted {
					err = setLastAnnouncedTime(db, server.Identifier, lastLoggedAt)
					if err != nil {
						log.Print(server.Identifier, err.Error())
					}
				}
			} else {
				log.Print("No new chapters for server ", server.Identifier, ".")
			}

			// Clear the "is announcing" flag back to false
			err = setAnnouncingServerFlag(db, server.Identifier, false)
			if err != nil {
				log.Print(server.Identifier, err.Error())
			}

			log.Print("Announcement process finished for server ", server.Identifier, ".")
			waiter.Done()
		}(s)
	}

	waiter.Wait()

	log.Print("Global announcement process finished.")
	return nil
}
