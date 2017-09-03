package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nlopes/slack"
	"github.com/zmb3/spotify"
)

var results *spotify.SearchResult

func nowPlaying(rtm *slack.RTM, channel string, client *spotify.Client) error {
	postMsgParams := slack.NewPostMessageParameters()
	postMsgParams.AsUser = true
	currentlyPlaying, err := client.PlayerCurrentlyPlaying()
	if err != nil {
		rtm.PostMessage(channel, "Unable to get currently playing track.", postMsgParams)
		return err
	}
	artist := currentlyPlaying.Item.Artists[0].Name
	track := currentlyPlaying.Item.Name
	album := currentlyPlaying.Item.Album.Name
	_, _, err = rtm.PostMessage(channel, fmt.Sprintf("Artist: %s\nTrack: %s\nAlbum: %s", artist, track, album), postMsgParams)
	if err != nil {
		return err
	}
	return nil
}

func search(rtm *slack.RTM, text, channel string, client *spotify.Client) error {
	var searchMap map[string]string
	var searchSlice []string
	searchMap = make(map[string]string)

	postParams := slack.NewPostMessageParameters()
	postParams.AsUser = true

	// First we remove the term search from the beginning of the string
	// and trim any trailing space
	text = strings.TrimPrefix(text, "search")
	text = strings.TrimSpace(text)

	// Split on the seperator to turn the string into a slice of strings
	// for each section of the track to be searched
	searchSlice = strings.Split(text, ",")

	for _, str := range searchSlice {
		// Split the string into parts on the seperator
		// e.g. for the string "Artist: Some Person"
		parts := strings.Split(str, ":")

		// Place the parts into the map with index 0 as the key and index 1 as the value
		// trimming any trailing space
		parts[0] = strings.ToLower(parts[0])
		parts[1] = strings.ToLower(parts[1])
		searchMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	// Check to ensure that all of the required parameters have been specified in the command
	if _, ok := searchMap["artist"]; !ok {
		rtm.PostMessage(channel, fmt.Sprintf("Missing parameter Artist in search command\nSearch should be in the format: `@bot search Artist: Someone, Track: Some Track`"), postParams)
		return fmt.Errorf("Missing parameter Artist in search command\nSearch should be in the format: `@bot search Artist: Someone, Track: Some Track`")
	}

	if _, ok := searchMap["track"]; !ok {
		rtm.PostMessage(channel, fmt.Sprintf("Missing parameter Track in search command\nSearch should be in the format: `@bot search Artist: Someone, Track: Some Track`"), postParams)
		return fmt.Errorf("Missing parameter Track in search command\nSearch should be in the format: `@bot search Artist: Someone, Track: Some Track`")
	}

	searchQuery := fmt.Sprintf("artist:%s track:%s", searchMap["artist"], searchMap["track"])
	results, err := client.Search(searchQuery, spotify.SearchTypeTrack)
	if err != nil {
		return err
	}

	// If no results are found send a message to notify of this and return an error
	if len(results.Tracks.Tracks) < 1 {
		rtm.PostMessage(channel, fmt.Sprintf("No results found for Artist: %s and Track: %s", searchMap["artist"], searchMap["track"]), postParams)
		return errors.New("no results found for query")
	}

	// Range over the results, append them to the results message and send a slack message to display these
	var resultsMsg string
	for _, result := range results.Tracks.Tracks {
		/*postParams.Attachments = []slack.Attachment{
			{Actions: []slack.AttachmentAction{
				{Name: "Add",
					Text: "Add",
					Type: "select",
					Options: []slack.AttachmentActionOption{
						{Text: "Song Text",
							Value: "Song Value"},
					}},
			}},
		}*/
		resultsMsg = fmt.Sprintf("%s\nTrack Title: %s\nArtist: %s\nAlbum: %s\nTrackID: %s\n", resultsMsg, result.Name, result.Artists[0].Name, result.Album.Name, result.ID)
	}

	_, _, err = rtm.PostMessage(channel, resultsMsg, postParams)
	if err != nil {
		return err
	}

	return nil
}

func addTrackToPlayList(rtm *slack.RTM, text, channel string, client *spotify.Client, playlist spotify.SimplePlaylist) error {
	text = strings.TrimPrefix(text, "add")
	text = strings.TrimSpace(text)

	postParams := slack.NewPostMessageParameters()
	postParams.AsUser = true

	id := spotify.ID(text)

	user, err := client.CurrentUser()

	_, err = client.AddTracksToPlaylist(user.ID, playlist.ID, id)

	if err != nil {
		return err
	}

	var addedMsg string

	addedMsg = "Song successfully added"

	_, _, err = rtm.PostMessage(channel, addedMsg, postParams)
	if err != nil {
		return err
	}

	return nil
}
