package raw_test

import (
	"encoding/hex"
	"testing"

	modsv1 "kyzmet.ai/go/proto/mods/v1"
	routerv1 "kyzmet.ai/go/proto/router/v1"
)

// An inference router's payloads are Protocol's messages in the canonical
// binary encoding. The expected bytes are google.golang.org/protobuf's
// encoding of the same message, so the reflection-free codec stays
// interchangeable with every other Protocol consumer.
func TestARouterPayloadEncodesAsProtocolDoes(t *testing.T) {
	response := &routerv1.RecommendModelsResponse{
		SchemaVersion: 1,
		Defaults: &routerv1.RouterModelSelections{
			Chat: &routerv1.RouterModelSelection{
				ModelId:   "qwen3-8b",
				ProfileId: "balanced",
				ExecutionConfig: &modsv1.ModelExecutionConfig{
					SchemaVersion: 1,
					ModelId:       "qwen3-8b",
					OptionId:      "q4",
				},
			},
		},
		Preferred: &routerv1.RouterModelSelections{},
	}
	const canonical = "080112280a260a087177656e332d3862120862616c616e6365641a10080112087177656e332d38621a0271341a00"

	encoded, err := response.MarshalVT()
	if err != nil {
		t.Fatalf("MarshalVT: %v", err)
	}
	if got := hex.EncodeToString(encoded); got != canonical {
		t.Fatalf("encoding: got %s, want %s", got, canonical)
	}
	bytes, _ := hex.DecodeString(canonical)
	decoded := &routerv1.RecommendModelsResponse{}
	if err := decoded.UnmarshalVT(bytes); err != nil {
		t.Fatalf("UnmarshalVT: %v", err)
	}
	if !decoded.EqualVT(response) {
		t.Fatalf("decoded %v, want %v", decoded, response)
	}
}

// Bytes that are not the message fail to decode rather than yield a partial value.
func TestARouterPayloadRefusesBytesThatAreNotTheMessage(t *testing.T) {
	if err := (&routerv1.RecommendModelsRequest{}).UnmarshalVT([]byte{0xff, 0xff, 0xff}); err == nil {
		t.Fatal("UnmarshalVT accepted bytes that are not a RecommendModelsRequest")
	}
}
