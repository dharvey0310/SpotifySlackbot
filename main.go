package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
	"github.com/zmb3/spotify"
)

var (
	auth          = spotify.NewAuthenticator("http://localhost:8080/callback", spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState, spotify.ScopePlaylistModifyPublic)
	ch            = make(chan *spotify.Client)
	state         = "testslackbot"
	slacktoken    string
	playlistName  string
	bannedArtists map[string]string
)

func init() {
	slacktoken = os.Getenv("SLACK_TOKEN")
	playlistName = os.Getenv("SPOTIFY_PLAYLIST")
}

func main() {
	var client *spotify.Client
	var playerState *spotify.PlayerState
	var userPlaylists *spotify.SimplePlaylistPage
	var user *spotify.PrivateUser
	var playlist spotify.SimplePlaylist
	bannedArtists = make(map[string]string)

	http.HandleFunc("/callback", completeAuth)
	go http.ListenAndServe(":8080", nil)

	go func(userPlaylists *spotify.SimplePlaylistPage, user *spotify.PrivateUser) {
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

		userPlaylists, err = client.GetPlaylistsForUser(user.ID)

		if err != nil {
			fmt.Println(err)
		}

		for _, p := range userPlaylists.Playlists {
			if p.Name == playlistName {
				playlist = p
			}
		}

		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
	}(userPlaylists, user)

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
			//fmt.Println("hello")

		case *slack.ConnectedEvent:
			/*fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)

			*/

		case *slack.MessageEvent:

			info := rtm.GetInfo()
			prefix := fmt.Sprintf("<@%s> ", info.User.ID)

			if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
				respond(rtm, ev, prefix, client, playlist)
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

func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string, client *spotify.Client, playlist spotify.SimplePlaylist) {
	var err error
	text := msg.Text

	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	if ok := strings.HasPrefix(text, "search"); ok {
		err = search(rtm, text, msg.Channel, client)
	}

	if strings.HasPrefix(text, "add") {
		err = addTrackToPlayList(rtm, text, msg.Channel, client, playlist)
	}

	if ok := strings.HasPrefix(text, "volume"); ok {
		text = strings.TrimPrefix(text, "volume")
		text = strings.TrimSpace(text)

		volume, err := strconv.Atoi(text)

		if err != nil {
			log.Println(err)
		}

		client.Volume(volume)
	}

	if ok := strings.HasPrefix(text, "ban"); ok {
		err = addToBannedList(rtm, text, msg.Channel, msg.User, client, playlist)
	}

	text = strings.ToLower(text)
	switch text {
	case "play":
		err = client.Play()
	case "pause":
		err = client.Pause()
	case "next":
		err = client.Next()
	case "previous":
		err = client.Previous()
	case "now playing":
		err = nowPlaying(rtm, msg.Channel, client)
	}
	if err != nil {
		log.Print(err)
	}
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
