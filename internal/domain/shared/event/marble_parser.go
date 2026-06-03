package event

import (
	"fmt"
	"strconv"
	"strings"
)

type MarbleEntryKind int

const (
	MarbleEntryTick MarbleEntryKind = iota
	MarbleEntryEvent
	MarbleEntryGroup
	MarbleEntryFollowup
	MarbleEntryOrdering // a->b
	MarbleEntryWindow   // a[1-3]
	MarbleEntryRelative // a--b
)

type MarbleEntry struct {
	Tick  int
	Label string
	Kind  MarbleEntryKind

	// For followup:
	Trigger string

	// For ordering:
	Before string // a->b: a.Before = "b"
	After  string // a->b: b.After = "a"

	// For windows:
	WindowStart int // a[1-3]: WindowStart = 1
	WindowEnd   int // a[1-3]: WindowEnd = 3

	// For relative time:
	Offset int // a--b: b.Offset = 2 (relative to a)
}

type MarbleDiagram struct {
	Entries []MarbleEntry
}

type MarbleParser struct{}

func (p *MarbleParser) Parse(marble string) (*MarbleDiagram, error) {
	ops, err := parseMarble(marble)
	if err != nil {
		return nil, err
	}

	diagram := &MarbleDiagram{}
	currentTick := 0
	for _, o := range ops {
		p.applyOp(o, &currentTick, diagram)
	}

	return diagram, nil
}

// opKind defines the type of operation parsed from a marble string.
type opKind int

const (
	opTick opKind = iota
	opEvent
	opFollowup
	opGroup
	opOrdering // a->b
	opWindow   // a[1-3]
	opRelative // a--b
)

// op represents a single operation in a marble sequence.
type op struct {
	kind   opKind
	label  string
	target string
	subOps []op // for opGroup

	// For ordering:
	before string
	after  string

	// For windows:
	windowStart int
	windowEnd   int

	// For relative time:
	offset int
}

func parseMarble(marble string) ([]op, error) {
	var ops []op
	i := 0
	for i < len(marble) {
		c := marble[i]
		switch {
		case c == '-':
			ops = append(ops, op{kind: opTick})
			i++
		case c == '_':
			for i < len(marble) && marble[i] == '_' {
				i++
			}
			ops = append(ops, op{kind: opTick})
		case c == ' ':
			i++
		case c == '(':
			i++
			start := i
			for i < len(marble) && marble[i] != ')' { // nested groups are not allowed
				i++
			}
			if i >= len(marble) {
				return nil, fmt.Errorf("invalid marble: unclosed group '('")
			}
			groupContent := marble[start:i]
			i++
			subOps, err := parseMarble(groupContent)
			if err != nil {
				return nil, err
			}
			// Filter out ticks from group
			var filtered []op
			for _, so := range subOps {
				if so.kind == opTick {
					return nil, fmt.Errorf("ticks '-' or '_' are not allowed inside groups '(ab)'")
				}
				if so.kind == opGroup {
					return nil, fmt.Errorf("nested groups are not allowed")
				}
				filtered = append(filtered, so)
			}
			ops = append(ops, op{kind: opGroup, subOps: filtered})
		case c == '\'':
			labelOps, err := parseLabel(marble, i)
			if err != nil {
				return nil, err
			}
			ops = append(ops, labelOps...)
		default:
			var label string
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				label = string(c)
				i++
			} else {
				return nil, fmt.Errorf("invalid character in marble at position %d: %c", i, c)
			}

			//// Check for constraints starting with the label we just parsed
			//if i < len(marble) {
			//	next := marble[i]
			//	switch next {
			//	case '-':
			//		// Check for -> or --
			//		if i+1 < len(marble) {
			//			if marble[i+1] == '>' {
			//				// a->b
			//				i += 2
			//				if i < len(marble) {
			//					target := string(marble[i])
			//					i++
			//					ops = append(ops, op{kind: opEvent, label: label})
			//					ops = append(ops, op{kind: opOrdering, label: label, before: target})
			//					// Note: b will be added by the next iteration if it's a simple label
			//					// But we need to handle the target label here to skip it or it will be added as opEvent too.
			//					ops = append(ops, op{kind: opEvent, label: target})
			//					continue
			//				}
			//			} else if marble[i+1] == '-' { // TODO: remove this part of the syntax: a--b is already handled by the parser, no need to an additional opRelative op. Moreover, managing two-tick separated events that way wouldn't be more relevant than a single-tick or a three-tick separated event.
			//				// a--b
			//				i += 2
			//				if i < len(marble) {
			//					target := string(marble[i])
			//					i++
			//					ops = append(ops, op{kind: opEvent, label: label})
			//					ops = append(ops, op{kind: opTick})
			//					ops = append(ops, op{kind: opTick})
			//					ops = append(ops, op{kind: opEvent, label: target})
			//					ops = append(ops, op{kind: opRelative, label: label, after: target, offset: 2})
			//					continue
			//				}
			//			}
			//		}
			//	case '[':
			//		// a[1-3]
			//		i++
			//		start := i
			//		for i < len(marble) && marble[i] != ']' {
			//			i++
			//		}
			//		if i >= len(marble) {
			//			return nil, fmt.Errorf("invalid marble: unclosed window '['")
			//		}
			//		windowContent := marble[start:i]
			//		i++
			//		parts := strings.Split(windowContent, "-")
			//		if len(parts) != 2 {
			//			return nil, fmt.Errorf("invalid window syntax: %s", windowContent)
			//		}
			//		wStart, err := strconv.Atoi(parts[0])
			//		if err != nil {
			//			return nil, fmt.Errorf("invalid window start: %s", parts[0])
			//		}
			//		wEnd, err := strconv.Atoi(parts[1])
			//		if err != nil {
			//			return nil, fmt.Errorf("invalid window end: %s", parts[1])
			//		}
			//		ops = append(ops, op{kind: opWindow, label: label, windowStart: wStart, windowEnd: wEnd})
			//		continue
			//	}
			//}
			ops = append(ops, op{kind: opEvent, label: label})
		}
	}
	return ops, nil
}

