// Command check is the fixed acceptance program for the nfa2dfa exercise.
//
// It runs a fixed list of scenarios against the library and prints one line per
// scenario, either
//
//	PASS <group>/<name>
//	FAIL <group>/<name>  期望=… 实际=…
//
// followed by a summary line "结果：通过 x/N". A scenario that panics or that
// does not return is reported as a failure and the run continues.
//
//	go run ./check                  # all scenarios
//	go run ./check -list            # list the scenario ids
//	go run ./check --only closure   # only one group
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"nfa2dfa"
)

// scenarioTimeout bounds a single scenario so that a non terminating
// implementation is reported instead of hanging the whole run.
const scenarioTimeout = 3 * time.Second

type scenario struct {
	group string
	name  string
	run   func() error
}

func (s scenario) id() string { return s.group + "/" + s.name }

// mismatch renders the expected/actual pair of a failing scenario.
type mismatch struct {
	want string
	got  string
}

func (m mismatch) Error() string {
	return fmt.Sprintf("期望=%s 实际=%s", m.want, m.got)
}

func fail(want, got string) error { return mismatch{want: want, got: got} }

func scenarios() []scenario {
	return []scenario{
		{"closure", "multi-hop-after-char", checkMultiHop},
		{"closure", "pure-eps-cycle", checkEpsCycle},
		{"table", "alphabet-complete", checkAlphabetComplete},
		{"table", "state-count-stable", checkStateCountStable},
		{"table", "bytes-deterministic", checkBytesDeterministic},
		{"accept", "list-ascending", checkAcceptAscending},
		{"accept", "post-init-subset", checkLateAccept},
		{"robust", "nil-nfa", checkNilNFA},
		{"robust", "empty-transitions", checkEmptyTransitions},
		{"equiv", "short-strings", checkEquiv},
	}
}

func main() {
	list := flag.Bool("list", false, "list the scenario ids and exit")
	only := flag.String("only", "", "run the scenarios of this group only")
	flag.Parse()

	all := scenarios()
	if *list {
		for _, s := range all {
			fmt.Println(s.id())
		}
		return
	}

	selected := all
	if *only != "" {
		selected = nil
		for _, s := range all {
			if s.group == *only {
				selected = append(selected, s)
			}
		}
		if len(selected) == 0 {
			fmt.Fprintf(os.Stderr, "未知场景组：%s\n", *only)
			os.Exit(2)
		}
	}

	passed := 0
	for _, s := range selected {
		if err := invoke(s); err != nil {
			fmt.Printf("FAIL %s  %s\n", s.id(), err)
			continue
		}
		passed++
		fmt.Printf("PASS %s\n", s.id())
	}
	fmt.Printf("结果：通过 %d/%d\n", passed, len(selected))
	if passed != len(selected) {
		os.Exit(1)
	}
}

// invoke runs one scenario, catching panics and bounding its running time.
func invoke(s scenario) error {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fail("场景正常结束", fmt.Sprintf("panic: %v", r))
			}
		}()
		done <- s.run()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(scenarioTimeout):
		return fail(fmt.Sprintf("%v 内返回结果", scenarioTimeout), "超时未返回")
	}
}

// --- fixtures ---------------------------------------------------------------

func sharedSubsetNFA() *nfa2dfa.NFA {
	n := nfa2dfa.NewNFA(7)
	for _, to := range []int{1, 2, 3, 4, 5} {
		n.AddEdge(0, "a", to)
		n.AddEdge(to, "b", 6)
		n.AddEdge(6, "a", to)
	}
	n.SetAccept(5, true)
	return n
}

func equivNFA() *nfa2dfa.NFA {
	n := nfa2dfa.NewNFA(5)
	n.AddEdge(0, "a", 1)
	n.AddEdge(0, "b", 2)
	n.AddEdge(1, nfa2dfa.Epsilon, 3)
	n.AddEdge(3, nfa2dfa.Epsilon, 4)
	n.AddEdge(2, "b", 2)
	n.AddEdge(2, "a", 4)
	n.SetAccept(4, true)
	return n
}

// --- reference interpreter --------------------------------------------------

