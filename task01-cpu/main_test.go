package main

import (
	"reflect"
	"testing"
)

// Тест на корректность. Он должен оставаться зелёным и ДО, и ПОСЛЕ оптимизации.
func TestTopWords(t *testing.T) {
	text := "Go go GO! И снова go.\n" +
		"Профилирование - это просто. Профилирование и бенчмарки.\n" +
		"бенчмарки, pprof, pprof\n" +
		"the end"

	got := TopWords(text, 4)
	// При равной частоте - по алфавиту (обычное сравнение строк в Go):
	// латиница идёт раньше кириллицы. "и" и "the" - стоп-слова.
	want := []WordCount{
		{"go", 4},
		{"pprof", 2},
		{"бенчмарки", 2},
		{"профилирование", 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TopWords:\n got  %v\n want %v", got, want)
	}
}

func TestTopWordsBig(t *testing.T) {
	text := GenerateText(500, 20, 2000)
	got := TopWords(text, 50)
	if len(got) != 50 {
		t.Fatalf("ожидали 50 слов, получили %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if less(got[i], got[i-1]) {
			t.Fatalf("нарушен порядок сортировки на позиции %d: %v перед %v", i, got[i-1], got[i])
		}
	}
}

// Бенчмарк, на котором снимаем профиль:
//
//	go test -bench=TopWords -benchmem -cpuprofile=cpu.prof
func BenchmarkTopWords(b *testing.B) {
	text := GenerateText(20000, 10, 3000)
	for b.Loop() {
		TopWords(text, 10)
	}
}
