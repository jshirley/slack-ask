package asker

// Patching nlopes/slack to add dialog support mostly, because I was too lazy to fork it right now

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

var SLACK_API string = "https://slack.com/api/"

type HTTPRequester interface {
	Do(*http.Request) (*http.Response, error)
}

var customHTTPClient HTTPRequester

func getHTTPClient() HTTPRequester {
	if customHTTPClient != nil {
		return customHTTPClient
	}

	return HTTPClient
}

var HTTPClient = &http.Client{}

func parseResponseBody(body io.ReadCloser, intf *interface{}, debug bool) error {
	response, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	// FIXME: will be api.Debugf
	if debug {
		log.Printf("parseResponseBody: %s\n", string(response))
	}

	err = json.Unmarshal(response, &intf)
	if err != nil {
		return err
	}

	return nil
}

func postForm(ctx context.Context, endpoint string, values url.Values, intf interface{}, debug bool) error {
	reqBody := strings.NewReader(values.Encode())
	req, err := http.NewRequest("POST", endpoint, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = req.WithContext(ctx)
	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Slack seems to send an HTML body along with 5xx error codes. Don't parse it.
	if resp.StatusCode != 200 {
		logResponse(resp, debug)
		return fmt.Errorf("Slack server error: %s.", resp.Status)
	}

	return parseResponseBody(resp.Body, &intf, debug)
}

func post(ctx context.Context, path string, values url.Values, intf interface{}, debug bool) error {
	return postForm(ctx, SLACK_API+path, values, intf, debug)
}

func logResponse(resp *http.Response, debug bool) error {
	if debug {
		text, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}

		log.Print(string(text))
	}

	return nil
}
