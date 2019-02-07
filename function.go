// Package webhookconverter contains an HTTP Cloud Function.
package webhookconverter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
)

const slackWebHookURL = "SLACK_WEBHOOK_URL"

func sendWebhook(message map[string]interface{}) error {
	url, ok := os.LookupEnv(slackWebHookURL)
	if !ok {
		return fmt.Errorf("%s is not defined in environment", slackWebHookURL)
	}
	bs, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = http.Post(url, "application/json", strings.NewReader(string(bs))) /* #nosec G107 */
	if err != nil {
		return err
	}

	return nil
}

func convertConversationUserReplied(o map[string]interface{}) (map[string]interface{}, error) {
	if o["data"] == nil {
		return nil, fmt.Errorf("data is not found")
	}
	data := o["data"].(map[string]interface{})

	if data["item"] == nil {
		return nil, fmt.Errorf("data.item is not found")
	}
	item := data["item"].(map[string]interface{})

	//if item["conversation_message"] == nil {
	//	return nil, fmt.Errorf("data.item.conversation_message is not found")
	//}
	//message := item["conversation_message"].(map[string]interface{})

	//if message["body"] == nil {
	//	return nil, fmt.Errorf("data.item.conversation_message.body is not found")
	//}
	//messageBody := message["body"].(string)

	if item["conversation_parts"] == nil {
		return nil, fmt.Errorf("data.item.conversation_parts is not found")
	}
	parts := item["conversation_parts"].(map[string]interface{})

	if parts["conversation_parts"] == nil {
		return nil, fmt.Errorf("data.item.conversation_parts.conversation_parts is not found")
	}
	partsParts := parts["conversation_parts"].([]interface{})

	msg := ""
	for _, r := range partsParts {
		if r == nil {
			continue
		}
		p := r.(map[string]interface{})
		if p["body"] == nil {
			continue
		}
		if len(msg) != 0 {
			msg += "\n"
		}
		msg += strip.StripTags(p["body"].(string))
	}

	rval := make(map[string]interface{})
	rval["text"] = msg
	//rval["thread_ts"] = message["id"].(string)

	return rval, nil
}

func convertNotificationEventToSlack(o map[string]interface{}) (map[string]interface{}, error) {
	if o["topic"] == nil {
		return nil, fmt.Errorf("topic is not found")
	}
	topic := o["topic"].(string)

	switch topic {
	case "conversation.user.replied":
		return convertConversationUserReplied(o)
	}

	return nil, fmt.Errorf("unsupported topic in notification event: %s", topic)
}

func convertToSlack(bb []byte) (map[string]interface{}, error) {
	var o map[string]interface{}
	if err := json.Unmarshal(bb, &o); err != nil {
		return nil, err
	}

	if o["type"] == nil {
		return nil, fmt.Errorf("type not found")
	}
	t := o["type"].(string)
	switch t {
	case "notification_event":
		return convertNotificationEventToSlack(o)
	}

	return nil, fmt.Errorf("unsupported type: %s", t)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	fmt.Println(msg)
	fmt.Fprintf(w, msg)
}

// WebHookConverter convert the received data for slack.
func WebHookConverter(w http.ResponseWriter, r *http.Request) {
	bb, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		writeError(w, http.StatusNoContent, fmt.Sprintf("failed to read body: %v", err))
		return
	}

	fmt.Printf("WebHookConverter: %s\n", string(bb))

	msg, err := convertToSlack(bb)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to convert to slack: %v: %s", err, string(bb)))
		return
	}

	fmt.Printf("slack: %s\n", msg)

	if err := sendWebhook(msg); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to send webhook: %v\n", err))
		return
	}
	w.WriteHeader(http.StatusOK)
}
