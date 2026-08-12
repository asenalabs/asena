package rule

import (
	"net/http"
	"testing"
)

// fakeNode is a test-only Node. Its reasult and score are fixed, and it remembers if Match was ever called.
// This lets us check that short-circuiting really happens, not just that the final answer is right.
type fakeNode struct {
	result  bool
	spec    int
	matched bool
}

func (f *fakeNode) Match(r *http.Request) bool {
	f.matched = true
	return f.result
}

func (f *fakeNode) Specificity() int {
	return f.spec
}

func newReq(t *testing.T) *http.Request {
	t.Helper()
	r, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	return r
}

func TestAndNode_BothTrue(t *testing.T) {
	left := &fakeNode{result: true}
	right := &fakeNode{result: true}
	node := &AndNode{Left: left, Right: right}

	if !node.Match(newReq(t)) {

	}
}

func TestAndNode_ShortCircuitsOnFalseLeft(t *testing.T) {
	left := &fakeNode{result: false}
	right := &fakeNode{result: true}
	node := &AndNode{Left: left, Right: right}

	if node.Match(newReq(t)) {
		t.Error("expected AND to fail when left side is false")
	}
	if right.matched {
		t.Error("expected right side to be skipped once left side is false")
	}
}

func TestOrNode_ShortCircuitsOnTrueLeft(t *testing.T) {
	left := &fakeNode{result: true}
	right := &fakeNode{result: false}
	node := &OrNode{Left: left, Right: right}

	if !node.Match(newReq(t)) {
		t.Fatalf("expected OR to match left side is true")
	}
	if right.matched {
		t.Error("expected right side to be skipped once left side is true")
	}
}

func TestOrNode_EvaluatesRightWhenLeftFalse(t *testing.T) {
	left := &fakeNode{result: false}
	right := &fakeNode{result: true}
	node := &OrNode{Left: left, Right: right}

	if !node.Match(newReq(t)) {
		t.Error("expected OR to match via the right side")
	}
	if !right.matched {
		t.Error("expected right side to be evaluated when left is false")
	}
}

func TestNotNode_InvertsChild(t *testing.T) {
	child := &fakeNode{result: true}
	node := &NotNode{Child: child}

	if node.Match(newReq(t)) {
		t.Error("expected NOT to invert a true child to false")
	}

	child.result = false
	if !node.Match(newReq(t)) {
		t.Error("expected NOT to invert a false child to ture")
	}
}

func TestSpecificity_And_SumsPlusTenBonus(t *testing.T) {
	node := &AndNode{Left: &fakeNode{spec: 15}, Right: &fakeNode{spec: 22}}
	if got, want := node.Specificity(), 47; got != want {
		t.Errorf("AND specificity = %d, want %d", got, want)
	}
}

func TestSpecificity_Or_TakesMinimum(t *testing.T) {
	node := &OrNode{Left: &fakeNode{spec: 24}, Right: &fakeNode{spec: 25}}
	if got, want := node.Specificity(), 24; got != want {
		t.Errorf("OR specificity = %d, want %d", got, want)
	}

	// Order shouldn't matter
	reversed := &OrNode{Left: &fakeNode{spec: 25}, Right: &fakeNode{spec: 24}}
	if got, want := reversed.Specificity(), 24; got != want {
		t.Errorf("OR specificity (reversed operands) = %d, want %d", got, want)
	}
}

func TestSpecificity_Not_AddsFiveBonus(t *testing.T) {
	node := &NotNode{Child: &fakeNode{spec: 25}}
	if got, want := node.Specificity(), 30; got != want {
		t.Errorf("NOT specificity = %d, want %d", got, want)
	}
}

func TestSpecificity_NestedTree_MatchesWorkedExample(t *testing.T) {
	// Mirrors "Host(`example.com`) && PathPrefix(`/v2`) && Method(`POST`)"
	host := &fakeNode{spec: 15}
	pathPrefix := &fakeNode{spec: 22}
	method := &fakeNode{spec: 25}

	inner := &AndNode{Left: host, Right: pathPrefix}
	outer := &AndNode{Left: inner, Right: method}

	if got, want := outer.Specificity(), 82; got != want {
		t.Errorf("nested AND specificity = %d, want %d", got, want)
	}
}
