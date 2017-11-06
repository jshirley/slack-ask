package asker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/schema"
	"github.com/nlopes/slack"
)

type SlackTeam struct {
	Id     string `json:"id"`
	Domain string `json:"domain"`
}

type SlackTuple struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type TicketRequest struct {
	Username    string
	ProjectKey  string
	Summary     string
	Description string
	Priority    string
	Components  []string
}

type InteractiveRequest struct {
	Type       string            `json:"type"`
	Submission map[string]string `json:"submission"`
	CallbackID string            `json:"callback_id"`
	Team       SlackTeam         `json:"team"`
	User       SlackTuple        `json:"user"`
	Channel    SlackTuple        `json:"channel"`
	Timestamp  string            `json:"action_ts"`
	Token      string            `json:"token"`
}

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
	Config         *ChannelConfig
	Timestamp      int64
}

type DialogOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type DialogElement struct {
	Type        string         `json:"type"`
	Label       string         `json:"label"`
	Name        string         `json:"name"`
	Placeholder string         `json:"placeholder,omitempty"`
	Value       string         `json:"value,omitempty"`
	Options     []DialogOption `json:"options,omitempty"`
}

type Dialog struct {
	CallbackID  string          `json:"callback_id"`
	Title       string          `json:"title"`
	SubmitLabel string          `json:"submit_label"`
	Elements    []DialogElement `json:"elements"`
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

	command.Timestamp = time.Now().Unix()
	return command, nil
}

func (a *Asker) parseInteractiveRequest(r *http.Request) (*InteractiveRequest, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	request := new(InteractiveRequest)
	if err := json.Unmarshal([]byte(r.FormValue("payload")), request); err != nil {
		return nil, err
	}

	if request.Token != a.Token {
		return nil, fmt.Errorf("Invalid token on dialog submission, check configuration or ensure someone isn't sending you bogus data")
	}
	return request, nil
}

func (a *Asker) OpenDialog(callback string, config *ChannelConfig, triggerId string) error {
	dialog := Dialog{
		CallbackID:  callback,
		Title:       "Ask a Question", //fmt.Sprintf("Ask a Question!", config.Project),
		SubmitLabel: "Ask!",
		Elements: []DialogElement{
			DialogElement{Type: "text", Label: "The one liner...", Name: "summary"},
			DialogElement{Type: "textarea", Label: "The details", Name: "description", Placeholder: "What have you tried so far, stack traces, etc."},
			DialogElement{
				Type:        "select",
				Label:       "Blocking?",
				Name:        "blocking",
				Placeholder: "Are you actively blocked and need someone now?",
				Value:       "no",
				Options: []DialogOption{
					DialogOption{"No", "no"},
					DialogOption{"Yes", "yes"},
					DialogOption{"I'm about to page!", "911"},
				},
			},
		},
	}

	dialogJson, err := json.Marshal(dialog)
	if err != nil {
		log.Printf("Error encoding Dialog JSON: %+v\n", err)
		return err
	}

	values := url.Values{
		"token":      {a.OAuth},
		"trigger_id": {triggerId},
		"dialog":     {string(dialogJson)},
	}

	response := slack.SlackResponse{}
	err = post(context.Background(), "dialog.open", values, response, true)
	if err != nil {
		return err
	}
	log.Printf("Got a response from Slack: %+v\n", response)
	return nil
}

type SlackResponseResult struct {
	ResponseType string             `json:"response_type"`
	Text         string             `json:"text"`
	Attachments  []slack.Attachment `json:"attachments"`
}

func (a *Asker) PostAskResult(originalAsk *SlashCommand, request *InteractiveRequest) error {
	ticket := TicketRequest{
		Username:    originalAsk.UserName,
		Summary:     request.Submission["summary"],
		Description: request.Submission["description"],
		ProjectKey:  originalAsk.Config.Project,
		Components:  originalAsk.Config.Components,
	}
	log.Printf("Creating a JIRA ticket in %s by %s\n", ticket.ProjectKey, ticket.Username)
	issue, err := a.Jira.CreateIssue(&ticket)

	response := SlackResponseResult{}
	if err != nil {
		response = SlackResponseResult{
			Text: fmt.Sprintf("Sorry! We failed to create an issue for that... please try again, and if it is helpful the error is `%v`", err),
		}
	} else {
		response = SlackResponseResult{
			ResponseType: "in_channel",
			Text:         fmt.Sprintf("<@%s> is asking \"%s\" (<%s|%s>)", originalAsk.UserID, request.Submission["summary"], a.Jira.GetTicketURL(issue.Key), issue.Key),
		}
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error encoding JSON: %+v\n", err)
		return err
	}

	req, err := http.NewRequest("POST", originalAsk.ResponseURL, bytes.NewBuffer(responseJson))
	req.Header.Set("Content-Type", "application/json")

	req = req.WithContext(context.Background())
	resp, err := getHTTPClient().Do(req)
	if err != nil {
		log.Printf("Error posting response back to Slack: %s\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logResponse(resp, true)
		return fmt.Errorf("Slack server error: %s.", resp.Status)
	}

	return nil
}
