package asker

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
)

type Asker struct {
	OAuth   string
	Token   string
	api     *slack.Client
	storage *Storage
}

//var AskQueue map[string]*SlashCommand = make(map[string]int)

var AskQueue = make(map[string]*SlashCommand)

func NewAsker(oAuthToken string, token string, mongodb string) (*Asker, error) {
	storage, err := NewStorage(mongodb, "slack-asker", "channel_configs")
	if err != nil {
		return nil, err
	}

	client := Asker{
		OAuth:   oAuthToken,
		Token:   token,
		api:     slack.New(oAuthToken),
		storage: storage,
	}

	return &client, nil
}

func (a *Asker) Listen(addr string) {
	defer a.storage.CloseStorage()

	r := mux.NewRouter()
	r.HandleFunc("/events/ask", a.AskHandler)
	r.HandleFunc("/events/request", a.RequestHandler)
	r.HandleFunc("/events/options", a.OptionsHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    addr,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func (a *Asker) handleChannelLink(command *SlashCommand) (string, error) {
	s := strings.Split(command.Text, " ")
	cmd, project := s[0], s[1]
	if cmd != "link" {
		return "", fmt.Errorf("Invalid command, use /%s link <jira project>", command.Command)
	}

	err := a.storage.SetChannelProject(command.ChannelID, project)
	return project, err
}

func (a *Asker) AskHandler(w http.ResponseWriter, r *http.Request) {
	command, err := a.parseSlashCommand(r)
	if err != nil {
		log.Printf("Failed parsing slash command: %+v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
	}

	// Handle a link command, which doesn't open a dialog
	if strings.HasPrefix(command.Text, "link ") {
		project, err := a.handleChannelLink(command)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, fmt.Sprintf("Unable to set the channel's project: %+v", err))
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, fmt.Sprintf("Got it, set %s JIRA project to %s", command.ChannelName, project))
		return
	}

	var callbackID = fmt.Sprintf("ask-%d", time.Now().UnixNano())
	AskQueue[callbackID] = command

	log.Printf("Got incoming /events/ask request, deserialize request:\n%+v\n", command)
	a.OpenDialog(callbackID, command.TriggerID)
	w.WriteHeader(http.StatusOK)
	// Send an empty response, because we'll use the responseURL later
	fmt.Fprintf(w, "")
}

func (a *Asker) RequestHandler(w http.ResponseWriter, r *http.Request) {
	request, err := a.parseInteractiveRequest(r)
	if err != nil {
		log.Printf("Failed parsing slash command: %+v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
	}

	log.Printf("Looking in AskQueue for %s\n", request.CallbackID)
	if originalAsk, ok := AskQueue[request.CallbackID]; ok {
		log.Printf("I found the original ask! Post something to response_url: %v\n", originalAsk.ResponseURL)
		if err = a.PostAskResult(originalAsk, request); err != nil {
			log.Printf("Unable to post response back: %v\n", err)
		}
	}

	log.Printf("Got a request coming in for the dialog: \"%s\"\n", request.Submission["summary"])
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "")
}

func (a *Asker) OptionsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Got an options request")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Ok")
}

func (a *Asker) GetGroups() {
	groups, err := a.api.GetGroups(false)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for _, group := range groups {
		fmt.Printf("ID: %s, Name: %s\n", group.ID, group.Name)
	}
}
