# Задачи: профилирование Go-программ с pprof

Представь: сервис начал тормозить. Или жрать память. Или его перенесли на сервер с 16 ядрами, а RPS не вырос. Что ты делаешь?

Новичок открывает код и начинает "оптимизировать то, что выглядит медленно". Через два дня у него переписанный модуль, новые баги и тот же самый тормоз - потому что проблема была в другом месте.

Опытный инженер сначала **измеряет**. В Go для этого есть `pprof` - профилировщик, встроенный прямо в рантайм и стандартную библиотеку. Он отвечает на вопросы:
- **где** программа тратит процессор (CPU-профиль);
- **кто** выделяет память и **кто её держит** (heap / allocs);
- **сколько** горутин и **где** они висят (goroutine);
- **где** горутины ждут друг друга (block / mutex).

В этих задачах в каждой программе специально спрятана проблема. Твоя работа - не угадать её, а **найти профилировщиком**, исправить и **доказать** цифрами, что стало лучше.

> **Теория** - в курсе: [`09-performance`](https://github.com/Go-Courses-Tasks/go-course) (особенно `9.2-profiling`). Здесь - практика.

---

## Что будем делать

| Задача | О чём | Что поймёшь |
|--------|-------|-------------|
| [task01-cpu](./task01-cpu/) | CPU-профиль бенчмарка | `-cpuprofile`, `top`, `list`, flat vs cum, flame graph |
| [task02-memory](./task02-memory/) | Профиль памяти | `-memprofile`, `alloc_space` vs `alloc_objects`, почему `+=` для строк - зло |
| [task03-http-pprof](./task03-http-pprof/) | pprof в живом сервисе | `net/http/pprof`, CPU под нагрузкой, утечка памяти через `inuse_space` |
| [task04-goroutine-leak](./task04-goroutine-leak/) | Утечка горутин | `/debug/pprof/goroutine?debug=1/2`, как выглядит зависшая горутина |
| [task05-contention](./task05-contention/) | Block и mutex профили | почему 8 ядер не дают ускорения и как это увидеть |
| [task06-project](./task06-project/) | Финальный проект | найти несколько проблем самому и доказать ускорение через `benchstat` |

Иди по порядку - каждая следующая опирается на предыдущую. Каждая задача - отдельный Go-модуль со своим `README.md`: шаги, команды, на что смотреть и вопросы.

---

## Что тебе понадобится

- **Go 1.25+** - бенчмарки используют `b.Loop()`.
- **Браузер** - для `go tool pprof -http` (flame graph).
- **graphviz** (по желанию) - для вида Graph и команды `web`: `brew install graphviz`. Flame graph работает и без него.
- **benchstat** - для task06: `go install golang.org/x/perf/cmd/benchstat@latest`.
- **hey** (по желанию) - генератор нагрузки: `brew install hey`. В task03 и task04 есть свои встроенные генераторы, так что можно обойтись без него.

Никакого Docker - только Go.

---

## Как устроены задачи

В каждой задаче:
- **рабочий код с проблемой** - он компилируется, тесты (кроме task04) зелёные, но работает медленно / течёт / ждёт;
- **тесты на корректность** - после оптимизации они должны остаться зелёными. Ускорить, сломав результат, - не считается;
- **бенчмарк** или **генератор нагрузки** - чтобы профилю было что показать;
- **README** - шаги, точные команды, что искать, подсказки (спрятаны под спойлер - не открывай раньше времени) и вопросы на подумать.

Готовых решений нет - только подсказки. Это нарочно: смысл не в исправленном коде, а в навыке **найти** проблему по профилю.

---

## Шпаргалка

```bash
# бенчмарк + профили
go test -run='^$' -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof
go test -run='^$' -bench=. -blockprofile=block.prof -mutexprofile=mutex.prof

# смотреть профиль
go tool pprof cpu.prof                          # консоль: top, top -cum, list <func>, peek, traces
go tool pprof -http=:8090 cpu.prof              # браузер: flame graph, source, graph
go tool pprof -top -nodecount=15 cpu.prof       # без интерактива
go tool pprof -top -cum -focus=MyFunc -relative_percentages cpu.prof  # только то, что внутри MyFunc (спасает от шума на macOS)
go tool pprof -sample_index=alloc_space mem.prof   # alloc_space | alloc_objects | inuse_space | inuse_objects

# живой сервис с import _ "net/http/pprof"
go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=10"   # CPU
go tool pprof http://localhost:6060/debug/pprof/heap                    # память
curl "http://localhost:6060/debug/pprof/goroutine?debug=1"             # горутины, сгруппированные по стеку
curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=3"  # трейс -> go tool trace trace.out

# сравнить два профиля
go tool pprof -top -diff_base=old.prof new.prof

# сравнить бенчмарки до/после
go test -run='^$' -bench=. -benchmem -count=10 > old.txt
# ... правим код ...
go test -run='^$' -bench=. -benchmem -count=10 > new.txt
benchstat old.txt new.txt
```

Внутри консоли pprof:

| Команда | Что делает |
|---------|-----------|
| `top` / `top -cum` | самые дорогие функции по собственному / накопленному времени |
| `list <regexp>` | исходник функции построчно со временем на каждой строке |
| `peek <regexp>` | кто вызывает функцию и кого вызывает она |
| `traces` | все уникальные стеки (удобно для goroutine-профиля) |
| `web` | граф вызовов в браузере (нужен graphviz) |
| `help` | всё остальное |

---

## Как сдавать
1. Делаем форк этого репозитория.
2. Решаем задачи по порядку, на каждую - отдельный коммит.
3. В описании коммита / PR - что нашёл в профиле (топ функций, проценты) и цифры до/после.
4. Создаём Pull Request.
5. Отправляем ссылку на PR в SkillSpace.
