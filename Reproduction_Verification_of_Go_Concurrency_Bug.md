## 1. Experiment Description

Report note:

Because the complete Docker and gRPC‑go source projects depend on a large number of system components, historical protobuf versions and vendor dependencies, the full projects are not compiled in this experiment. Minimal reproduction programs are used to replicate the bug logic of the original commits, and these programs rely only on the Go standard library.

Original project commits obtained from the dataset repository:

- Docker bug: blocking-bugs/docker/4d2d2ea39336aade783c5c415b83d129bdd339bb.txt
- gRPC‑go bug: non-blocking-bugs/grpc-go/40e7a759a035a36611d6c9a3c2acb11ce1497727.txt

## 2. Environment

```text
(base) zhongqinxuan@192 ~ % uname -a
sw_vers
Darwin 192.168.5.237 23.5.0 Darwin Kernel Version 23.5.0: Wed May  1 20:14:59 PDT 2024; root:xnu-10063.121.3~5/RELEASE_ARM64_T8122 arm64

ProductName:	macOS
ProductVersion:	14.5
BuildVersion:	23F79

(base) zhongqinxuan@192 ~ % go version
go env
go version go1.23.4 darwin/arm64

GO111MODULE=''
GOARCH='arm64'
GOBIN=''
GOCACHE='/Users/zhongqinxuan/Library/Caches/go-build'
GOENV='/Users/zhongqinxuan/Library/Application Support/go/env'
GOEXE=''
GOEXPERIMENT=''
GOFLAGS=''
GOHOSTARCH='arm64'
GOHOSTOS='darwin'
GOINSECURE=''
GOMODCACHE='/Users/zhongqinxuan/go/pkg/mod'
GONOPROXY=''
GONOSUMDB=''
GOOS='darwin'
GOPATH='/Users/zhongqinxuan/go'
GOPRIVATE=''
GOPROXY='https://proxy.golang.org,direct'
GOROOT='/usr/local/go'
GOSUMDB='sum.golang.org'
GOTMPDIR=''
GOTOOLCHAIN='auto'
GOTOOLDIR='/usr/local/go/pkg/tool/darwin_arm64'
GOVCS=''
GOVERSION='go1.23.4'
GODEBUG=''
GOTELEMETRY='local'
GOTELEMETRYDIR='/Users/zhongqinxuan/Library/Application Support/go/telemetry'
GCCGO='gccgo'
GOARM64='v8.0'
AR='ar'
CC='clang'
CXX='clang++'
CGO_ENABLED='1'
GOMOD='/dev/null'
GOWORK=''
CGO_CFLAGS='-O2 -g'
CGO_CPPFLAGS=''
CGO_CXXFLAGS='-O2 -g'
CGO_FFLAGS='-O2 -g'
CGO_LDFLAGS='-O2 -g'
PKG_CONFIG='pkg-config'
GOGCCFLAGS='-fPIC -arch arm64 -pthread -fno-caret-diagnostics -Qunused-arguments -fmessage-length=0 -ffile-prefix-map=/var/folders/8d/t6nn0p0j5bng9xqc1rdy4r2h0000gn/T/go-build1265342862=/tmp/go-build -gno-record-gcc-switches -fno-common'

(base) zhongqinxuan@192 ~ % git --version
git version 2.39.3 (Apple Git-146)
```

## 3. Experiment 1: Docker Goroutine Leak Reproduction (repro_docker_wait.go)

Two logic branches are implemented inside the source code: buggy (buf=0, unbuffered channel) / fixed (buf=1, single-slot buffered channel). Each version runs independently, with at least 3 test rounds per version.

> Screenshot: Output of 3 loop runs of repro_docker_wait.go

Key observation: the buggy version does not trigger Go's built-in deadlock detector; the main goroutine exits normally while the child goroutine leaks silently. This is fully consistent with the conclusion in Section 5 of the paper.

## 4. Experiment 2: gRPC‑go Data Race Reproduction (repro_grpc_race.go)

This bug is a non-blocking data race, so the ‑race flag must be enabled to run the detector. buggy: read and write without locking; fixed: protect shared variables with sync.Mutex. At least 3 test rounds per version.

> Screenshot 1: gRPC‑go buggy, Round 1, ‑race run outputting a DATA RACE warning
> Screenshot 2: gRPC‑go buggy, Round 2, ‑race run outputting a DATA RACE warning
> Screenshot 3: gRPC‑go buggy, Round 3, ‑race run outputting a DATA RACE warning

Experimental Analysis: The buggy version of gRPC‑go triggers `DATA RACE` warnings, as multiple goroutines concurrently read and write shared counter variables without locks. This bug belongs to the non-blocking category in the paper: the program neither hangs nor exits abnormally, so it cannot be captured automatically by the Go runtime, which makes it highly concealed. The fixed version protects the shared variables with a mutex and eliminates the data race; no race is detected in any of the three rounds, which matches the characteristics of data-race concurrency defects described in the paper.

> （注：部分内容由豆包 AI 生成）