package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const (
	EnvSlackWebhook   = "SLACK_WEBHOOK"
	EnvSlackIcon      = "SLACK_ICON"
	EnvSlackIconEmoji = "SLACK_ICON_EMOJI"
	EnvSlackChannel   = "SLACK_CHANNEL"
	EnvSlackTitle     = "SLACK_TITLE"
	EnvSlackMessage   = "SLACK_MESSAGE"
	EnvSlackColor     = "SLACK_COLOR"
	EnvSlackUserName  = "SLACK_USERNAME"
	EnvSlackFooter    = "SLACK_FOOTER"
	EnvGithubActor    = "GITHUB_ACTOR"
	EnvSiteName       = "SITE_NAME"
	EnvHostName       = "HOST_NAME"
	EnvMinimal        = "MSG_MINIMAL"
	EnvSlackLinkNames = "SLACK_LINK_NAMES"
	EnvPRTitle        = "PR_TITLE"
	EnvPRNumber       = "PR_NUMBER"
	EnvPRBody         = "PR_BODY"
)

type Webhook struct {
	Text        string          `json:"text,omitempty"`
	UserName    string          `json:"username,omitempty"`
	IconURL     string          `json:"icon_url,omitempty"`
	IconEmoji   string          `json:"icon_emoji,omitempty"`
	Channel     string          `json:"channel,omitempty"`
	LinkNames   string          `json:"link_names,omitempty"`
	UnfurlLinks bool            `json:"unfurl_links"`
	Attachments []Attachment    `json:"attachments,omitempty"`
	Blocks      json.RawMessage `json:"blocks,omitempty"`
}

type Attachment struct {
	Fallback   string  `json:"fallback"`
	Pretext    string  `json:"pretext,omitempty"`
	Color      string  `json:"color,omitempty"`
	AuthorName string  `json:"author_name,omitempty"`
	AuthorLink string  `json:"author_link,omitempty"`
	AuthorIcon string  `json:"author_icon,omitempty"`
	Footer     string  `json:"footer,omitempty"`
	Fields     []Field `json:"fields,omitempty"`
}

type Field struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short,omitempty"`
}

