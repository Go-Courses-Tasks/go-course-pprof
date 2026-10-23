package main

import (
	"testing"
)

// Тесты на корректность. После оптимизации должны остаться зелёными.

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"/api/users/42":         "/api/users/:id",
		"/api/users/42/orders":  "/api/users/:id/orders",
		"/api/orders/1/items/2": "/api/orders/:id/items/:id",
		"/api/1/2/3":            "/api/:id/:id/:id",
		"/api/products":         "/api/products",
		"/api/v2/users":         "/api/v2/users",
		"/healthz":              "/healthz",
		"/api/users/42abc":      "/api/users/42abc",
	}
	for in, want := range cases {
		if got := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnalyzeSmall(t *testing.T) {
	lines := []string{
		"2026-10-23T10:00:00Z GET /api/users/1 200 10ms",
		"2026-10-23T10:00:01Z GET /api/users/2 200 30ms",
		"2026-10-23T10:00:02Z GET /api/users/3 503 20ms",
		"какой-то мусор",
		"2026-10-23T10:00:03Z POST /api/orders 201 100ms",
		"2026-10-23T09:59:59Z GET /api/users/4 200 40ms",
	}
	rep := Analyze(lines)

	if rep.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", rep.Skipped)
	}
	if got := rep.From.Format("15:04:05"); got != "09:59:59" {
		t.Errorf("From = %s, want 09:59:59", got)
	}
	if got := rep.To.Format("15:04:05"); got != "10:00:03" {
		t.Errorf("To = %s, want 10:00:03", got)
	}
	if len(rep.Endpoints) != 2 {
		t.Fatalf("эндпоинтов %d, want 2: %v", len(rep.Endpoints), rep.Endpoints)
	}
	u := rep.Endpoints["GET /api/users/:id"]
	if u == nil || u.Count != 4 || u.Errors != 1 || u.P95 != 40 {
		t.Errorf("GET /api/users/:id = %+v, want Count=4 Errors=1 P95=40", u)
	}
	o := rep.Endpoints["POST /api/orders"]
	if o == nil || o.Count != 1 || o.Errors != 0 || o.P95 != 100 {
		t.Errorf("POST /api/orders = %+v, want Count=1 Errors=0 P95=100", o)
	}
}

// "Золотой" тест: на большом детерминированном логе отчёт должен совпасть
// с тем, что выдаёт исходная (медленная) версия.
func TestAnalyzeGolden(t *testing.T) {
	got := Analyze(GenerateLog(20000)).Format()
	if got != golden {
		t.Fatalf("отчёт изменился!\n--- got:\n%s\n--- want:\n%s", got, golden)
	}
}

func TestPercentile(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if got := percentile(s, 95); got != 10 {
		t.Errorf("p95 = %d, want 10", got)
	}
	if got := percentile(s, 50); got != 5 {
		t.Errorf("p50 = %d, want 5", got)
	}
	if got := percentile(nil, 95); got != 0 {
		t.Errorf("p95(nil) = %d, want 0", got)
	}
}

// Бенчмарк для профилей и benchstat:
//
//	go test -run='^$' -bench=Analyze -benchmem -count=10 | tee old.txt
func BenchmarkAnalyze(b *testing.B) {
	lines := GenerateLog(20000)
	b.ReportAllocs()
	for b.Loop() {
		Analyze(lines)
	}
}
