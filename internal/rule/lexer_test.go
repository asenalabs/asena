package rule

import "testing"

func TestLex_SingleHostRule(t *testing.T) {
	tokens, err := Lex("Host(`example.com`)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d: %+v", len(tokens), tokens)
	}
	if tokens[0].Type != FUNC || tokens[0].Value != "Host(`example.com`)" {
		t.Errorf("unexpected token: %+v", tokens[0])
	}
}

func TestLex_CompoundRule(t *testing.T) {
	// Mirrors the worked example from the design doc.
	tokens, err := Lex("Host(`example.com`) && (PathPrefix(`/v2`) || !Method(`DELETE`))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []Token{
		{FUNC, "Host(`example.com`)"},
		{AND, "&&"},
		{LPAREN, "("},
		{FUNC, "PathPrefix(`/v2`)"},
		{OR, "||"},
		{NOT, "!"},
		{FUNC, "Method(`DELETE`)"},
		{RPAREN, ")"},
	}

	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d: %+v", len(want), len(tokens), tokens)
	}
	for i, tok := range tokens {
		if tok != want[i] {
			t.Errorf("token %d: expected %+v, got %+v", i, want[i], tok)
		}
	}
}

func TestLex_NestedParensInArgument(t *testing.T) {
	// A pathological but valid input: the argument itself contains parens.
	// This exercises the depth-counting logic in lexFunc directly.
	tokens, err := Lex("Header(`X-Debug`, `(on)`)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Value != "Header(`X-Debug`, `(on)`)" {
		t.Errorf("unexpected result: %+v", tokens)
	}
}

func TestLex_UnclosedParenthesis(t *testing.T) {
	_, err := Lex("Host(`example.com`")
	if err == nil {
		t.Fatal("expected an error for unclosed parenthesis")
	}
}

func TestLex_MissingParenAfterName(t *testing.T) {
	_, err := Lex("Host `example.com`")
	if err == nil {
		t.Fatal("expected an error for a matcher name with no '(' following it")
	}
}

func TestLex_UnexpectedCharacter(t *testing.T) {
	_, err := Lex("Host(`example.com`) & Method(`GET`)")
	if err == nil {
		t.Fatal("expected an error for a stray single '&'")
	}
}

func TestLex_WhitespaceIsSkipped(t *testing.T) {
	tokens, err := Lex("   Host(`example.com`)   &&   Method(`GET`)  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %+v", len(tokens), tokens)
	}
}
