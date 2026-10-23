package main

import (
	"fmt"
	"sync"
	"testing"
)

// Тесты на корректность - должны оставаться зелёными после оптимизации.
// Запускай с -race: go test -race -v
func TestGet(t *testing.T) {
	s := NewService()
	if got, want := s.Get("a"), render("a"); got != want {
		t.Fatalf("Get(a) = %q, want %q", got, want)
	}
	s.Get("a")
	s.Get("b")
	hits, misses := s.Stats()
	if hits != 1 || misses != 2 {
		t.Fatalf("Stats() = %d, %d; want 1, 2", hits, misses)
	}
	if n := s.Close(); n != 3 {
		t.Fatalf("audited = %d, want 3", n)
	}
}

func TestGetConcurrent(t *testing.T) {
	s := NewService()
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				key := fmt.Sprintf("k%d", i%50)
				if got := s.Get(key); got != render(key) {
					t.Errorf("Get(%s) вернул чужое значение", key)
					return
				}
			}
		}()
	}
	wg.Wait()
	hits, misses := s.Stats()
	if hits+misses != 8*500 {
		t.Fatalf("hits+misses = %d, want %d", hits+misses, 8*500)
	}
	// Каждый ключ должен быть отрендерен хотя бы раз и не больше,
	// чем числом горутин раз.
	if misses < 50 || misses > 50*8 {
		t.Fatalf("misses = %d, ожидали от 50 до 400", misses)
	}
	if n := s.Close(); n != 8*500 {
		t.Fatalf("audited = %d, want %d", n, 8*500)
	}
}

func keys(n int) []string {
	ks := make([]string, n)
	for i := range ks {
		ks[i] = fmt.Sprintf("item-%d", i)
	}
	return ks
}

// Однопоточный бенчмарк - для сравнения.
func BenchmarkGetSerial(b *testing.B) {
	s := NewService()
	defer s.Close()
	ks := keys(1000)
	i := 0
	for b.Loop() {
		s.Get(ks[i%len(ks)])
		i++
	}
}

// Параллельный бенчмарк: GOMAXPROCS горутин долбят Get одновременно.
//
//	go test -bench=Parallel -blockprofile=block.prof -mutexprofile=mutex.prof
func BenchmarkGetParallel(b *testing.B) {
	s := NewService()
	defer s.Close()
	ks := keys(1000)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Get(ks[i%len(ks)])
			i++
		}
	})
}
