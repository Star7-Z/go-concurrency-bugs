// ============================================================
// 【C2】Kubernetes pkg/watch Mux —— 持锁向无缓冲 channel 阻塞发送
// 文件：repro_k8s_mux.go（Week3 第 4 版，在同学 B 三版冒烟测试的基础上完善）
// 原始修复提交：blocking-bugs/kubernetes/435e0b73bb99862f9dedf56a50260ff3dfef14ff.txt
//   "pkg/watch: fix potential deadlock"（kubernetes，2014-09-15）
//
// 用法：
//   go run repro_k8s_mux.go -version=buggy   → 原始 Bug 行为
//       结尾触发 fatal error: all goroutines are asleep - deadlock!
//       退出码 2 属预期现象，不是程序写错了
//   go run repro_k8s_mux.go -version=fixed   → 官方修复后行为，正常退出（退出码 0）
//
// 本版（v4）相对 v3 改了什么，为什么改：
//   1. 两个场景拆开，用 -version 参数各跑各的。
//      v3 是一个进程先跑 fixed 再跑 buggy，buggy 结尾的全局死锁会把整个
//      进程杀掉，两份输出混在一起、退出码永远是 2。本周要求 buggy/fixed
//      各留 3 轮独立日志，只能拆开。
//   2. 关键节点都打了时间戳（程序启动起算）。
//      v3 只能看出"卡住了"，看不出"谁先卡、卡了多久"。加时间戳后死锁链
//      的时序可以直接从日志里读出来。
//   3. distribute 在给每个 watcher 发送前先打一行日志。
//      这样 buggy 版卡死时能精确看到它卡在给谁发送上（v3 卡住后就 silent 了）。
//
// 历史踩坑（v1→v3，来自 Week2 同学 B 的冒烟测试，继承过来防止再踩）：
//   v1：Stop 在 distribute 之前调用 → B 被提前删掉，死锁复现不出来。
//       教训：复现并发 bug 必须先画时序图确认事件先后。
//   v2：时序对了但 Stop 启动太晚 → fixed 场景误报"卡住 1 秒"。
//   v3：distribute 先启动，50ms 后 Stop 启动，两个动作并发（最终形态）。
// ============================================================
package main

import (
	"flag"
	"fmt"
	"sync"
	"time"
)

// 程序启动时刻，日志里打的是相对时间戳，方便看时序
var t0 = time.Now()

func logf(format string, args ...interface{}) {
	fmt.Printf("[%8.1fms] %s\n",
		float64(time.Since(t0).Nanoseconds())/1e6,
		fmt.Sprintf(format, args...))
}

type Event struct{ Msg string }

type muxWatcher struct {
	result   chan Event    // 无缓冲，和修复前一致
	stopped  chan struct{} // 官方修复新增（patch 里的 stopped）
	stopOnce sync.Once     // 官方修复新增（patch 里的 stop sync.Once）
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
	logf("[distribute] 拿到锁，开始分发 %q（修复前逻辑：裸发送）", ev.Msg)
	for _, w := range m.watchers {
		logf("[distribute] → 向 watcher %q 发送……", w.name)
		w.result <- ev // 没人接收就卡死，且锁被一直持有
	}
}

// distributeFixed 对应修复后：select 双路 + stopped 信号（官方 patch 的写法）
func (m *Mux) distributeFixed(ev Event) {
	m.lock.Lock()
	defer m.lock.Unlock()
	logf("[distribute] 拿到锁，开始分发 %q（修复后逻辑：select 双路）", ev.Msg)
	for _, w := range m.watchers {
		logf("[distribute] → 向 watcher %q 发送（select 同时等发送成功/已停止）", w.name)
		select {
		case w.result <- ev:
		case <-w.stopped:
			logf("[distribute] watcher %q 已停止，跳过发送", w.name)
		}
	}
}

// stopBuggy 对应修复前的 Stop()：直接调 stopWatching（内部要先拿锁）
func (w *muxWatcher) stopBuggy(m *Mux) {
	logf("[Stop(B)] 尝试拿锁清理 watcher B……")
	m.lock.Lock() // ← 修复前死锁链的最后一环：锁在卡死的 distribute 手里
	delete(m.watchers, w.name)
	m.lock.Unlock()
}

