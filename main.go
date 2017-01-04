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
	// The timer in milliseconds after witch the status is shown.
	showTimer = 100
)

// The type posts is a list of ports. Its used for the command line parsing.
// The type satisfies the flag.Value interface with the methods String()
// and Set()
type ports []int

// Returns the ports as string.
func (p *ports) String() string {
	return fmt.Sprintf("%d", *p)
}

// Appends the value to the list of ports.
func (p *ports) Set(value string) error {
	tmp, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("the value %s can not be converted to an integer", value)
	}
	*p = append(*p, tmp)
	return nil
}

// Varialbes for the flags. They are set in the init function and should not be
// changed afterwards.
var (
	hostFlag      string
	portsFlag     ports
	projectorFlag int
	clientsFlag   int
	usernameFlag  string
	passwordFlag  string
	path          string
	showMessages  bool
)

// Parse command line args
func init() {
	flag.StringVar(&hostFlag, "host", "localhost", "Host of OpenSlides daphne server.")
	flag.Var(
		&portsFlag,
		"port",
		"Port of OpenSlides daphne server. Multiple ports can be given to connect "+
			"to more then one daphne server. (default 8000)")

	flag.IntVar(
		&projectorFlag,
		"projector",
		0,
		"ID of the projector you want to connect to. Default is 0 to connect "+
			"to site instead of projector.")

	flag.IntVar(&clientsFlag, "clients", 500, "Number of clients that should connect to server.")
	flag.StringVar(
		&usernameFlag,
		"username",
		"",
		"Connect with this username. Empty string for anonymous. %i is replaced by "+
			"an number between 1 and the max count of clients.")

	flag.StringVar(
		&passwordFlag,
		"password",
		"",
		"Use this password for the connection. %i is replaced by a number between 1 "+
			"and max count of clients.")

	flag.BoolVar(
		&showMessages,
		"showmessages",
		false,
		"Show all messages that the server sends. You should only use this flag with --clients 1.")

	flag.Parse()
	if len(portsFlag) == 0 {
		portsFlag.Set("8000")
	}

	if projectorFlag == 0 {
		path = "/ws/site/"
	} else {
		path = fmt.Sprintf("/ws/projector/%d/", projectorFlag)
	}
}

func getWebsocketURL(port int) string {
	return fmt.Sprintf("ws://%s:%d%s", hostFlag, port, path)
}

// getLoginURL returns the url for login. When more then one port was specified
// this function will only use the first port.
func getLoginURL() string {
	return fmt.Sprintf("http://%s:%d/users/login/", hostFlag, portsFlag[0])
}

// Logs in with a specific username and password. Returns a session id
func login(username, password string, retry int) (cookie string, err error) {
	resp, err := http.Post(
		getLoginURL(),
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
		return login(username, password, retry-1)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("login failed: StatusCode: %d", resp.StatusCode)
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "OpenSlidesSessionID" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("no Session cookie in login response")
}

// Connects to the websocket url
func connectToWebsocket(port int, sessionID string, receiveChannel chan bool, wsOpenChannel chan bool) {
	header := make(http.Header)
	header.Set("Cookie", "OpenSlidesSessionID="+sessionID)
	ws, _, err := websocket.DefaultDialer.Dial(getWebsocketURL(port), header)
	if err != nil {
		log.Fatal("Websocket error:", err)
	}
	defer ws.Close()
	wsOpenChannel <- true

	for {
		_, m, err := ws.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}
		if showMessages {
			fmt.Printf("%s\n", m)
		}
		receiveChannel <- true
	}
}

// Creates a lot of websocket connections to the server
func main() {
	// Connect to server via websocket
	var sessionID string
	var err error
	receiveChannel := make(chan bool, clientsFlag)
	wsOpenChannel := make(chan bool, clientsFlag)
	if len(portsFlag) == 1 {
		fmt.Printf("Try to connect %d clients to %s\n", clientsFlag, getWebsocketURL(portsFlag[0]))
	} else {
		fmt.Println("Try to connect:")
		for i, port := range portsFlag {
			clientCount := clientsFlag / len(portsFlag)
			if clientsFlag%len(portsFlag) > i {
				clientCount++
			}
			fmt.Printf("\t%d clients to %s\n", clientCount, getWebsocketURL(port))
		}
	}

	// Create a sessionID. If username is empty (use anonymous) then we need no
	// login at all. If it contains the placeholder %i then we can not use a global
	// session for all connections and have to set it individualy
	if strings.Contains(usernameFlag, "%i") || usernameFlag == "" {
		sessionID = ""
	} else {
		sessionID, err = login(usernameFlag, passwordFlag, 3)
		if err != nil {
			log.Fatal("Login error: ", err)
		}
	}
	for i := 0; i < clientsFlag; i++ {
		// Connect to the websocket in a different goroutine.
		// If the placeholder is used in the username, then an individualy session
		// will be created for each connection.
		// If more then one port was given, the connections will be spread.
		go func(sessionID string, clientCount int, port int) {
			if sessionID == "" && usernameFlag != "" {
				// If the sessionID was not set in the lines above but the username
				// is not empty, then we it contains the placeholder %i and we have to
				// make the login request for each connection.
				sessionID, err = login(
					strings.Replace(usernameFlag, "%i", strconv.Itoa(clientCount+1), 1),
					strings.Replace(passwordFlag, "%i", strconv.Itoa(clientCount+1), 1),
					3)
				if err != nil {
					log.Fatal("Login error: ", err)
				}
			}
			connectToWebsocket(port, sessionID, receiveChannel, wsOpenChannel)
		}(sessionID, i, portsFlag[i%len(portsFlag)])
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
			if wsOpenCounter >= clientsFlag {
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
