package lexer

import (
	"strings"

	"github.com/codecrafters-io/shell-starter-go/pkg/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	var tok token.Token

	if l.ch == 0 {
		tok.Literal = ""
		tok.Type = token.EOF
		return tok
	}

	// Handle Pipe
	if l.ch == '|' {
		tok = token.Token{Type: token.PIPE, Literal: "|"}
		l.readChar()
		return tok
	}

	if l.ch == ';' {
		tok = token.Token{Type: token.SEMICOLON, Literal: ";"}
		l.readChar()
		return tok
	}

	if isRedirectStart(l.ch) || (isDigit(l.ch) && (l.peekChar() == '>')) {
		literal := l.readRedirect()
		// Double check it wasn't just a number like "123"
		if strings.Contains(literal, ">") || strings.Contains(literal, "<") {
			tok.Type = token.REDIRECT
			tok.Literal = literal
			return tok
		}
	}
	
	if l.ch == '>' || l.ch == '<' {
		tok.Type = token.REDIRECT
		tok.Literal = l.readRedirect()
		return tok
	}

	tok.Literal = l.readWord()

	tok.Type = token.LookupIdent(tok.Literal)
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.readChar()
	}
}

func (l *Lexer) readWord() string {
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for l.ch != 0 {
		ch := l.ch

		if !inSingle && !inDouble && !escaped {
			if ch == ' ' || ch == '\t' || ch == '|' {
				break
			}

			if ch == '>' || ch == '<' {
				break
			}
		}

		if escaped {
			current.WriteByte(ch)
			escaped = false
			l.readChar()
			continue
		}

		if ch == '\\' {
			if inSingle {
				current.WriteByte(ch)
			} else if inDouble {
				peek := l.peekChar()
				if peek == '"' || peek == '\\' || peek == '$' || peek == '`' || peek == '\n' {
					escaped = true
				} else {
					current.WriteByte(ch)
				}
			} else {
				escaped = true
			}
			l.readChar()
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			l.readChar()
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			l.readChar()
			continue
		}

		current.WriteByte(ch)
		l.readChar()
	}

	return current.String()
}

func (l *Lexer) readRedirect() string {
	var res strings.Builder
	for isDigit(l.ch) {
		res.WriteByte(l.ch)
		l.readChar()
	}

	if l.ch == '>' {
		res.WriteByte(l.ch)
		l.readChar()
		if l.ch == '>' {
			res.WriteByte(l.ch)
			l.readChar()
		}
	} else if l.ch == '<' {
		res.WriteByte(l.ch)
		l.readChar()
	}
	return res.String()
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isRedirectStart(ch byte) bool {
	return ch == '>' || ch == '<'
}