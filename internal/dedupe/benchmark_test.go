package dedupe

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"filemill/internal/walker"
)

// FILEMILL_BENCH_FILES=100000 go test ./internal/dedupe -run '^$' -bench BenchmarkTree -benchtime=1x -benchmem
func BenchmarkTree(b *testing.B) {
	count := 1000
	if value := os.Getenv("FILEMILL_BENCH_FILES"); value != "" {
		var err error
		count, err = strconv.Atoi(value)
		if err != nil || count < 2 {
			b.Fatal("FILEMILL_BENCH_FILES must be >= 2")
		}
	}
	dir := b.TempDir()
	// Every file has a matching size, exercising the full hashing path.
	for i := 0; i < count; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%08d.txt", i)), []byte("duplicate content\n"), 0600); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := walker.Collect(dir, walker.Options{})
		if err != nil {
			b.Fatal(err)
		}
		r := Find(files, 8)
		if len(r.Errors) != 0 || r.Hashed != count || len(r.Groups) != 1 || len(r.Groups[0].Duplicates) != count-1 {
			b.Fatalf("incorrect large-tree result: scanned=%d hashed=%d errors=%v", r.Scanned, r.Hashed, r.Errors)
		}
	}
	b.ReportMetric(float64(count), "files/op")
}
