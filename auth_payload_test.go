package raw_test

import (
	"encoding/hex"
	"testing"

	hostv1 "kyzmet.ai/go/proto/host/v1"
)

// What every generated payload message does, without reflection.
type authPayload interface {
	MarshalVT() ([]byte, error)
	UnmarshalVT([]byte) error
	EqualMessageVT(any) bool
}

// An auth-type's payloads are Protocol's messages in the canonical binary
// encoding. The expected bytes are google.golang.org/protobuf's encoding of
// the same messages, and the Rust crate's test pins them too, so every SDK and
// every Protocol consumer reads what the others write.
func TestAuthPayloadsEncodeAsProtocolDoes(t *testing.T) {
	enabled := true
	cases := []struct {
		name      string
		sent      authPayload
		received  authPayload
		canonical string
	}{
		{
			// A verification's complete after a SAML HTTP-POST redirect: nested
			// config, a oneof message callback and the stored credentials.
			name: "AuthOperationRequest",
			sent: &hostv1.AuthOperationRequest{
				IdentityId: "identity",
				Config: &hostv1.AuthProviderConfig{
					ProviderId: "okta",
					TypeId:     "com.example.okta-sso",
					Enabled:    &enabled,
					ClientId:   "client",
					Scopes:     []string{"openid", "email"},
				},
				FlowId:     "abcdefghijABCDEFGHIJ0123456789-_abcdefghijk",
				FlowSecret: []byte("secret"),
				FlowCallback: &hostv1.AuthFlowCallback{
					Callback: &hostv1.AuthFlowCallback_FormPost{FormPost: &hostv1.AuthFormPost{
						Fields: []*hostv1.AuthFormField{{Name: "SAMLResponse", Value: "PHNhbWw+"}},
					}},
				},
				StoredCredentials: []*hostv1.AuthStoredCredential{
					{CredentialId: "factor", ProviderPublicData: []byte("user and factor")},
				},
			},
			received:  &hostv1.AuthOperationRequest{},
			canonical: "0a086964656e7469747912350a046f6b74611214636f6d2e6578616d706c652e6f6b74612d73736f28013206636c69656e7472066f70656e69647205656d61696c322b6162636465666768696a4142434445464748494a303132333435363738392d5f6162636465666768696a6b3a06736563726574421c121a0a180a0c53414d4c526573706f6e7365120850484e686257772b4a190a06666163746f72120f7573657220616e6420666163746f72",
		},
		{
			// A regular field beside a oneof member: the shape Protocol numbers
			// so that Go and prost write the same bytes.
			name: "AuthRegistrationChallenge",
			sent: &hostv1.AuthRegistrationChallenge{
				FlowSecret: []byte("secret"),
				Challenge: &hostv1.AuthRegistrationChallenge_Prompt{Prompt: &hostv1.AuthPrompt{
					Prompt: &hostv1.AuthPrompt_TextInput{TextInput: &hostv1.AuthTextInput{
						Label: "Code",
						Kind:  hostv1.AuthTextInputKind_AUTH_TEXT_INPUT_KIND_CODE,
					}},
				}},
			},
			received:  &hostv1.AuthRegistrationChallenge{},
			canonical: "12067365637265742a0a0a080a04436f64651003",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded, err := c.sent.MarshalVT()
			if err != nil {
				t.Fatalf("MarshalVT: %v", err)
			}
			if got := hex.EncodeToString(encoded); got != c.canonical {
				t.Fatalf("encoding: got %s, want %s", got, c.canonical)
			}
			bytes, _ := hex.DecodeString(c.canonical)
			if err := c.received.UnmarshalVT(bytes); err != nil {
				t.Fatalf("UnmarshalVT: %v", err)
			}
			if !c.received.EqualMessageVT(c.sent) {
				t.Fatalf("decoded %v, want %v", c.received, c.sent)
			}
		})
	}
}

// Bytes that are not the message fail to decode rather than yield a partial value.
func TestAnAuthPayloadRefusesBytesThatAreNotTheMessage(t *testing.T) {
	if err := (&hostv1.AuthOperationRequest{}).UnmarshalVT([]byte{0xff, 0xff, 0xff}); err == nil {
		t.Fatal("UnmarshalVT accepted bytes that are not an AuthOperationRequest")
	}
}
