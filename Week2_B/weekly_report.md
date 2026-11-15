\# Week 2 Individual Weekly Report – Student B



\## 1. Work Completed



During Week 2, I was responsible for setting up the experimental environment and conducting Data Race tests on the gRPC-go project.



The following tasks were completed:



1\. \*\*Development Environment Setup\*\*

&#x20;  - Installed and configured Go 1.23.4.

&#x20;  - Configured GCC 16.2.0 and enabled CGO.

&#x20;  - Set up Go Modules and configured the dependencies required by the legacy gRPC-go project.

&#x20;  - Verified that the transport package could be compiled with the Race Detector enabled.



2\. \*\*Historical Bug Analysis\*\*

&#x20;  - Investigated the historical Data Race issue in gRPC-go.

&#x20;  - Identified the two involved fields: `bytesSent` and `bytesReceived`.

&#x20;  - Examined the original fix commit `40e7a759a035a36611d6c9a3c2acb11ce1497727`.

&#x20;  - Compared the source code before and after the patch.



3\. \*\*Data Race Testing\*\*

&#x20;  - Implemented four field-level tests in `transport/race\_repro\_test.go`.

&#x20;  - Conducted unsynchronized write tests for both fields.

&#x20;  - Conducted synchronized access tests using the same mutex.

&#x20;  - Used Go's Race Detector to identify data races.

&#x20;  - Saved all four test outputs in the `test\_logs` directory.



4\. \*\*Documentation\*\*

&#x20;  - Completed `environment\_setup.md`.

&#x20;  - Completed `grpc\_race\_test.md`.

&#x20;  - Organized the experimental source code and test logs.



\## 2. Key Findings



The experiment focused on the concurrency safety of `bytesSent` and `bytesReceived`.



The test results showed that:



\- Unsynchronized writes combined with synchronized reads could trigger Data Race warnings.

\- When both read and write operations used the same mutex, the tests passed without Race Detector warnings.

\- The historical patch addressed the issue by adding mutex protection to the two write operations.



These findings are consistent with the synchronization mechanism introduced in the original patch.



\## 3. Problems Encountered



Several compatibility and testing issues were encountered during the experiment.



1\. \*\*Legacy Project Compatibility\*\*



&#x20;  The original gRPC-go repository did not include a modern Go Modules configuration. A separate experimental copy was therefore created, with a `go.mod` file and compatible dependency versions.



2\. \*\*CGO Configuration\*\*



&#x20;  The Go Race Detector required CGO support in the Windows environment. GCC was installed and configured to enable the required functionality.



3\. \*\*Go Vet Compatibility\*\*



&#x20;  Some legacy code triggered warnings under the modern Go vet checks. The `-vet=off` option was used during testing to avoid unrelated compatibility issues.



4\. \*\*Test Scope\*\*



&#x20;  The original historical network communication path was not reproduced end to end. Instead, field-level minimal tests were implemented to verify the synchronization problem and the effectiveness of mutex protection.



\## 4. Next Week's Plan



1\. Review and refine the experimental results and documentation.

2\. Coordinate with other team members to integrate individual findings.

3\. Continue testing and verification according to the group's next-week task allocation.

4\. Improve the reproducibility and organization of the experimental materials.



\## 5. Deliverables



| Deliverable | Status |

|---|---|

| `environment\_setup.md` | Completed |

| `grpc\_race\_test.md` | Completed |

| `transport/race\_repro\_test.go` | Completed |

| `test\_logs/01\_race\_bytes\_sent.txt` | Saved |

| `test\_logs/02\_safe\_bytes\_sent.txt` | Saved |

| `test\_logs/03\_race\_bytes\_received.txt` | Saved |

| `test\_logs/04\_safe\_bytes\_received.txt` | Saved |

| `weekly\_report.md` | Completed |





