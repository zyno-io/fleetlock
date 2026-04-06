package fleetlock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// SlackNotifier sends notifications to a Slack channel.
type SlackNotifier struct {
	botToken  string
	channelID string
	log       *logrus.Logger
	client    *http.Client
}

// NewSlackNotifier creates a new SlackNotifier.
func NewSlackNotifier(botToken, channelID string, log *logrus.Logger) *SlackNotifier {
	return &SlackNotifier{
		botToken:  botToken,
		channelID: channelID,
		log:       log,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// slackMessage represents a Slack chat.postMessage request.
type slackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

// Notify sends a message to the configured Slack channel.
func (s *SlackNotifier) Notify(text string) error {
	msg := slackMessage{
		Channel: s.channelID,
		Text:    text,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("slack: error marshaling message: %v", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: error sending message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack: unexpected status code %d", resp.StatusCode)
	}
	return nil
}

// notifySlack sends a Slack notification for lock events. It is best-effort
// and logs errors rather than returning them.
func (s *Server) notifySlack(event, group, nodeID string) {
	if s.slackNotifier == nil {
		return
	}

	var text string
	switch event {
	case "lock_granted":
		text = fmt.Sprintf(":lock: Reboot lock *granted* for node `%s` in group `%s`", nodeID, group)
	case "lock_released":
		text = fmt.Sprintf(":unlock: Reboot lock *released* for node `%s` in group `%s`", nodeID, group)
	default:
		text = fmt.Sprintf("Reboot lock event `%s` for node `%s` in group `%s`", event, nodeID, group)
	}

	go func() {
		if err := s.slackNotifier.Notify(text); err != nil {
			s.log.Errorf("fleetlock: slack notification error: %v", err)
		}
	}()
}
