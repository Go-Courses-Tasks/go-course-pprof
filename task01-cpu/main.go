// task01: CPU-профиль. Ищем, на что программа тратит процессор.
//
// Здесь лежит функция TopWords - она считает самые частые слова в тексте.
// Работает правильно (тесты зелёные), но медленно. Твоя задача - НЕ угадывать,
// почему она медленная, а СНЯТЬ CPU-профиль и увидеть это своими глазами.
//
// Запуск программы (просто посмотреть, что она делает):
//   go run .
//
// Бенчмарк + CPU-профиль:
//   go test -bench=TopWords -benchmem -cpuprofile=cpu.prof
//   go tool pprof cpu.prof
//
// Подробные шаги - в README.md.

package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// WordCount - слово и сколько раз оно встретилось.
type WordCount struct {
	Word  string
	Count int
}

// stopWords - служебные слова, которые не считаем.
var stopWords = []string{
	"и", "в", "во", "не", "что", "он", "на", "я", "с", "со", "как", "а", "то",
	"все", "она", "так", "его", "но", "да", "ты", "к", "у", "же", "вы", "за",
	"бы", "по", "только", "ее", "мне", "было", "вот", "от", "меня", "еще",
	"нет", "о", "из", "ему", "the", "a", "an", "and", "or", "of", "to", "in",
}

// TopWords возвращает n самых частых слов текста.
//
// Правила:
//   - слово - это последовательность букв (латиница или кириллица);
//   - регистр не важен: "Go" и "go" - одно слово;
//   - стоп-слова (stopWords) не считаем;
//   - сортировка: по убыванию частоты, при равенстве - по алфавиту.
//
// TODO (после того как найдёшь узкие места в профиле): ускорь функцию.
// Поведение менять нельзя - main_test.go должен остаться зелёным.
func TopWords(text string, n int) []WordCount {
	counts := make(map[string]int)

	for _, line := range strings.Split(text, "\n") {
		wordRe := regexp.MustCompile(`[a-zа-яё]+`)
		for _, w := range wordRe.FindAllString(strings.ToLower(line), -1) {
			if isStopWord(w) {
				continue
			}
			counts[w]++
		}
	}

	result := make([]WordCount, 0, len(counts))
	for w, c := range counts {
		result = append(result, WordCount{Word: w, Count: c})
	}

	sortWordCounts(result)

	if len(result) > n {
		result = result[:n]
	}
	return result
}

func isStopWord(w string) bool {
	for _, s := range stopWords {
		if s == w {
			return true
		}
	}
	return false
}

// sortWordCounts сортирует по убыванию Count, при равенстве - по Word.
func sortWordCounts(ws []WordCount) {
	for i := 0; i < len(ws); i++ {
		for j := 0; j < len(ws)-1-i; j++ {
			if less(ws[j+1], ws[j]) {
				ws[j], ws[j+1] = ws[j+1], ws[j]
			}
		}
	}
}

func less(a, b WordCount) bool {
	if a.Count != b.Count {
		return a.Count > b.Count
	}
	return a.Word < b.Word
}

// GenerateText генерирует детерминированный "текст": lines строк по
// wordsPerLine слов из словаря размером vocab. Нужен для бенчмарка.
func GenerateText(lines, wordsPerLine, vocab int) string {
	r := rand.New(rand.NewSource(42))
	words := make([]string, vocab)
	letters := []rune("абвгдежзиклмнопрстуфхцчшэюяabcdefghijklmnopqrstuvwxyz")
	for i := range words {
		l := 3 + r.Intn(8)
		var sb strings.Builder
		for j := 0; j < l; j++ {
			sb.WriteRune(letters[r.Intn(len(letters))])
		}
		words[i] = sb.String()
	}

	var sb strings.Builder
	for i := 0; i < lines; i++ {
		for j := 0; j < wordsPerLine; j++ {
			if j > 0 {
				sb.WriteString(", ")
			}
			// Степенное распределение: часть слов встречается часто, большинство - редко.
			idx := int(float64(vocab) * r.Float64() * r.Float64())
			w := words[idx]
			if r.Intn(10) == 0 {
				w = strings.ToUpper(w)
			}
			sb.WriteString(w)
		}
		sb.WriteString(".\n")
	}
	return sb.String()
}

func main() {
	text := GenerateText(20000, 10, 3000)

	start := time.Now()
	top := TopWords(text, 10)
	fmt.Printf("TopWords отработала за %v\n\n", time.Since(start))

	for i, wc := range top {
		fmt.Printf("%2d. %-12s %d\n", i+1, wc.Word, wc.Count)
	}
}
