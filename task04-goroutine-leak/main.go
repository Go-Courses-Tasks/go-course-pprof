// task04: утечка горутин.
//
// Сервис поиска. На каждый запрос он параллельно спрашивает три реплики
// бэкенда и отдаёт ответ той, которая ответила первой (классический
// "hedged request"). Если за 50 мс никто не ответил - отдаёт таймаут.
//
// Под нагрузкой сервис работает, отвечает быстро... но число горутин
// растёт и никогда не уменьшается. Через сутки - OOM.
//
// Найди утечки через /debug/pprof/goroutine и исправь их.
// Подсказка: их там больше одной.
//
// Запуск:
//   go run .                      - сервис (:8081) + pprof (:6060)
//   go test -run Leak -v          - тест на утечку (сейчас ПАДАЕТ - это и есть баг)
//
// Подробные шаги - в README.md.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync/atomic"
	"time"
)

// ErrTimeout - ни одна реплика не ответила вовремя.
var ErrTimeout = errors.New("search: timeout")

const searchTimeout = 50 * time.Millisecond

// queryReplica имитирует запрос к реплике: случайная задержка 10-100 мс.
func queryReplica(replica int, q string) string {
	time.Sleep(time.Duration(10+rand.Intn(90)) * time.Millisecond)
	return fmt.Sprintf("реплика %d: результаты по запросу %q", replica, q)
}

// inflight - сколько запросов сейчас в обработке (для /stats).
var inflight atomic.Int64

// trackRequest пишет "пульс" запроса раз в 10 мс - типа для дебага
// долгих запросов. Должна работать, пока запрос обрабатывается.
func trackRequest(ctx context.Context, q string) {
	ticker := time.NewTicker(10 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			_ = q // тут могла бы быть запись в лог/метрику
		}
	}
}

// Search спрашивает три реплики параллельно и возвращает первый ответ.
//
// TODO: найди через pprof, какие горутины остаются висеть после того,
// как Search вернул результат, и почини.
func Search(ctx context.Context, q string) (string, error) {
	inflight.Add(1)
	defer inflight.Add(-1)

	ctx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()

	go trackRequest(ctx, q)

	results := make(chan string)
	for i := 1; i <= 3; i++ {
		go func(replica int) {
			results <- queryReplica(replica, q)
		}(i)
	}

	select {
	case r := <-results:
		return r, nil
	case <-ctx.Done():
		return "", ErrTimeout
	}
}

// =====================================================================
// HTTP
// =====================================================================

func searchHandler(w http.ResponseWriter, r *http.Request) {
	res, err := Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusGatewayTimeout)
		return
	}
	fmt.Fprintln(w, res)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "goroutines: %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "inflight:   %d\n", inflight.Load())
}

func main() {
	go func() {
		log.Println("pprof:  http://localhost:6060/debug/pprof/")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", searchHandler)
	mux.HandleFunc("GET /stats", statsHandler)

	// Встроенный мини-генератор нагрузки, чтобы не ставить ничего лишнего:
	// go run . -selfload  - сервис сам себе шлёт ~200 запросов в секунду.
	if selfLoad() {
		go generateLoad("http://localhost:8081/search?q=go")
	}

	log.Println("сервис: http://localhost:8081  (GET /search?q=..., GET /stats)")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
