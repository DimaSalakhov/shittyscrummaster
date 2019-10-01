package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const TOKEN = `Bearer xoxb-XXXXXXXXXXXXXXXXX`

func main() {
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02T15:04:05.999-0700"})
	log.SetLevel(log.DebugLevel)
	log.Info("Starting...")

	log.Fatalln(http.ListenAndServe(":8080", router()))
}

func router() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/slack-events", slackEventsHanlder)
	return router
}

type SlackEvent struct {
	Token       string `json:"token"`
	Challenge   string `json:"challenge"` // only used by verification call
	Type        string `json:"type"`
	TeamID      string `json:"team_id"`
	APIAppID    string `json:"api_app_id"`
	Event       json.RawMessage
	AuthedTeams []string `json:"authed_teams"`
	EventID     string   `json:"event_id"`
	EventTime   int      `json:"event_time"`
}

func slackEventsHanlder(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var msg SlackEvent
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		log.WithError(err).Error("failed to decode a slack message")
	}

	log.WithField("event", msg).Debug("Received slack event")

	switch msg.Type {
	case "url_verification":
		respondToVerification(w, msg)
		return
	case "event_callback":
		handleEventCallback(msg.Event)
		w.WriteHeader(http.StatusOK)
		return
	}

}

func respondToVerification(w http.ResponseWriter, msg SlackEvent) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(msg.Challenge))
}

type IMMessage struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	BotID       string `json:"bot_id"`
	Channel     string `json:"channel"`
	User        string `json:"user"`
	Text        string `json:"text"`
	TS          string `json:"ts"`
	EventTS     string `json:"event_ts"`
	ChannelType string `json:"channel_type"`
}

func handleEventCallback(event json.RawMessage) {
	var msg IMMessage
	if err := json.Unmarshal(event, &msg); err != nil {
		log.WithError(err).WithField("event", event).Error("failed to unmarshal event")
	}

	if msg.Subtype == "bot_message" {
		return
	}

	post(msg.Channel, "I'm a certified Scrum Master")
}

func post(channel string, text string) {
	log.WithFields(log.Fields{"text": text}).Debug("sending a message")

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	body, err := json.Marshal(map[string]string{
		"channel": channel,
		"text":    text,
		// "token":   TOKEN,
	})
	if err != nil {
		log.WithError(err).Error("failed to marshal post body")
	}

	req, err := http.NewRequest(http.MethodPost, `https://slack.com/api/chat.postMessage`, bytes.NewBuffer(body))
	if err != nil {
		log.WithError(err).Error("failed to create post request")
	}
	req.Header.Set("Authorization", TOKEN)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("failed to post a message")
		return
	}

	responseBody, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	log.WithField("response", string(responseBody)).Debug("response to a post message")
}
