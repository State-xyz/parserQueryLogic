package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Expr interface{}

type Condition struct {
	Field  string
	Op     string
	Value  string
	Values []string
}

type AndExpr struct {
	Left  Expr
	Right Expr
}

type OrExpr struct {
	Left  Expr
	Right Expr
}

// Hỗ trợ identifier chứa - và +
var tokenRegex = regexp.MustCompile(`\s*([A-Za-z_][A-Za-z0-9_\-\+]*|!=|<=|>=|=|<|>|\(|\)|AND|OR|IN|NOT IN|'[^']*'|"[^"]*"|\d+|,|\S)\s*`)

func tokenize(input string) []string {
	matches := tokenRegex.FindAllStringSubmatch(input, -1)
	tokens := []string{}

	for _, m := range matches {
		tokens = append(tokens, strings.TrimSpace(m[1]))
	}

	return tokens
}

type parser struct {
	tokens []string
	pos    int
}

func (p *parser) current() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

func (p *parser) eat(expected string) bool {
	if strings.ToUpper(p.current()) == expected {
		p.pos++
		return true
	}
	return false
}

func (p *parser) next() {
	p.pos++
}

func (p *parser) parseExpr() Expr {
	left := p.parseAnd()

	for strings.ToUpper(p.current()) == "OR" {
		p.next()
		right := p.parseAnd()
		left = OrExpr{Left: left, Right: right}
	}

	return left
}

func (p *parser) parseAnd() Expr {
	left := p.parsePrimary()

	for {
		token := strings.ToUpper(p.current())

		if token == "AND" {
			p.next()
			right := p.parsePrimary()
			left = AndExpr{Left: left, Right: right}
			continue
		}

		// Nếu gặp OR hoặc ) hoặc hết chuỗi => OK
		if token == "OR" || token == ")" || token == "" {
			return left
		}

		// Nếu tiếp theo lại là identifier/value => lỗi thiếu AND/OR
		if isValueOrIdentifier(p.current()) {
			panic("missing AND/OR between conditions near: " + p.current())
		}

		return left
	}
}

func (p *parser) parsePrimary() Expr {

	// (subexpression)
	if p.eat("(") {
		expr := p.parseExpr()
		if !p.eat(")") {
			panic("missing closing parenthesis")
		}
		return expr
	}

	field := p.current()
	if field == "" {
		panic("unexpected end of input")
	}
	p.next()

	op := strings.ToUpper(p.current())

	// Handle IN / NOT IN
	if op == "IN" || (op == "NOT" && strings.ToUpper(p.tokens[p.pos+1]) == "IN") {

		if op == "NOT" {
			p.next()
			p.eat("IN")
			op = "NOT IN"
		} else {
			p.next()
		}

		if !p.eat("(") {
			panic("Expected '(' after IN / NOT IN")
		}

		values := []string{}
		for {
			val := p.current()
			if val == "" {
				panic("unexpected end of IN list")
			}

			values = append(values, val)
			p.next()

			if p.eat(")") {
				break
			}

			if !p.eat(",") {
				panic("Expected ',' in IN list")
			}
		}

		return Condition{
			Field:  field,
			Op:     op,
			Values: values,
		}
	}

	// Basic comparison operators
	if op != "=" && op != "!=" && op != "<" && op != ">" && op != "<=" && op != ">=" {
		panic("invalid operator: " + op)
	}
	p.next()

	value := p.current()
	if value == "" {
		panic("missing value after operator")
	}
	p.next()

	return Condition{
		Field: field,
		Op:    op,
		Value: value,
	}
}

func isValueOrIdentifier(tok string) bool {
	if tok == "" {
		return false
	}

	// String literal
	if (strings.HasPrefix(tok, "'") && strings.HasSuffix(tok, "'")) ||
		(strings.HasPrefix(tok, "\"") && strings.HasSuffix(tok, "\"")) {
		return true
	}

	// Number
	if regexp.MustCompile(`^\d+$`).MatchString(tok) {
		return true
	}

	// Identifier chứa chữ, số, _, -, +
	if regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_\-\+]*$`).MatchString(tok) {
		return true
	}

	return false
}

func ParseQuery(input string) Expr {
	tokens := tokenize(input)
	p := &parser{tokens: tokens}
	return p.parseExpr()
}

// Demo
func main() {
	query := `
		(A = hello-world AND B >= 5)
		OR C IN (x-y, "a+b", test-1)
		AND D NOT IN ('x-y', foo-bar+123)
	`

	expr := ParseQuery(query)
	fmt.Printf("\nParsed:\n%#v\n\n", expr)
}
