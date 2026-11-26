# Лабораторная работа №7

Статический анализ VitaLine Clinic (`project/VitaLineClinic`, Go): метрики сложности, code style, проверка типов. Проверки запускаются перед каждым коммитом (git-хук) и в GitHub Actions перед тестами.

## Что проверяется

| Пункт задания | Чем | Где настроено |
| --- | --- | --- |
| 1a. Цикломатическая сложность, максимум 10 | `gocyclo` в golangci-lint (`min-complexity: 10` — ошибка при CC > 10) и собственный анализатор `lab7/cmd/complexity` | `project/VitaLineClinic/.golangci.yml`, `lab7/scripts/check.sh` |
| 1b. Сложность по Холстеду | `lab7/cmd/complexity`: n1, n2, N1, N2, словарь, длина, объём V, трудность D, усилия E, время T, ожидаемые ошибки B, индекс сопровождаемости | `lab7/internal/metrics` |
| 2. Code style | `gofmt`, `goimports`, `revive` (преемник golint), `gocritic`, `misspell`, `nolintlint` | `.golangci.yml` |
| 2. Ошибки и типы | компилятор Go (golangci-lint падает на ошибках типов), `govet`, `staticcheck`, `errcheck`, `errorlint`, `ineffassign`, `unused`, `unconvert`, `bodyclose`, `gosec` | `.golangci.yml` |
| 3. Перед каждым коммитом, обход `--no-verify` | git-хук `pre-commit` | `.githooks/pre-commit`, `lab7/scripts/pre-commit.sh` |
| 4. В CI/CD перед тестами | job `static-analysis`, от него зависит job `test` (`needs`) | `.github/workflows/lab7.yml` |

