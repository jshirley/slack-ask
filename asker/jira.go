package asker

import (
	jira "github.com/andygrunwald/go-jira"
)

type JiraClient struct {
	client *jira.Client
}

func NewJira(endpoint string, username string, password string) (*JiraClient, error) {
	client, err := jira.NewClient(nil, endpoint)
	if err != nil {
		panic(err)
	}
	if username != "" {
		client.Authentication.SetBasicAuth(username, password)
	}

	return &JiraClient{client: client}, nil
}

func (j *JiraClient) CreateIssue() {
}

func (j *JiraClient) GetComponents(project string) {
}
