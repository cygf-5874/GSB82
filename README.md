# nfa2dfa

Go 1.24，只用标准库（`go.mod` 里没有 `require`）。把非确定有限自动机（NFA）按子集构造转成
确定有限自动机（DFA）。

## 用法

```go
n := nfa2dfa.NewNFA(3)
n.AddEdge(0, "a", 1)
n.AddEdge(1, nfa2dfa.Epsilon, 2)
n.SetAccept(2, true)

d := nfa2dfa.ToDFA(n)
d.Accepts("a") // true
d.String()     // 逐字节确定的渲染
```

## 对外契约

1. **子集构造与编号**：`ToDFA(n *NFA) *DFA` 做子集构造。DFA 状态编号按 **BFS 首次发现顺序**
   从 0 开始，起点恒为 0。
2. **ε 传递闭包**：`ε` 边必须做**传递闭包**（多跳 ε 链要走到头），闭包里的所有状态共同决定
   接受性。
3. **转移表覆盖**：DFA 的转移表必须覆盖**所有出现过的输入字符**（含只在边条件里出现、不出现在
   任何接受路径上的字符）。`Alphabet` 升序，`Trans[s][j]` 是状态 `s` 读到 `Alphabet[j]` 之后的
   状态；没有任何后继时记 `-1`。
4. **子集相等**：两个子集是同一个状态，当且仅当**状态编号集合相等**（与遍历顺序无关、与 `map`
   迭代顺序无关）。
5. **接受性**：接受状态 = 子集里**含任意一个 NFA 接受状态**。
6. **逐字节确定**：同一 `NFA` 多次转换得到相同的状态数、相同的转移表、相同的接受状态列表
   （升序）。
7. **退化输入**：`ToDFA(nil)` 返回单状态 DFA（状态 0，无转移，不接受）；空转移表合法。
8. **不改入参、可重入**：不得修改传入的 `NFA`；`ToDFA` 必须可重入。

## 数据结构

- `nfa.go`：`NFA` / `State` / `Edge` 的定义、`NewNFA` / `AddEdge` / `SetAccept` / `IsAccept` /
  `Accepting` / `EdgeList` / `Clone` / `Validate` 等访问与校验入口。`Edge.Cond == nfa2dfa.Epsilon`
  （空串）表示 ε 边。
- `subset.go`：`DFA` 与 `ToDFA`，以及 `DFA.Accepts` / `DFA.Step` / `DFA.String` / `DFA.Clone`。
- `minimize.go`：`DFA.Complete` / `Minimize` / `Equivalent` 等 DFA 后处理工具。

`DFA` 字段：`NumStates`、`Start`、`Alphabet`、`Trans`、`Accept`。

## 自检

```bash
go build ./...
go test ./...
go run ./repro          # 调用方给的最小输入
bash scripts/check.sh   # 固定验收程序，等价于 go run ./check
```

`check/` 与 `repro/` 是外部给的程序，不要修改。`scripts/check.sh` 只是 `go run ./check` 的薄封装：

```bash
bash scripts/check.sh              # 跑全部场景
bash scripts/check.sh -list        # 列出场景
bash scripts/check.sh --only table # 只跑某一组
```
