// ============================================================
// 【Case 2】Kubernetes pkg/watch Mux —— 持锁向无缓冲 channel 阻塞发送
// 文件：repro_k8s_mux.go
// Bug 类型：blocking bug（死锁链：持锁阻塞发送 → Stop 等锁 → 全局卡死）
// 原始修复提交：blocking-bugs/kubernetes/435e0b73bb99862f9dedf56a50260ff3dfef14ff.txt
// 运行：go run repro_k8s_mux.go
// 预期：buggy 场景结尾触发运行时 fatal error: all goroutines are asleep - deadlock!
//       退出码为 2 属预期现象，不是程序写错了
// ============================================================
//
// 场景设计说明（第一版踩的坑）：
// 一开始我把 stop(B) 放在 distribute 之前调用，结果 B 被提前从 watchers 里删掉，
// distribute 根本碰不到 B，死锁复现不出来。真实 Bug 的时序应该是：
// 先有人分发事件（此时 B 已"放弃阅读"但还没被移除），分发卡住持锁之后，
// 才有人来调 Stop 想清理 → 卡在拿锁上。
package main

import (
	"fmt"
	"sync"
	"time"
)

type Event struct{ Msg string }

type muxWatcher struct {
	result   chan Event    // 无缓冲，和修复前一致
	stopped  chan struct{} // 官方修复新增
	stopOnce sync.Once     // 官方修复新增
	name     string
}

type Mux struct {
	lock     sync.Mutex
	watchers map[string]*muxWatcher
}

func newMux() *Mux {
	return &Mux{watchers: make(map[string]*muxWatcher)}
}

func (m *Mux) Watch(name string) *muxWatcher {
	w := &muxWatcher{
		result:  make(chan Event),
		stopped: make(chan struct{}),
		name:    name,
	}
	m.lock.Lock()
	m.watchers[name] = w
	m.lock.Unlock()
	return w
}

// distributeBuggy 对应修复前：持锁 + 无 select 的阻塞发送
func (m *Mux) distributeBuggy(ev Event) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, w := range m.watchers {
		w.result <- ev // 没人接收就卡死，且锁被一直持有
	}
}

// distributeFixed 对应修复后：select 双路 + stopped 信号
func (m *Mux) distributeFixed(ev Event) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, w := range m.watchers {
		select {
		case w.result <- ev:
		case <-w.stopped:
			fmt.Printf("[distribute] watcher %q 已停止，跳过发送\n", w.name)
		}
	}
}

// stopBuggy 对应修复前的 Stop()：直接调 stopWatching（内部要先拿锁）
func (w *muxWatcher) stopBuggy(m *Mux) {
	m.lock.Lock()
	delete(m.watchers, w.name)
	m.lock.Unlock()
}

// stopFixed 对应修复后的 Stop()：先 close(stopped) 再去拿锁清理，用 Once 防重入
func (w *muxWatcher) stopFixed(m *Mux) {
	w.stopOnce.Do(func() {
		close(w.stopped)
		m.lock.Lock()
		delete(m.watchers, w.name)
		m.lock.Unlock()
	})
}

func runCase(title string, fixed bool) {
	fmt.Printf("\n========== %s ==========\n", title)
	m := newMux()

	// watcher A：一直在读
	a := m.Watch("A")
	aDone := make(chan struct{})
	go func() {
		for ev := range a.result {
			fmt.Printf("[watcher A] 收到: %s\n", ev.Msg)
		}
		close(aDone)
	}()

	// watcher B：注册了但调用方已经"放弃读"（result 无人接收，人也联系不上了）
	b := m.Watch("B")
	fmt.Println("[场景] watcher B 已注册但不再读取 result channel（人还没被移除）")

	// 1. 先分发事件 —— buggy 版会卡在给 B 发送上，并持有锁
	distDone := make(chan struct{})
	go func() {
		if fixed {
			m.distributeFixed(Event{Msg: "pod-updated"})
		} else {
			m.distributeBuggy(Event{Msg: "pod-updated"})
		}
		close(distDone)
	}()

	// 2. 50ms 后有人想清理 B（修复后 stop 能靠 stopped 信号解围；修复前只能干等锁）
	// 注意 stop 必须在"检查 distribute 是否卡住"之前就启动：真实系统里两个动作来自不同协程，
	// 第一版我把 stop 放到 1 秒检查之后才启动，导致修复后场景也白等 1 秒、误报"卡住"
	time.Sleep(50 * time.Millisecond)
	stopDone := make(chan bool, 1)
	go func() {
		start := time.Now()
		if fixed {
			b.stopFixed(m)
		} else {
			b.stopBuggy(m)
		}
		stopDone <- time.Since(start) < 500*time.Millisecond
	}()

	// 3. 分别检查 distribute 和 Stop 的结果
	select {
	case <-distDone:
		fmt.Println("[检查] distribute 正常完成")
	case <-time.After(1 * time.Second):
		fmt.Println("[检查] ✗ distribute 卡住超过 1 秒 —— 阻塞在 w.result <- ev 上，且 m.lock 未释放")
	}

	select {
	case ok := <-stopDone:
		if ok {
			fmt.Println("[检查] Stop(B) 正常返回")
		} else {
			fmt.Println("[检查] ✗ Stop(B) 超过 0.5 秒（死锁链：Stop 等锁 → 锁被卡死的 distribute 持有）")
		}
	case <-time.After(2 * time.Second):
		fmt.Println("[检查] ✗ Stop(B) 2 秒都没返回 → 死锁确认")
	}

	// 3. 收尾清理 + 关闭 A 的接收
	// 注意：buggy 场景走到这里时，main 也要拿锁 → 也被卡住 → 所有协程全部阻塞，
	// Go 运行时内置 deadlock detector 会判定"全局死锁"直接终止进程（见运行输出）。
	// 这恰好演示了论文 Section 8 的结论：内置 detector 只能抓"全部协程都卡死"的情况，
	// 而且抓到的方式是把整个进程杀掉，不是修复。
	fmt.Println("[收尾] 尝试清理并退出……")
	m.lock.Lock()
	for n, w := range m.watchers {
		w.stopOnce.Do(func() { close(w.stopped) })
		delete(m.watchers, n)
	}
	m.lock.Unlock()
	close(a.result)
	<-aDone
	fmt.Println("[收尾] 本场景正常结束")
}

func main() {
	// 修复后的场景先跑（能干净退出）；修复前的场景最后跑，以运行时的 fatal error 收场
	runCase("修复后：select + stopped 信号（官方补丁）", true)
	runCase("修复前：持锁 + 裸发送（原始 Bug）", false)
	// buggy 场景结尾会触发 "fatal error: all goroutines are asleep - deadlock!"，
	// 程序不会走到这里——这正是这个 bug 最可怕的地方：一旦死锁，整个进程陪葬。
	fmt.Println("\n（如果你看到这行，说明 buggy 场景没有触发全局死锁，检查复现逻辑！）")
}
