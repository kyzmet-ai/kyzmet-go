package raw_test

import (
	"encoding/json"
	"testing"
	"time"

	raw "kyzmet.ai/go"
)

func TestTriggerPayloadDecodeHelpers(t *testing.T) {
	payload := raw.TriggerPayload{
		MediaType:       "application/json",
		Data:            []byte(`{"triggerId":"prepare-meeting-brief"}`),
		Classifications: []string{"calendar.metadata"},
	}

	if got := raw.DecodeTextPayload(payload); got != `{"triggerId":"prepare-meeting-brief"}` {
		t.Fatalf("unexpected text payload: %s", got)
	}

	var decoded struct {
		TriggerID string `json:"triggerId"`
	}
	if err := raw.DecodeJSONPayload(payload, &decoded); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if decoded.TriggerID != "prepare-meeting-brief" {
		t.Fatalf("unexpected trigger id: %s", decoded.TriggerID)
	}
}

func TestTriggerResultBuilderAndIdempotencyHelpers(t *testing.T) {
	invocation := raw.TriggerInvocation{
		InvocationID: "inv-1",
		TriggerType:  raw.PlatformTrigger(raw.PlatformEventActionRequested),
		BindingID:    "prepare-meeting-brief",
		AppID:        "ai.kyzmet.workplace",
		SphereID:     "work",
		Source:       "platform-action-provider",
		Payload: raw.TriggerPayload{
			MediaType: "application/json",
			Data:      []byte(`{}`),
		},
		OccurredAt: "2026-06-24T05:00:00.000Z",
		InvokedAt:  "2026-06-24T05:00:01.000Z",
		ReceiptRequest: raw.TriggerReceiptRequest{
			UserVisible: true,
		},
	}

	if got := raw.EventIdempotencyKey(invocation); got != "inv-1" {
		t.Fatalf("unexpected event idempotency key: %s", got)
	}
	if got := raw.BindingIdempotencyKey(invocation); got != "prepare-meeting-brief" {
		t.Fatalf("unexpected binding idempotency key: %s", got)
	}

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
