package rule

import "net/http"

// Node is what every part of rule can do, check a request, and say how exact it is.
// This is true for AndNode, OrNode, and NotNode below, and also for the matcher nodes in matchers.go (HostNode, and so on).
//
// Because every part uses the same Node interface, a big tree can be built out of small pieces without any special-case code.
// An AndNode does not need to know if its children are single matchers or whole sub-trees, it just calls Match on them.
type Node interface {
	Match(r *http.Request) bool
	Specificity() int
}

// AndNode matches only if both sides match.
type AndNode struct {
	Left, Right Node
}

// Match uses Go's own "&&", so if Left is false, Right is never checked.
func (n *AndNode) Match(r *http.Request) bool {
	return n.Left.Match(r) && n.Right.Match(r)
}

// Specificity adds both sides together, plus 10. The 10 makes an AND always score
// higher than either side alone, since combining two checks is always more exact than using just one.
func (n *AndNode) Specificity() int {
	return n.Left.Specificity() + n.Right.Specificity() + 10
}

// OrNode matches if either side matches.
type OrNode struct {
	Left, Right Node
}

// Match short-circuits the other way, if Left is true, Right is never checked.
func (n *OrNode) Match(r *http.Request) bool {
	return n.Left.Match(r) || n.Right.Match(r)
}

// Specificity takes the smaller of the two sides, not the sum.
//
// Why: "Host(`example.com`) || PathPrefix(`/`)"" is not a tight rule, it matches almost anything, because PathPrefix(`/`)
// alone matches every path. An OR rule is only as tight as its weakest side, since either side alone is enough to match.
// Using the smaller score keeps this rule from looking tighter than it really is.
func (n *OrNode) Specificity() int {
	l, r := n.Left.Specificity(), n.Right.Specificity()
	if l < r {
		return l
	}
	return r
}

// NotNode flips its child's result.
type NotNode struct {
	Child Node
}

func (n *NotNode) Match(r *http.Request) bool {
	return !n.Child.Match(r)
}

// Specificity adds a small bonus (5) over the child's score. A NOT is a real extra condition, so it should score
// a bit higher than the child alone, but it is not a second full condition like AND has, so the bonus is smaller.
func (n *NotNode) Specificity() int {
	return n.Child.Specificity() + 5
}
