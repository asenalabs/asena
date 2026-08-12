package rule

// TokenType says what kind of token this is.
type TokenType int

const (
	// AND is "&&".
	AND TokenType = iota
	// OR is "||".
	OR
	// NOT is "!".
	NOT
	// LPAREN is "(".
	LPAREN
	// RPAREN is ")".
	RPAREN
	// FUNC is one full matcher call, like "Host(`example.com`)".
	FUNC
)

// Token is one small piece of a rule.
type Token struct {
	Type TokenType
	// Value is the original text, like "&&" or "Host(`example.com`)".
	// We keep it so error messages can show the user's own words.
	Value string
}
