// Package nfa2dfa converts nondeterministic finite automata (NFA) into
// deterministic finite automata (DFA) by subset construction.
//
// The package only depends on the Go standard library. The public contract is
// written down in README.md; check/ is a fixed acceptance program.
package nfa2dfa

import (
	"errors"
	"fmt"
	"sort"
)

// Epsilon is the edge label used by epsilon transitions.
const Epsilon = ""

// State is a single NFA state.
type State struct {
	// ID is the state number. IDs must be unique and lie in [0, len(States)).
	ID int
	// Accept reports whether the state is accepting.
	Accept bool
}

// Edge is one labelled transition of an NFA.
type Edge struct {
	// From and To are state numbers.
	From int
	To   int
	// Cond is the input character carried by the edge. Epsilon means that the
	// edge can be taken without consuming any input.
	Cond string
}

// NFA is a nondeterministic finite automaton.
//
// The zero value is a legal, empty automaton; ToDFA treats a nil *NFA the same
// way it treats an automaton without state 0.
type NFA struct {
	States []State
	Edges  []Edge
}

// NewNFA returns an automaton with size non-accepting states numbered 0..size-1.
func NewNFA(size int) *NFA {
	n := &NFA{}
	for i := 0; i < size; i++ {
		n.States = append(n.States, State{ID: i})
	}
	return n
}

// AddEdge appends one transition from -> cond -> to and returns n, so that
// calls can be chained. An empty cond means an epsilon transition.
func (n *NFA) AddEdge(from int, cond string, to int) *NFA {
	n.Edges = append(n.Edges, Edge{From: from, Cond: cond, To: to})
	return n
}

// SetAccept marks the state with the given ID as accepting (accept == true) or
// non-accepting (accept == false) and returns n.
func (n *NFA) SetAccept(id int, accept bool) *NFA {
	for i := range n.States {
		if n.States[i].ID == id {
			n.States[i].Accept = accept
			return n
		}
	}
	return n
}

// Size returns the number of states declared by n. A nil receiver has size 0.
func (n *NFA) Size() int {
	if n == nil {
		return 0
	}
	return len(n.States)
}

// IsAccept reports whether the state with the given ID is accepting. Unknown
// IDs and a nil receiver are reported as false.
func (n *NFA) IsAccept(id int) bool {
	if n == nil {
		return false
	}
	for _, st := range n.States {
		if st.ID == id {
			return st.Accept
		}
	}
	return false
}

// Accepting returns the accepting state IDs in ascending order. The result is
// always a non-nil slice.
func (n *NFA) Accepting() []int {
	out := []int{}
	if n == nil {
		return out
	}
	for _, st := range n.States {
		if st.Accept {
			out = append(out, st.ID)
		}
	}
	sort.Ints(out)
	return out
}

// epsilonTargets returns the direct epsilon successors of the state id in
// ascending order without duplicates. A nil receiver has no successor.
func (n *NFA) epsilonTargets(id int) []int {
	if n == nil {
		return nil
	}
	var out []int
	for _, e := range n.Edges {
		if e.Cond == Epsilon && e.From == id {
			out = append(out, e.To)
		}
	}
	return sortedUnique(out)
}

// charTargets returns the direct successors of the state id that consume the
// character cond, in ascending order without duplicates. A nil receiver has no
// successor.
func (n *NFA) charTargets(id int, cond string) []int {
	if n == nil || cond == Epsilon {
		return nil
	}
	var out []int
	for _, e := range n.Edges {
		if e.Cond == cond && e.From == id {
			out = append(out, e.To)
		}
	}
	return sortedUnique(out)
}

// EdgeList returns a copy of the edge list of n. A nil receiver has no edges.
func (n *NFA) EdgeList() []Edge {
	if n == nil {
		return nil
	}
	out := make([]Edge, len(n.Edges))
	copy(out, n.Edges)
	return out
}

// Clone returns a deep copy of n. A nil receiver stays nil.
func (n *NFA) Clone() *NFA {
	if n == nil {
		return nil
	}
	out := &NFA{
		States: make([]State, len(n.States)),
		Edges:  make([]Edge, len(n.Edges)),
	}
	copy(out.States, n.States)
	copy(out.Edges, n.Edges)
	return out
}

// Validate reports the first structural problem found in n. A nil receiver, an
// automaton without states and an automaton whose edges only touch known states
// are all considered well formed.
func (n *NFA) Validate() error {
	if n == nil {
		return nil
	}
	seen := make(map[int]bool, len(n.States))
	for _, st := range n.States {
		if st.ID < 0 {
			return fmt.Errorf("nfa2dfa: state id %d is negative", st.ID)
		}
		if seen[st.ID] {
			return fmt.Errorf("nfa2dfa: duplicate state id %d", st.ID)
		}
		seen[st.ID] = true
	}
	for i, e := range n.Edges {
		if e.From < 0 || e.To < 0 {
			return fmt.Errorf("nfa2dfa: edge %d has a negative endpoint", i)
		}
		if !seen[e.From] {
			return fmt.Errorf("nfa2dfa: edge %d starts at unknown state %d", i, e.From)
		}
		if !seen[e.To] {
			return fmt.Errorf("nfa2dfa: edge %d ends at unknown state %d", i, e.To)
		}
	}
	return nil
}

// ErrNoStartState is reported by StartState when state 0 is missing.
var ErrNoStartState = errors.New("nfa2dfa: state 0 is missing")

// StartState returns 0 when the automaton declares that state.
func (n *NFA) StartState() (int, error) {
	if n == nil {
		return 0, nil
	}
	for _, st := range n.States {
		if st.ID == 0 {
			return 0, nil
		}
	}
	if len(n.States) == 0 {
		return 0, ErrNoStartState
	}
	return 0, ErrNoStartState
}

// sortedUnique returns the ascending, duplicate free form of in.
func sortedUnique(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	out := make([]int, len(in))
	copy(out, in)
	sort.Ints(out)
	w := 1
	for i := 1; i < len(out); i++ {
		if out[i] != out[w-1] {
			out[w] = out[i]
			w++
		}
	}
	return out[:w]
}