func parseLabel(marble string, i *int) ([]op, error) {
	var ops []op
	*i++
	start := *i
	for *i < len(marble) && marble[*i] != '\'' {
		*i++
	}
	if *i >= len(marble) {
		return nil, fmt.Errorf("invalid marble: unclosed quote")
	}
	label := marble[start:*i]
	*i++
	if strings.Contains(label, "<-") {
		parts := strings.Split(label, "<-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid followup syntax: %s", label)
		}
		ops = append(ops, op{kind: opFollowup, label: parts[0], target: parts[1]})
	} else {
		ops = append(ops, op{kind: opEvent, label: label})
	}
	return ops, nil
}

func parseLabelConstraint(marble, label string, i *int) ([]op, error) {
	var ops []op
	// Check for constraints starting with the label we just parsed
	if *i < len(marble) {
		next := marble[*i]
		switch next {
		case '-':
			// Check for ->
			if *i+1 < len(marble) {
				if marble[*i+1] == '>' {
					// a->b
					*i += 2
					if i < len(marble) {
						target := string(marble[i])
						i++
						ops = append(ops, op{kind: opEvent, label: label})
						ops = append(ops, op{kind: opOrdering, label: label, before: target})
						// Note: b will be added by the next iteration if it's a simple label
						// But we need to handle the target label here to skip it or it will be added as opEvent too.
						ops = append(ops, op{kind: opEvent, label: target})
						continue
					}
				}
			}
		case '[':
			// a[1-3]
			i++
			start := i
			for i < len(marble) && marble[i] != ']' {
				i++
			}
			if i >= len(marble) {
				return nil, fmt.Errorf("invalid marble: unclosed window '['")
			}
			windowContent := marble[start:i]
			i++
			parts := strings.Split(windowContent, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid window syntax: %s", windowContent)
			}
			wStart, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid window start: %s", parts[0])
			}
			wEnd, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid window end: %s", parts[1])
			}
			ops = append(ops, op{kind: opWindow, label: label, windowStart: wStart, windowEnd: wEnd})
			continue
		}
	}
}

func (p *MarbleParser) applyOp(o op, currentTick *int, diagram *MarbleDiagram) {
	switch o.kind {
	case opTick:
		*currentTick++
	case opEvent:
		diagram.Entries = append(diagram.Entries, MarbleEntry{
			Tick:  *currentTick,
			Label: o.label,
			Kind:  MarbleEntryEvent,
		})
		*currentTick++
	case opFollowup:
		diagram.Entries = append(diagram.Entries, MarbleEntry{
			Tick:    *currentTick,
			Label:   o.label,
			Kind:    MarbleEntryFollowup,
			Trigger: o.target,
		})
		*currentTick++
	case opGroup:
		for _, sub := range o.subOps {
			switch sub.kind {
			case opEvent:
				diagram.Entries = append(diagram.Entries, MarbleEntry{
					Tick:  *currentTick,
					Label: sub.label,
					Kind:  MarbleEntryEvent,
				})
			case opFollowup:
				diagram.Entries = append(diagram.Entries, MarbleEntry{
					Tick:    *currentTick,
					Label:   sub.label,
					Kind:    MarbleEntryFollowup,
					Trigger: sub.target,
				})
			case opOrdering:
				diagram.Entries = append(diagram.Entries, MarbleEntry{
					Tick:   *currentTick,
					Label:  sub.label,
					Kind:   MarbleEntryOrdering,
					Before: sub.before,
				})
			case opWindow:
				diagram.Entries = append(diagram.Entries, MarbleEntry{
					Tick:        *currentTick,
					Label:       sub.label,
					Kind:        MarbleEntryWindow,
					WindowStart: sub.windowStart,
					WindowEnd:   sub.windowEnd,
				})
			case opRelative:
				diagram.Entries = append(diagram.Entries, MarbleEntry{
					Tick:   *currentTick,
					Label:  sub.label,
					Kind:   MarbleEntryRelative,
					After:  sub.after,
					Offset: sub.offset,
				})
			}
		}
		*currentTick++
	case opOrdering:
		diagram.Entries = append(diagram.Entries, MarbleEntry{
			Tick:   *currentTick,
			Label:  o.label,
			Kind:   MarbleEntryOrdering,
			Before: o.before,
		})
		// We don't increment tick for ordering constraint itself if it's just a constraint?
		// Usually constraints are separate from events.
		// But in marble testing, everything happens at some tick.
		// Let's say a->b doesn't consume time by itself, it's a property of events a and b.
		// However, a and b might be defined elsewhere.
		// If I have "a->b", it might mean event a arrives now, and b must arrive later.
		// Let's assume it doesn't increment tick if it's just a constraint.
		// Wait, the plan says a->b: a must arrive before b.
	case opWindow:
		diagram.Entries = append(diagram.Entries, MarbleEntry{
			Tick:        *currentTick,
			Label:       o.label,
			Kind:        MarbleEntryWindow,
			WindowStart: o.windowStart,
			WindowEnd:   o.windowEnd,
		})
	case opRelative:
		diagram.Entries = append(diagram.Entries, MarbleEntry{
			Tick:   *currentTick,
			Label:  o.label,
			Kind:   MarbleEntryRelative,
			After:  o.after,
			Offset: o.offset,
		})
	}
}
