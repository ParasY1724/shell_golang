package parser

import (
	"strings"

	"github.com/codecrafters-io/shell-starter-go/pkg/ast"
	"github.com/codecrafters-io/shell-starter-go/pkg/lexer"
	"github.com/codecrafters-io/shell-starter-go/pkg/token"
)

type Parser struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Parse() ast.Node {
	return p.parsePipeline()
}

// parsePipeline handles "cmd | cmd | cmd"
func (p *Parser) parsePipeline() ast.Node {
	left := p.parseCommand()

	for p.curToken.Type == token.PIPE {
		p.nextToken() // consume '|'
		right := p.parseCommand()
		left = &ast.PipeNode{Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseCommand() ast.Node {
    cmd := &ast.CommandNode{Args: []string{}} 
    var result ast.Node = cmd

    for p.curToken.Type != token.EOF && p.curToken.Type != token.PIPE {
        if p.curToken.Type == token.REDIRECT {
            op := p.curToken.Literal
            p.nextToken()

            if p.curToken.Type != token.WORD {
                return result
            }
            filename := p.curToken.Literal
            p.nextToken()

            fd := 1
            if strings.HasPrefix(op, "2") {
                fd = 2
            }

            result = &ast.RedirectNode{
                Stmt:     result,
                Location: filename,
                Type:     op,
                Fd:       fd,
            }
        } else {
            cmd.Args = append(cmd.Args, p.curToken.Literal)
            p.nextToken()
        }
    }
    return result
}