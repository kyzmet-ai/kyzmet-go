// protobuf-policy-file: allowed external (1 site) [c573c9fac0ec] — trigger payload bytes declared JSON by their source media type.
package raw

import (
	"encoding/json"
	"time"
)

// TriggerStatusCode is the handler's outcome, named by its google.rpc.Code
// (coretap.grpc.status.v1.Code). The broker acts on each code: OK and
// ALREADY_EXISTS settle the invocation, UNAVAILABLE, RESOURCE_EXHAUSTED and
// ABORTED are retried, and the rest are retried, dead-lettered or failed as
// the trigger design describes.
type TriggerStatusCode string

const (
	TriggerStatusCodeOK                 TriggerStatusCode = "OK"
	TriggerStatusCodeCancelled          TriggerStatusCode = "CANCELLED"
	TriggerStatusCodeUnknown            TriggerStatusCode = "UNKNOWN"
	TriggerStatusCodeInvalidArgument    TriggerStatusCode = "INVALID_ARGUMENT"
	TriggerStatusCodeDeadlineExceeded   TriggerStatusCode = "DEADLINE_EXCEEDED"
	TriggerStatusCodeNotFound           TriggerStatusCode = "NOT_FOUND"
	TriggerStatusCodeAlreadyExists      TriggerStatusCode = "ALREADY_EXISTS"
	TriggerStatusCodePermissionDenied   TriggerStatusCode = "PERMISSION_DENIED"
	TriggerStatusCodeResourceExhausted  TriggerStatusCode = "RESOURCE_EXHAUSTED"
	TriggerStatusCodeFailedPrecondition TriggerStatusCode = "FAILED_PRECONDITION"
	TriggerStatusCodeAborted            TriggerStatusCode = "ABORTED"
	TriggerStatusCodeOutOfRange         TriggerStatusCode = "OUT_OF_RANGE"
	TriggerStatusCodeUnimplemented      TriggerStatusCode = "UNIMPLEMENTED"
	TriggerStatusCodeInternal           TriggerStatusCode = "INTERNAL"
	TriggerStatusCodeUnavailable        TriggerStatusCode = "UNAVAILABLE"
	TriggerStatusCodeDataLoss           TriggerStatusCode = "DATA_LOSS"
	TriggerStatusCodeUnauthenticated    TriggerStatusCode = "UNAUTHENTICATED"
)

// TriggerDuration is a google.protobuf.Duration: whole seconds plus nanoseconds.
type TriggerDuration struct {
	Seconds int64 `json:"seconds"`
	Nanos   int32 `json:"nanos"`
}

// TriggerRetryInfo is the handler's hint for when the broker may retry.
type TriggerRetryInfo struct {
	RetryDelay TriggerDuration `json:"retry_delay"`
}

// NewTriggerRetryInfo returns retry info asking for a retry after delay.
func NewTriggerRetryInfo(delay time.Duration) TriggerRetryInfo {
	return TriggerRetryInfo{RetryDelay: TriggerDuration{
		Seconds: int64(delay / time.Second),
		Nanos:   int32(delay % time.Second),
	}}
}

// TriggerPayload carries broker-minimized trigger data plus classification metadata.
type TriggerPayload struct {
	MediaType       string   `json:"media_type"`
	Schema          *string  `json:"schema,omitempty"`
	Data            []byte   `json:"data"`
	Classifications []string `json:"classifications,omitempty"`
}

// TriggerAuthoritySummary describes authority the broker granted for this invocation.
type TriggerAuthoritySummary struct {
	GrantedConsent      []string `json:"granted_consent,omitempty"`
	GrantedCapabilities []string `json:"granted_capabilities,omitempty"`
	PolicyDecisions     []string `json:"policy_decisions,omitempty"`
}

// TriggerReceiptRequest describes broker-requested receipt behavior.
type TriggerReceiptRequest struct {
	UserVisible bool `json:"user_visible"`
}

// TriggerInvocation is the typed envelope delivered to handle-trigger.
type TriggerInvocation struct {
	InvocationID     string                  `json:"invocation_id"`
	TriggerType      TriggerType             `json:"trigger_type"`
	BindingID        string                  `json:"binding_id"`
	AppID            string                  `json:"app_id"`
	SphereID         string                  `json:"sphere_id"`
	Source           string                  `json:"source"`
	IdempotencyKey   *string                 `json:"idempotency_key,omitempty"`
	Payload          TriggerPayload          `json:"payload"`
	OccurredAt       string                  `json:"occurred_at"`
	InvokedAt        string                  `json:"invoked_at"`
	AuthoritySummary TriggerAuthoritySummary `json:"authority_summary"`
	ReceiptRequest   TriggerReceiptRequest   `json:"receipt_request"`
}

// TriggerResultNextAction is a follow-up action the platform may present.
type TriggerResultNextAction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// TriggerResultReceiptMetadata is user-visible receipt metadata.
type TriggerResultReceiptMetadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TriggerResult is the structured response returned from handle-trigger.
type TriggerResult struct {
	Status          TriggerStatusCode              `json:"status"`
	Title           string                         `json:"title"`
	Summary         string                         `json:"summary"`
	ReceiptMetadata []TriggerResultReceiptMetadata `json:"receipt_metadata"`
	NextActions     []TriggerResultNextAction      `json:"next_actions"`
	RetryInfo       *TriggerRetryInfo              `json:"retry_info,omitempty"`
}

// NewTriggerResult returns a result with list fields initialized for stable JSON.
func NewTriggerResult(status TriggerStatusCode, title, summary string) TriggerResult {
	return TriggerResult{
		Status:          status,
		Title:           title,
		Summary:         summary,
		ReceiptMetadata: []TriggerResultReceiptMetadata{},
		NextActions:     []TriggerResultNextAction{},
	}
}

// WithRetryAfter returns the result asking the broker to retry after delay.
func (r TriggerResult) WithRetryAfter(delay time.Duration) TriggerResult {
	info := NewTriggerRetryInfo(delay)
	r.RetryInfo = &info
	return r
}

// EventIdempotencyKey derives the event-level idempotency key from the invocation ID.
func EventIdempotencyKey(invocation TriggerInvocation) string {
	return invocation.InvocationID
}

// BindingIdempotencyKey derives the binding-level idempotency key from the binding ID.
func BindingIdempotencyKey(invocation TriggerInvocation) string {
	return invocation.BindingID
}

// DecodeTextPayload decodes trigger payload bytes as UTF-8 text.
func DecodeTextPayload(payload TriggerPayload) string {
	return string(payload.Data)
}

// DecodeJSONPayload decodes trigger payload bytes as JSON.
func DecodeJSONPayload(payload TriggerPayload, out interface{}) error {
	return json.Unmarshal(payload.Data, out)
}
