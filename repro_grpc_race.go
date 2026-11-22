// ============================================================
// 【Case 3】gRPC-go bytesSent/bytesReceived —— 无锁读写 data race
// 文件：repro_grpc_race.go
// Bug 类型：non-blocking bug（data race）
// 原始修复提交：non-blocking-bugs/grpc-go/40e7a759a035a36611d6c9a3c2acb11ce1497727.txt
// （官方修复：给 s.bytesSent / s.bytesReceived 的写入加 s.mu.Lock()/Unlock()）
//
// 运行方式（必须用 -race 才能看到现象）：
//   go run -race repro_grpc_race.go buggy   ← 修复前：race detector 报 DATA RACE
//   go run -race repro_grpc_race.go fixed   ← 修复后：干净退出
//
// 注意：不带 -race 直接 go run 两个版本都"看起来正常"——这正是这类 bug 危险的地方，
// 它不崩溃、不卡死，只留下不可预期的内存可见性错误。
//
// 踩坑记录（第一版为什么复现不出来）：
// 我最初让两个协程分别写 bytesSent 和 bytesReceived 两个【不同】的字段，以为这就是竞争——
// 但 data race 的定义是"无同步地并发访问同一内存位置"，两个协程各写各的字段根本不构成 race，
// race detector 静静地看完全程没吭声。原 bug 里真正的竞争是【写】和【读】之间：
// NewStream/operateHeaders 协程在写这两个布尔字段时，别的协程（如统计、连接关闭路径）在读它们。
// 所以本程序加了第三个协程来读。这个坑记在初步运行结果.md 里了。
package main

import (
	"fmt"
	"os"
	"sync"
)

// Stream 对应 gRPC-go transport/http2_client.go 里的 http2Client.stream
type Stream struct {
	// mu 是官方修复提交新加的锁（原代码写这两个字段时完全没同步）
	mu            sync.Mutex
	bytesSent     bool
	bytesReceived bool
}

func (s *Stream) markSent(fixed bool) {
	if fixed {
		s.mu.Lock()
		s.bytesSent = true
		s.mu.Unlock()
	} else {
		s.bytesSent = true // ← 无锁写
	}
}

func (s *Stream) markReceived(fixed bool) {
	if fixed {
		s.mu.Lock()
		s.bytesReceived = true
		s.mu.Unlock()
	} else {
		s.bytesReceived = true // ← 无锁写
	}
}

// statsReader 对应 gRPC 里其他读这两个字段的路径（统计 / 关闭连接检查）
func (s *Stream) statsReader(fixed bool) bool {
	if fixed {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.bytesSent && s.bytesReceived
	}
	return s.bytesSent && s.bytesReceived // ← 无锁读，和写入并发 = DATA RACE
}

func main() {
	mode := "buggy"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	fixed := mode == "fixed"
	fmt.Printf("运行模式：%s（官方修复=%v）\n", mode, fixed)

	s := &Stream{}
	var wg sync.WaitGroup
	const N = 10000 // 多跑几轮，确保读写充分交错

	// 协程1：模拟 NewStream 写 bytesSent
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			s.markSent(fixed)
		}
	}()

	// 协程2：模拟 operateHeaders 写 bytesReceived
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			s.markReceived(fixed)
		}
	}()

	// 协程3：模拟其他路径在读这两个字段（这才是和写入构成 race 的一方）
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			_ = s.statsReader(fixed)
		}
	}()

	wg.Wait()
	fmt.Println("结束：读写各 1 万次全部完成，程序正常退出")
	if !fixed {
		fmt.Println("（但期间存在大量无锁并发读写——race detector 应该已经报了 DATA RACE，往上翻输出）")
	}
}
