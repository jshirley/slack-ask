package asker

import (
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
)

type SlashCommand struct {
	Token          string `schema:"token"`
	TeamID         string `schema:"team_id"`
	TeamDomain     string `schema:"team_domain"`
	EnterpriseID   string `schema:"enterprise_id"`
	EnterpriseName string `schema:"enterprise_name"`
	ChannelID      string `schema:"channel_id"`
	ChannelName    string `schema:"channel_name"`
	UserID         string `schema:"user_id"`
	UserName       string `schema:"user_name"`
	Command        string `schema:"command"`
	Text           string `schema:"text"`
	ResponseURL    string `schema:"response_url"`
	TriggerID      string `schema:"trigger_id"`
}

func (a *Asker) parseSlashCommand(r *http.Request) (*SlashCommand, error) {
	err := r.ParseForm()

	if err != nil {
		return nil, err
	}

	command := new(SlashCommand)

	decoder := schema.NewDecoder()
	err = decoder.Decode(command, r.PostForm)

	if err != nil {
		return nil, err
	}

	if command.Token != a.Token {
		return nil, fmt.Errorf("Invalid token, check configuration or ensure someone isn't sending you bogus data")
	}

	return command, nil
}
