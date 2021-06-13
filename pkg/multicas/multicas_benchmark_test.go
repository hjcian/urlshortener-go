package multicas

import "testing"

func Benchmark_MultiCAS_Set_same_key_by_single_GR(b *testing.B) {
	benchmarks := []struct {
		name  string
		class MultiCAS
	}{
		{
			"version 1 - use sync.Mutex",
			newMultiCAS_v1_forTest(),
		},
		{
			"version 2 - use sync.RWMutex",
			newMultiCAS_v2_forTest(),
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bm.class.Set(123)
			}
		})
	}
}

func Benchmark_MultiCAS_Set_same_key_by_multiple_GR(b *testing.B) {
	b.SetParallelism(8)
	benchmarks := []struct {
		name  string
		class MultiCAS
	}{
		{
			"version 1 - use sync.Mutex",
			newMultiCAS_v1_forTest(),
		},
		{
			"version 2 - use sync.RWMutex",
			newMultiCAS_v2_forTest(),
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					bm.class.Set(123)
				}
			})

		})
	}
}
