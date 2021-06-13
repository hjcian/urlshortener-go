# MutliCAS


**benchmark results**

```
$ make test-bench
go test -bench=. -benchmem -run=none .
goos: darwin
goarch: amd64
pkg: goshorturl/pkg/multicas
Benchmark_MultiCAS_Set_same_key_by_single_GR/version_1_-_use_sync.Mutex-4      39329145   41.0 ns/op   0 B/op   0 allocs/op
Benchmark_MultiCAS_Set_same_key_by_single_GR/version_2_-_use_sync.RWMutex-4    38026896   35.5 ns/op   0 B/op   0 allocs/op
Benchmark_MultiCAS_Set_same_key_by_multiple_GR/version_1_-_use_sync.Mutex-4    20610673   62.4 ns/op   0 B/op   0 allocs/op
Benchmark_MultiCAS_Set_same_key_by_multiple_GR/version_2_-_use_sync.RWMutex-4  27938036   62.9 ns/op   0 B/op   0 allocs/op
PASS
ok      goshorturl/pkg/multicas 7.254s
```
- 可觀察到 version 2 使用 RWLock 可提升 multiple goroutines 在 `Set()` 中檢查 `m.table[key]` 是否已經被設為 `true` 時的效率 (*每秒執行數：27938036 > 20610673*)