package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/itam/server/internal/model"
)

var severityEmoji = map[string]string{
	"CRITICAL": "🔴",
	"HIGH":     "🟠",
	"MEDIUM":   "🟡",
	"LOW":      "🔵",
}

// SendTelegram posts an HTML-formatted message to a Telegram chat via Bot API.
// botToken is the value returned by BotFather; chatID is the numeric chat ID.
func SendTelegram(ctx context.Context, botToken string, chatID int64, anomaly *model.Anomaly) error {
	emoji := severityEmoji[anomaly.Severity]
	if emoji == "" {
		emoji = "⚠️"
	}

	text := fmt.Sprintf(
		"%s <b>[%s]</b> %s\n\n"+
			"<b>Host:</b> %s\n"+
			"<b>Entity:</b> %s <code>%s</code>\n"+
			"<b>Detected:</b> %s",
		emoji,
		html.EscapeString(anomaly.Severity),
		html.EscapeString(anomaly.RuleID),
		html.EscapeString(anomaly.Hostname),
		html.EscapeString(anomaly.EntityType),
		html.EscapeString(anomaly.EntityKey),
		anomaly.DetectedAt.Format(time.RFC3339),
	)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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

	var result struct {
		OK bool `json:"ok"`
	}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if !result.OK {
		return fmt.Errorf("telegram API returned ok=false for chat_id %d", chatID)
	}
	return nil
}
