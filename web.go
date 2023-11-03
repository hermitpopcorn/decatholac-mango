// This file handles the web interface

package main

import (
	"log"
	"net/http"
	"os"
)

func startWebInterface() {
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
		if session != nil {
			go startAnnouncers(db)
			w.Write([]byte("Announcement process started."))
		} else {
			w.Write([]byte("Could not start announcement process: no Discord session."))
		}
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
}
