// task03: pprof в работающем HTTP-сервисе.
//
// Это маленький "сервис авторизации":
//   POST /login?user=<имя>  - проверяет пароль (хеширует его) и создаёт сессию
//   GET  /stats             - сколько сессий и горутин, сколько памяти в куче
//
// У сервиса две проблемы, которые видны только под нагрузкой:
//   1. он ест много CPU на каждый логин;
//   2. его память растёт и не возвращается.
//
// Найди обе через /debug/pprof/, не читая код заранее (честно!).
//
// Запуск:
//   go run .                 - сервис (:8081) + pprof (:6060)
//   go run ./cmd/load        - генератор нагрузки (в другом терминале)
//
// Подробные шаги - в README.md.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	// Пустой импорт ради побочного эффекта: пакет в init() регистрирует
	// хендлеры /debug/pprof/* в http.DefaultServeMux.
	// Сам по себе он ничего не "включает" и почти ничего не стоит,
	// пока никто не дёргает эндпоинты.
	_ "net/http/pprof"
)

// =====================================================================
// "Бизнес-логика"
// =====================================================================

// hashPassword - имитация медленного хеширования пароля (как bcrypt/argon2).
func hashPassword(password, salt string) string {
	sum := sha256.Sum256([]byte(salt + password))
	for i := 0; i < 50000; i++ {
		sum = sha256.Sum256(sum[:])
	}
	return hex.EncodeToString(sum[:])
}

// Session - данные сессии пользователя.
type Session struct {
	ID        string
	User      string
	CreatedAt time.Time
	// Буфер под "контекст" сессии: права, настройки, кэш профиля...
	Payload []byte
}

var (
	sessionsMu sync.Mutex
	sessions   = make(map[string]*Session)
	nextID     int
)

func newSession(user string) *Session {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	nextID++
	s := &Session{
		ID:        fmt.Sprintf("sess-%d", nextID),
		User:      user,
		CreatedAt: time.Now(),
		Payload:   make([]byte, 4*1024),
	}
	sessions[s.ID] = s
	return s
}

// =====================================================================
// HTTP
// =====================================================================

func loginHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		http.Error(w, "нужен параметр user", http.StatusBadRequest)
		return
	}
	// Пароль в query - только для учебного примера, в жизни так не делай.
	hash := hashPassword(r.URL.Query().Get("password"), user)
	if len(hash) == 0 {
		http.Error(w, "внутренняя ошибка", http.StatusInternalServerError)
		return
	}
	s := newSession(user)
	fmt.Fprintf(w, "ok %s\n", s.ID)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sessionsMu.Lock()
	n := len(sessions)
	sessionsMu.Unlock()

	fmt.Fprintf(w, "sessions:   %d\n", n)
	fmt.Fprintf(w, "goroutines: %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "heap_inuse: %.1f MB\n", float64(m.HeapInuse)/1024/1024)
	fmt.Fprintf(w, "num_gc:     %d\n", m.NumGC)
}

func main() {
	// pprof - на ОТДЕЛЬНОМ порту и только на localhost.
	// В проде /debug/pprof наружу не выставляют: там стеки, пути к файлам,
	// а снятие CPU-профиля само по себе нагружает сервис.
	go func() {
		log.Println("pprof:  http://localhost:6060/debug/pprof/")
		log.Println(http.ListenAndServe("localhost:6060", nil)) // nil = DefaultServeMux, туда зарегистрировался pprof
	}()

	// Бизнес-хендлеры - на своём mux-е, чтобы pprof случайно не утёк наружу вместе с ними.
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", loginHandler)
	mux.HandleFunc("GET /stats", statsHandler)

	log.Println("сервис: http://localhost:8081  (POST /login?user=..., GET /stats)")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