// refClosure returns the epsilon closure of the given states.
func refClosure(n *nfa2dfa.NFA, ids []int) map[int]bool {
	out := map[int]bool{}
	queue := append([]int{}, ids...)
	for _, id := range ids {
		out[id] = true
	}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for _, e := range n.EdgeList() {
			if e.From == s && e.Cond == nfa2dfa.Epsilon && !out[e.To] {
				out[e.To] = true
				queue = append(queue, e.To)
			}
		}
	}
	return out
}

// refAccepts is a plain, independent NFA interpreter used as the reference for
// the equivalence scenario.
func refAccepts(n *nfa2dfa.NFA, in string) bool {
	if n == nil {
		return false
	}
	cur := refClosure(n, []int{0})
	for _, r := range in {
		next := map[int]bool{}
		for s := range cur {
			for _, e := range n.EdgeList() {
				if e.From == s && e.Cond == string(r) {
					next[e.To] = true
				}
			}
		}
		if len(next) == 0 {
			return false
		}
		ids := make([]int, 0, len(next))
		for id := range next {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		cur = refClosure(n, ids)
	}
	for s := range cur {
		if n.IsAccept(s) {
			return true
		}
	}
	return false
}

// allStrings returns every string over alphabet up to max length, shortest
// first.
func allStrings(alphabet []string, max int) []string {
	out := []string{""}
	frontier := []string{""}
	for n := 1; n <= max; n++ {
		next := []string{}
		for _, prefix := range frontier {
			for _, c := range alphabet {
				next = append(next, prefix+c)
			}
		}
		out = append(out, next...)
		frontier = next
	}
	return out
}

// --- scenarios --------------------------------------------------------------

func checkMultiHop() error {
	n := nfa2dfa.NewNFA(4)
	n.AddEdge(0, "a", 1)
	n.AddEdge(1, nfa2dfa.Epsilon, 2)
	n.AddEdge(2, nfa2dfa.Epsilon, 3)
	n.SetAccept(3, true)

	d := nfa2dfa.ToDFA(n)
	if !d.Accepts("a") {
		return fail(`Accepts("a")=true`, `Accepts("a")=false`)
	}
	if d.Accepts("") {
		return fail(`Accepts("")=false`, `Accepts("")=true`)
	}
	return nil
}

func checkEpsCycle() error {
	n := nfa2dfa.NewNFA(2)
	n.AddEdge(0, nfa2dfa.Epsilon, 1)
	n.AddEdge(1, nfa2dfa.Epsilon, 0)
	n.SetAccept(1, true)

	d := nfa2dfa.ToDFA(n)
	if d.NumStates != 1 {
		return fail("NumStates=1", fmt.Sprintf("NumStates=%d", d.NumStates))
	}
	if !d.Accepts("") {
		return fail(`Accepts("")=true`, `Accepts("")=false`)
	}
	return nil
}

func checkAlphabetComplete() error {
	n := nfa2dfa.NewNFA(4)
	n.AddEdge(0, "a", 1)
	n.AddEdge(0, "a", 2)
	n.AddEdge(2, "b", 3)
	n.SetAccept(1, true)

	d := nfa2dfa.ToDFA(n)
	want := []string{"a", "b"}
	if !equalStrings(d.Alphabet, want) {
		return fail("Alphabet="+showStrings(want), "Alphabet="+showStrings(d.Alphabet))
	}
	return nil
}

func checkStateCountStable() error {
	const want = 3
	n := sharedSubsetNFA()
	counts := make([]int, 0, 5)
	for i := 0; i < 5; i++ {
		counts = append(counts, nfa2dfa.ToDFA(n).NumStates)
	}
	for _, c := range counts {
		if c != want {
			return fail(fmt.Sprintf("5 次转换的 NumStates 都是 %d", want),
				"NumStates="+showInts(counts))
		}
	}
	return nil
}

// canonicalSharedSubset is the byte exact rendering of the DFA that
// sharedSubsetNFA has to be converted into.
const canonicalSharedSubset = "states=3 start=0 alphabet=[a b] accept=[1]\n" +
	"  0: a->1 b->-1\n" +
	"  1: a->-1 b->2\n" +
	"  2: a->1 b->-1"

func checkBytesDeterministic() error {
	n := sharedSubsetNFA()
	first := nfa2dfa.ToDFA(n)
	second := nfa2dfa.ToDFA(n)
	if first.String() != second.String() {
		return fail("两次转换的渲染逐字节相同",
			fmt.Sprintf("第一次 %s ；第二次 %s", header(first), header(second)))
	}
	want := strings.Split(canonicalSharedSubset, "\n")
	got := strings.Split(first.String(), "\n")
	if len(want) != len(got) {
		return fail(fmt.Sprintf("渲染共 %d 行", len(want)),
			fmt.Sprintf("渲染共 %d 行", len(got)))
	}
	for i := range want {
		if want[i] != got[i] {
			return fail(fmt.Sprintf("第 %d 行=%s", i, want[i]),
				fmt.Sprintf("第 %d 行=%s", i, got[i]))
		}
	}
	return nil
}

func checkAcceptAscending() error {
	n := nfa2dfa.NewNFA(3)
	n.AddEdge(0, "a", 1)
	n.AddEdge(0, "a", 2)
	for _, id := range []int{0, 1, 2} {
		n.SetAccept(id, true)
	}

	d := nfa2dfa.ToDFA(n)
	want := []int{0, 1}
	if !equalInts(d.Accept, want) {
		return fail("Accept="+showInts(want)+"（升序）", "Accept="+showInts(d.Accept))
	}
	if !sort.IntsAreSorted(d.Accept) {
		return fail("Accept 升序", "Accept="+showInts(d.Accept))
	}
	return nil
}

func checkLateAccept() error {
	n := nfa2dfa.NewNFA(2)
	n.AddEdge(0, "a", 1)
	n.SetAccept(1, true)

	d := nfa2dfa.ToDFA(n)
	if !d.Accepts("a") {
		return fail(`Accepts("a")=true`, `Accepts("a")=false`)
	}
	if d.Accepts("") {
		return fail(`Accepts("")=false`, `Accepts("")=true`)
	}
	if !equalInts(d.Accept, []int{1}) {
		return fail("Accept=[1]", "Accept="+showInts(d.Accept))
	}
	return nil
}

func checkNilNFA() error {
	d := nfa2dfa.ToDFA(nil)
	if d.NumStates != 1 || d.Start != 0 {
		return fail("NumStates=1 且 Start=0",
			fmt.Sprintf("NumStates=%d 且 Start=%d", d.NumStates, d.Start))
	}
	if len(d.Alphabet) != 0 || len(d.Accept) != 0 {
		return fail("Alphabet=[] 且 Accept=[]",
			fmt.Sprintf("Alphabet=%s 且 Accept=%s", showStrings(d.Alphabet), showInts(d.Accept)))
	}
	if d.Accepts("") {
		return fail(`Accepts("")=false`, `Accepts("")=true`)
	}
	return nil
}

func checkEmptyTransitions() error {
	n := nfa2dfa.NewNFA(3)
	n.SetAccept(2, true)

	d := nfa2dfa.ToDFA(n)
	if d.NumStates != 1 || len(d.Alphabet) != 0 || len(d.Accept) != 0 {
		return fail("NumStates=1 且 Alphabet=[] 且 Accept=[]",
			fmt.Sprintf("NumStates=%d 且 Alphabet=%s 且 Accept=%s",
				d.NumStates, showStrings(d.Alphabet), showInts(d.Accept)))
	}
	if len(d.Trans) != 1 || len(d.Trans[0]) != 0 {
		return fail("空的转移表", fmt.Sprintf("Trans=%v", d.Trans))
	}
	return nil
}

func checkEquiv() error {
	n := equivNFA()
	d := nfa2dfa.ToDFA(n)
	for _, s := range allStrings([]string{"a", "b"}, 4) {
		want := refAccepts(n, s)
		got := d.Accepts(s)
		if want != got {
			return fail(fmt.Sprintf("NFA 解释器对 %q 判定 %v", s, want),
				fmt.Sprintf("DFA 对 %q 判定 %v", s, got))
		}
	}
	return nil
}

// --- rendering helpers ------------------------------------------------------

func header(d *nfa2dfa.DFA) string {
	return strings.SplitN(d.String(), "\n", 2)[0]
}

func showInts(v []int) string {
	if len(v) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(v))
	for _, x := range v {
		parts = append(parts, strconv.Itoa(x))
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func showStrings(v []string) string {
	if len(v) == 0 {
		return "[]"
	}
	return "[" + strings.Join(v, " ") + "]"
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
