值班同事报的：同一份 NFA 定义，喂短串能对上，喂长串就漏接受状态；
更怪的是连着跑两次，打印出来的 DFA 状态数还不一样。nfa2dfa 是 Go 1.24 的
自动机转换库，只用标准库，自检走 `scripts/check.sh`（`check/` 是固定验收程序，别改）。

`nfa.go` 的数据定义、`minimize.go` 的工具函数都是对的；出问题的是 `subset.go`。
`subset_test.go` 的 14 个用例当前全绿，`repro/main.go` 是调用方给的最小输入。

任务：把 `subset.go` 里违反 README「对外契约」的地方找出来并修好，
既有用例要一直保持全绿，固定件要全过。

验收：
- go build ./... 退出码 0；
- go test ./... 退出码 0；
- go run ./repro 退出码 0（那处说不通的地方不再复现）；
- go run ./check 退出码 0，10 个场景全过（closure 2 + table 3 + accept 2 + robust 2 + equiv 1）。

约束：
1. 不改 `check/`、不改 `repro/`、不改 `nfa.go`、不改 `minimize.go`、不改 `subset_test.go`。
2. 不引入第三方依赖；不许用 `time`、`rand`、`hash/maphash` 之类让结果不确定的东西。
3. 公开的函数签名不变（可新增函数与类型）。
