package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// The timer in milliseconds after witch the status is shown
	showTimer = 100
)

func getWebsocketUrl(host string, port int, path string) string {
	return fmt.Sprintf("ws://%s:%d%s", host, port, path)
}

func getLoginUrl(host string, port int) string {
	return fmt.Sprintf("http://%s:%d/users/login/", host, port)
}

// Logs in with a specific username and password. Returns a session id
func login(host string, port int, username, password string, retry int) (cookie string, err error) {
	resp, err := http.Post(
		getLoginUrl(host, port),
		"application/json",
		strings.NewReader(fmt.Sprintf(
			"{\"username\": \"%s\", \"password\": \"%s\"}",
			username,
			password)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 && resp.StatusCode < 600 && retry > 0 {
		// If the error is on the server side, then retry
		return login(host, port, username, password, retry-1)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("login failed: StatusCode: %d", resp.StatusCode)
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "OpenSlidesSessionID" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("no Session cookie in login response.")
}

// Connects to the websocket url
func connectToWebsocket(host string, port int, path, sessionId string, receiveChannel chan bool, wsOpenChannel chan bool) {
	header := make(http.Header)
	header.Set("Cookie", "OpenSlidesSessionID="+sessionId)
	ws, _, err := websocket.DefaultDialer.Dial(getWebsocketUrl(host, port, path), header)
	if err != nil {
		log.Fatal("Websocket error:", err)
	}
	defer ws.Close()
	wsOpenChannel <- true

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
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
	username := flag.String(
		"username",
		"",
		"Connect with this username. Empty string for anonymous. %i is replaced by "+
			"an number between 1 and the max count of clients.")
	password := flag.String(
		"password",
		"",
		"Use this password for the connection. %i is replaced by a number between 1 "+
			"and max count of clients.")
	flag.Parse()

	// Connect to server via websocket
	var path, sessionId string
	var err error
	if *projector == 0 {
		path = "/ws/site/"
	} else {
		path = fmt.Sprintf("/ws/projector/%d/", *projector)
	}
	receiveChannel := make(chan bool, *numberOfWSClients)
	wsOpenChannel := make(chan bool, *numberOfWSClients)
	fmt.Printf("Try to connect %d clients to %s\n", *numberOfWSClients, getWebsocketUrl(*host, *port, path))

	// Create a sessionId. If username is empty (use anonymous) then we need no
	// login at all. If it contains the placeholder %i then we can not use a global
	// session for all connections and have to set it individualy
	if strings.Contains(*username, "%i") || *username == "" {
		sessionId = ""
	} else {
		sessionId, err = login(*host, *port, *username, *password, 3)
		if err != nil {
			log.Fatal("Login error: ", err)
		}
	}
	for i := 0; i < *numberOfWSClients; i++ {
		go func(sessionId string, clientcount int) {
			if sessionId == "" && *username != "" {
				// If the sessionId was not set in the lines above but the username
				// is not empty, then we it contains the placeholder %i and we have to
				// make the login request for each connection.
				sessionId, err = login(
					*host,
					*port,
					strings.Replace(*username, "%i", strconv.Itoa(clientcount+1), 1),
					strings.Replace(*password, "%i", strconv.Itoa(clientcount+1), 1),
					3)
				if err != nil {
					// Do not exit the test if some connection fail to login. Only print
					// the error.
					log.Print("Login error: ", err)
				}
			}
			connectToWebsocket(*host, *port, path, sessionId, receiveChannel, wsOpenChannel)
		}(sessionId, i)
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
