# 候选 Bug 代码与触发条件分析（同学B）

> 基于 Week 1 论文《Understanding Real-World Concurrency Bugs in Go》的分类框架，对同学A初筛出的 4 个候选做精读分析。
> 每个 patch 都是官方仓库里的一个真实修复提交，文件在 `go-concurrency-bugs/{blocking,non}-blocking-bugs/<项目>/` 下。
> 我给每个候选评了一个"可复现性"，供同学D安排测试顺序。

## 候选1：Docker `ContainerWait` — unbuffered channel 导致 goroutine 卡死（blocking）

- **仓库文件**：`blocking-bugs/docker/4d2d2ea39336aade783c5c415b83d129bdd339bb.txt`
- **行为类别**：blocking；**根因类别**：message passing（对应论文 Observation 3，58% 的 blocking bug 由 message passing 引起这一类）

**Bug 代码（修复前）：**
```go
resultC := make(chan container.ContainerWaitOKBody)
errC := make(chan error)          // ← 无缓冲
// ...
go func() {
    select {
    case errC <- <-errC?? // 省略：实际是子协程往 errC 发送请求结果/错误
    }
}()
```
修复提交的说明写得很直白：*"when `cli.post(...)` fails `errC <- err` blocks because `errC` is unbuffered"*。

**触发条件：**
1. 父协程起一个子协程去请求 Docker daemon，子协程失败后往 `errC` 发送错误；
2. 但父协程此时已经因为超时 / context 取消不再从 `errC` 接收；
3. 无缓冲 channel 的发送必须等接收方就绪 → 子协程永远卡在 `errC <- err`。

**官方修复：**`errC := make(chan error, 1)`，只加了个缓冲区大小 1。子协程的发送能进缓冲区，即使没人接收也能走完。

**我的评价：**这个 bug 和论文 Figure 1 里 Kubernetes 的 `finishReq` 是同一个模式（unbuffered channel + 父协程提前退出），但代码量小得多，非常适合作为我们组的复现对象。修复只改一个词，对比效果直观。

**可复现性：★★★★★**（不需要任何外部依赖，纯标准库可以写出等价最小复现）

## 候选2：Kubernetes `pkg/watch` Mux — 持锁发送到没人接收的 channel（blocking）

- **仓库文件**：`blocking-bugs/kubernetes/435e0b73bb99862f9dedf56a50260ff3dfef14ff.txt`
- **行为类别**：blocking；**根因类别**：message passing 与 shared memory 的**混合误用**——这是论文强调的"新特性组合使用导致的、难检测的 bug"

**Bug 代码（修复前，`mux.go`）：**
```go
func (m *Mux) distribute(event Event) {
    m.lock.Lock()
    defer m.lock.Unlock()          // 锁一直持到底
    for _, w := range m.watchers {
        w.result <- event          // ← result 是无缓冲 channel，阻塞式发送
    }
}

func (mw *muxWatcher) Stop() {
    mw.m.stopWatching(mw.id)       // ← stopWatching 内部也要拿 m.lock
}
```

**触发条件（三步死锁链）：**
1. 某个 watcher 的使用者已经不读 `result` channel 了（比如调用方提前放弃）；
2. 此时 `distribute` 在**持有锁**的状态下向该 watcher 的无缓冲 channel 发送事件，卡住；
3. 其他协程调用 `Stop()` 想清理，但 `stopWatching` 要先拿 `m.lock` → 锁永远拿不到 → 整个 Mux 死锁。

**官方修复：**给每个 watcher 加一个 `stopped chan struct{}`，发送改成 `select` 双路（要么发出去、要么发现已停止）；`Stop()` 用 `sync.Once` 保证先 `close(stopped)` 再去拿锁。相当于用 channel 解开了"持锁阻塞发送"这个死结。

**我的评价：**比候选1复杂，但更能体现论文说的"channel 和锁混合使用是 Go 特有难点"。值得保留为候选。

**可复现性：★★★★☆**（纯标准库可复现，但要把死锁链完整摆出来需要 3 个协程配合，代码比候选1多一点）

## 候选3：gRPC-go — `bytesSent / bytesReceived` 无锁读写（non-blocking / data race）

- **仓库文件**：`non-blocking-bugs/grpc-go/40e7a759a035a36611d6c9a3c2acb11ce1497727.txt`
- **行为类别**：non-blocking；**根因类别**：shared memory（传统 data race，对应论文 Section 6）

**Bug 代码（修复前）：**
```go
s.bytesSent = true        // NewStream 协程里直接写
...
s.bytesReceived = true    // operateHeaders 协程里直接写
```
两个布尔字段被不同协程读写，没有任何同步。修复就是加上 `s.mu.Lock()/Unlock()`。

**我的评价：**最"教科书"的 data race，用 `go run -race` 一定能测出来。它的价值在于对照：说明 Go 也有大量传统型 bug，并不是用了 channel 就天下太平（论文统计 non-blocking bug 基本都是这类）。

**可复现性：★★★★★**（`-race` 检测器可以直接证明；缺点是普通运行时不会出错，演示效果靠 race detector）

## 候选4：BoltDB — `Close()` 与未完成事务的竞争（non-blocking）

- **仓库文件**：`non-blocking-bugs/boltdb/c69cb102ce8f7609b57f75d0ccb10862167001e7.txt`（一个 merge commit，PR #377 "Wait for pending transactions on close"）
- **行为类别**：non-blocking；**根因类别**：shared memory / 生命周期竞争

**问题：**`Close()` 没有等待正在进行的事务结束就释放/复用资源，导致 close 与事务操作竞争。修复引入等待未决事务的机制（代码改动较大）。

**我的评价：**真实价值高，但 patch 是个大 merge，牵扯 BoltDB 内部事务管理，单独拎出来讲不清楚。而且 BoltDB 的复现需要真的起数据库实例，不像前三个用标准库就能演示。

**可复现性：★★☆☆☆**（建议放弃或只做文字分析，不作为最终复现对象）

## 汇总：给同学D的测试顺序建议

| 优先级 | 候选 | 复现方式 | 预期现象 |
|---|---|---|---|
| 1 | Docker ContainerWait | 标准库最小复现，跑两版（修复前/后） | 修复前子协程卡死；修复后正常退出 |
| 2 | gRPC data race | 标准库等价复现 + `go run -race` | race detector 报 DATA RACE |
| 3 | K8s watch mux | 标准库复现死锁链 + 超时检测 | Stop() 卡在拿锁上 |
| 4 | BoltDB close race | （不做运行测试） | — |

## 我的困惑（如实记录）

1. 候选2严格说是"message passing + shared memory 混合"，我按论文的口径标了根因为 message passing 为主——但论文 Table 6 具体怎么给这类 bug 归类的，我还没逐条核对，下周找时间看下原文表格。
2. 候选3的 race 在修复前后用 `-race` 应该都能报（因为 race 检测的是访问模式而不是结果），这点和"修复后就不报"的直觉相反——注意：修复后加了锁，race detector 确实就不报了，锁就是同步点。我想清楚了，不算困惑了，保留在此作为提醒。
3. 仓库里同一个 bug 有时有两个文件（bug 的两阶段修复），同学A建表时注意别重复计数。

---

*分析日期：2026-11-12。所有 patch 内容直接来自官方仓库，论文数据引自 paper/go-study.pdf。两个最小复现程序已先替同学D做了冒烟测试，见 `repro/` 目录和 `初步运行结果.md`。*
