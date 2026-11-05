package raw_test

import (
	"encoding/json"
	"testing"
	"time"

	raw "kyzmet.ai/go"
)

func TestTriggerResultBuilder(t *testing.T) {
	result := raw.NewTriggerResult(raw.TriggerStatusCodeOK, "Prepared", "Done")
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if string(encoded) != `{"status":"OK","title":"Prepared","summary":"Done","receipt_metadata":[],"next_actions":[]}` {
		t.Fatalf("unexpected result json: %s", encoded)
	}
}

func TestTriggerResultRetryAfterEncodesRetryInfo(t *testing.T) {
	result := raw.NewTriggerResult(raw.TriggerStatusCodeUnavailable, "Busy", "Try later.").
		WithRetryAfter(1500 * time.Millisecond)
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	want := `{"status":"UNAVAILABLE","title":"Busy","summary":"Try later.","receipt_metadata":[],"next_actions":[],"retry_info":{"retry_delay":{"seconds":1,"nanos":500000000}}}`
	if string(encoded) != want {
		t.Fatalf("unexpected result json: %s", encoded)
	}
}
