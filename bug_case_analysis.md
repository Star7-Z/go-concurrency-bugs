**Preliminary Investigation of Bug Cases**

Thesis Repository: https://github.com/system-pclub/go-concurrency-bugs

The warehouse is mainly divided into two folders: ‘blocking-bugs/’ and ‘non-blocking-bugs/’. And in the README.md, the author directly posted the minimalist code of four typical bugs. This can better assist us in conducting research:

1.	blocking-bugs/docker#25384: Corresponding to WaitGroup deadlock
```
1 var group sync.WaitGroup
2 group.Add(len(pm.plugins))
3 for _, p := range pm.plugins {
4   go func(p *plugin) {
5    defer group.Done()
6   }
7 - group.Wait()
8 }
9 + group.Wait()
```
2.	blocking-bugs/Kubernetes: Corresponding channel timeout blocking
```
1 func finishReq(timeout time.Duration) r ob {
2 -   ch := make(chan ob)
3 +   ch := make(chan ob, 1)
4   go func() {
5     result := fn()
6     ch <- result // block
7   } ()
8   select {
9     case result = <- ch:
10       return result
11     case <- time.After(timeout):
12       return nil
13   }
14 }
```
3.	non-blocking-bugs/docker: Corresponding to the competition of closure variable i
```
1  for i := 17; i <= 21; i++ { // write
2 -   go func() { /* Create a new goroutine */
3 +   go func(i int) {
4            apiVersion := fmt.Sprintf("v1.%d", i) // read
5            ...
6 -       }()
7 +       }(i)
8   }
```
4.	non-blocking-bugs/docker#24007: Repeatedly close the panic for the corresponding channel
```
1 - select {
2 -   case <- c.closed:
3 -   default:
4 +     Once.Do(func() {
5         close(c.closed)
6 +     })
7 - }
```

The above four case codes are minimalist, removing complex engineering dependencies. In the future, you can set up a Go ring locally, write test cases to trigger these bugs, and verify whether the Race Detector or built-in deadlock detector can capture them.
