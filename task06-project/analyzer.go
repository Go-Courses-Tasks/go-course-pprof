// task06: финальный проект. Найди и исправь узкие места - с доказательствами.
//
// Analyze разбирает access-лог HTTP-сервиса и считает по каждому эндпоинту:
// число запросов, число 5xx и p95 латентности. Такой отчёт строится
// каждую минуту по свежему логу - и он перестал успевать.
//
// В этом файле НЕСКОЛЬКО проблем с производительностью. Какие - не скажем.
// Найди их профилями (CPU + память), исправь и докажи ускорение benchstat-ом.
//
// Подробные шаги - в README.md.

package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// EndpointStats - статистика по одному эндпоинту.
type EndpointStats struct {
	Count  int
	Errors int // ответы 5xx
	P95    int // p95 латентности, мс

	latencies []int // отсортированные латентности
}

// Report - итог анализа лога.
type Report struct {
	From, To  time.Time
	Skipped   int // строки, которые не удалось разобрать
	Endpoints map[string]*EndpointStats
}

// Analyze разбирает строки лога вида
//
//	2026-10-23T10:00:00Z GET /api/users/42 200 15ms
//
// Ключ эндпоинта - метод + путь, в котором числовые сегменты заменены на :id:
//
//	"GET /api/users/:id"
func Analyze(lines []string) *Report {
	rep := &Report{Endpoints: make(map[string]*EndpointStats)}

	for _, line := range lines {
		var ts, method, path string
		var status, latency int
		_, err := fmt.Sscanf(line, "%s %s %s %d %dms", &ts, &method, &path, &status, &latency)
		if err != nil {
			rep.Skipped++
			continue
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			rep.Skipped++
			continue
		}
		if rep.From.IsZero() || t.Before(rep.From) {
			rep.From = t
		}
		if t.After(rep.To) {
			rep.To = t
		}

		key := fmt.Sprintf("%s %s", method, normalizePath(path))
		st, ok := rep.Endpoints[key]
		if !ok {
			st = &EndpointStats{}
			rep.Endpoints[key] = st
		}
		st.Count++
		if status >= 500 {
			st.Errors++
		}
		st.latencies = insertSorted(st.latencies, latency)
	}

	for _, st := range rep.Endpoints {
		st.P95 = percentile(st.latencies, 95)
	}
	return rep
}

// normalizePath заменяет числовые сегменты пути на :id.
//
//	/api/users/42/orders/7 -> /api/users/:id/orders/:id
func normalizePath(path string) string {
	re := regexp.MustCompile(`/[0-9]+(/|$)`)
	// ReplaceAll не находит перекрывающиеся совпадения ("/1/2"), поэтому крутим до упора.
	for {
		next := re.ReplaceAllString(path, "/:id$1")
		if next == path {
			return next
		}
		path = next
	}
}

// insertSorted вставляет v в отсортированный слайс, сохраняя порядок.
func insertSorted(s []int, v int) []int {
	i := sort.SearchInts(s, v)
	res := make([]int, 0, len(s)+1)
	res = append(res, s[:i]...)
	res = append(res, v)
	res = append(res, s[i:]...)
	return res
}

// percentile - перцентиль методом ближайшего ранга по ОТСОРТИРОВАННОМУ слайсу.
func percentile(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	rank := (p*len(sorted) + 99) / 100 // ceil(p/100 * n)
	if rank < 1 {
		rank = 1
	}
	return sorted[rank-1]
}

// Format печатает отчёт: эндпоинты по алфавиту.
func (r *Report) Format() string {
	keys := make([]string, 0, len(r.Endpoints))
	for k := range r.Endpoints {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	fmt.Fprintf(&sb, "период: %s - %s, пропущено строк: %d\n",
		r.From.Format(time.RFC3339), r.To.Format(time.RFC3339), r.Skipped)
	for _, k := range keys {
		st := r.Endpoints[k]
		fmt.Fprintf(&sb, "%-30s count=%-6d 5xx=%-5d p95=%dms\n", k, st.Count, st.Errors, st.P95)
	}
	return sb.String()
}
