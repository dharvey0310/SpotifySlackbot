package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nlopes/slack"
	"github.com/zmb3/spotify"
)

var (
	auth       = spotify.NewAuthenticator("http://localhost:8080/callback", spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	ch         = make(chan *spotify.Client)
	state      = "testapp"
	slacktoken string
)

func init() {
	slacktoken = os.Getenv("SLACK_TOKEN")
}

func main() {
	var client *spotify.Client
	var playerState *spotify.PlayerState

	http.HandleFunc("/callback", completeAuth)
	go http.ListenAndServe(":8080", nil)

	go func() {
		url := auth.AuthURL(state)
		fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

		// wait for auth to complete
		client = <-ch

		// use the client to make calls that require authorization
		user, err := client.CurrentUser()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("You are logged in as:", user.ID)

		playerState, err = client.PlayerState()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
	}()

	api := slack.New(slacktoken)
	//logger := log.New(os.Stdout, "slack", log.Lshortfile|log.LstdFlags)
	//slack.SetLogger(logger)
	//api.SetDebug(true)

	rtm := api.NewRTM()

	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		//fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			/*fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)

			*/

		case *slack.MessageEvent:

			info := rtm.GetInfo()
			prefix := fmt.Sprintf("<@%s> ", info.User.ID)

			if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
				respond(rtm, ev, prefix, client)
			}

		case *slack.PresenceChangeEvent:
			//fmt.Printf("Presence Change: %v\n", ev)

		case *slack.LatencyReport:
			//fmt.Printf("Current latency: %v\n", ev.Value)

		case *slack.RTMError:
			//fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			//fmt.Printf("Invalid credentials")
			return

		default:

			// Ignore other events..
			// fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}
}

func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string, client *spotify.Client) {
	text := msg.Text
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)
	var err error
	switch text {
	case "play":
		err = client.Play()
	case "pause":
		err = client.Pause()
	case "next":
		err = client.Next()
	case "previous":
		err = client.Previous()
	}
	if err != nil {
		log.Print(err)
	}

	client.Play()

	fmt.Printf("%v", text)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}
