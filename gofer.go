// Gofers are small processes that fetches, parses, and saves chapter data.

package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hermitpopcorn/decatholac-mango/database"
	"github.com/hermitpopcorn/decatholac-mango/helpers"
	"github.com/hermitpopcorn/decatholac-mango/parsers"
	"github.com/hermitpopcorn/decatholac-mango/types"
)

// To make sure not more than one gofer is running for one target,
// this flag is raised to "true" when startGofers is called,
// and startGofers can't run unless it's set to false.
var currentlyFetchingTargets = false

// This is just a custom error that's thrown whenever
// startGofers() is called when the above flag is still up.
type PreoccupiedError struct{}

func (e *PreoccupiedError) Error() string {
	return "The gofer is not done fetching yet"
}

// This turns a source URL into a string containing the response body.
func fetchBody(url string, headers map[string]string) (string, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// This fetches the source and then parses it according to the specified mode.
func fetchChapters(target *types.Target) ([]types.Chapter, error) {
	var body string
	var chapters []types.Chapter
	var err error

	if target.Mode == "json" {
		body, err = fetchBody(target.Source, target.RequestHeaders)
		if err != nil {
			return nil, err
		}
		chapters, err = parseJson(target, &body)
		if err != nil {
			return nil, err
		}
	} else if target.Mode == "rss" {
		body, err = fetchBody(target.Source, target.RequestHeaders)
		if err != nil {
			return nil, err
		}
		chapters, err = parseRss(target, &body)
		if err != nil {
			return nil, err
		}
	} else if target.Mode == "html" {
		body, err = fetchBody(target.Source, target.RequestHeaders)
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
func startGofer(waiter *sync.WaitGroup, db database.Database, target types.Target) {
	var chapters []types.Chapter
	var err error

	fmt.Println(helpers.FormattedNow(), "Gofer started for", target.Name)

	// Try fetching the source five times
	var attempts uint = 5
	for attempts = 5; attempts > 0; attempts-- {
		chapters, err = fetchChapters(&target)
		if err != nil {
			fmt.Println(helpers.FormattedNow(), target.Name+":", "Failed fetching:", err.Error(), "| Remaining attempt(s):", attempts)
			time.Sleep(5 * time.Second)
			continue
		}

		break
	}
	if attempts == 0 {
		fmt.Println(helpers.FormattedNow(), target.Name+":", "Failed all fetching attempts")
		waiter.Done()
		return
	}

	// Save the chapters to DB
	var retry = 10
	var saved = false
	for retry > 0 {
		err = db.SaveChapters(&chapters)
		if err == nil {
			retry = 0
			saved = true
		} else {
			if strings.HasPrefix(err.Error(), "database is locked") {
				fmt.Println(helpers.FormattedNow(), target.Name+":", "Thread busy. Retrying...")
				retry -= 1
			} else {
				retry = 0
			}
		}
	}

	if saved {
		fmt.Println(helpers.FormattedNow(), target.Name+":", "Gofer finished")
	} else {
		fmt.Println(helpers.FormattedNow(), target.Name+":", "Failed saving chapters:", err.Error())
	}

	waiter.Done()
}

// This is the "mother" gofer process.
// It runs one gofer for every target.
func startGofers(db database.Database, targets *[]types.Target) error {
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
	fmt.Println(helpers.FormattedNow(), "Fetch process finished")
	return nil
}
