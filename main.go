package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

type SlackUsersListResponse struct {
	Ok      bool          `json:"ok"`
	Error   string        `json:"error"`
	Members []SlackMember `json:"members"`
	Meta    struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

type SlackMember struct {
	ID          string `json:"id"`
	Deactivated bool   `json:"deleted"`
	Profile     struct {
		Email string `json:"email"`
	} `json:"profile"`
}

func loadLookupTable() (map[string]string, error) {
	apiToken := os.Getenv("SLACK_API_KEY")

	var slackResponse SlackUsersListResponse
	lookupTable := map[string]string{}
	for page := 0; slackResponse.Meta.NextCursor != "" || page == 0; page++ {
		resp, err := http.PostForm("https://slack.com/api/users.list",
			url.Values{"token": {apiToken}, "cursor": {slackResponse.Meta.NextCursor}})
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&slackResponse)
		if err != nil {
			return nil, err
		}
		if !slackResponse.Ok {
			return nil, fmt.Errorf("Slack returned not ok: %s", slackResponse.Error)
		}

		fmt.Printf("Page %d returned %d members\n", page+1, len(slackResponse.Members))

		for _, v := range slackResponse.Members {
			lookupTable[v.Profile.Email] = v.ID
		}
	}
	return lookupTable, nil
}

func main() {
	email := os.Getenv("SLACK_EMAIL")

	fmt.Printf("Searching Slack for user with email address %s\n", email)

	var username string

	lookupTable, err := loadLookupTable()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		var ok bool
		username, ok = lookupTable[email]
		if !ok {
			username = email
		}
	}
	fmt.Println(username)

	//
	// --- Step Outputs: Export Environment Variables for other Steps:
	// You can export Environment Variables for other Steps with
	//  envman, which is automatically installed by `bitrise setup`.
	// A very simple example:
	cmdLog, err := exec.Command("bitrise", "envman", "add", "--key", "SLACK_NAME", "--value", username).CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to expose output with envman, error: %#v | output: %s", err, cmdLog)
		os.Exit(1)
	}
	// You can find more usage examples on envman's GitHub page
	//  at: https://github.com/bitrise-io/envman

	//
	// --- Exit codes:
	// The exit code of your Step is very important. If you return
	//  with a 0 exit code `bitrise` will register your Step as "successful".
	// Any non zero exit code will be registered as "failed" by `bitrise`.
	os.Exit(0)
}
