package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
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
	projector := flag.Int(
		"projector",
		0,
		"ID of the projector you want to connect to. Default is 0 to connect "+
			"to site instead of projector.")
	numberOfWSClients := flag.Int("clients", 500, "Number of clients that should connect to server")
	flag.Parse()

	// Connect to server via websocket
	var path string
	if *projector == 0 {
		path = "/ws/site/"
	} else {
		path = fmt.Sprintf("/ws/projector/%d/", *projector)
	}
	url := fmt.Sprintf("ws://%s:%d%s", *host, *port, path)
	receiveChannel := make(chan bool, *numberOfWSClients)
	wsOpenChannel := make(chan bool, *numberOfWSClients)
	fmt.Printf("Try to connect %d clients to %s\n", *numberOfWSClients, url)
	for i := 0; i < *numberOfWSClients; i++ {
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
			if wsOpenCounter >= *numberOfWSClients {
				fmt.Println("Connections established.")
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
