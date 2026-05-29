package event

import (
	"fmt"
	"strings"
)

func parseMarble(marble string) []op {
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
			for i < len(marble) && marble[i] != ')' {
				i++
			}
			if i >= len(marble) {
				panic("invalid marble: unclosed group '('")
			}
			groupContent := marble[start:i]
			i++
			subOps := parseMarble(groupContent)
			// Filter out ticks from group
			var filtered []op
			for _, so := range subOps {
				if so.kind == opTick {
					panic("ticks '-' or '_' are not allowed inside groups '(ab)'")
				}
				if so.kind == opGroup {
					panic("nested groups are not allowed")
				}
				filtered = append(filtered, so)
			}
			ops = append(ops, op{kind: opGroup, subOps: filtered})
		case c == '\'':
			i++
			start := i
			for i < len(marble) && marble[i] != '\'' {
				i++
			}
			if i >= len(marble) {
				panic("invalid marble: unclosed quote")
			}
			label := marble[start:i]
			i++
			if strings.Contains(label, "<-") {
				parts := strings.Split(label, "<-")
				if len(parts) != 2 {
					panic("invalid followup syntax: " + label)
				}
				ops = append(ops, op{kind: opFollowup, label: parts[0], target: parts[1]})
			} else {
				ops = append(ops, op{kind: opEvent, label: label})
			}
		default:
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				ops = append(ops, op{kind: opEvent, label: string(c)})
				i++
			} else {
				panic(fmt.Sprintf("invalid character in marble at position %d: %c", i, c))
			}
		}
	}
	return ops
}
