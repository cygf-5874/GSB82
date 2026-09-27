// Command repro is the minimal input handed over by the caller. It feeds one
// fixed automaton to the library and reports what it sees.
//
// On the current library it prints two observations and exits with status 1.
package main

import (
	"fmt"
	"os"
	"sort"

	"nfa2dfa"
)

// probe is a string the definition of the automaton below accepts.
const probe = "yyyyy"

// tail walks from the start state to the accepting state with five characters
// in between, separated by epsilon moves.
var probeChars = []string{"y", "y", "y", "y", "y"}

// build assembles the automaton used by this program.
//
// For every pair of characters (c, d) the automaton has three states u, v, w:
//
//	0 --c--> u  0 --c--> v
//	u --d--> w  v --d--> w
//	w --c--> u  w --c--> v
//
// and w is accepting. The state set {u, v} is therefore reachable in two
// different ways. A separate chain of five characters and epsilon moves leads
// to the last accepting state.
func build() *nfa2dfa.NFA {
	pairs := [][2]string{
		{"a", "b"}, {"c", "d"}, {"e", "f"}, {"g", "h"},
		{"i", "j"}, {"k", "l"}, {"m", "n"}, {"o", "p"},
		{"q", "r"}, {"s", "t"}, {"u", "v"}, {"w", "x"},
	}
	tail := 1 + 3*len(pairs)
	n := nfa2dfa.NewNFA(tail + 2*len(probeChars) + 1)

	for i, p := range pairs {
		u, v, w := 1+3*i, 2+3*i, 3+3*i
		c, d := p[0], p[1]
		n.AddEdge(0, c, u)
		n.AddEdge(0, c, v)
		n.AddEdge(u, d, w)
		n.AddEdge(v, d, w)
		n.AddEdge(w, c, u)
		n.AddEdge(w, c, v)
		n.SetAccept(w, true)
	}

	cur := 0
	next := tail
	for i := 0; i < len(probeChars); i++ {
		n.AddEdge(cur, probeChars[i], next)
		n.AddEdge(next, nfa2dfa.Epsilon, next+1)
		cur = next + 1
		next += 2
	}
	n.SetAccept(next-1, true)
	return n
}

func main() {
	n := build()

	counts := make([]int, 0, 12)
	for i := 0; i < 12; i++ {
		counts = append(counts, nfa2dfa.ToDFA(n).NumStates)
	}
	distinct := unique(counts)

	unstable := len(distinct) > 1
	accepted := nfa2dfa.ToDFA(n).Accepts(probe)

	fmt.Println("调用方给的最小输入：同一份 NFA 定义，跑两遍看结果。")
	if unstable {
		fmt.Printf("  · DFA 状态数：连续转换 12 次得到 %s，多次之间不一样。\n", show(distinct))
	} else {
		fmt.Printf("  · DFA 状态数：连续转换 12 次都是 %d。\n", distinct[0])
	}
	if accepted {
		fmt.Printf("  · 串 %q：NFA 判定接受，DFA 判定接受，两者一致。\n", probe)
	} else {
		fmt.Printf("  · 串 %q：NFA 判定接受，DFA 判定拒绝，两者不一致。\n", probe)
	}

	if !unstable && accepted {
		fmt.Println("上面两处现象都没有出现。")
		return
	}
	fmt.Println("上面两处现象已复现。")
	os.Exit(1)
}

func unique(in []int) []int {
	seen := map[int]bool{}
	for _, v := range in {
		seen[v] = true
	}
	out := make([]int, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

func show(in []int) string {
	s := ""
	for i, v := range in {
		if i > 0 {
			s += "、"
		}
		s += fmt.Sprintf("%d", v)
	}
	return s
}
