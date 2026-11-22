// ============================================================
// 【Case 1】Docker ContainerWait —— unbuffered channel 泄漏
// 文件：repro_docker_wait.go
// Bug 类型：blocking bug（goroutine 泄漏）
// 对应论文：Table 4 中 message passing 引起的 blocking bug
// 原始修复提交：blocking-bugs/docker/4d2d2ea39336aade783c5c415b83d129bdd339bb.txt
// 运行：go run repro_docker_wait.go
// 预期：修复前子协程永久卡死（泄漏）；修复后（缓冲区 0→1）正常退出
// ============================================================
package main

import (
	"errors"
	"fmt"
	"time"
)

// doRequest 模拟 cli.post(...) 调 Docker daemon 失败
// 注意：耗时 300ms 故意比父协程的 200ms 超时长，才能触发"父协程先返回"的场景
// （第一版写成 100ms，结果父协程在超时前就收到了错误，复现不出来，这里记录一下这个坑）
func doRequest() error {
	time.Sleep(300 * time.Millisecond)
	return errors.New("post http://daemon/containers/xxx/wait: connection refused")
}

// waitOnce 对应 ContainerWait 的核心逻辑，buf 是 errC 的缓冲大小
// buggy 版本 buf=0（未修复），fixed 版本 buf=1（官方修复）
func waitOnce(name string, buf int) {
	errC := make(chan error, buf)
	childDone := make(chan struct{})

	go func() {
		err := doRequest()
		fmt.Printf("[%s][子协程] 请求失败，准备发送错误到 errC\n", name)
		errC <- err // buf=0 且没人接收时，永远卡在这里
		close(childDone)
	}()

	// 模拟父协程：select 等结果或超时，超时先返回（不再接收 errC）
	timeout := time.After(200 * time.Millisecond)
	select {
	case <-errC:
		fmt.Printf("[%s][父协程] 收到了结果（本例不会走到这）\n", name)
	case <-timeout:
		fmt.Printf("[%s][父协程] 超时，直接返回，不再接收 errC\n", name)
	}

	// 检查子协程是否还活着（1 秒观察窗口）
	select {
	case <-childDone:
		fmt.Printf("[%s][检查] 子协程已正常退出\n", name)
	case <-time.After(1 * time.Second):
		fmt.Printf("[%s][检查] ✗ 1 秒后子协程仍卡在 errC 发送 → goroutine 泄漏\n", name)
	}
}

func main() {
	fmt.Println("========== 修复前：errC := make(chan error) 无缓冲 ==========")
	waitOnce("buggy", 0)
	fmt.Println()
	fmt.Println("========== 修复后：errC := make(chan error, 1) 官方修复 ==========")
	waitOnce("fixed", 1)
	fmt.Println()
	fmt.Println("两次运行的唯一区别就是缓冲区大小 0 → 1，对应官方补丁的一行改动")
}
