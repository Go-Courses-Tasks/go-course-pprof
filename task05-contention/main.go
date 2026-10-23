// task05: block- и mutex-профили. Ищем, где горутины ЖДУТ.
//
// Service - кэш "отрендеренных" карточек товаров с аудитом обращений.
// На один поток он быстрый. Даёшь ему 8 ядер - быстрее не становится,
// а CPU при этом почти не загружен. CPU-профиль тут почти бесполезен:
// горутины не работают, они стоят в очереди.
//
// Найди, где именно они стоят, с помощью -blockprofile и -mutexprofile.
//
// Бенчмарк + профили:
//   go test -bench=Parallel -blockprofile=block.prof -mutexprofile=mutex.prof
//   go tool pprof block.prof
//   go tool pprof mutex.prof
//
// Подробные шаги - в README.md.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Service кэширует результат render(key) и пишет аудит каждого обращения.
type Service struct {
	mu     sync.Mutex
	cache  map[string]string
	hits   int
	misses int

	events  chan string
	audited int // трогает только auditLoop
	done    chan struct{}
}

// NewService создаёт сервис и запускает фоновый аудит.
func NewService() *Service {
	s := &Service{
		cache:  make(map[string]string),
		events: make(chan string),
		done:   make(chan struct{}),
	}
	go s.auditLoop()
	return s
}

// Get возвращает карточку товара по ключу.
//
// TODO (после профилирования): убери лишнее ожидание.
// Поведение Get и Stats менять нельзя - main_test.go должен остаться зелёным.
func (s *Service) Get(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.cache[key]
	if ok {
		s.hits++
	} else {
		v = render(key)
		s.cache[key] = v
		s.misses++
	}

	s.events <- key
	return v
}

// Stats возвращает число попаданий и промахов кэша.
func (s *Service) Stats() (hits, misses int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits, s.misses
}

// Close останавливает аудит и возвращает, сколько событий он обработал.
func (s *Service) Close() int {
	close(s.events)
	<-s.done
	return s.audited
}

func (s *Service) auditLoop() {
	defer close(s.done)
	for key := range s.events {
		s.audit(key)
	}
}

// audit - "запись в журнал аудита": немного CPU на каждое событие.
func (s *Service) audit(key string) {
	sum := sha256.Sum256([]byte(key))
	for i := 0; i < 20; i++ {
		sum = sha256.Sum256(sum[:])
	}
	_ = sum
	s.audited++
}

// render - "дорогой" рендер карточки товара.
func render(key string) string {
	sum := sha256.Sum256([]byte(key))
	for i := 0; i < 200; i++ {
		sum = sha256.Sum256(sum[:])
	}
	return "card:" + key + ":" + hex.EncodeToString(sum[:8])
}

func main() {
	s := NewService()

	const total = 200_000
	workers := runtime.GOMAXPROCS(0)

	start := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < total/workers; i++ {
				s.Get(fmt.Sprintf("item-%d", (i*7+w)%1000))
			}
		}(w)
	}
	wg.Wait()
	elapsed := time.Since(start)

	hits, misses := s.Stats()
	audited := s.Close()
	fmt.Printf("%d горутин, %d запросов за %v (%.0f req/s)\n",
		workers, hits+misses, elapsed, float64(hits+misses)/elapsed.Seconds())
	fmt.Printf("hits=%d misses=%d audited=%d\n", hits, misses, audited)
}
