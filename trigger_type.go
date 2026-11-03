package raw

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// TriggerFamily names a trigger family: the first segment of a canonical key.
type TriggerFamily string

// The trigger families this SDK knows. TriggerFamilyUnknown is a family, or
// an event of a family, newer than the SDK: a trigger the handler does not
// know.
const (
	TriggerFamilyUnknown   TriggerFamily = ""
	TriggerFamilyPlatform  TriggerFamily = "platform"
	TriggerFamilyDev       TriggerFamily = "dev"
	TriggerFamilyTime      TriggerFamily = "time"
	TriggerFamilyConnector TriggerFamily = "connector"
	TriggerFamilyDevice    TriggerFamily = "device"
)

// TriggerType is a decoded kyzmet.triggers.v1.TriggerType: the trigger in
// trigger-invocation.trigger-type and consent-summary.trigger-type, and the
// argument to get-consent-readiness and consent-scope.triggers.
//
// Event is the family's event specifier ("battery.level", "push"); Provider
// is set only for connector triggers. Compare triggers with ==, or by Key.
type TriggerType struct {
	Family   TriggerFamily
	Provider string
	Event    string
}

// Kyzmet-owned trigger events, by family, with their canonical specifiers.
const (
	PlatformEventActionRequested  = "action.requested"
	DevEventTestEvent             = "test.event"
	TimeEventSchedule             = "schedule"
	DeviceEventBatteryCharging    = "battery.charging"
	DeviceEventBatteryDischarging = "battery.discharging"
	DeviceEventBatteryLevel       = "battery.level"
	DeviceEventNetworkAvailable   = "network.available"
	DeviceEventAppForeground      = "app.foreground"
	DeviceEventAppBackground      = "app.background"
	DeviceEventLocked             = "locked"
	DeviceEventUnlocked           = "unlocked"
)

// The specifier of each enum value, indexed by its protobuf number.
var triggerEvents = map[TriggerFamily][]string{
	TriggerFamilyPlatform: {"", PlatformEventActionRequested},
	TriggerFamilyDev:      {"", DevEventTestEvent},
	TriggerFamilyTime:     {"", TimeEventSchedule},
	TriggerFamilyDevice: {"", DeviceEventBatteryCharging, DeviceEventBatteryDischarging, DeviceEventBatteryLevel,
		DeviceEventNetworkAvailable, DeviceEventAppForeground, DeviceEventAppBackground, DeviceEventLocked, DeviceEventUnlocked},
}

var triggerFamilyNumbers = []TriggerFamily{"", TriggerFamilyPlatform, TriggerFamilyDev, TriggerFamilyTime, TriggerFamilyConnector, TriggerFamilyDevice}

// Errors from the trigger-identifier codec.
var (
	ErrMalformedTriggerType = errors.New("malformed trigger type")
	ErrUnknownTriggerType   = errors.New("unknown trigger type")
	ErrInvalidTriggerType   = errors.New("invalid trigger specifier")
	ErrNotCanonicalTrigger  = errors.New("not a canonical trigger key")
)

// PlatformTrigger, DevTrigger, TimeTrigger, DeviceTrigger and
// ConnectorTrigger build trigger types.
func PlatformTrigger(event string) TriggerType {
	return TriggerType{Family: TriggerFamilyPlatform, Event: event}
}
func DevTrigger(event string) TriggerType { return TriggerType{Family: TriggerFamilyDev, Event: event} }
func TimeTrigger(event string) TriggerType {
	return TriggerType{Family: TriggerFamilyTime, Event: event}
}
func DeviceTrigger(event string) TriggerType {
	return TriggerType{Family: TriggerFamilyDevice, Event: event}
}
func ConnectorTrigger(provider, event string) TriggerType {
	return TriggerType{Family: TriggerFamilyConnector, Provider: provider, Event: event}
}

// Known reports whether the trigger is one this SDK knows.
func (t TriggerType) Known() bool { _, ok := t.Key(); return ok }

