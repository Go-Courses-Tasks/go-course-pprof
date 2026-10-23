// Генератор нагрузки для task03.
//
// Шлёт POST /login с несколькими параллельными клиентами и печатает RPS.
// Нужен, чтобы профилю было что показать: pprof на простаивающем сервисе пуст.
//
// Запуск (из папки task03-http-pprof):
//   go run ./cmd/load                       - 30 секунд, 8 клиентов
//   go run ./cmd/load -d 60s -c 16          - подольше и пожёстче
//
// Вместо него можно взять hey:
//   hey -z 30s -c 8 -m POST "http://localhost:8081/login?user=bob&password=secret"

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	addr := flag.String("addr", "http://localhost:8081", "адрес сервиса")
	dur := flag.Duration("d", 30*time.Second, "длительность нагрузки")
	conc := flag.Int("c", 8, "число параллельных клиентов")
	flag.Parse()

	var ok, fail atomic.Int64
	deadline := time.Now().Add(*dur)
	client := &http.Client{Timeout: 5 * time.Second}

	var wg sync.WaitGroup
	for c := 0; c < *conc; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				url := fmt.Sprintf("%s/login?user=user%d-%d&password=secret", *addr, c, i)
				resp, err := client.Post(url, "text/plain", nil)
				if err != nil {
					fail.Add(1)
					time.Sleep(100 * time.Millisecond)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					ok.Add(1)
				} else {
					fail.Add(1)
				}
			}
		}(c)
	}

	// Раз в секунду печатаем прогресс.
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		var prev int64
		for {
			select {
			case <-done:
				return
			case <-t.C:
				cur := ok.Load()
				log.Printf("rps=%d  ok=%d  fail=%d", cur-prev, cur, fail.Load())
				prev = cur
			}
		}
	}()

	wg.Wait()
	close(done)
	fmt.Printf("готово: ok=%d fail=%d за %v\n", ok.Load(), fail.Load(), *dur)
}
