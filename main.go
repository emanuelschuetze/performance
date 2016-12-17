package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// URL of the websocket server
	url = "ws://localhost:8000/ws/site/"

	// Number of websocket clients that connect to the server
	wsClientCount = 500

	// The timer in milliseconds after witch the status is shown
	showTimer = 100
)

// Connects to the websocket url
func connectToWebsocket(receiveChannel chan bool, wsOpenChannel chan bool) {
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
	receiveChannel := make(chan bool, wsClientCount)
	wsOpenChannel := make(chan bool, wsClientCount)
	for i := 0; i < wsClientCount; i++ {
		go connectToWebsocket(receiveChannel, wsOpenChannel)
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
				fmt.Println("All clients are connected")
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
