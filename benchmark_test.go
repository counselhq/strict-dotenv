package strictdotenv

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "testdata", name)
}

func BenchmarkEnvStoreSetFromRequiredDotEnv(b *testing.B) {
	pathBenchmark1 := testdataPath(".env.benchmark1")

	// verify the file parses without error before benchmarking
	store := NewEnvStore()
	cfg := NewParseConfig().WithRecommendedDefaults()
	if err := store.SetFromRequiredDotEnv(pathBenchmark1, cfg); err != nil {
		b.Fatalf("setup: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		s := NewEnvStore()
		if err := s.SetFromRequiredDotEnv(pathBenchmark1, cfg); err != nil {
			b.Fatal(err)
		}
	}
}
