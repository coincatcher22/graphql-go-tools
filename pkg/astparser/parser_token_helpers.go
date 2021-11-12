package astparser

import (
	"github.com/jensneuse/graphql-go-tools/pkg/ast"
	"github.com/jensneuse/graphql-go-tools/pkg/lexer/identkeyword"
	"github.com/jensneuse/graphql-go-tools/pkg/lexer/keyword"
	"github.com/jensneuse/graphql-go-tools/pkg/lexer/token"
)

// hasNextToken - checks that we haven't reached eof
func (p *Parser) hasNextToken() bool {
	return p.currentToken+1 < p.maxTokens
}

// next - increments current token index if hasNextToken
// otherwise returns current token
func (p *Parser) next() int {
	if p.hasNextToken() {
		p.currentToken++
	}
	return p.currentToken
}

// read - increments currentToken index and return token if hasNextToken
// otherwise returns keyword.EOF
func (p *Parser) read() token.Token {
	if p.hasNextToken() {
		return p.tokens[p.next()]
	}

	return token.Token{
		Keyword: keyword.EOF,
	}
}

// peek - returns token next to currentToken if hasNextToken
// otherwise returns keyword.EOF
func (p *Parser) peek() keyword.Keyword {
	if p.hasNextToken() {
		nextIndex := p.currentToken + 1
		return p.tokens[nextIndex].Keyword
	}
	return keyword.EOF
}

// peekLiteral - returns token next to currentToken and token name as a ast.ByteSliceReference if hasNextToken
// otherwise returns keyword.EOF
func (p *Parser) peekLiteral() (keyword.Keyword, ast.ByteSliceReference) {
	if p.hasNextToken() {
		nextIndex := p.currentToken + 1
		return p.tokens[nextIndex].Keyword, p.tokens[nextIndex].Literal
	}
	return keyword.EOF, ast.ByteSliceReference{}
}

// peekEquals - checks that next token is equal to key
func (p *Parser) peekEquals(key keyword.Keyword) bool {
	return p.peek() == key
}

// peekEqualsIdentKey - checks that next token is an identifier
func (p *Parser) peekEqualsIdentKey(identKey identkeyword.IdentKeyword) bool {
	key, literal := p.peekLiteral()
	if key != keyword.IDENT {
		return false
	}
	actualKey := p.identKeywordSliceRef(literal)
	return actualKey == identKey
}

func (p *Parser) mustRead(key keyword.Keyword) (next token.Token) {
	next = p.read()
	if next.Keyword != key {
		p.errUnexpectedToken(next, key)
	}
	return
}

func (p *Parser) mustReadIdentKey(key identkeyword.IdentKeyword) (next token.Token) {
	next = p.read()
	if next.Keyword != keyword.IDENT {
		p.errUnexpectedToken(next, keyword.IDENT)
	}
	identKey := p.identKeywordToken(next)
	if identKey != key {
		p.errUnexpectedIdentKey(next, identKey, key)
	}
	return
}

func (p *Parser) mustReadExceptIdentKey(key identkeyword.IdentKeyword) (next token.Token) {
	next = p.read()
	if next.Keyword != keyword.IDENT {
		p.errUnexpectedToken(next, keyword.IDENT)
	}
	identKey := p.identKeywordToken(next)
	if identKey == key {
		p.errUnexpectedIdentKey(next, identKey, key)
	}
	return
}

func (p *Parser) mustReadOneOf(keys ...identkeyword.IdentKeyword) (token.Token, identkeyword.IdentKeyword) {
	next := p.read()

	identKey := p.identKeywordToken(next)
	for _, expectation := range keys {
		if identKey == expectation {
			return next, identKey
		}
	}
	p.errUnexpectedToken(next)
	return next, identKey
}
