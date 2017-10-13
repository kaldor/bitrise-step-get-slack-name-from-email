package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"github.com/texttheater/golang-levenshtein/levenshtein"
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
	Name    string `json:"name"`
	Profile struct {
		RealName string `json:"real_name"`
		Email    string `json:"email"`
	} `json:"profile"`
}

func loadCachedLookupTable() map[string]SlackMember {
	return map[string]SlackMember{}
}

func updateLookupTable() (map[string]SlackMember, error) {
	apiToken := os.Getenv("SLACK_API_KEY")

	var slackResponse SlackUsersListResponse
	lookupTable := map[string]SlackMember{}
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
			lookupTable[v.Profile.Email] = v
		}
	}
	return lookupTable, nil
}

func saveLookupTableToCache(lookupTable map[string]SlackMember) {
	// TODO Cache this stuff so we don't have to hit Slack everytime
}

func findClosestMatch(email string, members []SlackMember) string {
	fmt.Println("Using fuzzy matching")
	var minIndex = -1
	var minDistance = math.MaxInt64
	for i, member := range members {
		emailToName := levenshtein.DistanceForStrings([]rune(email), []rune(member.Name), levenshtein.DefaultOptions)
		emailToRealName := levenshtein.DistanceForStrings([]rune(email), []rune(member.Profile.RealName), levenshtein.DefaultOptions)
		distance := emailToName * emailToRealName
		fmt.Printf("%s username: %d full name: %d total: %d\n", member.Name, emailToName, emailToRealName, distance)
		if distance < minDistance {
			minDistance = distance
			minIndex = i
		}
	}
	if minIndex > 0 {
		fmt.Printf("SELECTED %s distance: %d\n", members[minIndex].Name, minDistance)
		return members[minIndex].Name
	} else {
		return email
	}
}

func main() {
	email := os.Getenv("SLACK_EMAIL")

	fmt.Printf("Searching Slack for user with email address %s\n", email)

	lookupTable := loadCachedLookupTable()
	member, ok := lookupTable[email]

	var username string

	if !ok {
		lookupTable, err := updateLookupTable()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			member, ok = lookupTable[email]
			if !ok {
				var members = []SlackMember{}
				for _, v := range lookupTable {
					members = append(members, v)
				}
				username = findClosestMatch(email, members)
			} else {
				username = member.Name
			}
			saveLookupTableToCache(lookupTable)
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
