package nfa2dfa

import "sort"

// This file holds DFA post-processing helpers. They are independent of the
// subset construction and are written on top of the DFA value only.

// Complete returns an equivalent DFA in which every state has a transition for
// every input character. Missing entries (-1) are routed to a freshly appended
// sink state which is not accepting.
//
// The receiver is never modified.
func (d *DFA) Complete() *DFA {
	if d == nil {
		return &DFA{Start: 0, Alphabet: []string{}, Trans: [][]int{}, Accept: []int{}}
	}
	out := d.Clone()
	needsSink := false
	for _, row := range out.Trans {
		for _, to := range row {
			if to < 0 {
				needsSink = true
			}
		}
	}
	if !needsSink {
		return out
	}
	sink := out.NumStates
	out.NumStates++
	for i := range out.Trans {
		for j, to := range out.Trans[i] {
			if to < 0 {
				out.Trans[i][j] = sink
			}
		}
	}
	row := make([]int, len(out.Alphabet))
	for j := range row {
		row[j] = sink
	}
	out.Trans = append(out.Trans, row)
	return out
}

// Minimize returns the smallest DFA accepting the same language as d. State
// numbering of the result follows the order in which classes are first reached
// from the start state.
//
// The receiver is never modified.
func Minimize(d *DFA) *DFA {
	if d == nil || d.NumStates == 0 {
		return &DFA{Start: 0, Alphabet: []string{}, Trans: [][]int{}, Accept: []int{}}
	}
	work := d.Complete()
	size := work.NumStates
	width := len(work.Alphabet)

	accepting := make([]bool, size)
	for _, a := range work.Accept {
		if a >= 0 && a < size {
			accepting[a] = true
		}
	}

	class := make([]int, size)
	for i := 0; i < size; i++ {
		if accepting[i] {
			class[i] = 1
		}
	}
	for {
		seen := make(map[string]int)
		next := make([]int, size)
		changed := false
		for i := 0; i < size; i++ {
			sig := make([]byte, 0, 8)
			sig = appendKey(sig, class[i])
			for j := 0; j < width; j++ {
				sig = appendKey(sig, class[work.Trans[i][j]])
			}
			key := string(sig)
			if _, ok := seen[key]; !ok {
				seen[key] = len(seen)
			}
			next[i] = seen[key]
			if next[i] != class[i] {
				changed = true
			}
		}
		class = next
		if !changed {
			break
		}
	}

	// Rebuild the automaton with one state per class, then renumber by BFS.
	classes := map[int]bool{}
	for _, c := range class {
		classes[c] = true
	}
	order := []int{}
	index := map[int]int{}
	start := class[work.Start]
	order = append(order, start)
	index[start] = 0
	for i := 0; i < len(order); i++ {
		rep := -1
		for s := 0; s < size; s++ {
			if class[s] == order[i] {
				rep = s
				break
			}
		}
		if rep < 0 {
			continue
		}
		for j := 0; j < width; j++ {
			c := class[work.Trans[rep][j]]
			if _, ok := index[c]; !ok {
				index[c] = len(order)
				order = append(order, c)
			}
		}
	}

	out := &DFA{
		NumStates: len(order),
		Start:     0,
		Alphabet:  append([]string(nil), work.Alphabet...),
		Trans:     make([][]int, len(order)),
		Accept:    []int{},
	}
	for i, c := range order {
		row := make([]int, width)
		rep := -1
		for s := 0; s < size; s++ {
			if class[s] == c {
				rep = s
				break
			}
		}
		for j := 0; j < width; j++ {
			row[j] = index[class[work.Trans[rep][j]]]
		}
		out.Trans[i] = row
		if accepting[rep] {
			out.Accept = append(out.Accept, i)
		}
	}
	return out
}

// Equivalent reports whether two DFAs accept exactly the same set of strings.
// A nil automaton accepts nothing but the empty string only when it is
// accepting, i.e. a nil DFA accepts nothing.
func Equivalent(a, b *DFA) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	type pair struct{ l, r int }
	start := pair{a.Start, b.Start}
	seen := map[pair]bool{start: true}
	queue := []pair{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if isAccepting(a, cur.l) != isAccepting(b, cur.r) {
			return false
		}
		for _, ch := range unionAlphabet(a.Alphabet, b.Alphabet) {
			next := pair{stepAny(a, cur.l, ch), stepAny(b, cur.r, ch)}
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return true
}

// isAccepting reports whether the state s of d is accepting.
func isAccepting(d *DFA, s int) bool {
	for _, a := range d.Accept {
		if a == s {
			return true
		}
	}
	return false
}

// stepAny returns the successor of s under ch, or -1 when there is none.
func stepAny(d *DFA, s int, ch string) int {
	if s < 0 || s >= d.NumStates {
		return -1
	}
	for j, c := range d.Alphabet {
		if c == ch {
			return d.Trans[s][j]
		}
	}
	return -1
}

// unionAlphabet returns the ascending union of two alphabets.
func unionAlphabet(x, y []string) []string {
	seen := map[string]bool{}
	for _, c := range x {
		seen[c] = true
	}
	for _, c := range y {
		seen[c] = true
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// appendKey appends the decimal form of v plus a separator to buf.
func appendKey(buf []byte, v int) []byte {
	if v < 0 {
		buf = append(buf, '-')
		v = -v
	}
	if v >= 10 {
		buf = appendKey(buf, v/10)
	}
	return append(buf, byte('0'+v%10), '.')
}
