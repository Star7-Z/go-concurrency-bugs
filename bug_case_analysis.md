**Preliminary Investigation of Bug Cases**

In line with Sections 5 and 6 of the paper "Understanding Real-World Concurrency Bugs in Go," conduct an investigation into the case studies and the official repositories; this will serve as preparation for the bug selection and testing tasks scheduled for the second week.
Thesis Repository: https://github.com/system-pclub/go-concurrency-bugs

The warehouse is mainly divided into two folders: ‘blocking-bugs/’ and ‘non-blocking-bugs/’:
```text
go-concurrency-bugs/
├── blocking-bugs/
│   ├── docker/
│   ├── kubernetes/
│   ├── etcd/
│   ├── cockroachdb/
│   ├── grpc-go/
│   └── boltdb/
├── non-blocking-bugs/
│   ├── docker/
│   ├── kubernetes/
│   ├── etcd/
│   ├── cockroachdb/
│   ├── grpc-go/
│   └── boltdb/
└── bug-table/

File format: All are commit patch files (containing the commit message, author, and code diff).
