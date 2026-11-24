package gamestrings

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var ErrSyntax = errors.New("gamestrings: message syntax")

type argumentKind int

const (
	simpleArgument argumentKind = iota
	pluralArgument
	selectArgument
)

type argument struct {
	kind      argumentKind
	selectors []string
}

type parsed struct {
	arguments map[string][]argument
	length    int
}

type parser struct {
	source string
	pos    int
	result parsed
}

func parseMessage(text string) (parsed, error) {
	p := &parser{source: text, result: parsed{arguments: map[string][]argument{}}}
	length, err := p.message(false)
	if err != nil {
		return parsed{}, err
	}
	if p.pos < len(p.source) {
		return parsed{}, p.fail("unexpected }")
	}
	p.result.length = length
	return p.result, nil
}

func (p *parser) fail(reason string) error {
	return fmt.Errorf("%w: %s at byte %d", ErrSyntax, reason, p.pos)
}

func (p *parser) message(inPlural bool) (int, error) {
	length := 0
	literal := strings.Builder{}
	flush := func() {
		length += visibleLength(literal.String())
		literal.Reset()
	}
	for p.pos < len(p.source) {
		r, size := utf8.DecodeRuneInString(p.source[p.pos:])
		switch {
		case r == '}':
			flush()
			return length, nil
		case r == '{':
			flush()
			branchLength, err := p.argument()
			if err != nil {
				return 0, err
			}
			length += branchLength
		case r == '#' && inPlural:
			flush()
			p.pos += size
		default:
			literal.WriteRune(r)
			p.pos += size
		}
	}
	flush()
	return length, nil
}

func (p *parser) argument() (int, error) {
	p.pos++
	name := p.word()
	if !validName(name) {
		return 0, p.fail("bad placeholder name")
	}
	p.spaces()
	if p.take('}') {
		p.result.arguments[name] = append(p.result.arguments[name], argument{kind: simpleArgument})
		return 0, nil
	}
	if !p.take(',') {
		return 0, p.fail("expected , or }")
	}
	p.spaces()
	kindName := p.word()
	kinds := map[string]argumentKind{"plural": pluralArgument, "select": selectArgument}
	kind, known := kinds[kindName]
	if !known {
		return 0, p.fail("expected plural or select")
	}
	p.spaces()
	if !p.take(',') {
		return 0, p.fail("expected ,")
	}
	return p.branches(name, kind)
}

func (p *parser) branches(name string, kind argumentKind) (int, error) {
	longest := 0
	var selectors []string
	for {
		p.spaces()
		if p.take('}') {
			break
		}
		selector := p.word()
		if selector == "" {
			return 0, p.fail("expected a selector")
		}
		p.spaces()
		if !p.take('{') {
			return 0, p.fail("expected { after " + selector)
		}
		branchLength, err := p.message(kind == pluralArgument)
		if err != nil {
			return 0, err
		}
		if !p.take('}') {
			return 0, p.fail("unclosed branch " + selector)
		}
		selectors = append(selectors, selector)
		longest = max(longest, branchLength)
	}
	if len(selectors) == 0 {
		return 0, p.fail("no branches")
	}
	p.result.arguments[name] = append(p.result.arguments[name], argument{kind: kind, selectors: selectors})
	return longest, nil
}

func (p *parser) word() string {
	start := p.pos
	for p.pos < len(p.source) {
		r, size := utf8.DecodeRuneInString(p.source[p.pos:])
		if r == '{' || r == '}' || r == ',' || unicode.IsSpace(r) {
			break
		}
		p.pos += size
	}
	return p.source[start:p.pos]
}

func (p *parser) spaces() {
	for p.pos < len(p.source) && (p.source[p.pos] == ' ' || p.source[p.pos] == '\n' || p.source[p.pos] == '\t') {
		p.pos++
	}
}

func (p *parser) take(char byte) bool {
	if p.pos < len(p.source) && p.source[p.pos] == char {
		p.pos++
		return true
	}
	return false
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		if !letter && (i == 0 || r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func visibleLength(text string) int {
	length := 0
	for _, r := range norm.NFC.String(text) {
		if !unicode.Is(unicode.Mn, r) {
			length++
		}
	}
	return length
}
