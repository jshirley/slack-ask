package asker

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	jira "github.com/andygrunwald/go-jira"
)

type JiraClient struct {
	endpoint       string
	publicEndpoint string
	client         *jira.Client
}

func (ask *Asker) NewJira(endpoint string, username string, password string, publicEndpoint string) (*JiraClient, error) {
	log.Printf("Got public endpoint to use: %s\n", publicEndpoint)
	client, err := jira.NewClient(nil, endpoint)
	if err != nil {
		panic(err)
	}
	if username != "" {
		client.Authentication.SetBasicAuth(username, password)
	}

	return &JiraClient{endpoint: endpoint, client: client, publicEndpoint: publicEndpoint}, nil
}

func (j *JiraClient) CreateIssue(issueRequest *TicketRequest) (*jira.Issue, error) {
	project, _, err := j.client.Project.Get(issueRequest.ProjectKey)
	if err != nil {
		log.Printf("Unable to fetch JIRA project `%s`: %s\n", issueRequest.ProjectKey, err)
		return nil, err
	}

	components, err := j.getComponentsForRequest(project, issueRequest)
	if err != nil {
		log.Printf("Unable to fetch JIRA components for `%s`: %s\n", issueRequest.ProjectKey, err)
		return nil, err
	}

	i := &jira.Issue{
		Fields: &jira.IssueFields{
			Reporter:    &jira.User{Name: issueRequest.Username},
			Type:        jira.IssueType{Name: project.IssueTypes[0].Name},
			Project:     jira.Project{Key: issueRequest.ProjectKey},
			Summary:     issueRequest.Summary,
			Description: issueRequest.Description,
			Components:  components,
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

func (j *JiraClient) getComponentsForRequest(project *jira.Project, issueRequest *TicketRequest) ([]*jira.Component, error) {
	var components []*jira.Component

	for _, compName := range issueRequest.Components {
		for _, projectComponent := range project.Components {
			if strings.ToLower(projectComponent.Name) == strings.ToLower(compName) {
				components = append(components, &jira.Component{ID: projectComponent.ID, Name: projectComponent.Name})
			}
		}
	}
	return components, nil
}

func (j *JiraClient) GetTicketURL(key string) string {
	if j.publicEndpoint != "" {
		return fmt.Sprintf("%s/browse/%s", j.publicEndpoint, key)
	} else {
		return fmt.Sprintf("%s/browse/%s", j.endpoint, key)
	}
}

func (j *JiraClient) GetComponents(projectKey string) ([]jira.ProjectComponent, error) {
	project, _, err := j.client.Project.Get(projectKey)

	return project.Components, err
}
