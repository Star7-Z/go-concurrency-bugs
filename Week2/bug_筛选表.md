# Bug 筛选表（Week2）

> 从官方仓库 github.com/system-pclub/go-concurrency-bugs 的 171 个 bug patch 中初筛的结果。
> 分类沿用论文的两个正交维度：行为（blocking / non-blocking）× 根因（shared memory / message passing）。
> 详细代码分析见 `bug_代码与触发条件分析.md`（同学B）。

## 最终筛选结果

| 编号 | 项目 | 仓库文件 | 行为 | 根因 | 可复现性 | 状态 |
|---|---|---|---|---|---|---|
| C1 | Docker | `blocking-bugs/docker/4d2d2ea39336aade783c5c415b83d129bdd339bb.txt` | blocking | message passing（unbuffered channel） | ★★★★★ | ✅ 入选，已冒烟复现 |
| C2 | Kubernetes | `blocking-bugs/kubernetes/435e0b73bb99862f9dedf56a50260ff3dfef14ff.txt` | blocking | message passing + shared memory 混合 | ★★★★☆ | ✅ 入选，已冒烟复现 |
| C3 | gRPC-go | `non-blocking-bugs/grpc-go/40e7a759a035a36611d6c9a3c2acb11ce1497727.txt` | non-blocking | shared memory（data race） | ★★★★★ | ✅ 入选，待 `-race` 测试 |
| C4 | BoltDB | `non-blocking-bugs/boltdb/c69cb102ce8f7609b57f75d0ccb10862167001e7.txt` | non-blocking | shared memory（生命周期竞争） | ★★☆☆☆ | ❌ 落选（patch 是大 merge，需起数据库实例，复现代价高） |

## 浏览过但未入选的例子（记录筛选过程）

| 项目 | 仓库文件 | 一句话描述 | 未入选原因 |
|---|---|---|---|
| Docker | `blocking-bugs/docker/58befe3081726ef74ea09198cd9488fb42c51f51.txt` | Fix a deadlock in CmdStream (2013) | 和 C1 同为 channel 死锁，模式重复，留 C1 即可 |
| Docker | `blocking-bugs/docker/a69a59ffc7e3d028a72d1195c2c1535f447eaa84.txt` | Decouple removing fileWatcher from reading | 涉及文件系统 watcher，复现依赖外部环境 |
| Kubernetes | `blocking-bugs/kubernetes/010a127314a935d8d038f8dd4559fc5b249813e4.txt` | Fix quota controller worker deadlock | 逻辑嵌在 controller 框架里，抽不出独立小场景 |
| etcd | `blocking-bugs/etcd/22797c718504868665b78f143a595138cb521edf.txt` | clientv3 mutex 用 session lease id 做锁 key | 需要 etcd 服务端，超出我们环境配置能力 |
| BoltDB | `non-blocking-bugs/boltdb/7fb06feea4012543b6a1b0726a09e9a4c4a84933.txt` | Add parallel usage test and race detector | 这是加测试的提交不是修 bug 的，性质不符 |

## 筛选标准（组内讨论后定的）

1. **代码短**：patch 核心改动一眼能看完；
2. **触发条件清楚**：能画出协程交互时序图；
3. **纯标准库可复现**：不需要起 Docker daemon / etcd / 数据库（学生作业环境有限）；
4. **类别有代表性**：blocking（message passing）、blocking（混合）、non-blocking（data race）各至少一个。

## 一点说明

官方仓库每个 bug 一个 txt（修复提交的 patch），同名目录下按项目分组。浏览时发现同一个 bug 有时对应两个阶段修复，数的时候容易重复计——组里核对过我们表里没有这种情况。
