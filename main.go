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

// REGEX token hóa:
// 1. Field trong backtick  → `[^`]+`
// 2. Operators
// 3. String literal
// 4. Identifier (value) chứa chữ-số-_-+-, nhưng KHÔNG cho backtick
var tokenRegex = regexp.MustCompile(`\s*(` + 
	"`[^`]+`" +                  // field
	`|!=|<=|>=|=|<|>` +          // operators
	`|\(|\)|AND|OR|IN|NOT IN` +  // keywords
	`|'[^']*'|"[^"]*"` +         // string literal
	`|[A-Za-z0-9_\-\+]+` +       // identifier value
	`|,|\S)\s*`)                 // comma + fallback


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

		// OK khi gặp OR, ), hoặc hết
		if token == "OR" || token == ")" || token == "" {
			return left
		}

		// Nếu token tiếp theo là identifier/value => lỗi thiếu AND/OR
		if isValueOrIdentifier(p.current()) {
			panic("missing AND/OR between conditions near: " + p.current())
		}

		return left
	}
}

func (p *parser) parsePrimary() Expr {

	// ( SUBEXPR )
	if p.eat("(") {
		expr := p.parseExpr()
		if !p.eat(")") {
			panic("missing closing parenthesis")
		}
		return expr
	}

	// Field bắt buộc là backtick
	fieldToken := p.current()

	if !isBacktickField(fieldToken) {
		panic("field name must be inside backticks, example: `A`")
	}

	field := fieldToken[1 : len(fieldToken)-1] // bỏ dấu `
	p.next()

	op := strings.ToUpper(p.current())

	// IN / NOT IN
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

	// Basic operators
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

// ---------------------- Helpers --------------------------

func isBacktickField(tok string) bool {
	return strings.HasPrefix(tok, "`") && strings.HasSuffix(tok, "`")
}

// Value hoặc identifier hợp lệ
func isValueOrIdentifier(tok string) bool {
	if tok == "" {
		return false
	}

	// string literal
	if (strings.HasPrefix(tok, "'") && strings.HasSuffix(tok, "'")) ||
		(strings.HasPrefix(tok, "\"") && strings.HasSuffix(tok, "\"")) {
		return true
	}

	// number
	if regexp.MustCompile(`^\d+$`).MatchString(tok) {
		return true
	}

	// identifier không backtick, có chữ số _ - +
	if regexp.MustCompile(`^[A-Za-z0-9_\-\+]+$`).MatchString(tok) {
		return true
	}

	return false
}

func ParseQuery(input string) Expr {
	tokens := tokenize(input)
	p := &parser{tokens: tokens}
	return p.parseExpr()
}

// ----------------------- Demo ----------------------------

func main() {
	query := `
		(` + "`A-B`" + ` = 1 AND ` + "`C+D`" + ` >= 5)
		OR ` + "`X`" + ` IN (1, abc-xyz, "hello-world")
		AND ` + "`Y-Z`" + ` NOT IN ('x-y', foo+bar)
	`

	expr := ParseQuery(query)
	fmt.Printf("\nParsed:\n%#v\n\n", expr)
}
