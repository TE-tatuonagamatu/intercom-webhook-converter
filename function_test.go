package webhookconverter

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var webhookInputData string

var echoHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	webhookInputData = string(bs)
	w.WriteHeader(http.StatusOK)
})

func TestWebHookConverter(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{
			body: `{"type":"notification_event","app_id":"u3lubtax","data":{"type":"notification_event_data","item":{"type":"conversation","id":"20711452426","created_at":1549426675,"updated_at":1549433270,"user":{"type":"lead","id":"5c5a5ff3d717580071c93537","user_id":"ddc4e495-cfc2-4098-abd7-5fb4e9a43e63","name":"Cyan Megaphone from Shinagawa","email":"","do_not_track":null},"assignee":{"type":"nobody_admin","id":null},"conversation_message":{"type":"conversation_message","id":"309729238","url":null,"subject":"","body":"<p>Hi 😄 Have a look around! Let us know if you have any questions.</p>","author":{"type":"admin","id":"2925745"},"attachments":[]},"conversation_parts":{"type":"conversation_part.list","conversation_parts":[{"type":"conversation_part","id":"2621237688","part_type":"comment","body":"<p>I have a question about pricing</p>","created_at":1549433270,"updated_at":1549433270,"notified_at":1549433270,"assigned_to":null,"author":{"type":"user","id":"5c5a5ff3d717580071c93537","name":null,"email":""},"attachments":[],"external_id":null}],"total_count":1},"open":true,"state":"open","snoozed_until":null,"read":true,"metadata":{},"tags":{"type":"tag.list","tags":[]},"tags_added":{"type":"tag.list","tags":[]},"links":{"conversation_web":"https://app.intercom.io/a/apps/u3lubtax/conversations/20711452426"}}},"links":{},"id":"notif_32b8595a-93ab-437b-8f87-8ea5b784552c","topic":"conversation.user.replied","delivery_status":"pending","delivery_attempts":1,"delivered_at":0,"first_sent_at":1549433271,"created_at":1549433271,"self":null}`,
			want: `{"text":"I have a question about pricing"}`,
		},
	}

	ts := httptest.NewServer(echoHandler)
	defer ts.Close()

	os.Setenv("SLACK_WEBHOOK_URL", ts.URL)
	for _, test := range tests {
		req := httptest.NewRequest("POST", "/", strings.NewReader(test.body))
		req.Header.Add("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		WebHookConverter(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("WebHookConverter(%q) returns error: %d", test.body, rr.Code)
		}

		out, err := ioutil.ReadAll(rr.Result().Body)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if got := string(out); got != "" {
			t.Errorf("WebHookConverter(%q) = %q, want empty", test.body, got)
		}
		if webhookInputData != test.want {
			t.Errorf("WebHookConverter(%q) = %q, want %q", test.body, webhookInputData, test.want)
		}
	}
}
