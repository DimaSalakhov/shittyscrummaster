package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const TOKEN = `Bearer xoxb-XXXXXXXXXXXXXXXXX`

var reportChannel string

func main() {
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02T15:04:05.999-0700", PrettyPrint: true})
	log.SetLevel(log.DebugLevel)
	log.Info("Starting...")

	log.Fatalln(http.ListenAndServe(":8080", router()))
}

func router() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/slack-events", slackEventsHanlder)
	router.HandleFunc("/slash-config", slashConfigHanlder)
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

	s, ok := inmemoryStore[msg.User]
	if !ok && msg.Text != "start" {
		post(msg.Channel, "Type `start` when you're ready")
		return
	}

	if msg.Text == "start" {
		inmemoryStore[msg.User] = &summary{}
		post(msg.Channel, "1. What did you accomplish yesterday?")
		return
	}

	switch s.Progress {
	case Started:
		s.Yesterday = msg.Text
		s.Progress = YesterdayDone
		post(msg.Channel, "2. What are you working on today?")
		return
	case YesterdayDone:
		s.Today = msg.Text
		s.Progress = TodayDone
		post(msg.Channel, "3. Is anything standing in your way?")
		return
	case TodayDone:
		s.Misc = msg.Text
		s.Progress = Finished

		var echo strings.Builder
		echo.WriteString(fmt.Sprintf("<@%s>\n", msg.User))
		echo.WriteString("1. What did you accomplish yesterday?\n")
		echo.WriteString(s.Yesterday)
		echo.WriteString("\n2. What are you working on today?\n")
		echo.WriteString(s.Today)
		echo.WriteString("\n3. Is anything standing in your way?\n")
		echo.WriteString(s.Misc)
		post(reportChannel, echo.String())
		post(msg.Channel, "Well done! If you want to start again, just type `start`")

		delete(inmemoryStore, msg.User)
		return
	}

	post(msg.Channel, "I'm a certified Scrum Master")
}

var inmemoryStore = make(map[string]*summary, 10)

type summary struct {
	Yesterday string
	Today     string
	Misc      string
	Progress  progress
}

type progress int

const (
	Started       progress = 0
	YesterdayDone progress = 1
	TodayDone     progress = 2
	Finished      progress = 3
)

func post(channel string, text string) {
	log.WithFields(log.Fields{"text": text}).Debug("sending a message")

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	body, err := json.Marshal(map[string]string{
		"channel": channel,
		"text":    text,
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

func slashConfigHanlder(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"url": r.URL.RequestURI()}).Debug("received slash command")

	w.WriteHeader(http.StatusOK)

	err := r.ParseForm()
	if err != nil {
		log.WithError(err).Errorf("failed to parse slash message @q", r.URL.RequestURI())
	}

	channel := r.FormValue("channel_id")
	text := r.FormValue("text")

	const reportHereCommand = "report here"

	if text != "report here" {
		w.Write([]byte(fmt.Sprintf("Sorry, I don't know this command. Please use `@s` command", reportHereCommand)))
		return
	}

	reportChannel = channel

	body := []byte("Thanks! I'll ask to report to that channel")
	w.Write(body)
}
