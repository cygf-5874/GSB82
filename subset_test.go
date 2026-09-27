package nfa2dfa

import (
	"reflect"
	"testing"
)

// acceptAll marks every state of n accepting and returns n.
func acceptAll(n *NFA) *NFA {
	for i := range n.States {
		n.States[i].Accept = true
	}
	return n
}

// chain builds a size-state automaton with the edges given as (from, cond, to)
// triples and returns it.
func chain(t *testing.T, size int, triples ...[3]string) *NFA {
	t.Helper()
	n := NewNFA(size)
	for _, tr := range triples {
		from := mustAtoi(t, tr[0])
		to := mustAtoi(t, tr[2])
		n.AddEdge(from, tr[1], to)
	}
	return n
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	switch s {
	case "0":
		return 0
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	}
	t.Fatalf("unsupported state %q", s)
	return -1
}

func TestNilNFA(t *testing.T) {
	d := ToDFA(nil)
	if d.NumStates != 1 || d.Start != 0 {
		t.Fatalf("want 1 state starting at 0, got %d states starting at %d", d.NumStates, d.Start)
	}
	if len(d.Alphabet) != 0 {
		t.Fatalf("want empty alphabet, got %v", d.Alphabet)
	}
	if len(d.Accept) != 0 {
		t.Fatalf("want no accepting state, got %v", d.Accept)
	}
	if !reflect.DeepEqual(d.Trans, [][]int{{}}) {
		t.Fatalf("want a single empty transition row, got %v", d.Trans)
	}
}

func TestEmptyAutomatonAcceptsNothing(t *testing.T) {
	d := ToDFA(NewNFA(1))
	if d.NumStates != 1 {
		t.Fatalf("want 1 state, got %d", d.NumStates)
	}
	if d.Accepts("") || d.Accepts("a") {
		t.Fatalf("an automaton without accepting states must accept nothing")
	}
}

func TestSingleAcceptingState(t *testing.T) {
	n := NewNFA(1)
	n.SetAccept(0, true)
	d := ToDFA(n)
	if !d.Accepts("") {
		t.Fatalf("the single accepting state must accept the empty string")
	}
	if !reflect.DeepEqual(d.Accept, []int{0}) {
		t.Fatalf("want accepting state [0], got %v", d.Accept)
	}
}

func TestStartStateIsZero(t *testing.T) {
	n := acceptAll(chain(t, 2, [3]string{"0", "a", "1"}))
	d := ToDFA(n)
	if d.Start != 0 {
		t.Fatalf("want start state 0, got %d", d.Start)
	}
}

func TestTwoStateChain(t *testing.T) {
	n := acceptAll(chain(t, 2, [3]string{"0", "a", "1"}))
	d := ToDFA(n)
	if d.NumStates != 2 {
		t.Fatalf("want 2 states, got %d", d.NumStates)
	}
	if !reflect.DeepEqual(d.Alphabet, []string{"a"}) {
		t.Fatalf("want alphabet [a], got %v", d.Alphabet)
	}
	if !reflect.DeepEqual(d.Accept, []int{0, 1}) {
		t.Fatalf("want accepting states [0 1], got %v", d.Accept)
	}
	if !d.Accepts("a") {
		t.Fatalf("the automaton must accept \"a\"")
	}
}

func TestAlphabetIsAscending(t *testing.T) {
	n := acceptAll(chain(t, 3, [3]string{"0", "c", "1"}, [3]string{"0", "a", "2"}))
	d := ToDFA(n)
	if !reflect.DeepEqual(d.Alphabet, []string{"a", "c"}) {
		t.Fatalf("want alphabet [a c], got %v", d.Alphabet)
	}
}

func TestBFSDiscoveryOrder(t *testing.T) {
	n := acceptAll(chain(t, 3, [3]string{"0", "a", "1"}, [3]string{"1", "b", "2"}))
	d := ToDFA(n)
	if d.NumStates != 3 {
		t.Fatalf("want 3 states, got %d", d.NumStates)
	}
	want := [][]int{{1, -1}, {-1, 2}, {-1, -1}}
	if !reflect.DeepEqual(d.Trans, want) {
		t.Fatalf("want transitions %v, got %v", want, d.Trans)
	}
}

func TestRenderingIsStable(t *testing.T) {
	n := acceptAll(chain(t, 3, [3]string{"0", "a", "1"}, [3]string{"1", "b", "2"}))
	first := ToDFA(n).String()
	second := ToDFA(n).String()
	if first != second {
		t.Fatalf("two conversions of the same automaton rendered differently:\n%s\n%s", first, second)
	}
}

func TestLongerInputAccepted(t *testing.T) {
	n := acceptAll(chain(t, 3, [3]string{"0", "a", "1"}, [3]string{"1", "b", "2"}))
	d := ToDFA(n)
	if !d.Accepts("ab") {
		t.Fatalf("the automaton must accept \"ab\"")
	}
	if d.Accepts("ba") {
		t.Fatalf("the automaton must not accept \"ba\"")
	}
}

func TestUnknownCharacterRejected(t *testing.T) {
	n := acceptAll(chain(t, 2, [3]string{"0", "a", "1"}))
	d := ToDFA(n)
	if d.Accepts("b") {
		t.Fatalf("an input character outside the alphabet must be rejected")
	}
}

func TestEpsilonHopFromStart(t *testing.T) {
	n := acceptAll(NewNFA(2))
	n.AddEdge(0, Epsilon, 1)
	d := ToDFA(n)
	if d.NumStates != 1 {
		t.Fatalf("want 1 state, got %d", d.NumStates)
	}
	if !reflect.DeepEqual(d.Accept, []int{0}) {
		t.Fatalf("want accepting state [0], got %v", d.Accept)
	}
	if !d.Accepts("") {
		t.Fatalf("the epsilon successor is accepting, so the empty string is accepted")
	}
}

func TestValidateWellFormed(t *testing.T) {
	n := NewNFA(3)
	n.AddEdge(0, "a", 1)
	n.AddEdge(1, Epsilon, 2)
	if err := n.Validate(); err != nil {
		t.Fatalf("want a well formed automaton, got %v", err)
	}
}

func TestValidateRejectsUnknownEndpoint(t *testing.T) {
	n := NewNFA(2)
	n.AddEdge(0, "a", 7)
	if err := n.Validate(); err == nil {
		t.Fatalf("want an error for an edge that ends outside the state set")
	}
}

func TestToDFADoesNotModifyInput(t *testing.T) {
	n := acceptAll(chain(t, 3, [3]string{"0", "a", "1"}, [3]string{"1", "b", "2"}))
	before := n.Clone()
	ToDFA(n)
	if !reflect.DeepEqual(n.States, before.States) || !reflect.DeepEqual(n.Edges, before.Edges) {
		t.Fatalf("ToDFA modified its argument")
	}
}
