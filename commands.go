package main

import (
	"fmt"

	"github.com/nlopes/slack"
	"github.com/zmb3/spotify"
)

func nowPlaying(rtm *slack.RTM, channel string, playerState *spotify.PlayerState) error {
	artist := playerState.CurrentlyPlaying.Item.Artists[0].Name
	track := playerState.CurrentlyPlaying.Item.Name
	album := playerState.CurrentlyPlaying.Item.Album.Name
	postMsgParams := slack.NewPostMessageParameters()
	postMsgParams.AsUser = true
	_, _, err := rtm.PostMessage(channel, fmt.Sprintf("Artist: %s\nTrack: %s\nAlbum: %s", artist, track, album), postMsgParams)
	if err != nil {
		return err
	}
	return nil
}