// stopFixed 对应修复后的 Stop()：先 close(stopped) 再去拿锁，Once 防重入
func (w *muxWatcher) stopFixed(m *Mux) {
	w.stopOnce.Do(func() {
		close(w.stopped) // 先解围：让 distribute 的 select 能从 stopped 分支逃出去
		logf("[Stop(B)] 已 close(stopped)，再去拿锁清理")
		m.lock.Lock()
		delete(m.watchers, w.name)
		m.lock.Unlock()
	})
}

func runCase(fixed bool) {
	if fixed {
		fmt.Println("========== 修复后：select 双路 + stopped 信号（官方补丁 435e0b73） ==========")
	} else {
		fmt.Println("========== 修复前：持锁 + 裸发送（原始 Bug） ==========")
	}
	m := newMux()

	// watcher A：一直在读
	a := m.Watch("A")
	aDone := make(chan struct{})
	go func() {
		for ev := range a.result {
			logf("[watcher A] 收到: %s", ev.Msg)
		}
		close(aDone)
	}()

	// watcher B：注册了但调用方已经"放弃读"（result 无人接收，人也联系不上了）
	b := m.Watch("B")
	logf("[场景] watcher B 已注册但不再读取 result channel（人还没被移除）")

	// 1. 先分发事件 —— buggy 版会卡在给 B 发送上，并持有锁
	//    （时序要求见文件头 v1 踩坑记录：Stop 不能在 distribute 之前）
	distDone := make(chan struct{})
	go func() {
		if fixed {
			m.distributeFixed(Event{Msg: "pod-updated"})
		} else {
			m.distributeBuggy(Event{Msg: "pod-updated"})
		}
		close(distDone)
	}()

	// 2. 50ms 后有人想清理 B（v2 的教训：stop 启动太晚会让 fixed 场景误报）
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
		logf("[检查] distribute 正常完成")
	case <-time.After(1 * time.Second):
		logf("[检查] ✗ distribute 卡住超过 1 秒 —— 阻塞在 w.result <- ev 上，且 m.lock 未释放")
	}

	select {
	case ok := <-stopDone:
		if ok {
			logf("[检查] Stop(B) 正常返回")
		} else {
			logf("[检查] ✗ Stop(B) 超过 0.5 秒（死锁链：Stop 等锁 → 锁被卡死的 distribute 持有）")
		}
	case <-time.After(2 * time.Second):
		logf("[检查] ✗ Stop(B) 2 秒都没返回 → 死锁确认")
	}

	// 4. 收尾清理 + 关闭 A 的接收
	//    buggy 场景走到这里时，main 也要拿锁 → 也被卡住 → 所有协程全部阻塞，
	//    Go 运行时内置 deadlock detector 判定"全局死锁"，直接杀进程（见运行日志）。
	//    这实证了论文 Section 8：内置 detector 只抓"全部协程都卡死"，且处理方式
	//    是杀进程而不是修复。
	logf("[收尾] 尝试清理并退出……")
	m.lock.Lock()
	for n, w := range m.watchers {
		w.stopOnce.Do(func() { close(w.stopped) })
		delete(m.watchers, n)
	}
	m.lock.Unlock()
	close(a.result)
	<-aDone
	logf("[收尾] 本场景正常结束")
}

func main() {
	version := flag.String("version", "fixed", "buggy 或 fixed：跑原始 Bug 版还是官方修复版")
	flag.Parse()

	if *version == "buggy" {
		// buggy 场景结尾会触发 "fatal error: all goroutines are asleep - deadlock!"，
		// 程序不会走到下面的提示——这正是这个 bug 最可怕的地方：一旦死锁，整个进程陪葬。
		runCase(false)
		fmt.Println("\n（如果你看到这行，说明 buggy 场景没有触发全局死锁，检查复现逻辑！）")
	} else if *version == "fixed" {
		runCase(true)
	} else {
		fmt.Printf("未知版本 %q，可用值：buggy / fixed\n", *version)
	}
}
