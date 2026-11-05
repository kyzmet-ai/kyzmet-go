package raw_test

import (
	"encoding/hex"
	"testing"

	timestamppb "github.com/aperturerobotics/protobuf-go-lite/types/known/timestamppb"
	raw "kyzmet.ai/go"
	triggersv1 "kyzmet.ai/go/proto/triggers/v1"
)

// protoc --encode=kyzmet.triggers.v1.TriggerEvent of platformAction's text
// format; the Rust crate's test pins the same bytes, so every SDK reads what
// any host writes.
const platformActionHex = "0a076576656e742d3112040a0208011a08706c6174666f726d2204776f726b2a0608d0c8edd106320c6d6565742d3132332d6b65793a1163616c656e6461722e6d657461646174616a5e0a15707265706172652d6d656574696e672d6272696566120a6c6f63616c2d757365721a12776f726b706c6163652d6f766572766965772a250a0a746578742f706c61696e20022a154167656e64613a206c61756e636820726576696577"

func platformAction() *triggersv1.TriggerEvent {
	key := "meet-123-key"
	return &triggersv1.TriggerEvent{
		EventId: "event-1",
		Type: &triggersv1.TriggerType{Family: &triggersv1.TriggerType_Platform{
			Platform: &triggersv1.PlatformTrigger{Event: triggersv1.PlatformEvent_PLATFORM_EVENT_ACTION_REQUESTED},
		}},
		Source:          "platform",
		SphereId:        "work",
		OccurredAt:      &timestamppb.Timestamp{Seconds: 1782277200},
		IdempotencyKey:  &key,
		Classifications: []string{"calendar.metadata"},
		Details: &triggersv1.TriggerEvent_PlatformAction{PlatformAction: &triggersv1.PlatformActionEvent{
			ActionId:      "prepare-meeting-brief",
			ActorUserId:   "local-user",
			SourceSurface: "workplace-overview",
			Input: []*triggersv1.ContentPart{{
				MediaType: "text/plain",
				Origin:    triggersv1.ContentOrigin_CONTENT_ORIGIN_EXTERNAL,
				Data:      []byte("Agenda: launch review"),
			}},
		}},
	}
}

func mustUnhex(t *testing.T, text string) []byte {
	t.Helper()
	b, err := hex.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAPlatformActionEventIsProtocolsEncodingAndDecodes(t *testing.T) {
	encoded, err := platformAction().MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(encoded); got != platformActionHex {
		t.Fatalf("encoded %s", got)
	}
	event, err := raw.DecodeTriggerEvent(mustUnhex(t, platformActionHex))
	if err != nil {
		t.Fatal(err)
	}
	if !event.EqualVT(platformAction()) {
		t.Fatalf("decoded %+v", event)
	}
	if action := event.GetPlatformAction(); action.GetActionId() != "prepare-meeting-brief" ||
		string(action.GetInput()[0].GetData()) != "Agenda: launch review" {
		t.Fatalf("platform action %+v", action)
	}
	if got := raw.EventTrigger(event); got != raw.PlatformTrigger(raw.PlatformEventActionRequested) {
		t.Fatalf("trigger %v", got)
	}
}

func TestADetailsArmNewerThanTheSDKDecodesAsUnset(t *testing.T) {
	without := platformAction()
	without.Details = nil
	bytes, err := without.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	// Field 14, a length-delimited arm this SDK has never seen.
	bytes = append(bytes, 0x72, 0x03, 0x0a, 0x01, 'x')
	event, err := raw.DecodeTriggerEvent(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if event.Details != nil {
		t.Fatalf("details %+v", event.Details)
	}
	if event.GetEventId() != "event-1" || event.GetIdempotencyKey() != "meet-123-key" {
		t.Fatalf("event %+v", event)
	}
}

func TestATriggerFamilyNewerThanTheSDKIsUnknown(t *testing.T) {
	bytes, err := (&triggersv1.TriggerEvent{EventId: "event-2"}).MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	// A type whose only family is field 8, which this SDK does not know.
	bytes = append(bytes, 0x12, 0x04, 0x42, 0x02, 0x08, 0x01)
	event, err := raw.DecodeTriggerEvent(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if event.GetEventId() != "event-2" {
		t.Fatalf("event %+v", event)
	}
	if got := raw.EventTrigger(event); got.Known() {
		t.Fatalf("trigger %v is known", got)
	}
	if got := raw.EventTrigger(&triggersv1.TriggerEvent{}); got.Known() {
		t.Fatalf("an event with no type is %v", got)
	}
}

func TestBytesThatAreNoEventFail(t *testing.T) {
	if event, err := raw.DecodeTriggerEvent([]byte{0xff, 0xff, 0xff}); err == nil {
		t.Fatalf("decoded %+v", event)
	}
}
