package event

type Matcher interface {
	Match(event Event) bool
}

type isMatcher struct {
	baseType Type
}

func Is(event Type) Matcher {
	return &isMatcher{event}
}

func (m *isMatcher) Match(event Event) bool {
	return m.baseType == event.Type()
}

type isOneOfMatcher struct {
	types []Type
}

func IsOneOf(eventTypes ...Type) Matcher {
	return &isOneOfMatcher{types: eventTypes}
}

func (m *isOneOfMatcher) Match(event Event) bool {
	for _, t := range m.types {
		if event.Type() == t {
			return true
		}
	}
	return false
}
