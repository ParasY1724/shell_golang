package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	WORD      = "WORD"      // ls, -la, filename, "quoted string"
	PIPE      = "|"         // |
	REDIRECT  = "REDIRECT"  // >, >>, <, 1>, 2>, 2>>
)

type Token struct {
	Type    TokenType
	Literal string
}