// Key is the canonical key ("device.battery.level", "connector.github.push").
// It is false for an unknown trigger or one outside Protocol's constraints.
func (t TriggerType) Key() (string, bool) {
	switch t.Family {
	case TriggerFamilyConnector:
		if !validTriggerProvider(t.Provider) || !validTriggerEvent(t.Event) {
			return "", false
		}
		return "connector." + t.Provider + "." + t.Event, true
	case TriggerFamilyPlatform, TriggerFamilyDev, TriggerFamilyTime, TriggerFamilyDevice:
		if t.Provider != "" || triggerEventNumber(t.Family, t.Event) == 0 {
			return "", false
		}
		return string(t.Family) + "." + t.Event, true
	}
	return "", false
}

// String is the canonical key, or "unknown".
func (t TriggerType) String() string {
	if key, ok := t.Key(); ok {
		return key
	}
	return "unknown"
}

// UnknownTriggerText is the text an unknown trigger marshals as. It has no
// ".", so no canonical key can be mistaken for it.
const UnknownTriggerText = "unknown"

// MarshalText writes the canonical key, or UnknownTriggerText for an unknown
// trigger.
func (t TriggerType) MarshalText() ([]byte, error) {
	key, ok := t.Key()
	if !ok {
		return []byte(UnknownTriggerText), nil
	}
	return []byte(key), nil
}

