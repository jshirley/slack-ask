package asker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jshirley/slack-ask/storage"

	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
)

type Asker struct {
	OAuth          string
	Token          string
	api            *slack.Client
	storage        storage.Session
	Jira           *JiraClient
	dialogElements []DialogElement
}

func NewAsker(oAuthToken string, token string, mongodb string) (*Asker, error) {
	client := Asker{
		OAuth:   oAuthToken,
		Token:   token,
		api:     slack.New(oAuthToken),
		storage: storage.NewSession(mongodb),
	}

	return &client, nil
}

func (a *Asker) Listen(addr string) {
	defer a.storage.Close()

	r := mux.NewRouter()
	r.HandleFunc("/", a.RootHandler)
	r.HandleFunc("/events/ask", a.AskHandler)
	r.HandleFunc("/events/request", a.DialogRequestHandler)
	r.HandleFunc("/events/options", a.OptionsHandler)

	//http.Handle("/", StorageMiddleware(r, a.storage))
	http.ListenAndServe(addr, StorageMiddleware(r, a.storage))

	/*
		srv := &http.Server{
			Handler: r,
			Addr:    addr,
			// Good practice: enforce timeouts for servers you create!
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}

		log.Fatal(srv.ListenAndServe())
	*/
}

func StorageMiddleware(next http.Handler, session storage.Session) http.Handler {
	log.Println("Setting up storage middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbSession := session.Copy()

		r = r.WithContext(context.WithValue(r.Context(), "db", dbSession))
		next.ServeHTTP(w, r)

		dbSession.Close()
	})
}

// MgoSessionFromCtx takes a context argument and return the related *mgo.session.
func MgoSessionFromCtx(ctx context.Context) storage.Session {
	mgoSession, ok := ctx.Value("db").(storage.Session)
	if ok != true {
		panic(fmt.Errorf("Unable to cast value from context to mgoSession (value is %+v)", ctx.Value("db")))
	}
	return mgoSession
}

// MgoDBFromR takes a request argument and return the extracted *mgo.session.
func MgoDBFromRequest(r *http.Request) storage.DataLayer {
	return MgoSessionFromCtx(r.Context()).DB("slack-ask")
}

func (a *Asker) handleChannelLink(db storage.DataLayer, command *storage.SlashCommand) (string, error) {
	s := strings.Split(command.Text, " ")
	cmd, project := s[0], s[1]
	if cmd != "link" {
		return "", fmt.Errorf("Invalid command, use /%s link <jira project>", command.Command)
	}

	err := db.SetChannelProject(command.ChannelID, project)
	return project, err
}

func (a *Asker) RootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "I am alive, but there is nothing to see here")
}

func (a *Asker) handleConfigCommand(db storage.DataLayer, command *storage.SlashCommand, w http.ResponseWriter, r *http.Request) {
	commands := strings.Split(command.Text, " ")

	config, err := db.GetChannelConfig(command.ChannelID)
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
		err := db.SetChannelConfig(config)
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
	command, err := a.parseSlashCommand(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
	}

	db := MgoDBFromRequest(r)
	if strings.HasPrefix(command.Text, "config") {
		a.handleConfigCommand(db, command, w, r)
		return
	} else if strings.HasPrefix(command.Text, "link ") {
		project, err := a.handleChannelLink(db, command)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, fmt.Sprintf("Unable to set the channel's project: %+v", err))
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, fmt.Sprintf("Got it, set %s JIRA project to %s", command.ChannelName, project))
		return
	}

	config, err := db.GetChannelConfig(command.ChannelID)
	if err != nil {
		log.Printf("Failed fetching channel configuration from storage: +%v\n", err)
		w.WriteHeader(http.StatusOK)
		// Send an empty response, because we'll use the responseURL later
		fmt.Fprintf(w, "There is no /ask project configured for this channel. Use /ask link <PROJECT KEY> to link this channel to a JIRA project.")
		return
	}

	command.Config = config

	var callbackID = fmt.Sprintf("ask-%s-%d", command.ChannelID, time.Now().UnixNano())
	err = db.StoreCallback(callbackID, command)
	log.Printf("Storing %s into Mongo: %+v\n", callbackID, err)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		// Send an empty response, because we'll use the responseURL later
		fmt.Fprintf(w, "Internal Error storing callback: %+v", err)
		return
	}

	log.Printf("Got incoming /ask request, deserialize request:\n%+v\n", command)
	a.OpenDialog(callbackID, config, command.TriggerID)
	w.WriteHeader(http.StatusOK)
	// Send an empty response, because we'll use the responseURL later
	fmt.Fprintf(w, "")
}

func (a *Asker) DialogRequestHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling incoming response for dialog, verifying authenticity\n")
	request, err := a.parseInteractiveRequest(r)
	if err != nil {
		log.Printf("Failed verifying interactive request: %+v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
	}

	db := MgoDBFromRequest(r)
	config, err := db.GetChannelConfig(request.Channel.Id)
	if err != nil || config == nil {
		log.Printf("Unable to fetch channel configuration for %s: %+v\n", request.Channel.Id, err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request, unable to find configuration for the channel requested")
	}

	if originalAsk, err := db.GetCallback(request.CallbackID); err == nil {
		originalAsk.Config = config
		if err = a.PostAskResult(originalAsk, request); err != nil {
			log.Printf("Unable to post response back: %v\n", err)
		}
		db.RemoveCallback(request.CallbackID)
	} else {
		log.Printf("This is strange! We have a dialog with request ID %s, but that is not in the storage queue\n", request.CallbackID)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "I seem to have lost this request, which is unfortunate. I tried to store it in Mongo but now cannot find it.")
		return
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

const TIMEOUT = 5 * time.Minute

func (a *Asker) CleanQueue() {
	for {
		<-time.After(TIMEOUT)
		dbSession := a.storage.Copy()
		go dbSession.DB("slack-ask").RemoveStaleCallbacks(int64(TIMEOUT / time.Second))
		dbSession.Close()
	}
}
