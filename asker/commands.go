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

	"github.com/jshirley/slack-ask/storage"

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

func (a *Asker) parseSlashCommand(r *http.Request) (*storage.SlashCommand, error) {
	err := r.ParseForm()

	if err != nil {
		return nil, err
	}

	command := new(storage.SlashCommand)

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

func (a *Asker) OpenDialog(callback string, config *storage.ChannelConfig, triggerId string) error {
	dialog := a.GetDialog(callback)

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
	log.Printf("Got a response from Slack, are things ok? %+v\n", response.Ok)
	return nil
}

type SlackResponseResult struct {
	ResponseType string             `json:"response_type"`
	Text         string             `json:"text"`
	Attachments  []slack.Attachment `json:"attachments"`
}

func (a *Asker) PostAskResult(originalAsk *storage.SlashCommand, request *InteractiveRequest) error {
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
			Text:         fmt.Sprintf("<@%s> is `/ask`ing \"%s\" (<%s|%s>)", originalAsk.UserID, request.Submission["summary"], a.Jira.GetTicketURL(issue.Key), issue.Key),
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
