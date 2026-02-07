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
	return p.parseBlock()
}

//{stmt ; stmt ; stmt;} 
func (p *Parser) parseBlock() *ast.BlockNode {
    block := &ast.BlockNode{Statements: []ast.Node{}}

    for p.curToken.Type != token.EOF &&
		p.curToken.Type != token.FI &&
		p.curToken.Type != token.ELSE &&
		p.curToken.Type != token.THEN {

		stmt := p.parseLogical()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}

		if p.curToken.Type == token.SEMICOLON || p.curToken.Type == token.NEWLINE {
			p.nextToken()
		}
	}
	return block
}

func (p *Parser) parseLogical() ast.Node {
    left := p.parsePipeline()

    for p.curToken.Type == token.AND || p.curToken.Type == token.OR {
        operator := p.curToken.Literal
        p.nextToken()
        right := p.parsePipeline()
        
        left = &ast.BinaryNode{
            Left:     left,
            Operator: operator,
            Right:    right,
        }
    }
    return left
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
    if p.curToken.Type == token.IF {
        return p.parseIf()
    }
    cmd := &ast.CommandNode{Args: []string{}} 
    var result ast.Node = cmd

    for p.curToken.Type != token.EOF && 
        p.curToken.Type != token.PIPE && 
        p.curToken.Type != token.SEMICOLON &&
        p.curToken.Type != token.THEN &&  
        p.curToken.Type != token.ELSE &&  
        p.curToken.Type != token.FI &&
        p.curToken.Type != token.AND &&
        p.curToken.Type != token.OR  {    
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

func (p *Parser) parseIf() ast.Node {
    p.nextToken() // consume 'if'
    condition := p.parseBlock()

    if p.curToken.Type != token.THEN {
        return nil
    }
    p.nextToken() // consume 'then'
    consequence := p.parseBlock()

    var alternative ast.Node = nil

    // Check if we hit an 'ELSE' before 'FI'
    if p.curToken.Type == token.ELSE {
        p.nextToken() // consume 'else'
        alternative = p.parseBlock()
    }

    if p.curToken.Type != token.FI {
        return nil
    }
    p.nextToken() // consume 'fi'

    return &ast.IfNode{
        Condition: condition, 
        Then:      consequence, 
        Else:      alternative,
    }
}