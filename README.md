# Slack Ask: Ask Questions on Slack

This is a conduit to connect a `/ask` slash command to a JIRA endpoint, to open tickets based on a simple dialog.

This is somewhat opinionated at this point because I am experimenting with the idea. I'd love to expand on the
configuration options more.

# Requirements

## JIRA

Right now this only backs against JIRA. I'd love to have other backing stores for managing incoming questions

## MongoDB

Easy DB to manage what essentially just needs to be key/value stores.

## Patience

I am sure there are bugs, this is me learning the slack API more and the dialog system
