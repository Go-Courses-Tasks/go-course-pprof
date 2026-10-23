package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"time"
)

var selfLoadFlag = flag.Bool("selfload", false, "сервис сам генерирует себе нагрузку (~200 rps)")

func selfLoad() bool {
	flag.Parse()
	return *selfLoadFlag
}

// generateLoad шлёт запросы с постоянной скоростью (~200 rps).
// Сам ничего не печатает - смотри /stats и pprof.
func generateLoad(url string) {
	time.Sleep(500 * time.Millisecond) // даём серверу подняться
	log.Println("selfload: шлём ~200 rps на", url)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for range tick.C {
		go func() {
			resp, err := http.Get(url)
			if err != nil {
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
}
