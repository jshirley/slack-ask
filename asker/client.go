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
	Jira    *JiraClient
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
	r.HandleFunc("/", a.RootHandler)
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

func (a *Asker) RootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "I am alive, but there is nothing to see here.")
}

func (a *Asker) handleConfigCommand(command *SlashCommand, w http.ResponseWriter, r *http.Request) {
	commands := strings.Split(command.Text, " ")

	config, err := a.storage.GetChannelConfig(command.ChannelID)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, fmt.Sprintf("Unable to load configuration (do you need to `/ask link <PROJECTKEY>` first?)\nThe error from storage is: %+v", err))
		return
	}

	if len(commands) == 1 {
		w.WriteHeader(http.StatusOK)
		components := strings.Join(config.Components, ", ")
		if components == "" {
			components = "None! Use `/ask config components Component1 Component2` to set them"
		}
		fmt.Fprintf(w, fmt.Sprintf("This channel is set to JIRA project: %s\nDefault components: %s", config.Project, components))
	} else if commands[1] == "components" {
		config.Components = commands[2:len(commands)]
		err := a.storage.SetChannelConfig(config)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, fmt.Sprintf("Unable to store configuration: %+v", err))
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, fmt.Sprintf("Got it, default components for `%s` are now %v!", config.Project, config.Components))
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, fmt.Sprintf("Invalid config command. Available options are `/ask config`, `/ask config components Comp1 Comp2`, and maybe more. Patches welcome!"))
	}
}

func (a *Asker) AskHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got an incoming /ask command, validating\n")
	command, err := a.parseSlashCommand(r)
	if err != nil {
		log.Printf("Failed parsing or verifying slash command: %+v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
	}

	log.Printf("Handling valid /ask command\n")
	if strings.HasPrefix(command.Text, "config") {
		a.handleConfigCommand(command, w, r)
		return
	} else if strings.HasPrefix(command.Text, "link ") {
		project, err := a.handleChannelLink(command)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, fmt.Sprintf("Unable to set the channel's project: %+v", err))
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, fmt.Sprintf("Got it, set %s JIRA project to %s", command.ChannelName, project))
		return
	}

	config, err := a.storage.GetChannelConfig(command.ChannelID)
	if err != nil {
		log.Printf("Failed fetching channel configuration from storage: +%v\n", err)
		w.WriteHeader(http.StatusOK)
		// Send an empty response, because we'll use the responseURL later
		fmt.Fprintf(w, "There is no /ask project configured for this channel. Use /ask link <PROJECT KEY> to link this channel to a JIRA project.")
		return
	}

	command.Config = config
	var callbackID = fmt.Sprintf("ask-%d", time.Now().UnixNano())
	AskQueue[callbackID] = command

	log.Printf("Got incoming /ask request, deserialize request:\n%+v\n", command)
	a.OpenDialog(callbackID, config, command.TriggerID)
	w.WriteHeader(http.StatusOK)
	// Send an empty response, because we'll use the responseURL later
	fmt.Fprintf(w, "")
}

func (a *Asker) RequestHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling incoming response for dialog, verifying authenticity\n")
	request, err := a.parseInteractiveRequest(r)
	if err != nil {
		log.Printf("Failed verifying interactive request: %+v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
	}

	if originalAsk, ok := AskQueue[request.CallbackID]; ok {
		if err = a.PostAskResult(originalAsk, request); err != nil {
			log.Printf("Unable to post response back: %v\n", err)
		}
		delete(AskQueue, request.CallbackID)
	} else {
		log.Printf("This is strange! We have a dialog with request ID %s, but that is not in the queue\n", request.CallbackID)
	}

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

const TIMEOUT = 60 * 5 // 5 Minutes
func (a *Asker) CleanQueue() {
	for {
		<-time.After(5 * time.Minute)
		go a.removeQueueItems()
	}
}

func (a *Asker) removeQueueItems() {
	for callbackID, command := range AskQueue {
		if command.Timestamp < time.Now().Unix()-TIMEOUT {
			delete(AskQueue, callbackID)
		}
	}
}
