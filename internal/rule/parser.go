package rule

import "fmt"

// ParseRule turns a rule string into a Node tree, ready to check
// requests against.
//
// It reads the rule in this order, from loosest to tightest:
//
//	parseOr    -> handles "||"
//	parseAnd   -> handles "&&"
//	parseUnary -> handles "!"
//	parsePrimary -> one matcher call, or "( ... )"
//
// Each step calls the next step first, before looking for its own
// operator. This is what makes "&&" bind tighter than "||", the same as
// in Go itself: "A || B && C" becomes "A || (B && C)", because parseOr's
// right side comes from parseAnd, and parseAnd already grabs "B && C" as
// one piece before parseOr ever sees the "||".
func ParseRule(raw string) (Node, error) {
	tokens, err := Lex(raw)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("rule: empty rule")
	}

	p := &parser{tokens: tokens}
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	// A good rule uses every token. Anything left over means there was
	// extra text after a complete rule, like two matchers with no
	// operator between them.
	if p.pos != len(p.tokens) {
		return nil, fmt.Errorf("rule: unexpected token %q after end of expression in %q", p.tokens[p.pos].Value, raw)
	}
	return node, nil
}

// parser walks the token list with one cursor. This grammar never needs
// to go back, so a simple index is enough.
type parser struct {
	tokens []Token
	pos    int
}

// peek looks at the current token without moving past it. It returns
// false if there are no tokens left.
func (p *parser) peek() (Token, bool) {
	if p.pos >= len(p.tokens) {
		return Token{}, false
	}
	return p.tokens[p.pos], true
}

func (p *parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := p.peek()
		if !ok || tok.Type != OR {
			break
		}
		p.pos++
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrNode{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := p.peek()
		if !ok || tok.Type != AND {
			break
		}
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &AndNode{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (Node, error) {
	tok, ok := p.peek()
	if ok && tok.Type == NOT {
		p.pos++
		// Calling parseUnary again (not parsePrimary) lets "!!Host(...)" parse too. 
		// Nobody writes that on purpose, but it comes for free this way, so there is 
		// no need to block it.
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &NotNode{Child: child}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Node, error) {
	tok, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("rule: unexpected end of expression")
	}

	switch tok.Type {
	case FUNC:
		p.pos++
		return buildLeaf(tok.Value)

	case LPAREN:
		p.pos++
		// Going back to parseOr here is what makes parentheses able to override the normal order,
		// whatever is inside "(...)" is read as its own full rule, then treated as one single 
		// piece by whoever called us.
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		closeTok, ok := p.peek()
		if !ok || closeTok.Type != RPAREN {
			return nil, fmt.Errorf("rule: expected ')' to close group")
		}
		p.pos++
		return node, nil

	default:
		return nil, fmt.Errorf("rule: unexpected token %q", tok.Value)
	}
}