// UnmarshalText parses a canonical key, or UnknownTriggerText as an unknown
// trigger.
func (t *TriggerType) UnmarshalText(text []byte) error {
	if string(text) == UnknownTriggerText {
		*t = TriggerType{}
		return nil
	}
	parsed, err := ParseTriggerType(string(text))
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// ParseTriggerType parses a canonical key, such as one written in a manifest.
func ParseTriggerType(key string) (TriggerType, error) {
	family, specifier, ok := strings.Cut(key, ".")
	if !ok {
		return TriggerType{}, ErrNotCanonicalTrigger
	}
	t := TriggerType{Family: TriggerFamily(family), Event: specifier}
	if t.Family == TriggerFamilyConnector {
		t.Provider, t.Event, ok = strings.Cut(specifier, ".")
		if !ok {
			return TriggerType{}, ErrNotCanonicalTrigger
		}
	}
	if canonical, ok := t.Key(); !ok || canonical != key {
		return TriggerType{}, ErrNotCanonicalTrigger
	}
	return t, nil
}

// DecodeTriggerType decodes the encoded TriggerType the host sends. A family
// or an event newer than this SDK decodes as an unknown trigger (Family
// TriggerFamilyUnknown); only bytes that are not a protobuf encoding fail.
func DecodeTriggerType(b []byte) (TriggerType, error) {
	var t TriggerType
	r := protoReader(b)
	for !r.done() {
		number, wireType, err := r.key()
		if err != nil {
			return TriggerType{}, err
		}
		known := number >= 1 && number < uint64(len(triggerFamilyNumbers))
		if wireType != 2 {
			if known {
				return TriggerType{}, ErrMalformedTriggerType
			}
			if err := r.skip(wireType); err != nil {
				return TriggerType{}, err
			}
			continue
		}
		body, err := r.bytes()
		if err != nil {
			return TriggerType{}, err
		}
		if !known {
			continue
		}
		family := triggerFamilyNumbers[number]
		if family == TriggerFamilyConnector {
			t, err = decodeConnectorTrigger(body)
		} else {
			t, err = decodeEnumTrigger(family, body)
		}
		if err != nil {
			return TriggerType{}, err
		}
	}
	return t, nil
}

// Encode encodes the trigger for the host.
func (t TriggerType) Encode() ([]byte, error) {
	if _, ok := t.Key(); !ok {
		if t.Family == TriggerFamilyConnector {
			return nil, ErrInvalidTriggerType
		}
		return nil, ErrUnknownTriggerType
	}
	var body []byte
	if t.Family == TriggerFamilyConnector {
		body = appendProtoBytes(body, 1, []byte(t.Provider))
		body = appendProtoBytes(body, 2, []byte(t.Event))
	} else {
		body = appendProtoVarint(body, 1<<3)
		body = appendProtoVarint(body, triggerEventNumber(t.Family, t.Event))
	}
	for number, family := range triggerFamilyNumbers {
		if family == t.Family {
			return appendProtoBytes(nil, uint64(number), body), nil
		}
	}
	return nil, ErrUnknownTriggerType
}

func triggerEventNumber(family TriggerFamily, event string) uint64 {
	for number, specifier := range triggerEvents[family] {
		if number > 0 && specifier == event {
			return uint64(number)
		}
	}
	return 0
}

func decodeEnumTrigger(family TriggerFamily, body []byte) (TriggerType, error) {
	var event uint64
	r := protoReader(body)
	for !r.done() {
		number, wireType, err := r.key()
		if err != nil {
			return TriggerType{}, err
		}
		if number == 1 && wireType == 0 {
			if event, err = r.varint(); err != nil {
				return TriggerType{}, err
			}
		} else if err := r.skip(wireType); err != nil {
			return TriggerType{}, err
		}
	}
	events := triggerEvents[family]
	if event == 0 || event >= uint64(len(events)) {
		return TriggerType{}, nil
	}
	return TriggerType{Family: family, Event: events[event]}, nil
}

func decodeConnectorTrigger(body []byte) (TriggerType, error) {
	t := TriggerType{Family: TriggerFamilyConnector}
	r := protoReader(body)
	for !r.done() {
		number, wireType, err := r.key()
		if err != nil {
			return TriggerType{}, err
		}
		if (number == 1 || number == 2) && wireType == 2 {
			value, err := r.bytes()
			if err != nil {
				return TriggerType{}, err
			}
			if !utf8.Valid(value) {
				return TriggerType{}, ErrMalformedTriggerType
			}
			if number == 1 {
				t.Provider = string(value)
			} else {
				t.Event = string(value)
			}
		} else if err := r.skip(wireType); err != nil {
			return TriggerType{}, err
		}
	}
	// A specifier outside Protocol's constraints names no trigger this SDK
	// can key: the handler does not know it.
	if _, ok := t.Key(); !ok {
		return TriggerType{}, nil
	}
	return t, nil
}

func validTriggerProvider(v string) bool {
	if v == "" || len(v) > 32 || v[0] < 'a' || v[0] > 'z' {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

func validTriggerEvent(v string) bool {
	if v == "" || len(v) > 96 {
		return false
	}
	for _, segment := range strings.Split(v, ".") {
		if segment == "" || segment[0] < 'a' || segment[0] > 'z' {
			return false
		}
		for i := 0; i < len(segment); i++ {
			c := segment[i]
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				return false
			}
		}
	}
	return true
}

type protoReader []byte

func (r *protoReader) done() bool { return len(*r) == 0 }

func (r *protoReader) varint() (uint64, error) {
	var value uint64
	for shift := uint(0); shift < 64; shift += 7 {
		if len(*r) == 0 {
			return 0, ErrMalformedTriggerType
		}
		b := (*r)[0]
		*r = (*r)[1:]
		if shift == 63 && b > 1 {
			return 0, ErrMalformedTriggerType
		}
		value |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return value, nil
		}
	}
	return 0, ErrMalformedTriggerType
}

func (r *protoReader) key() (uint64, uint64, error) {
	key, err := r.varint()
	// Protobuf field numbers run from 1 to 2^29-1.
	if err != nil || key>>3 == 0 || key>>3 > 1<<29-1 {
		return 0, 0, ErrMalformedTriggerType
	}
	return key >> 3, key & 7, nil
}

func (r *protoReader) take(n uint64) ([]byte, error) {
	if uint64(len(*r)) < n {
		return nil, ErrMalformedTriggerType
	}
	out := (*r)[:n]
	*r = (*r)[n:]
	return out, nil
}

func (r *protoReader) bytes() ([]byte, error) {
	n, err := r.varint()
	if err != nil {
		return nil, err
	}
	return r.take(n)
}

func (r *protoReader) skip(wireType uint64) error {
	var err error
	switch wireType {
	case 0:
		_, err = r.varint()
	case 1:
		_, err = r.take(8)
	case 2:
		_, err = r.bytes()
	case 5:
		_, err = r.take(4)
	default:
		err = ErrMalformedTriggerType
	}
	return err
}

func appendProtoVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func appendProtoBytes(b []byte, number uint64, value []byte) []byte {
	b = appendProtoVarint(b, number<<3|2)
	b = appendProtoVarint(b, uint64(len(value)))
	return append(b, value...)
}
