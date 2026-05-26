// Package alert handles external notification dispatch to Slack and Telegram.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/itam/server/internal/model"
)

var severityColor = map[string]string{
	"CRITICAL": "#FF0000",
	"HIGH":     "#FF6600",
	"MEDIUM":   "#FFAA00",
	"LOW":      "#0099FF",
}

// SendSlack posts a Block Kit message to the incoming webhook URL.
func SendSlack(ctx context.Context, webhookURL string, anomaly *model.Anomaly) error {
	color := severityColor[anomaly.Severity]
	if color == "" {
		color = "#808080"
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"blocks": []map[string]interface{}{
					{
						"type": "header",
						"text": map[string]interface{}{
							"type": "plain_text",
							"text": fmt.Sprintf("[%s] %s on %s", anomaly.Severity, anomaly.RuleID, anomaly.Hostname),
						},
					},
					{
						"type": "section",
						"fields": []map[string]interface{}{
							{"type": "mrkdwn", "text": fmt.Sprintf("*Host:*\n%s", anomaly.Hostname)},
							{"type": "mrkdwn", "text": fmt.Sprintf("*Rule:*\n%s", anomaly.RuleID)},
							{"type": "mrkdwn", "text": fmt.Sprintf("*Entity:*\n%s `%s`", anomaly.EntityType, anomaly.EntityKey)},
							{"type": "mrkdwn", "text": fmt.Sprintf("*Detected:*\n%s", anomaly.DetectedAt.Format(time.RFC3339))},
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook: status %d", resp.StatusCode)
	}
	return nil
}
