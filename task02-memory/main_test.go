package main

import (
	"reflect"
	"strings"
	"testing"
)

// Тесты на корректность - должны оставаться зелёными после оптимизации.
func TestParseOrders(t *testing.T) {
	lines := []string{
		"1;Иванов;чай, кофе;150.50",
		"",                 // пустая - пропускаем
		"битая строка",     // нет полей - пропускаем
		"x;Петрова;сыр;10", // id не число - пропускаем
		"2;Петрова;сыр;10",
	}
	got := ParseOrders(lines)
	want := []Order{
		{ID: 1, Customer: "Иванов", Items: []string{"чай", "кофе"}, Total: 150.50},
		{ID: 2, Customer: "Петрова", Items: []string{"сыр"}, Total: 10},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseOrders:\n got  %+v\n want %+v", got, want)
	}
}

func TestRenderReport(t *testing.T) {
	orders := []Order{
		{ID: 1, Customer: "Иванов", Items: []string{"чай", "кофе"}, Total: 150.5},
		{ID: 2, Customer: "Петрова", Items: []string{"сыр"}, Total: 10},
	}
	got := RenderReport(orders)
	want := "ОТЧЁТ ПО ЗАКАЗАМ\n" +
		"#1 Иванов: чай, кофе = 150.50\n" +
		"#2 Петрова: сыр = 10.00\n" +
		"ИТОГО: 2 заказов на 160.50\n"
	if got != want {
		t.Fatalf("RenderReport:\n got  %q\n want %q", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	lines := GenerateCSV(100)
	report := RenderReport(ParseOrders(lines))
	if n := strings.Count(report, "\n"); n != 102 {
		t.Fatalf("ожидали 102 строки в отчёте, получили %d", n)
	}
}

// go test -bench=. -benchmem -memprofile=mem.prof
func BenchmarkParseOrders(b *testing.B) {
	lines := GenerateCSV(5000)
	b.ReportAllocs()
	for b.Loop() {
		ParseOrders(lines)
	}
}

func BenchmarkRenderReport(b *testing.B) {
	orders := ParseOrders(GenerateCSV(5000))
	b.ReportAllocs()
	for b.Loop() {
		RenderReport(orders)
	}
}
