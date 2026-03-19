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

func BenchmarkStoreParseRequiredDotEnv(b *testing.B) {
	pathBenchmark1 := testdataPath(".env.benchmark1")

	// verify the file parses without error before benchmarking
	store := NewStore(0)
	cfg := new(Config)
	cfg.MergeGlobalOptions(Options{UnescapeBackslashDoubleQuote: new(true)})
	if err := store.ParseRequiredDotEnv(pathBenchmark1, cfg); err != nil {
		b.Fatalf("setup: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		s := NewStore(0)
		if err := s.ParseRequiredDotEnv(pathBenchmark1, cfg); err != nil {
			b.Fatal(err)
		}
	}
}
