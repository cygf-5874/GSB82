package nfa2dfa

import (
	"fmt"
	"sort"
	"strings"
)

// DFA is a deterministic finite automaton produced by ToDFA.
//
// Alphabet is ascending and Trans is indexed by state first, alphabet position
// second: Trans[s][j] is the state reached from s by reading Alphabet[j], or -1
// when the automaton has no successor for that character.
type DFA struct {
	NumStates int
	Start     int
	Alphabet  []string
	Trans     [][]int
	Accept    []int
}

// Clone returns a deep copy of d. A nil receiver stays nil.
func (d *DFA) Clone() *DFA {
	if d == nil {
		return nil
	}
	out := &DFA{
		NumStates: d.NumStates,
		Start:     d.Start,
		Alphabet:  append([]string{}, d.Alphabet...),
		Trans:     make([][]int, len(d.Trans)),
		Accept:    append([]int{}, d.Accept...),
	}
	for i, row := range d.Trans {
		out.Trans[i] = append([]int{}, row...)
	}
	return out
}

// Step returns the successor of state s under the character ch, or -1 when
// there is none.
func (d *DFA) Step(s int, ch string) int {
	if d == nil || s < 0 || s >= d.NumStates {
		return -1
	}
	for j, c := range d.Alphabet {
		if c == ch {
			return d.Trans[s][j]
		}
	}
	return -1
}

// Accepts reports whether the DFA accepts the string in.
func (d *DFA) Accepts(in string) bool {
	if d == nil {
		return false
	}
	s := d.Start
	for _, r := range in {
		s = d.Step(s, string(r))
		if s < 0 {
			return false
		}
	}
	for _, a := range d.Accept {
		if a == s {
			return true
		}
	}
	return false
}

// String renders the DFA in a fixed, line oriented form: a header followed by
// one line per state in ascending state order, with the transitions listed in
// alphabet order.
func (d *DFA) String() string {
	if d == nil {
		return "<nil dfa>"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "states=%d start=%d alphabet=%v accept=%v",
		d.NumStates, d.Start, d.Alphabet, d.Accept)
	for s := 0; s < d.NumStates; s++ {
		fmt.Fprintf(&b, "\n  %d:", s)
		for j, ch := range d.Alphabet {
			fmt.Fprintf(&b, " %s->%d", ch, d.Trans[s][j])
		}
	}
	return b.String()
}

// ToDFA converts n into an equivalent DFA by subset construction.
//
// The start state of the result is 0 and the remaining states are numbered in
// the order in which the construction first reaches them. The result never
// shares memory with n, and n is not modified.
func ToDFA(n *NFA) *DFA {
	if n == nil {
		return &DFA{
			NumStates: 1,
			Start:     0,
			Alphabet:  []string{},
			Trans:     [][]int{{}},
			Accept:    []int{},
		}
	}

	start := closure(n, startSet(n))

	sets := [][]int{start}
	index := map[string]int{key(start): 0}
	alphabet := alphabetOf(n)

	trans := make([][]int, 0, 4)
	for i := 0; i < len(sets); i++ {
		cur := sets[i]
		row := make([]int, len(alphabet))
		for j, ch := range alphabet {
			next := moveSet(n, cur, ch)
			if len(next) == 0 {
				row[j] = -1
				continue
			}
			k := key(next)
			id, ok := index[k]
			if !ok {
				id = len(sets)
				index[k] = id
				sets = append(sets, next)
			}
			row[j] = id
		}
		trans = append(trans, row)
	}

	accepts := []int{}
	for i, set := range sets {
		if containsAccept(n, set) {
			accepts = append(accepts, i)
		}
	}
	sort.Ints(accepts)

	return &DFA{
		NumStates: len(sets),
		Start:     0,
		Alphabet:  alphabet,
		Trans:     trans,
		Accept:    accepts,
	}
}

// startSet returns the NFA states the construction starts from.
func startSet(n *NFA) []int {
	return []int{0}
}

// closure returns the epsilon closure of a set of NFA states.
func closure(n *NFA, set []int) []int {
	if n == nil {
		return sortedInts(set)
	}
	seen := make(map[int]bool)
	queue := make([]int, 0, len(set))
	for _, s := range set {
		if !seen[s] {
			seen[s] = true
			queue = append(queue, s)
		}
	}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for _, t := range n.epsilonTargets(s) {
			if !seen[t] {
				seen[t] = true
				queue = append(queue, t)
			}
		}
	}
	out := make([]int, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Ints(out)
	return out
}

// moveSet returns the states reachable from set by reading ch and then taking
// the epsilon moves that follow the consumed character.
func moveSet(n *NFA, set []int, ch string) []int {
	stepped := make(map[int]bool)
	for _, s := range set {
		for _, t := range n.charTargets(s, ch) {
			stepped[t] = true
		}
	}
	if len(stepped) == 0 {
		return nil
	}
	next := make([]int, 0, len(stepped))
	for s := range stepped {
		next = append(next, s)
	}
	return closure(n, next)
}

// key renders a subset of NFA states.
func key(set []int) string {
	return fmt.Sprint(sortedInts(set))
}

// alphabetOf returns the input characters the construction has to handle.
func alphabetOf(n *NFA) []string {
	if n == nil {
		return []string{}
	}
	seen := make(map[string]bool)
	for _, e := range n.Edges {
		if e.Cond != Epsilon {
			seen[e.Cond] = true
		}
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// sortedInts returns set in ascending order without duplicates.
func sortedInts(set []int) []int {
	seen := make(map[int]bool, len(set))
	for _, v := range set {
		seen[v] = true
	}
	out := make([]int, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

// containsAccept reports whether set holds an accepting NFA state.
func containsAccept(n *NFA, set []int) bool {
	for _, s := range set {
		if n.IsAccept(s) {
			return true
		}
	}
	return false
}
