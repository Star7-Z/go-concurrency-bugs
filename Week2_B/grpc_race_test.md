\# gRPC-go Data Race Bug Reproduction and Verification Report



\## 1. Experimental Objectives



This experiment investigates a historical Data Race issue in the gRPC-go project.



The issue involves two fields in the `Stream` structure:



\- `bytesSent`

\- `bytesReceived`



The main objectives are:



1\. Analyze the cause of the historical Data Race bug.

2\. Compare the source code before and after the fix.

3\. Use field-level tests to demonstrate the data race caused by an unsynchronized write and a synchronized read.

4\. Verify that using the same mutex to protect both read and write operations prevents the data race.



\## 2. Bug Information



| Item | Description |

|---|---|

| Project | gRPC-go |

| Fix Commit | `40e7a759a035a36611d6c9a3c2acb11ce1497727` |

| Fix Date | 2017-06-19 |

| Modified File | `transport/http2\_client.go` |

| Fix Description | Protect `bytesSent` and `bytesReceived` with a mutex |



Commit message:



`Protect bytesSent and bytesReceived with mutex to avoid datarace (#1318)`



\## 3. Bug Analysis



\### 3.1 Involved Fields



The `Stream` structure contains two status fields:



\- `bytesSent`: Indicates whether data has been sent.

\- `bytesReceived`: Indicates whether data has been received.



These fields are accessed by different execution paths.



\### 3.2 Cause of the Bug



Before the fix, the program had the following situation:



1\. Some execution paths directly modified `bytesSent` and `bytesReceived`.

2\. The `BytesSent()` and `BytesReceived()` methods used a mutex when reading these fields.

3\. However, the corresponding write operations did not use the same mutex.



As a result, concurrent access to the same fields by multiple goroutines could cause a data race.



Using a mutex only for read operations is not sufficient to ensure concurrency safety. Both read and write operations must use the same synchronization mechanism.



\## 4. Patch Analysis



Before the fix, the relevant field assignments were performed without mutex protection.



After the fix, mutex protection was added to the two write operations in `transport/http2\_client.go`.



For `bytesSent`:



```go

s.mu.Lock()

s.bytesSent = true

s.mu.Unlock()

```



For `bytesReceived`:



```go

s.mu.Lock()

s.bytesReceived = true

s.mu.Unlock()

```



The corresponding read operations are performed through `BytesSent()` and `BytesReceived()`, which also use `s.mu` to protect access.



Therefore, after the fix, both read and write operations use the same mutex, preventing concurrent access from causing a data race.



\## 5. Experimental Environment



| Item | Configuration |

|---|---|

| Operating System | Windows |

| Go Version | Go 1.23.4 |

| GCC Version | GCC 16.2.0 |

| CGO | Enabled |

| Race Detector | Go `-race` |

| Project | gRPC-go |

| Experimental Version | Commit `40e7a75` |



Since this project is an older Go project and does not include the modern Go Modules configuration, a `go.mod` file and dependency configuration were added to the independent experimental copy to support testing in the current Go environment.



These configuration changes were kept separate from the original repository.



\## 6. Testing Method



Four field-level tests were implemented in `transport/race\_repro\_test.go`.



The tests are divided into two categories:



\- \*\*Race Tests:\*\* One goroutine writes to a field without using a mutex, while another goroutine reads the field through the existing Getter method.

\- \*\*Safe Tests:\*\* Both read and write operations use the same mutex.



Go's Race Detector was used to identify concurrent data access problems.



Example test command:



```powershell

go test -race -vet=off -count=1 -run "^TestRaceBytesSent$" ./transport

```



Command options:



\- `-race`: Enables the data race detector.

\- `-vet=off`: Disables vet checks to avoid compatibility warnings from legacy code under the modern Go environment.

\- `-count=1`: Forces the test to run without using cached results.

\- `-run`: Specifies the test to execute.



\## 7. Test Results



\### 7.1 bytesSent Tests



| Test Name | Testing Method | Result |

|---|---|---|

| `TestRaceBytesSent` | Unsynchronized write and synchronized read | Data race detected |

| `TestSafeBytesSent` | Synchronized write and synchronized read | Passed |



In `TestRaceBytesSent`, the Race Detector identified a data race between the unsynchronized write to `bytesSent` in the test code and the read operation in `BytesSent()`.



In `TestSafeBytesSent`, both operations used the same mutex, and the test passed.



\### 7.2 bytesReceived Tests



| Test Name | Testing Method | Result |

|---|---|---|

| `TestRaceBytesReceived` | Unsynchronized write and synchronized read | Data race detected |

| `TestSafeBytesReceived` | Synchronized write and synchronized read | Passed |



In `TestRaceBytesReceived`, the Race Detector identified a data race between the unsynchronized write to `bytesReceived` in the test code and the read operation in `BytesReceived()`.



In `TestSafeBytesReceived`, both operations used the same mutex, and the test passed.



\### 7.3 Test Log Files



The complete terminal outputs of the four tests were saved in the `test\_logs` directory.



| Log File | Description |

|---|---|

| `test\_logs/01\_race\_bytes\_sent.txt` | Unsynchronized write test for bytesSent |

| `test\_logs/02\_safe\_bytes\_sent.txt` | Synchronized access test for bytesSent |

| `test\_logs/03\_race\_bytes\_received.txt` | Unsynchronized write test for bytesReceived |

| `test\_logs/04\_safe\_bytes\_received.txt` | Synchronized access test for bytesReceived |



\## 8. Result Analysis



The experimental results demonstrate that:



1\. For `bytesSent`, an unsynchronized write combined with a synchronized read can cause a data race.

2\. For `bytesReceived`, the same situation can also cause a data race.

3\. When both read and write operations use the same mutex, the Race Detector no longer reports a data race.



These results are consistent with the approach used in the historical patch.



By adding mutex protection to the two write operations, the patch ensures that both fields are accessed safely during concurrent execution.



\## 9. Experimental Limitations



This experiment uses field-level minimal reproduction tests.



In the Race tests, the test code directly modifies the fields without acquiring the mutex to simulate the unsynchronized write behavior of the historical bug.



The Safe tests simulate the synchronization mechanism introduced by the patch.



Therefore, this experiment verifies the synchronization mechanism and its effect on concurrent field access. It does not reproduce the complete original gRPC network communication path end to end.



In addition, the experiment uses a modern Go version and updated dependency configuration to run the legacy project. The environment is not identical to the original development environment in 2017.



\## 10. Conclusion



Through source code analysis and Race Detector testing, this experiment verifies that concurrent access to `bytesSent` and `bytesReceived` requires a consistent synchronization mechanism.



The historical patch adds `s.mu.Lock()` and `s.mu.Unlock()` around the two field assignments, ensuring that write operations are protected by the same mutex used by the Getter methods.



The field-level reproduction tests and safe synchronization tests produced the expected results, demonstrating the importance of consistent mutex protection in concurrent Go programs.

