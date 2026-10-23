# Тестирование программного обеспечения

Лабораторные работы на примере VitaLine Clinic.

## Ветки

- `lab1` — лабораторная №1: модульные тесты, сценарии UC/TC, Allure и покрытие.
- `lab2` — лабораторная №2: интеграционные и e2e-тесты, постоянный Docker-стенд и GitHub Actions.

Ветка `lab2` содержит `lab1`, потому что её unit-тесты — первый этап конвейера ЛР2.
Обе ветки содержат приложение `project/VitaLineClinic`, от которого зависят тесты.

## Лабораторная работа №2

Команды выполняются в Ubuntu/WSL с работающим Docker Compose:

```bash
cd lab2
make setup
make up
make test
```

- [Полная инструкция ЛР2](lab2/README.md)
- [Интеграционные тесты](lab2/integration)
- [E2E-тест](lab2/e2e/patient_journey_e2e_test.go)
- [GitHub Actions CI](.github/workflows/lab2.yml)
- [ЛР1 и сценарии UC/TC](lab1/README.md)

`make parallel` проверяет два параллельных прогона.
`LAB2_E2E_FAIL=1 make test` демонстрирует падение e2e и публикацию Allure.

GitHub Actions требует self-hosted Linux x64 runner с меткой `tiopo-lab2`.
Без подключённого runner задания ожидают выполнения. Инструкция настройки находится в README ЛР2.

Сгенерированные отчёты, резервные копии сессий, зависимости инструментов и локальные настройки не включены в Git.
