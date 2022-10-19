package main

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

var currentlyFetchingTargets = false

type PreoccupiedError struct{}

func (e *PreoccupiedError) Error() string {
	return "The gofer is not done fetching yet."
}

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
	}

	return chapters, nil
}

func startGofer(waiter *sync.WaitGroup, target target) {
	var chapters []chapter
	var err error

	log.Print("Gofer started for: ", target.Name)

	// Try fetching the source five times
	var attempts uint = 5
	for attempts = 5; attempts > 0; attempts-- {
		chapters, err = fetchChapters(&target)
		if err != nil {
			log.Print("Failed fetching", err.Error(), "remaining attempt(s):", attempts)
			continue
		}

		break
	}
	if attempts == 0 {
		log.Print("Failed all fetching attempts for", target.Name)
		return
	}

	// Save the chapters to DB
	err = saveChapters(db, &chapters)
	if err != nil {
		log.Print("Failed saving chapters:", err.Error())
		return
	}

	log.Print("Gofer finished for: ", target.Name)

	waiter.Done()
}

func startGofers(targets *map[string]target) error {
	// Set on progress flag; cancel if it's up
	if currentlyFetchingTargets {
		return &PreoccupiedError{}
	}
	currentlyFetchingTargets = true

	// Iterate through targets
	var waiter sync.WaitGroup
	for _, target := range *targets {
		waiter.Add(1)

		// Send gofer to work
		go startGofer(&waiter, target)
	}

	waiter.Wait()

	// Give it some time to rest
	time.Sleep(30 * time.Second)

	currentlyFetchingTargets = false
	return nil
}
