# C2 Buggy / Fixed 对照结果

Week 3｜同学A｜复现对象：Kubernetes `pkg/watch` Mux（patch `435e0b73`）
复现程序：`repro_k8s_mux.go`（v4，`-version=buggy/fixed` 分开跑）
日志：`logs/` 目录，buggy 和 fixed 各 3 轮独立留档；实跑截图：`screenshots/` 3 张（用户本机另跑的一轮，详见主报告 6.6）

## 1. 逐项现象对照

| 对照项 | Buggy（修复前） | Fixed（官方补丁 435e0b73） |
|---|---|---|
| distribute 给 A 发送 | 成功（0.5ms） | 成功（0.5ms） |
| distribute 给 B 发送 | **卡死在 `w.result <- ev`，锁被扣下** | select 双路等待，Stop 一 close(stopped) 立即走逃逸分支跳过 |
| Stop(B)（50ms 后到场） | **等锁 2 秒+没拿到** → 死锁确认 | 正常返回（约 1ms 内） |
| distribute 整体 | 1 秒检查超时，未完成 | 正常完成 |
| main 收尾 | 等锁 → 全部协程阻塞 | 正常清理退出 |
| 进程结局 | `fatal error: all goroutines are asleep - deadlock!`（go run 打印 exit status 2） | 正常退出，退出码 0 |
| 运行时死锁检测 | 触发（因为 main 也被拖下水，全部协程卡死） | 无 |
| goroutine 栈 | 四协程：chan send(持锁) / Mutex.Lock ×2 / chan receive | 无异常栈 |

## 2. 三轮结果汇总

| 轮次 | 日志文件 | fixed 退出码 | buggy 退出码（go run 打印） | buggy fatal 栈形态 |
|---|---|---|---|---|
| 1 | fixed_round1.txt / buggy_round1.txt | 0 | exit status 2 | 四协程，一致 |
| 2 | fixed_round2.txt / buggy_round2.txt | 0 | exit status 2 | 四协程，一致 |
| 3 | fixed_round3.txt / buggy_round3.txt | 0 | exit status 2 | 四协程，一致 |

三轮完全一致，复现稳定。

## 3. 关键日志对照（第 1 轮摘录，逐字来自 logs/round1 文件；← 注释是我加的）

**Buggy（buggy_round1.txt）—— 死锁链形成：**

```
[     0.5ms] [distribute] → 向 watcher "B" 发送……            ← 卡在这，锁被扣下
[    51.0ms] [Stop(B)] 尝试拿锁清理 watcher B……               ← 等锁，等不到
[  1051.3ms] [检查] ✗ distribute 卡住超过 1 秒
[  3052.0ms] [检查] ✗ Stop(B) 2 秒都没返回 → 死锁确认
[  3052.0ms] [收尾] 尝试清理并退出……                          ← main 陪葬
fatal error: all goroutines are asleep - deadlock!
```

**Fixed（fixed_round1.txt）—— 官方补丁解围（注意 50.9ms 前后）：**

```
[     0.0ms] [distribute] → 向 watcher "B" 发送（select 同时等发送成功/已停止）
[    50.9ms] [distribute] watcher "B" 已停止，跳过发送         ← close(stopped) 一关立即逃逸，锁释放
[    50.9ms] [检查] distribute 正常完成
[    50.9ms] [Stop(B)] 已 close(stopped)，再去拿锁清理
[    52.1ms] [检查] Stop(B) 正常返回
[    52.6ms] [收尾] 本场景正常结束
```

时间戳读法：Buggy 里 distribute 从 0.5ms 卡到进程死（3052ms 后被杀），整整 3 秒锁没放出来；Fixed 里同样的卡点只存在了约 50ms——Stop 一到场就解围。这就是一行 select + 一个 stopped 信号的差别。
（两协程并发时"跳过发送"和 Stop 的日志先后每轮略有交错，三轮日志可对比，结果一致。）

## 4. 补充：`-race` 模式对照（GCC 14.2.0 下实测）

环境：Go 1.23.4 + MinGW-Builds GCC 14.2.0（UCRT），CGO_ENABLED=1。
第一轮用本机老 GCC 8.1.0 跑 race 报 `exit status 0xc0000139`（DLL 入口点问题），换 14.2.0 后成功。

| | race_fixed.txt | race_buggy.txt |
|---|---|---|
| DATA RACE 告警数 | **0** | **0** |
| 程序行为 | 正常跑完，退出码 0 | 照样触发全局死锁，进程被运行时终止 |
| 备注 | 日志完整 | Windows 上 race 运行时走 failfast，fatal 栈没写进重定向文件，日志停在"收尾"行；死锁行为与非 race 版（第 3 节）一致 |

**结论：race detector 对 C2 这种 blocking bug 完全无感（buggy/fixed 都不报）**——它检测的是数据竞争，不是阻塞。对照 Week2_B 的 gRPC 实验（C3 无锁时必报 DATA RACE），两条证据合起来正好说明：两类 bug 要用不同的检测工具，没有万能检测器。
