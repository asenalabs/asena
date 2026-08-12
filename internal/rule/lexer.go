package rule

import (
	"fmt"
	"strings"
)

// Lex reads a rule string and splits it into tokens.
//
// It does not check if the rule makes sense (like "does && have a left
// and right side"). It only finds the pieces. Checking if they fit
// together is the parser's job, in parser.go.
func Lex(input string) ([]Token, error) {
	src := strings.TrimSpace(input)
	var tokens []Token
	i := 0

	for i < len(src) {
		switch {
		case src[i] == ' ' || src[i] == '\t':
			// Skip blank space.
			i++

		case strings.HasPrefix(src[i:], "&&"):
			tokens = append(tokens, Token{AND, "&&"})
			i += 2

		case strings.HasPrefix(src[i:], "||"):
			tokens = append(tokens, Token{OR, "||"})
			i += 2

		case src[i] == '!':
			tokens = append(tokens, Token{NOT, "!"})
			i++

		case src[i] == '(':
			// A matcher call like "Host(`example.com`)" is read whole, in one
			// go, by the branch below. So if we see a lone "(" here, it
			// can only be a grouping paren, like in "(A && B)".
			tokens = append(tokens, Token{LPAREN, "("})
			i++

		case src[i] == ')':
			tokens = append(tokens, Token{RPAREN, ")"})
			i++

		case isIdentStart(src[i]):
			// Looks like the start of a matcher name, e.g. "Host".
			tok, next, err := lexFunc(src, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
			i = next

		default:
			return nil, fmt.Errorf("rule: unexpected character %q at position %d in %q", src[i], i, src)
		}
	}

	return tokens, nil
}

// lexFunc reads one whole matcher call, starting at i, and returns it as
// one FUNC token: name, parentheses, and the quoted value(s) inside.
//
// We count how deep the parentheses go, instead of stopping at the first
// ")". This way, a value that itself contains parentheses does not cut
// the call short too early.
func lexFunc(src string, start int) (Token, int, error) {
	i := start
	for i < len(src) && isIdentChar(src[i]) {
		i++
	}
	name := src[start:i]

	if i >= len(src) || src[i] != '(' {
		return Token{}, 0, fmt.Errorf("rule: expected '(' after matcher name %q in %q", name, src)
	}

	depth := 0
	end := i
	for end < len(src) {
		switch src[end] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end++
				return Token{FUNC, src[start:end]}, end, nil
			}
		}
		end++
	}

	return Token{}, 0, fmt.Errorf("rule: unclosed parenthesis in %q", src[start:])
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isIdentChar(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}
