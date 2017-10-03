package asker

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
)

type Asker struct {
	Token string
	api   *slack.Client
}

func NewAsker(oAuthToken string, token string) (*Asker, error) {
	client := Asker{
		Token: token,
		api:   slack.New(oAuthToken),
	}

	return &client, nil
}

func (a *Asker) Listen(addr string) {
	r := mux.NewRouter()
	r.HandleFunc("/events/ask", a.AskHandler)
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

func (a *Asker) AskHandler(w http.ResponseWriter, r *http.Request) {
	command, err := a.parseSlashCommand(r)
	if err != nil {
		log.Printf("Failed parsing slash command: %+v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
	}

	log.Printf("Got incoming /events/ask request, deserialize request:\n%+v\n", command)
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
