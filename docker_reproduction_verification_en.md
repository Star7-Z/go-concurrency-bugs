# Reproduction Verification Report: Docker `ContainerWait`

> **TL;DR** — The goroutine leak in Docker's `ContainerWait` reproduces deterministically (10/10 runs) in a standard-library-only environment. The upstream fix (`errC` capacity changed from 0 to 1) removes the leak: `runtime.NumGoroutine()` drops from 2 back to 1. The bug produces **no error output and no non-zero exit code** — it is invisible unless an explicit observation point is instrumented.

## Material Provenance Statement (please read first)

- The reproduction code used in this report was therefore written by the author , based on the patch content and trigger logic of the upstream commit `blocking-bugs/docker/4d2d2ea39336aade783c5c415b83d129bdd339bb.txt`. It is an equivalent simulation of the real bug behavior.

---

## 1. Objective and Pass/Fail Criteria

### 1.1 Target bug

A goroutine leak in Docker's `ContainerWait`, caused by `errC` being an unbuffered channel (blocking bug; root cause: message passing).

| Field | Value |
|---|---|
| Upstream commit | `4d2d2ea39336aade783c5c415b83d129bdd339bb` |
| Commit message | `Remove deadlock in ContainerWait` |


```diff
 	resultC := make(chan container.ContainerWaitOKBody)
-	errC := make(chan error)
+	errC := make(chan error, 1)
```

### 1.2 Trigger chain

1. The parent goroutine spawns a child goroutine that issues a request to the Docker daemon.
2. The parent times out, gives up waiting, and **stops receiving from `errC`**.
3. Only then does the request fail, so the child executes `errC <- err`.
4. A send on an unbuffered channel requires a ready receiver — but the receiver has already left. The child blocks forever and the goroutine leaks.

The critical factor is **ordering**: the parent must time out *before* the child's request fails. If the request fails faster than the timeout, the parent receives the error immediately, the "early abandonment" scenario never happens, and the before/after-fix behavior becomes indistinguishable.

### 1.3 Why the exit code cannot be used as a criterion

When `main` returns, Go **does not wait** for other goroutines; the process simply terminates. As a result, the exit code is **0 in both the buggy and the fixed version**, and the leak is completely invisible from it. This is the central design constraint of this verification, and it directly reflects the paper's claim that blocking bugs are hard to expose with conventional testing.

Three-level criteria are therefore used:

| Criterion | Role | Buggy version (expected) | Fixed version (expected) |
|---|---|---|---|
| Whether the child goroutine finishes within 1 s (`childDone` channel + `time.After`) | Primary | ✗ not finished | ✓ finished |
| `runtime.NumGoroutine()` at the end of the run | Quantitative | 2 (main + blocked child) | 1 (main only) |
| Printed output | Textual corroboration | contains "stuck on chan send", does **not** contain "send completed" | contains "send completed" |

**Neither the exit code nor `fatal error` is used as a criterion.** This scenario is not a global deadlock: the Go runtime will not report `all goroutines are asleep`, and the process always exits with 0.

---

## 2. Environments

### Primary verification environment (local machine)

| Item | Value |
|---|---|
| Machine | MacBook M3 (macOS, Apple Silicon / arm64) |
| `go version` | `go1.23.4 darwin/arm64` |
| Dependencies | Standard library only: `fmt`, `runtime`, `time` |
| How it was run | VS Code integrated terminal (zsh), `go run <file>.go` |


## Appendix A: Source Before the Fix (`repro/docker_buggy.go`)

> Reproduced verbatim from the file that was actually executed; inline comments are kept in the original Chinese to preserve the exact artifact.

```go
package main

import (
	"fmt"
	"runtime"
	"time"
)

// Upstream Docker commit 4d2d2ea: client/container_wait.go
// Before fix: errC := make(chan error) -- unbuffered
func main() {
	errC := make(chan error) // <- pre-fix form

	childDone := make(chan struct{})
	go func() {
		defer close(childDone)
		// cli.post(...) takes 300ms, longer than the parent's 200ms timeout
		time.Sleep(300 * time.Millisecond)
		fmt.Println("[子协程] 请求失败，准备向 errC 发送错误")
		errC <- fmt.Errorf("post failed") // <- blocking point: unbuffered, no receiver
		fmt.Println("[子协程] 发送完成，子协程退出")
	}()

	select {
	case e := <-errC:
		fmt.Printf("[父协程] 收到错误：%v\n", e)
	case <-time.After(200 * time.Millisecond):
		fmt.Println("[父协程] 超时，放弃等待，不再读取 errC")
	}

	select {
	case <-childDone:
		fmt.Println("[检查] 子协程已正常退出")
	case <-time.After(time.Second):
		fmt.Println("[检查] ✗ 1 秒后子协程仍未退出 → 卡在 chan send，goroutine 泄漏")
	}

	fmt.Printf("[main] 当前 goroutine 数 = %d\n", runtime.NumGoroutine())
}
```

## Appendix B: Source After the Fix (`repro/docker_fixed.go`)

> The only difference from Appendix A is `make(chan error)` → `make(chan error, 1)`, which corresponds exactly to the upstream patch.

```go
package main

import (
	"fmt"
	"runtime"
	"time"
)

// Upstream Docker commit 4d2d2ea: client/container_wait.go
// After fix: errC := make(chan error, 1) -- capacity 1, the send fits in the buffer
func main() {
	errC := make(chan error, 1) // <- upstream fix: one buffer slot

	childDone := make(chan struct{})
	go func() {
		defer close(childDone)
		time.Sleep(300 * time.Millisecond)
		fmt.Println("[子协程] 请求失败，准备向 errC 发送错误")
		errC <- fmt.Errorf("post failed") // into the buffer; no receiver required
		fmt.Println("[子协程] 发送完成，子协程退出")
	}()

	select {
	case e := <-errC:
		fmt.Printf("[父协程] 收到错误：%v\n", e)
	case <-time.After(200 * time.Millisecond):
		fmt.Println("[父协程] 超时，放弃等待，不再读取 errC")
	}

	select {
	case <-childDone:
		fmt.Println("[检查] 子协程已正常退出")
	case <-time.After(time.Second):
		fmt.Println("[检查] ✗ 1 秒后子协程仍未退出 → 卡在 chan send，goroutine 泄漏")
	}

	fmt.Printf("[main] 当前 goroutine 数 = %d\n", runtime.NumGoroutine())
}
```
<img width="1470" height="956" alt="截屏2026-11-15 23 38 10" src="https://github.com/user-attachments/assets/207ec401-f6ab-4cf4-93f4-cf82388af2f9" /><img width="1470" height="956" alt="截屏2026-11-15 23 40 20" src="https://github.com/user-attachments/assets/df78a784-b350-4c14-8066-fa08d2a9c4f7" />

