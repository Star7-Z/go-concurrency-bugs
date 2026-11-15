\# Experimental Environment Setup



\## 1. Overview



This document describes the development environment configuration for the gRPC-go Data Race experiment.



The experiment was conducted on Windows using Go, GCC, CGO, and Go Modules.



The purpose of this setup was to provide a working environment for compiling and testing the legacy gRPC-go project with Go's Race Detector.



\## 2. Environment Information



| Component | Version / Configuration |

|---|---|

| Operating System | Windows |

| Go | 1.23.4 |

| GCC | 16.2.0 |

| CGO | Enabled |

| Go Modules | Enabled |

| Race Detector | Go `-race` |



\## 3. Go Installation



Go 1.23.4 was installed in the following directory:



```text

D:\\Development\\Go

```



The Go executable is located at:



```text

D:\\Development\\Go\\bin\\go.exe

```



The installed version was verified using:



```powershell

go version

```



Expected output:



```text

go version go1.23.4 windows/amd64

```



\## 4. GCC and CGO Configuration



GCC was installed through MSYS2 and is located in:



```text

D:\\Development\\msys64\\ucrt64\\bin

```



The GCC installation was verified using:



```powershell

gcc --version

```



The reported compiler version was GCC 16.2.0.



CGO was enabled to support Go's Race Detector on Windows.



The following environment variable was configured:



```powershell

$env:CGO\_ENABLED="1"

```



\## 5. Go Environment Configuration



The experiment used the following Go environment settings:



| Variable | Configuration |

|---|---|

| `CGO\_ENABLED` | `1` |

| `GOTOOLCHAIN` | `local` |

| `GOPATH` | `D:\\Development\\GoWorkspace` |



`GOTOOLCHAIN=local` was used to keep the experiment running with the locally installed Go version.



\## 6. Project Setup



The original gRPC-go repository was kept separate from the experimental copy.



The experimental project directory was:



```text

D:\\Development\\GoProjects\\grpc-race-test\\grpc-go\\grpc-go-module

```



A `go.mod` file and dependency configuration were added to the experimental copy because the original legacy repository did not include a modern Go Modules configuration.



This allowed the project to be tested in the current Go environment without modifying the original repository.



\## 7. Dependency Configuration



The experimental copy uses Go Modules to manage its dependencies.



The dependencies were configured in `go.mod` and resolved into `go.sum`.



The dependency configuration was adapted for compatibility with Go 1.23.4.



These dependencies are experimental compatibility settings and should not be interpreted as the original dependency versions used by gRPC-go in 2017.



\## 8. Environment Verification



The following command was used to verify that the transport package could be compiled with the Race Detector enabled:



```powershell

go test -race -vet=off -run "^$" ./transport

```



The command completed successfully.



This confirmed that the transport package could be compiled in the configured environment.



The `-vet=off` option was used because legacy code triggered a compatibility warning from the modern Go vet checks.



\## 9. Important Notes



1\. The experiment uses Go 1.23.4 rather than the original Go environment from 2017.

2\. The original repository was kept separate from the experimental copy.

3\. Go Modules configuration and dependency versions were adapted for compatibility.

4\. CGO was enabled to support the Race Detector.

5\. The environment was verified through a successful package compilation test.



\## 10. Conclusion



The Windows development environment was successfully configured for the gRPC-go Data Race experiment.



Go 1.23.4, GCC 16.2.0, CGO, and Go Modules were used to build and test the experimental copy of the project.



The configured environment successfully compiled the transport package with the Race Detector enabled, providing the necessary foundation for the subsequent concurrency tests.

