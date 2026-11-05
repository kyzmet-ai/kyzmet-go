package raw_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"

	raw "kyzmet.ai/go"
)

type triggerVectors struct {
	Valid []struct {
		Canonical  string `json:"canonical"`
		EncodedHex string `json:"encoded_hex"`
	} `json:"valid"`
	AlternativeEncodings []struct {
		Canonical  string `json:"canonical"`
		EncodedHex string `json:"encoded_hex"`
	} `json:"alternative_encodings"`
	InvalidEncodings []struct {
		EncodedHex string `json:"encoded_hex"`
		Error      string `json:"error"`
		Guest      string `json:"guest"`
		Note       string `json:"note"`
	} `json:"invalid_encodings"`
	InvalidCanonicalKeys []struct {
		Text string `json:"text"`
	} `json:"invalid_canonical_keys"`
}

func loadTriggerVectors(t *testing.T) triggerVectors {
	t.Helper()
	data, err := os.ReadFile("../../protocol-sdk/protocol/test-vectors/trigger-type-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var v triggerVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTriggerTypeValidVectors(t *testing.T) {
	v := loadTriggerVectors(t)
	for _, entry := range v.Valid {
		decoded, err := raw.DecodeTriggerType(mustHex(t, entry.EncodedHex))
		if err != nil {
			t.Fatalf("%s: %v", entry.Canonical, err)
		}
		if key, ok := decoded.Key(); !ok || key != entry.Canonical {
			t.Fatalf("%s: decoded key %q", entry.Canonical, key)
		}
		parsed, err := raw.ParseTriggerType(entry.Canonical)
		if err != nil || parsed != decoded {
			t.Fatalf("%s: parsed %+v, %v", entry.Canonical, parsed, err)
		}
		encoded, err := parsed.Encode()
		if err != nil || hex.EncodeToString(encoded) != entry.EncodedHex {
			t.Fatalf("%s: encoded %x, %v", entry.Canonical, encoded, err)
		}
	}
	for _, entry := range v.AlternativeEncodings {
		decoded, err := raw.DecodeTriggerType(mustHex(t, entry.EncodedHex))
		if err != nil || decoded.String() != entry.Canonical {
			t.Fatalf("alternative %s: %v, %v", entry.Canonical, decoded, err)
		}
	}
}

func TestTriggerTypeEveryInvalidEncodingReadsAsTheVectorSays(t *testing.T) {
	for _, entry := range loadTriggerVectors(t).InvalidEncodings {
		decoded, err := raw.DecodeTriggerType(mustHex(t, entry.EncodedHex))
		switch entry.Guest {
		case "malformed":
			if !errors.Is(err, raw.ErrMalformedTriggerType) {
				t.Fatalf("%s: got %+v, %v", entry.Note, decoded, err)
			}
		case "unknown":
			if err != nil || decoded.Known() || decoded != (raw.TriggerType{}) {
				t.Fatalf("%s: got %+v, %v", entry.Note, decoded, err)
			}
		default:
			if key, ok := decoded.Key(); err != nil || !ok || key != entry.Guest {
				t.Fatalf("%s: got %q, %v", entry.Note, key, err)
			}
		}
	}
	if _, err := (raw.TriggerType{}).Encode(); !errors.Is(err, raw.ErrUnknownTriggerType) {
		t.Fatalf("unknown encoded: %v", err)
	}
}

func TestTriggerTypeInvalidCanonicalKeys(t *testing.T) {
	for _, entry := range loadTriggerVectors(t).InvalidCanonicalKeys {
		if _, err := raw.ParseTriggerType(entry.Text); !errors.Is(err, raw.ErrNotCanonicalTrigger) {
			t.Fatalf("%q: %v", entry.Text, err)
		}
	}
	if _, err := raw.ConnectorTrigger("GitHub", "push").Encode(); !errors.Is(err, raw.ErrInvalidTriggerType) {
		t.Fatalf("invalid connector: %v", err)
	}
}

func TestAppAndChannelTriggersOutsideProtocolConstraintsDoNotEncode(t *testing.T) {
	for _, bad := range []raw.TriggerType{raw.AppTrigger("a.b"), raw.ChannelTrigger("chat")} {
		if _, err := bad.Encode(); !errors.Is(err, raw.ErrInvalidTriggerType) {
			t.Fatalf("%+v: %v", bad, err)
		}
		if bad.Known() {
			t.Fatalf("%+v is known", bad)
		}
	}
}
