package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type (
	Request   map[string][]string
	Response  []string
	inChannel struct {
		position int
		url      string
	}
	outChannel struct {
		position int
		status   string
	}
)

func main() {
	// Launching our server
	http.HandleFunc("/parse", parseURL)
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func parseURL(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Create workers and channels
	wg := new(sync.WaitGroup)
	inCH := make(chan inChannel)
	outCH := make(chan outChannel)
	// Start workers
	for i := 0; i < 250; i++ {
		// Increase the WaitGroup counter
		wg.Add(1)
		go getHTML(inCH, outCH, wg)
	}
	// Decoding the json request into a variable
	decoder := json.NewDecoder(r.Body)
	var request Request
	err := decoder.Decode(&request)
	if err != nil {
		log.Fatal(err)
	}
	// Getting the HTML statuses
	statuses := getStatuses(request["urls"], inCH, outCH, wg)
	// Encode the statuses variable into JSON
	resp, err := json.Marshal(statuses)
	if err != nil {
		panic(err)
	}
	// Ending the function
	elapsed := time.Since(start)
	log.Println("\n===================================\n Completed in:", elapsed, "\n===================================\n")
	fmt.Fprintln(w, string(resp))
}

func getStatuses(urls []string, inCH chan inChannel, outCH chan outChannel, wg *sync.WaitGroup) []string {
	statuses := make([]string, len(urls))
	// Create a goroutine that listens for the outChannel.
	go func() {
		// Infinite loop listening for incoming values on the outChannel. It will run until the outCH is closed
		for out := range outCH {
			statuses[out.position] = out.status
		}
	}()
	// Pushing through the inChannel the URLs along with the the URL position within the POST request
	for k, url := range urls {
		inCH <- inChannel{position: k, url: url}
	}
	// Close the inChannel
	close(inCH)
	// Wait for all goroutines to finish
	wg.Wait()
	// Closing the outChannel
	close(outCH)
	// Return the statuses
	return statuses
}

func getHTML(inCH chan inChannel, outCH chan outChannel, wg *sync.WaitGroup) {
	defer wg.Done()
	// Infinite loop listening for any value coming through the channel. This will run until the inChannel is closed.
	for in := range inCH {
		html, err := http.Get(in.url)
		if err != nil {
			log.Println(err)
		}
		// Log the url - for demonstration purposes only
		log.Println(in.url)
		// Push the HTML Status through the outChannel along with the URL position within the POST request
		outCH <- outChannel{position: in.position, status: html.Status}
	}
}
