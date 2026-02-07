package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	WORD      = "WORD"      // ls, -la, filename, "quoted string"
	PIPE      = "|"         // |
	REDIRECT  = "REDIRECT"  // >, >>, <, 1>, 2>, 2>>

	SEMICOLON = ";"

	IF    = "if"
	THEN  = "then"
	ELSE  = "else"
	ELIF  = "elif"
	FI    = "fi"

	NEWLINE = "NEWLINE"

	AND       = "&&"
    OR        = "||"
)

var keywords = map[string]TokenType{
	"if":    IF,
	"then":  THEN,
	"else":  ELSE,
	"elif":  ELIF,
	"fi":    FI,
}

type Token struct {
	Type    TokenType
	Literal string
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return WORD
}