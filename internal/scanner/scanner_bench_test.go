package scanner

import (
	"context"
	"strings"
	"testing"
)

func BenchmarkScan(b *testing.B) {
	root := b.TempDir()
	content := strings.Repeat("PORT=8080\nLOG_LEVEL=info\n", 4096) + "password=synthetic-only\n"
	writeTestFile(b, root, "app.env", content)
	settings := Defaults()
	settings.Workers = 1
	engine := newTestScanner(b, WithSettings(settings))
	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := engine.Scan(context.Background(), root)
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Findings) != 1 {
			b.Fatal("benchmark lost its detection")
		}
	}
}
