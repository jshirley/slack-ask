package asker

import (
	"fmt"
	"io/ioutil"

	jira "github.com/andygrunwald/go-jira"
)

type JiraClient struct {
	endpoint string
	client   *jira.Client
}

func (ask *Asker) NewJira(endpoint string, username string, password string) (*JiraClient, error) {
	client, err := jira.NewClient(nil, endpoint)
	if err != nil {
		panic(err)
	}
	if username != "" {
		client.Authentication.SetBasicAuth(username, password)
	}

	return &JiraClient{endpoint: endpoint, client: client}, nil
}

func (j *JiraClient) CreateIssue(issueRequest *TicketRequest) (*jira.Issue, error) {
	project, _, err := j.client.Project.Get(issueRequest.ProjectKey)
	if err != nil {
		return nil, err
	}

	i := &jira.Issue{
		Fields: &jira.IssueFields{
			Reporter:    &jira.User{Name: issueRequest.Username},
			Type:        jira.IssueType{Name: project.IssueTypes[0].Name},
			Project:     jira.Project{Key: issueRequest.ProjectKey},
			Summary:     issueRequest.Summary,
			Description: issueRequest.Description,
		},
	}
	issue, resp, err := j.client.Issue.Create(i)
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, fmt.Errorf(bodyString)
	}

	return issue, nil
}

func (j *JiraClient) GetTicketURL(key string) string {
	return fmt.Sprintf("%s/browse/%s", j.endpoint, key)
}

func (j *JiraClient) GetComponents(projectKey string) ([]jira.ProjectComponent, error) {
	project, _, err := j.client.Project.Get(projectKey)

	return project.Components, err
}
