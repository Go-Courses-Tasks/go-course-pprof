package main

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateLog генерирует n детерминированных строк access-лога.
func GenerateLog(n int) []string {
	r := rand.New(rand.NewSource(1))
	type ep struct {
		method, path string
		baseLatency  int
	}
	eps := []ep{
		{"GET", "/api/users/%d", 15},
		{"GET", "/api/users/%d/orders", 40},
		{"GET", "/api/orders/%d/items/%d", 25},
		{"POST", "/api/orders", 120},
		{"GET", "/api/products", 30},
		{"GET", "/api/products/%d", 10},
		{"DELETE", "/api/sessions/%d", 5},
		{"GET", "/healthz", 1},
	}
	start := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	lines := make([]string, 0, n)
	for i := 0; i < n; i++ {
		e := eps[r.Intn(len(eps))]
		path := e.path
		switch c := countVerbs(path); c {
		case 1:
			path = fmt.Sprintf(path, r.Intn(100000))
		case 2:
			path = fmt.Sprintf(path, r.Intn(100000), r.Intn(50))
		}
		status := 200
		if r.Intn(50) == 0 {
			status = 500 + r.Intn(4)
		}
		latency := e.baseLatency + r.Intn(e.baseLatency*3+1)
		if r.Intn(100) < 3 {
			latency *= 10 // редкие медленные запросы - ради них и нужен p95
		}
		ts := start.Add(time.Duration(i) * 10 * time.Millisecond).Format(time.RFC3339)
		lines = append(lines, fmt.Sprintf("%s %s %s %d %dms", ts, e.method, path, status, latency))
		if r.Intn(1000) == 0 {
			lines = append(lines, "garbage line") // битые строки бывают в любом логе
		}
	}
	return lines
}

func countVerbs(s string) int {
	n := 0
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '%' && s[i+1] == 'd' {
			n++
		}
	}
	return n
}

func main() {
	lines := GenerateLog(20000)

	start := time.Now()
	rep := Analyze(lines)
	elapsed := time.Since(start)

	fmt.Print(rep.Format())
	fmt.Printf("\nразобрано %d строк за %v\n", len(lines), elapsed)
}
