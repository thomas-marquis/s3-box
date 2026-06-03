package event

import (
	"fmt"
	"reflect"
)

type EventMatcher struct {
	payloads map[string]Payload
	matchers map[string]Matcher
}

func NewEventMatcher() *EventMatcher {
	return &EventMatcher{
		payloads: make(map[string]Payload),
		matchers: make(map[string]Matcher),
	}
}

func (m *EventMatcher) AddPayload(label string, payload Payload) {
	m.payloads[label] = payload
}

func (m *EventMatcher) AddMatcher(label string, matcher Matcher) {
	m.matchers[label] = matcher
}

func (m *EventMatcher) Match(evt Event, label string) bool {
	if matcher, ok := m.matchers[label]; ok {
		return matcher.Match(evt)
	}
	payload, ok := m.payloads[label]
	if !ok {
		return false
	}
	return evt.Type() == payload.Type() && reflect.DeepEqual(evt.Payload, payload)
}

func (m *EventMatcher) LabelForEvent(evt Event) string {
	for label, payload := range m.payloads {
		if evt.Type() == payload.Type() && reflect.DeepEqual(evt.Payload, payload) {
			return label
		}
	}
	return fmt.Sprintf("%v", evt.Payload)
}

// AnyMatcher matches any event
type AnyMatcher struct{}

func (m *AnyMatcher) Match(evt Event) bool {
	return true
}

// FieldMatcher matches events with a specific field value
type FieldMatcher struct {
	Field string
	Value any
}

func (m *FieldMatcher) Match(evt Event) bool {
	v := reflect.ValueOf(evt.Payload)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}
	field := v.FieldByName(m.Field)
	if !field.IsValid() {
		return false
	}
	return reflect.DeepEqual(field.Interface(), m.Value)
}
