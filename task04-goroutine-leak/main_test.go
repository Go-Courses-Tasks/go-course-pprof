package main

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSearchReturnsResultOrTimeout(t *testing.T) {
	for i := 0; i < 20; i++ {
		res, err := Search(context.Background(), "pprof")
		if err != nil && err != ErrTimeout {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if err == nil && !strings.Contains(res, "pprof") {
			t.Fatalf("странный ответ: %q", res)
		}
	}
}

// Тест на утечку горутин.
//
// Идея простая: запоминаем число горутин, делаем 100 поисков, ждём, пока
// всё "успокоится", и проверяем, что горутин стало примерно столько же.
//
// Сейчас тест ПАДАЕТ. Когда починишь утечки - станет зелёным.
func TestSearchNoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		Search(context.Background(), "leak")
	}

	// Даём "честным" горутинам время доработать: самые медленные реплики
	// отвечают за 100 мс.
	deadline := time.Now().Add(2 * time.Second)
	var after int
	for time.Now().Before(deadline) {
		after = runtime.NumGoroutine()
		if after <= before+5 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Печатаем, где висят горутины - тот же формат, что /debug/pprof/goroutine?debug=2.
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	t.Fatalf("утечка горутин: было %d, стало %d\n\nпервые стеки:\n%s", before, after, buf[:n])
}
