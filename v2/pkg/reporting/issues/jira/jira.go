package jira

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/projectdiscovery/nuclei/v2/pkg/output"
	"github.com/projectdiscovery/nuclei/v2/pkg/reporting/issues/format"
	"github.com/projectdiscovery/nuclei/v2/pkg/types"
)

// Integration is a client for a issue tracker integration
type Integration struct {
	jira    *jira.Client
	options *Options
}

// Options contains the configuration options for jira client
type Options struct {
	// URL is the URL of the jira server
	URL string `yaml:"url"`
	// AccountID is the accountID of the jira user.
	AccountID string `yaml:"account-id"`
	// Email is the email of the user for jira instance
	Email string `yaml:"email"`
	// Token is the token for jira instance.
	Token string `yaml:"token"`
	// ProjectName is the name of the project.
	ProjectName string `yaml:"project-name"`
	// IssueType is the name of the created issue type
	IssueType string `yaml:"issue-type"`
}

// New creates a new issue tracker integration client based on options.
func New(options *Options) (*Integration, error) {
	tp := jira.BasicAuthTransport{
		Username: options.Email,
		Password: options.Token,
	}
	jiraClient, err := jira.NewClient(tp.Client(), options.URL)
	if err != nil {
		return nil, err
	}
	return &Integration{jira: jiraClient, options: options}, nil
}

// CreateIssue creates an issue in the tracker
func (i *Integration) CreateIssue(event *output.ResultEvent) error {
	summary := format.Summary(event)

	issueData := &jira.Issue{
		Fields: &jira.IssueFields{
			Assignee:    &jira.User{AccountID: i.options.AccountID},
			Reporter:    &jira.User{AccountID: i.options.AccountID},
			Description: jiraFormatDescription(event),
			Type:        jira.IssueType{Name: i.options.IssueType},
			Project:     jira.Project{Key: i.options.ProjectName},
			Summary:     summary,
		},
	}
	_, resp, err := i.jira.Issue.Create(issueData)
	if err != nil {
		var data string
		if resp != nil && resp.Body != nil {
			d, _ := ioutil.ReadAll(resp.Body)
			data = string(d)
		}
		return fmt.Errorf("%s => %s", err, data)
	}
	return nil
}

// jiraFormatDescription formats a short description of the generated
// event by the nuclei scanner in Jira format.
func jiraFormatDescription(event *output.ResultEvent) string {
	template := format.GetMatchedTemplate(event)

	builder := &bytes.Buffer{}
	builder.WriteString("*Details*: *")
	builder.WriteString(template)
	builder.WriteString("* ")
	builder.WriteString(" matched at ")
	builder.WriteString(event.Host)
	builder.WriteString("\n\n*Protocol*: ")
	builder.WriteString(strings.ToUpper(event.Type))
	builder.WriteString("\n\n*Full URL*: ")
	builder.WriteString(event.Matched)
	builder.WriteString("\n\n*Timestamp*: ")
	builder.WriteString(event.Timestamp.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
	builder.WriteString("\n\n*Template Information*\n\n| Key | Value |\n")
	for k, v := range event.Info {
		builder.WriteString(fmt.Sprintf("| %s | %s |\n", k, v))
	}
	builder.WriteString("\n*Request*\n\n{code}\n")
	builder.WriteString(event.Request)
	builder.WriteString("\n{code}\n\n*Response*\n\n{code}\n")
	builder.WriteString(event.Response)
	builder.WriteString("\n{code}\n\n")

	if len(event.ExtractedResults) > 0 || len(event.Metadata) > 0 {
		builder.WriteString("*Extra Information*\n\n")
		if len(event.ExtractedResults) > 0 {
			builder.WriteString("*Extracted results*:\n\n")
			for _, v := range event.ExtractedResults {
				builder.WriteString("- ")
				builder.WriteString(v)
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
		if len(event.Metadata) > 0 {
			builder.WriteString("*Metadata*:\n\n")
			for k, v := range event.Metadata {
				builder.WriteString("- ")
				builder.WriteString(k)
				builder.WriteString(": ")
				builder.WriteString(types.ToString(v))
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
	}
	data := builder.String()
	return data
}
