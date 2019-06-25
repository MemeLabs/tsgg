package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const emoteEndpoint = "https://raw.githubusercontent.com/MemeLabs/chat-gui/master/assets/emotes.json"

type emoteEndpointResponse struct {
	Destiny []string `json:"destiny"`
	Twitch  []string `json:"twitch"`
}

func getEmotes() ([]string, error) {
	emotes := make([]string, 0)

	resp, err := http.Get(emoteEndpoint)
	if err != nil {
		return emotes, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return emotes, fmt.Errorf("Emote endpoint status code %d", resp.StatusCode)
	}

	var er emoteEndpointResponse

	err = json.NewDecoder(resp.Body).Decode(&er)
	if err != nil {
		return emotes, err
	}

	emotes = append(emotes, er.Destiny...)
	emotes = append(emotes, er.Twitch...)
	return emotes, nil
}
