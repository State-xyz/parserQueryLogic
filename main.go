package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Expr interface{}

type Condition struct {
	Field string
	Op    string
	Value string
}

type AndExpr struct {
	Left  Expr
	Right Expr
}

type OrExpr struct {
	Left  Expr
	Right Expr
}

var tokenRegex = regexp.MustCompile(`\s*([A-Za-z_][A-Za-z0-9_]*|!=|<=|>=|=|<|>|\(|\)|AND|OR|\'[^\']*\'|"[^"]*"|\d+|\S)\s*`)

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
	for strings.ToUpper(p.current()) == "AND" {
		p.next()
		right := p.parsePrimary()
		left = AndExpr{Left: left, Right: right}
	}
	return left
}

func (p *parser) parsePrimary() Expr {
	if p.eat("(") {
		expr := p.parseExpr()
		if !p.eat(")") {
			panic("missing closing parenthesis")
		}
		return expr
	}

	// Parse condition: Field Op Value
	field := p.current()
	p.next()

	op := p.current()
	if op != "=" && op != "!=" && op != "<" && op != ">" && op != "<=" && op != ">=" {
		panic("invalid operator: " + op)
	}
	p.next()

	value := p.current()
	p.next()

	return Condition{Field: field, Op: op, Value: value}
}

func ParseQuery(input string) Expr {
	tokens := tokenize(input)
	p := &parser{tokens: tokens}
	return p.parseExpr()
}

func main() {
	query := `((A > 1 AND B <= 5) OR C != "test") AND D = 'hello'`
	expr := ParseQuery(query)
	fmt.Printf("Result: \n")
	fmt.Printf("%#v\n", expr)
}
