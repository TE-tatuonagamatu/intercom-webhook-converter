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

func sendWebhook(message map[string]interface{}) error {
	url, ok := os.LookupEnv("SLACK_WEBHOOK_URL")
	if !ok {
		return fmt.Errorf("SLACK_WEBHOOK_URL is not defined in environment")
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

/*
func debugJSONObject(o map[string]interface{}, m string) {
	for k, v := range o {
		fmt.Printf("jsonObject[%s]: %s: %v\n", m, k, v)
	}
}
*/

func convertConversationUserReplied(o map[string]interface{}) (map[string]interface{}, error) {
	data := o["data"].(map[string]interface{})
	item := data["item"].(map[string]interface{})
	message := item["conversation_message"].(map[string]interface{})
	messageBody := message["body"].(string)
	parts := item["conversation_parts"].(map[string]interface{})
	partsParts := parts["conversation_parts"].([]interface{})

	fmt.Printf("convertConversationUserReplied: body: %d: %s\n", len(partsParts), messageBody)

	msg := ""
	for n, r := range partsParts {
		p := r.(map[string]interface{})
		if len(msg) != 0 {
			msg += "\n"
		}
		fmt.Printf("convertConversationUserReplied: %d: %s\n", n, p["body"].(string))
		msg += strip.StripTags(p["body"].(string))
	}

	rval := make(map[string]interface{})
	rval["text"] = msg
	//rval["thread_ts"] = message["id"].(string)

	return rval, nil
}

func convertNotificationEventToSlack(o map[string]interface{}) (map[string]interface{}, error) {
	topic := o["topic"].(string)

	fmt.Printf("convertNotificationEventToSlack: topic: %s\n", topic)
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

	fmt.Printf("convertToSlack: %s\n", string(bb))

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

// WebHookConverter convert the received data for slack.
func WebHookConverter(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("WebHookConverter is called\n")

	bb, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		fmt.Printf("failed to read body: %v\n", err)
		fmt.Fprintf(w, "failed to read body: %v", err)
		return
	}

	fmt.Printf("body: %s\n", string(bb))

	msg, err := convertToSlack(bb)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("failed to convert to slack: %v\n", err)
		fmt.Fprintf(w, "failed to convert to slack: %v", err)
		return
	}

	fmt.Printf("slack: %s\n", msg)

	if err := sendWebhook(msg); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("failed to send webhook: %v\n", err)
		fmt.Fprintf(w, "failed to send webhook: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
