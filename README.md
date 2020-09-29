# golang-cache

一个在大量并发的前提下, 性能比 go-cache 好的, 纯在内存中缓存的 Key-Value 容器.

❯ go test -bench .
goos: linux
goarch: amd64
pkg: github.com/black-desk/golang-cache
BenchmarkMultiThreadSetGetAddMy-8                      1        46580500100 ns/op
BenchmarkMultiThreadSetGetAddGoCache-8                 1        71747233300 ns/op
PASS
ok      github.com/black-desk/golang-cache      131.822s

具体测试内容看源码

go-cache 当一个 goroutine 在写的时候, 会将整个 hash map 上锁.

选择在 map 中存指向对象的指针, 读出指针之后锁住对象, hash map 就可以解锁了, 方便同进行其他的读/写