Code style — стандартный для Go: форматирование только `gofmt`/`goimports`, правила [Effective Go](https://go.dev/doc/effective_go) и [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), которые проверяют `revive` и `staticcheck`. Используется набор линтеров golangci-lint `standard` и стандартные пресеты исключений (`comments`, `std-error-handling`, `common-false-positives`): не требуются doc-комментарии у каждого экспортируемого имени и проверка ошибки `fmt.Fprintln`/`Close` в `defer`.

### Метрики Холстеда

Для Go нет поддерживаемого линтера метрик Холстеда, поэтому анализатор написан в `lab7/internal/metrics` (только стандартная библиотека: `go/ast`, `go/scanner`). Считается для каждой функции:

- **операнды** — идентификаторы и литералы;
- **операторы** — все остальные лексемы: ключевые слова, операции, разделители; парные скобки `() [] {}` считаются один раз, комментарии и автоматические `;` в конце строк не считаются;
- n = n1 + n2, N = N1 + N2, **V = N·log₂n**, **D = n1/2 · N2/n2**, **E = D·V**, T = E/18 с, **B = V/3000**;
- индекс сопровождаемости (вариант Microsoft): `max(0, (171 − 5.2·ln V − 0.23·CC − 16.2·ln LOC)·100/171)`;
- цикломатическая сложность по тем же правилам, что `gocyclo`: 1 + `if`, `for`, `range`, `case` (кроме `default`), `&&`, `||`.

Общепринятых порогов для метрик Холстеда нет, поэтому лимиты выбраны по распределению значений в проекте (V: медиана 369, p90 955, p99 2365; D: медиана 9, p99 23) так, чтобы отсекать выбросы:

| Метрика функции | Лимит | Максимум в проекте |
| --- | ---: | ---: |
| Цикломатическая сложность | 10 | 10 |
| Объём Холстеда V | 3000 | 2946 |
| Трудность Холстеда D | 30 | 26,2 |

Лимиты меняются переменными `MAX_CYCLOMATIC`, `MAX_VOLUME`, `MAX_DIFFICULTY`.

## Исправление кодовой базы

Изначально golangci-lint выдал 57 замечаний: 17 функций с CC > 10, 21 сравнение ошибок через `!=`, 9 замечаний `gosec` и др. Сейчас — 0. Самые сложные функции до и после:

| Функция | CC до | Что сделано | CC после (макс. по частям) |
| --- | ---: | --- | ---: |
| `cli.(*CommandApp).runAppointment` | 26 | Таблица подкоманд `dispatch`, каждая подкоманда — отдельный метод, общие `newFlagSet`, `parseIDFlag`, `printList`, `actionFailed` | 4 |
| `cli.(*CommandApp).runSchedule` | 26 | то же, разбор флагов слота в `slotFromFlags` | 5 |
| `cli.(*CommandApp).runPatient` | 21 | то же | 4 |
| `cli.(*CommandApp).runVisit` | 20 | то же | 4 |
| `appconfig.(Config).ValidateWithOptions` | 19 | Список правил `rule{failed, field, message}` и `firstFailure` | 4 |
| `appconfig.(*Config).applyDefaults` | 16 | Хелпер `defaultString`, `defaultDatabaseURL` | 3 |
| `auth.(*Handler).recover` | 12 | `validEmail`, `startRecovery` | 8 |
| `auth.(*Handler).checkCode` | 11 | `challenge.open`, `challenge.matches`, `wrongCode` | 5 |
| `mongodb.combineMonitors` | 11 | Обобщённая функция `fanOut[E]` | 5 |
| `gui.(*backendAPI).do` | 11 | Сборка запроса вынесена в `newRequest` | 7 |
| 7 тестов `memory`, `postgres`, `appconfig` | 12–27 | Хелперы `must`/`check`/`expect`, табличные проверки, подтесты | ≤ 5 |

По Холстеду выбросами были `bootstrap.New` (V = 5254 → 1999: открытие хранилища в `openStorage`/`openPostgres`/`openMongo`, сборка сервисов в `newUseCases`) и два теста (V ≈ 4900 и 3300: подготовка данных вынесена в `seedTechnicalUI` и `seedForDelete`).

Прочее: `errors.Is` вместо `!=` для ошибок, `ReadHeaderTimeout` у HTTP-серверов API и GUI (защита от Slowloris), права `0600` на файлы в тестах, явное `_ =` для ошибок `Disconnect` при уже возникшей ошибке, `ValidateOptions(opts)` вместо копирования полей. Там, где замечание `gosec` ложное, стоит `//nolint:gosec` с причиной (лог-файлы читаются на хосте через bind mount; SQL собирает squirrel с плейсхолдерами; URL бэкенда из конфига).

Поведение не изменилось: перед рефакторингом CLI добавлен тест `command_test.go`, который прогоняет все подкоманды, включая ошибки; вывод до и после рефакторинга совпал побайтно. Добавлены тесты валидации конфига. Интеграционные тесты PostgreSQL проверены на реальной базе.

## Как запустить

Всё выполняется в WSL (или Linux) из каталога `lab7`, нужен Go ≥ 1.26. golangci-lint v2.15.0 ставится автоматически в `lab7/.tools`.

```bash
cd /mnt/c/ITS/tiopo/lab7
make check            # golangci-lint и метрики сложности для project/VitaLineClinic и lab7
make complexity       # только отчёт по сложности, топ-20 функций
make test             # тесты анализатора и юнит-тесты клиники
make ci               # check, затем test — как в GitHub Actions
make install-hooks    # включить pre-commit хук в этом репозитории
make demo-hook        # демонстрация хука во временном репозитории
```

Отчёты в JSON (все функции со всеми метриками): `lab7/reports/complexity-<модуль>.json`.

## Хук pre-commit

`make install-hooks` выполняет `git config core.hooksPath .githooks`. Хук `.githooks/pre-commit`:

1. выгружает из индекса (`git checkout-index`) модули `project/VitaLineClinic` и `lab7` во временный каталог — проверяется ровно то, что попадёт в коммит, неиндексированные правки не влияют;
2. запускает `lab7/scripts/check.sh` — тот же скрипт, что в CI;
3. при любой ошибке возвращает код 1, и git не создаёт коммит.

Пропустить проверку: `git commit --no-verify`. Если коммит делается из Git for Windows, где нет Go, хук сам запускает проверку в WSL.

`make demo-hook` создаёт временный репозиторий и показывает три шага: чистый код коммитится; функция `Grade` с CC = 11 — коммит отклонён (сообщения gocyclo и анализатора); тот же коммит с `--no-verify` проходит.

## CI

Workflow `Lab 7 - static analysis and tests` (`.github/workflows/lab7.yml`), запускается на push и pull request при изменениях в `project/VitaLineClinic`, `lab7`, `.githooks`:

- job `static-analysis` — `lab7/scripts/check.sh`; таблица метрик пишется в сводку запуска, JSON-отчёты — в артефакт `lab7-complexity-<run>`;
- job `test` (`needs: static-analysis`) — тесты анализатора и `go test -race ./...` клиники вместе с интеграционными тестами на сервисе PostgreSQL 16. Если статический анализ упал, тесты не запускаются.

## Как показать преподавателю

1. `make demo-hook` — хук блокирует коммит с CC = 11, `--no-verify` его пропускает.
2. `make check` — 0 замечаний линтера, максимальная CC = 10, таблица самых объёмных по Холстеду функций.
3. GitHub Actions → `Lab 7 - static analysis and tests`: два последовательных job, сводка с метриками, артефакт с JSON.
4. Рефакторинг: `project/VitaLineClinic/internal/delivery/cli/command*.go`, `internal/appconfig/config.go`, `internal/bootstrap/bootstrap.go`, `internal/auth/http.go`.
