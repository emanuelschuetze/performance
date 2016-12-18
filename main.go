package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Number of websocket clients that connect to the server
	wsClientCount = 500

	// The timer in milliseconds after witch the status is shown
	showTimer = 100
)

// Connects to the websocket url
func connectToWebsocket(url string, receiveChannel chan bool, wsOpenChannel chan bool) {
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("websocket error:", err)
	}
	defer ws.Close()
	wsOpenChannel <- true

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}
		receiveChannel <- true
	}
}

// Creates a lot of websocket connections to the server
func main() {
	// Parse command line args
	host := flag.String("host", "localhost", "Host of OpenSlides daphne server")
	port := flag.Int("port", 8000, "Port of OpenSlides daphne server")
	flag.Parse()

	// Connect to server via websocket
	url := fmt.Sprintf("ws://%s:%d/ws/site/", *host, *port)
	receiveChannel := make(chan bool, wsClientCount)
	wsOpenChannel := make(chan bool, wsClientCount)
	fmt.Printf("Try to connect to %s ... ", url)
	for i := 0; i < wsClientCount; i++ {
		go connectToWebsocket(url, receiveChannel, wsOpenChannel)
	}

	wsOpenCounter := 0
	receiveCounter := 0
	receiveAllCounter := 0
	tickCounter := 0
	emptyCounter := 0
	tick := time.Tick(showTimer * time.Millisecond)

	for {
		select {
		case <-wsOpenChannel:
			wsOpenCounter++
			if wsOpenCounter >= wsClientCount {
				fmt.Println("done.")
			}
		case <-receiveChannel:
			receiveCounter++
			receiveAllCounter++
		case <-tick:
			tickCounter++
			if receiveCounter != 0 {
				if emptyCounter != 0 {
					fmt.Printf("--- %d ms without data ---\n", emptyCounter*showTimer)
				}
				fmt.Printf("%d\tReceived blobs: %d (all: %d)\n", tickCounter, receiveCounter, receiveAllCounter)
				receiveCounter = 0
				emptyCounter = 0
			} else {
				emptyCounter++
			}
		}
	}
}
