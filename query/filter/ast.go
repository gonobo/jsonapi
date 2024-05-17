package filter

import (
	"errors"
	"slices"
	"strings"
)

type assoc int

const (
	assocUndefined assoc = iota
	assocLeft
	assocRight
)

type symbol int

const (
	symbolUndefined symbol = iota
	symbolAND
	symbolOR
	symbolNOT
	symbolVAR
)

func (s symbol) isOperation() bool {
	return slices.Contains([]symbol{
		symbolAND,
		symbolOR,
		symbolNOT,
	}, s)
}

func (s symbol) precedence() int {
	switch s {
	case symbolNOT:
		return 3
	case symbolAND:
		return 2
	case symbolOR:
		return 1
	}
	return -1
}

func (s symbol) associativity() assoc {
	switch s {
	case symbolNOT:
		return assocLeft
	case symbolAND:
		return assocLeft
	case symbolOR:
		return assocLeft
	}
	return assocUndefined
}

type token struct {
	value  string
	symbol symbol
}

type node struct {
	token token
	left  *node
	right *node
}

type lex struct {
	position int
	tokens   []string
}

func (lex) tokenFromString(str string) token {
	token := token{value: str}
	switch Operation(strings.ToLower(str)) {
	case OperationAND:
		token.symbol = symbolAND
	case OperationNOT:
		token.symbol = symbolNOT
	case OperationOR:
		token.symbol = symbolOR
	default:
		token.symbol = symbolVAR
	}
	return token
}

func (l lex) peek() (token, bool) {
	var token token
	if l.position >= len(l.tokens) {
		return token, false
	}

	return l.tokenFromString(l.tokens[l.position]), true
}

func (l *lex) next() (token, bool) {
	token, ok := l.peek()
	l.position++
	return token, ok
}

type astBuilder struct {
	lexer lex
}

func (t astBuilder) build(tokens []string) (*node, error) {
	t.lexer.tokens = tokens

	primary, err := t.primary()
	if err != nil {
		return nil, err
	}

	return t.parseNode(primary, 0)
}

func (t *astBuilder) parseNode(left *node, minPrecendence int) (*node, error) {
	for {
		operator, ok := t.lexer.peek()
		if !ok || !operator.symbol.isOperation() || operator.symbol.precedence() < minPrecendence {
			break
		}

		t.lexer.next()

		right, err := t.primary()

		if err != nil {
			return nil, err
		}

		for {
			lookahead, ok := t.lexer.peek()
			if !ok {
				break
			} else if !lookahead.symbol.isOperation() {
				break
			}

			operatorPrecendence := operator.symbol.precedence()
			lookaheadPrecedence := lookahead.symbol.precedence()
			lookaheadAssoc := lookahead.symbol.associativity()

			if !(lookaheadPrecedence > operatorPrecendence ||
				lookaheadAssoc == assocRight && lookaheadPrecedence == operatorPrecendence) {
				break
			}

			right, err = t.parseNode(right, lookaheadPrecedence)

			if err != nil {
				return nil, err
			}
		}

		node := &node{token: operator}
		node.left = left
		node.right = right
		left = node
	}

	return left, nil
}

func (t *astBuilder) primary() (*node, error) {
	token, ok := t.lexer.next()
	if !ok {
		return nil, errors.New("empty token list")
	}

	root := &node{token: token}

	switch token.symbol {
	case symbolNOT:
		left, err := t.primary()
		root.left = left
		return root, err
	case symbolVAR:
		break
	default:
		return nil, errors.New("expected variable as first token")
	}

	return root, nil
}