func main() {
	prTitle := os.Getenv(EnvPRTitle)
	jiraTicketID := extractJiraID(prTitle)
	viewPrURL := os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/pull/" + os.Getenv(EnvPRNumber)
	viewJiraTicketURL := "https://makersoftware.atlassian.net/browse/" + jiraTicketID

	endpoint := os.Getenv(EnvSlackWebhook)
	if endpoint == "" {
		fmt.Fprintln(os.Stderr, "URL is required")
		os.Exit(1)
	}
	text := os.Getenv(EnvSlackMessage)
	if text == "" {
		fmt.Fprintln(os.Stderr, "Message is required")
		os.Exit(1)
	}
	if strings.HasPrefix(os.Getenv("GITHUB_WORKFLOW"), ".github") {
		os.Setenv("GITHUB_WORKFLOW", "Link to action run")
	}

	long_sha := os.Getenv("GITHUB_SHA")
	commit_sha := long_sha[0:6]

	minimal := os.Getenv(EnvMinimal)
	fields := []Field{}
	if minimal == "true" {
		mainFields := []Field{
			{
				Title: os.Getenv(EnvSlackTitle),
				Value: envOr(EnvSlackMessage, "EOM"),
				Short: false,
			},
		}
		fields = append(mainFields, fields...)
	} else if minimal != "" {
		requiredFields := strings.Split(minimal, ",")
		mainFields := []Field{
			{
				Title: os.Getenv(EnvSlackTitle),
				Value: envOr(EnvSlackMessage, "EOM"),
				Short: false,
			},
		}
		for _, requiredField := range requiredFields {
			switch strings.ToLower(requiredField) {
			case "ref":
				field := []Field{
					{
						Title: "Ref",
						Value: os.Getenv("GITHUB_REF"),
						Short: true,
					},
				}
				mainFields = append(field, mainFields...)
			case "event":
				field := []Field{
					{
						Title: "Event",
						Value: os.Getenv("GITHUB_EVENT_NAME"),
						Short: true,
					},
				}
				mainFields = append(field, mainFields...)
			case "actions url":
				field := []Field{
					{
						Title: "Actions URL",
						Value: "<" + os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/commit/" + os.Getenv("GITHUB_SHA") + "/checks|" + os.Getenv("GITHUB_WORKFLOW") + ">",
						Short: true,
					},
				}
				mainFields = append(field, mainFields...)
			case "commit":
				field := []Field{
					{
						Title: "Commit",
						Value: "<" + os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/commit/" + os.Getenv("GITHUB_SHA") + "|" + commit_sha + ">",
						Short: true,
					},
				}
				mainFields = append(field, mainFields...)
			}
		}
		fields = append(mainFields, fields...)
	} else {
		mainFields := []Field{
			{
				Title: "Ref",
				Value: os.Getenv("GITHUB_REF"),
				Short: true,
			}, {
				Title: "Event",
				Value: os.Getenv("GITHUB_EVENT_NAME"),
				Short: true,
			},
			{
				Title: "Actions URL",
				Value: "<" + os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/commit/" + os.Getenv("GITHUB_SHA") + "/checks|" + os.Getenv("GITHUB_WORKFLOW") + ">",
				Short: true,
			},
			{
				Title: "Commit",
				Value: "<" + os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/commit/" + os.Getenv("GITHUB_SHA") + "|" + commit_sha + ">",
				Short: true,
			},
			{
				Title: os.Getenv(EnvSlackTitle),
				Value: envOr(EnvSlackMessage, "EOM"),
				Short: false,
			},
		}
		fields = append(mainFields, fields...)
	}

	hostName := os.Getenv(EnvHostName)
	if hostName != "" {
		newfields := []Field{
			{
				Title: os.Getenv("SITE_TITLE"),
				Value: os.Getenv(EnvSiteName),
				Short: true,
			},
			{
				Title: os.Getenv("HOST_TITLE"),
				Value: os.Getenv(EnvHostName),
				Short: true,
			},
		}
		fields = append(newfields, fields...)
	}

	githubServerURL := os.Getenv("GITHUB_SERVER_URL")
	githubActor := os.Getenv("EnvGithubActor")
	githubFormattedImageSource := fmt.Sprintf("%s/%s.png?size=32", githubServerURL, githubActor)

	msg := Webhook{
		UserName:  os.Getenv(EnvSlackUserName),
		IconURL:   os.Getenv(EnvSlackIcon),
		IconEmoji: os.Getenv(EnvSlackIconEmoji),
		Channel:   os.Getenv(EnvSlackChannel),
		LinkNames: os.Getenv(EnvSlackLinkNames),
		Blocks: json.RawMessage(`[
			{
				"type": "context",
				"elements": [
					{
						"type": "image",
						"image_url": "` + githubFormattedImageSource + `",
						"alt_text": "github user"
					},
					{
						"type": "mrkdwn",
						"text": "<!here> *` + os.Getenv(EnvGithubActor) + `* has a pull request ready for review."
					}
				]
			},
			{
				"type": "header",
				"text": {
					"type": "plain_text",
					"text": "Ready for Review"
				}
			},
			{
				"type": "context",
				"elements": [
					{
						"type": "mrkdwn",
						"text": "*Repository:* ` + os.Getenv("GITHUB_REPOSITORY") + `"
					}
				]
			},
			{
				"type": "context",
				"elements": [
					{
						"type": "mrkdwn",
						"text": "*Title:* ` + prTitle + `"
					}
				]
			},
			{
				"type": "divider"
			},
			{
				"type": "section",
				"text": {
					"type": "mrkdwn",
					"text": "` + strings.ReplaceAll(os.Getenv(findAndFormatUserID(EnvPRBody)), "\n", "\\n") + `"
				}
			},
			{
				"type": "actions",
				"elements": [
					{
						"type": "button",
						"text": {
							"type": "plain_text",
							"text": "View Pull Request"
						},
						"style": "primary",
						"url": "` + viewPrURL + `"
					},
					{
						"type": "button",
						"text": {
							"type": "plain_text",
							"text": "View JIRA Ticket"
						},
						"style": "primary",
						"url": "` + viewJiraTicketURL + `",
						"value": "click_me_123"
					}
				]
			}
		]`),
	}

	fmt.Printf("Sending message to %s\n", endpoint)
	fmt.Printf("Message: %s\n", msg.Text)

	if err := send(endpoint, msg); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message: %s\n", err)
		os.Exit(2)
	}
}

func extractJiraID(prTitle string) string {
	re := regexp.MustCompile(`FOR-\d+`)
	matches := re.FindStringSubmatch(prTitle)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

var usernameToIDMap = map[string]string{
	"alex":       "U01FFMD8P7E",
	"Alex":       "U01FFMD8P7E",
	"twigs67":    "U01FFMD8P7E",
	"brad":       "U058HUUKZ6U",
	"Brad":       "U058HUUKZ6U",
	"dvrs-brad":  "U058HUUKZ6U",
	"josh":       "U061W1T6L0Y",
	"Josh":       "U061W1T6L0Y",
	"skyfriends": "U061W1T6L0Y",
	"bryer":      "U03HRTQ0LKW",
	"Bryer":      "U03HRTQ0LKW",
	"bryercowan": "U03HRTQ0LKW",
}

func findAndFormatUserID(input string) string {
	re := regexp.MustCompile(`@(\w+)`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		username := match[1:] // Remove the '@' from the match.
		if userID, ok := usernameToIDMap[username]; ok {
			return "<@" + userID + ">"
		}
		return match // Return the original match if no user ID is found.
	})
}

func envOr(name, def string) string {
	if d, ok := os.LookupEnv(name); ok {
		return d
	}
	return def
}

func send(endpoint string, msg Webhook) error {
	enc, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	b := bytes.NewBuffer(enc)
	res, err := http.Post(endpoint, "application/json", b)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("Error on message: %s\n", res.Status)
	}
	fmt.Println(res.Status)
	return nil
}
