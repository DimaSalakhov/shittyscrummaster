package main

import (
	"encoding/json"
	"net/http"

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
	}

}

func respondToVerification(w http.ResponseWriter, msg SlackEvent) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(msg.Challenge))
}
