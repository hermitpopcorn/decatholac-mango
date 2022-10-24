// Gofers are small processes that fetches, parses, and saves chapter data.

package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// To make sure not more than one gofer is running for one target,
// this flag is raised to "true" when startGofers is called,
// and startGofers can't run unless it's set to false.
var currentlyFetchingTargets = false

// This is just a custom error that's thrown whenever
// startGofers() is called when the above flag is still up.
type PreoccupiedError struct{}

func (e *PreoccupiedError) Error() string {
	return "The gofer is not done fetching yet."
}

// This turns a source URL into a string containing the response body.
func fetchBody(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// This fetches the source and then parses it according to the specified mode.
func fetchChapters(target *target) ([]chapter, error) {
	var body string
	var chapters []chapter
	var err error

	if target.Mode == "json" {
		body, err = fetchBody(target.Source)
		if err != nil {
			return nil, err
		}
		chapters, err = parseJson(target, &body)
		if err != nil {
			return nil, err
		}
	} else if target.Mode == "rss" {
		body, err = fetchBody(target.Source)
		if err != nil {
			return nil, err
		}
		chapters, err = parseRss(target, &body)
		if err != nil {
			return nil, err
		}
	} else if target.Mode == "html" {
		body, err = fetchBody(target.Source)
		if err != nil {
			return nil, err
		}
		chapters, err = parseHtml(target, &body)
		if err != nil {
			return nil, err
		}
	}

	return chapters, nil
}

// This starts a gofer process.
// It doesn't return anything because it's supposed to be called in a goroutine.
// It does take a *sync.WaitGroup as a parameter so it can tell the main process that it's done, though.
func startGofer(waiter *sync.WaitGroup, db *sql.DB, target target) {
	var chapters []chapter
	var err error

	log.Print("Gofer started for ", target.Name)

	// Try fetching the source five times
	var attempts uint = 5
	for attempts = 5; attempts > 0; attempts-- {
		chapters, err = fetchChapters(&target)
		if err != nil {
			log.Print(target.Name, ": ", "Failed fetching: ", err.Error(), "| Remaining attempt(s):", attempts)
			continue
		}

		break
	}
	if attempts == 0 {
		log.Print(target.Name, ": ", "Failed all fetching attempts.")
		waiter.Done()
		return
	}

	// Save the chapters to DB
	err = saveChapters(db, &chapters)
	if err != nil {
		log.Print(target.Name, ": ", "Failed saving chapters: ", err.Error())
		waiter.Done()
		return
	}

	log.Print(target.Name, ": ", "Gofer finished.")

	waiter.Done()
}

// This is the "mother" gofer process.
// It runs one gofer for every target.
func startGofers(targets *[]target, db *sql.DB) error {
	// Set on progress flag; cancel if it's up
	if currentlyFetchingTargets {
		return &PreoccupiedError{}
	}
	currentlyFetchingTargets = true

	// Iterate through targets
	var waiter sync.WaitGroup
	for _, target := range *targets {
		waiter.Add(1)

		// Send gofer to work in a parallel process
		go startGofer(&waiter, db, target)
	}

	waiter.Wait()

	// Give it some time to rest
	time.Sleep(30 * time.Second)

	// Take down flag and return
	currentlyFetchingTargets = false
	log.Print("Fetch process finished.")
	return nil
}
