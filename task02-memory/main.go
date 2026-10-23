// task02: профиль памяти. Ищем, кто и сколько аллоцирует.
//
// Здесь две функции из "сервиса отчётов":
//   ParseOrders  - разбирает строки CSV-выгрузки в []Order;
//   RenderReport - собирает из заказов текстовый отчёт.
//
// Обе работают правильно, но каждая аллоцирует так, что GC не успевает
// отдыхать. Твоя задача - снять профиль памяти, понять, ГДЕ и ПОЧЕМУ
// аллокации, и уменьшить их.
//
// Бенчмарк + профиль памяти:
//   go test -bench=. -benchmem -memprofile=mem.prof
//   go tool pprof -sample_index=alloc_space mem.prof
//
// Подробные шаги - в README.md.

package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// Order - один заказ из выгрузки.
type Order struct {
	ID       int
	Customer string
	Items    []string
	Total    float64
}

// ParseOrders разбирает строки вида
//
//	"<id>;<покупатель>;<товар1>,<товар2>,...;<сумма>"
//
// Пустые и битые строки пропускает.
//
// TODO (после профилирования): уменьши количество аллокаций.
func ParseOrders(lines []string) []Order {
	var orders []Order
	for _, line := range lines {
		parts := strings.Split(line, ";")
		if len(parts) != 4 {
			continue
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		total, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			continue
		}
		var items []string
		for _, it := range strings.Split(parts[2], ",") {
			items = append(items, strings.TrimSpace(it))
		}
		orders = append(orders, Order{
			ID:       id,
			Customer: fmt.Sprintf("%s", parts[1]),
			Items:    items,
			Total:    total,
		})
	}
	return orders
}

// RenderReport собирает отчёт: по строке на заказ плюс итог.
//
// TODO (после профилирования): уменьши количество аллокаций и байт.
func RenderReport(orders []Order) string {
	report := "ОТЧЁТ ПО ЗАКАЗАМ\n"
	var sum float64
	for _, o := range orders {
		line := fmt.Sprintf("#%d %s: %s = %.2f\n", o.ID, o.Customer, strings.Join(o.Items, ", "), o.Total)
		report += line
		sum += o.Total
	}
	report += fmt.Sprintf("ИТОГО: %d заказов на %.2f\n", len(orders), sum)
	return report
}

// GenerateCSV генерирует n детерминированных строк выгрузки для бенчмарков.
func GenerateCSV(n int) []string {
	r := rand.New(rand.NewSource(7))
	customers := []string{"Иванов", "Петрова", "Сидоров", "Козлова", "Smith", "Müller"}
	goods := []string{"чай", "кофе", "сахар", "молоко", "хлеб", "сыр", "яблоки", "печенье"}
	lines := make([]string, 0, n)
	for i := 0; i < n; i++ {
		k := 1 + r.Intn(5)
		items := make([]string, k)
		for j := range items {
			items[j] = goods[r.Intn(len(goods))]
		}
		lines = append(lines, fmt.Sprintf("%d;%s;%s;%.2f",
			i+1, customers[r.Intn(len(customers))], strings.Join(items, ", "), 100+r.Float64()*9900))
	}
	return lines
}

func main() {
	lines := GenerateCSV(5000)

	start := time.Now()
	orders := ParseOrders(lines)
	report := RenderReport(orders)
	fmt.Printf("обработано %d заказов за %v, размер отчёта %d байт\n\n",
		len(orders), time.Since(start), len(report))

	// Первые строки отчёта - просто посмотреть.
	fmt.Println(strings.Join(strings.SplitN(report, "\n", 6)[:5], "\n"))
}
