# Сценарии и тесты ЛР1

Use case (UC) описывает операцию и её результат. Test case (TC) задаёт конкретные входные данные и проверяемое ожидание. Одна операция имеет несколько тестовых случаев: успешный, ошибочный, граничный.

Каждая строка ниже ссылается на существующий Go-тест. Если ссылка указывает на родительский тест, запускаются все его перечисленные в условии подтесты. Полный путь после `Test...` указан в параметре «Go-тест» в Allure. Русское название карточки берётся из условия в таблице.

Запуск одной строки из каталога `lab1`:

```bash
make case CASE=TC-06-04
make show RESULTS=reports/cases/TC-06-04
```

Проверка всех ссылок по свежему полному прогону:

```bash
make test
make check-docs
```

## Как показать связь в Allure

Откройте раздел **«Функциональность»** → нужный **UC** → строку **TC** → карточку теста.
В карточке находятся условие, ожидаемый результат, команда запуска, параметр «Go-тест» и вложение с исходным кодом.

Например: **UC-06 Создание записи на приём → TC-06-04 Время занято**.
Покажите ожидание `ErrSlotBusy`, затем во вложении `usecase/errors_test.go` найдите
`TestAppointmentCreateErrors` и строку `busy_slot`. Проверки требуют нужную ошибку,
пустой DTO и отсутствие сохранения. Зелёный статус означает, что эти ожидания подтвердились.

## Содержание

- [UC-01. Регистрация пациента](#uc-01)
- [UC-02. Поиск пациентов](#uc-02)
- [UC-03. Удаление пациента](#uc-03)
- [UC-04. Создание слота расписания](#uc-04)
- [UC-05. Просмотр и удаление слотов](#uc-05)
- [UC-06. Создание записи на приём](#uc-06)
- [UC-07. Отмена записи](#uc-07)
- [UC-08. Просмотр записей](#uc-08)
- [UC-09. Начало приёма](#uc-09)
- [UC-10. Завершение приёма](#uc-10)
- [UC-11. Просмотр истории приёмов](#uc-11)
- [UC-12. Срочная заявка на свободное время](#uc-12)
- [UC-13. Срочная заявка на занятое время](#uc-13)
- [UC-14. Ответ на предложение переноса](#uc-14)
- [UC-15. Списки срочных заявок и предложений](#uc-15)
- [UC-16. Операции хранилищ](#uc-16)

<a id="uc-01"></a>

## UC-01. Регистрация пациента

Пациент передаёт ФИО, телефон, email и дату рождения. При корректных данных сервис возвращает новый ID и сохраняет пациента.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-01-01 | Корректные данные, мок репозитория | Созданы ID и DTO; на сохранение переданы имя и email | [TestPatientServiceLondon/Register/success](../usecase/patient_test.go) |
| TC-01-02 | Корректные данные, репозиторий в памяти | Пациент читается по выданному ID; email совпадает | [TestPatientServiceClassical/valid](../usecase/patient_test.go) |
| TC-01-03 | Пустые имя, телефон, email, дата рождения; неверные телефон и email | ValidationError нужного поля, пустой DTO, хранилище пустое | [TestPatientValidationErrors](../usecase/errors_test.go) |
| TC-01-04 | Телефон длиной 6, 7, 15 и 16 цифр | 6 и 16 отклоняются с полем Phone; 7 и 15 допускаются | [TestPatientPhoneBoundaries](../domain/domain_test.go) |
| TC-01-05 | Ошибка сохранения пациента | Возвращается исходная ошибка репозитория | [TestPatientServiceLondon/Register/patient.Save](../usecase/patient_test.go) |
| TC-01-06 | Доменная проверка пациента: корректные данные, пустое имя, неверные телефон и email | Корректные данные приняты; неверные возвращают ErrValidation | [TestPatientValidate](../domain/domain_test.go) |
| TC-01-07 | Регистрация с неверным email и репозиторием в памяти | ValidationError поля Email | [TestPatientServiceClassical/invalid](../usecase/patient_test.go) |

<a id="uc-02"></a>

## UC-02. Поиск пациентов

Сервис предоставляет поиск по ID, email и получение списка пациентов. Отсутствие одной записи отличается от пустого списка.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-02-01 | Известный ID | DTO с ожидаемыми ID и email | [TestPatientServiceLondon/GetByID/success](../usecase/patient_test.go) |
| TC-02-02 | Известный email | Найден ожидаемый ID | [TestPatientServiceLondon/FindByEmail/success](../usecase/patient_test.go) |
| TC-02-03 | Получение списка | В списке один подготовленный пациент | [TestPatientServiceLondon/GetAll/success](../usecase/patient_test.go) |
| TC-02-04 | Неизвестный ID в памяти | ErrNotFound | [TestPatientRepositoryContract/FindByIDMissing](../storage/memory/repository_test.go) |
| TC-02-05 | Неизвестный email в памяти | ErrNotFound | [TestPatientRepositoryContract/FindByEmailMissing](../storage/memory/repository_test.go) |
| TC-02-06 | Пустое хранилище | Пустой список без ошибки | [TestPatientRepositoryContract/FindAllMissing](../storage/memory/repository_test.go) |
| TC-02-07 | Репозиторий возвращает обёрнутые ErrNotFound, storageFailure или context.Canceled | errors.Is находит исходную причину; DTO пустой. Здесь проверяется передача ошибки, а не отмена реального запроса | [TestPatientRepositoryErrors](../usecase/errors_test.go) |
| TC-02-08 | Ошибка списка | Исходная ошибка репозитория возвращена сервисом | [TestPatientServiceLondon/GetAll/patient.FindAll](../usecase/patient_test.go) |
| TC-02-09 | Ошибка поиска по email | Исходная ошибка репозитория возвращена сервисом | [TestPatientServiceLondon/FindByEmail/patient.FindByEmail](../usecase/patient_test.go) |
| TC-02-10 | Ошибка поиска пациента по ID в моке | Исходная ошибка репозитория | [TestPatientServiceLondon/GetByID/patient.FindByID](../usecase/patient_test.go) |

<a id="uc-03"></a>

## UC-03. Удаление пациента

Удаление выполняется по ID. Успех подтверждается отсутствием записи при повторном чтении.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-03-01 | Удаление через сервис | Репозиторий получил запрошенный ID | [TestPatientServiceLondon/Delete/success](../usecase/patient_test.go) |
| TC-03-02 | Удаление существующего пациента из памяти | Повторное чтение возвращает ErrNotFound | [TestPatientRepositoryContract/DeletePresent](../storage/memory/repository_test.go) |
| TC-03-03 | Удаление отсутствующего пациента | ErrNotFound | [TestPatientRepositoryContract/DeleteMissing](../storage/memory/repository_test.go) |
| TC-03-04 | Ошибка удаления | Исходная ошибка репозитория возвращена сервисом | [TestPatientServiceLondon/Delete/patient.Delete](../usecase/patient_test.go) |

<a id="uc-04"></a>

## UC-04. Создание слота расписания

Врач, дата, начало, конец и кабинет обязательны. Конец должен быть строго позже начала. Новый слот свободен.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-04-01 | Корректный слот | Сохранён свободный слот с заданными врачом и кабинетом | [TestScheduleServiceLondon/CreateSlot/success](../usecase/schedule_test.go) |
| TC-04-02 | Пустые обязательные поля, равные времена или конец раньше начала | ValidationError с нужным полем; слот не сохранён | [TestScheduleValidationErrors](../usecase/errors_test.go) |
| TC-04-03 | Ошибка сохранения | Исходная ошибка репозитория возвращена сервисом | [TestScheduleServiceLondon/CreateSlot/schedule.Save](../usecase/schedule_test.go) |
| TC-04-04 | Доменная проверка слота: корректные данные, отсутствие врача или времени, равные начало и конец | Корректные данные приняты; неверные возвращают ErrValidation | [TestScheduleValidate](../domain/domain_test.go) |

<a id="uc-05"></a>

## UC-05. Просмотр и удаление слотов

Можно получить расписание врача, выбрать свободные слоты на дату и удалить слот по ID.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-05-01 | Расписание врача | Возвращается подготовленный слот | [TestScheduleServiceLondon/GetByDoctor/success](../usecase/schedule_test.go) |
| TC-05-02 | Подходящий свободный слот, другой врач, другая дата и занятый слот | Ровно один подходящий ID; остальные исключены из результата | [TestScheduleFreeSlotsFilter](../storage/memory/filter_test.go) |
| TC-05-03 | Ошибка получения свободных слотов | Исходная ошибка репозитория | [TestScheduleServiceLondon/GetFreeSlots/schedule.FindFreeSlots](../usecase/schedule_test.go) |
| TC-05-04 | Ошибка получения расписания | Исходная ошибка репозитория | [TestScheduleServiceLondon/GetByDoctor/schedule.FindByDoctorID](../usecase/schedule_test.go) |
| TC-05-05 | Удаление слота | Удаляется запрошенный ID | [TestScheduleServiceLondon/DeleteSlot/success](../usecase/schedule_test.go) |
| TC-05-06 | Ошибка удаления слота | Исходная ошибка репозитория | [TestScheduleServiceLondon/DeleteSlot/schedule.Delete](../usecase/schedule_test.go) |
| TC-05-07 | Сервис получает свободный слот из мока | Возвращён один слот с IsBooked=false | [TestScheduleServiceLondon/GetFreeSlots/success](../usecase/schedule_test.go) |

<a id="uc-06"></a>

## UC-06. Создание записи на приём

Нужны пациент, врач и дата. Прошлая дата, недостаточный интервал до приёма и занятое время отклоняются.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-06-01 | Свободное время и корректные данные | Запись CREATED передана на сохранение с заданными пациентом, врачом и датой | [TestAppointmentServiceLondon/Create/success](../usecase/appointment_test.go) |
| TC-06-02 | Не заданы пациент, врач или дата | ValidationError нужного поля; DTO пустой, записи нет | [TestAppointmentValidationErrors](../usecase/errors_test.go) |
| TC-06-03 | Дата в прошлом | ErrPastDate; DTO пустой, записи нет | [TestAppointmentCreateErrors/past_date](../usecase/errors_test.go) |
| TC-06-04 | Время занято | ErrSlotBusy; DTO пустой, записи нет | [TestAppointmentCreateErrors/busy_slot](../usecase/errors_test.go) |
| TC-06-05 | Приём через час, минимальный интервал 24 часа | ErrAppointmentLeadTime; DTO пустой, записи нет. Это проверка нарушения интервала, не точной границы по часам | [TestAppointmentCreateErrors/insufficient_lead_time](../usecase/errors_test.go) |
| TC-06-06 | Ошибка проверки занятости | storageFailure; DTO пустой, записи нет | [TestAppointmentCreateErrors/storage_failure](../usecase/errors_test.go) |
| TC-06-07 | Ошибка сохранения записи | Исходная ошибка репозитория | [TestAppointmentServiceLondon/Create/appointment.Save](../usecase/appointment_test.go) |
| TC-06-08 | Недопустимый статус доменной записи | ErrValidation | [TestAppointmentValidate](../domain/domain_test.go) |

<a id="uc-07"></a>

## UC-07. Отмена записи

Созданная запись переходит в CANCELLED. Уже отменённую или завершённую запись отменять нельзя.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-07-01 | Отмена через сервис | Сохранён статус CANCELLED | [TestAppointmentServiceLondon/Cancel/success](../usecase/appointment_test.go) |
| TC-07-02 | Доменная запись CREATED, CANCELLED или COMPLETED | CREATED отменяется; для остальных ErrInvalidStatus и прежний статус | [TestAppointmentCancelTransitions](../domain/domain_test.go) |
| TC-07-03 | Ошибка сохранения отмены | Исходная ошибка репозитория | [TestAppointmentServiceLondon/Cancel/appointment.Update](../usecase/appointment_test.go) |
| TC-07-04 | Проверка занятости после отмены в памяти | Отменённая запись не занимает время | [TestAppointmentRepositoryContract/CheckSlotBusyCancelled](../storage/memory/repository_test.go) |

<a id="uc-08"></a>

## UC-08. Просмотр записей

Сервис возвращает запись по ID, список всех записей и записи выбранного пациента.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-08-01 | Поиск по ID | Ожидаемый ID в DTO | [TestAppointmentServiceLondon/GetByID/success](../usecase/appointment_test.go) |
| TC-08-02 | Все записи | Одна подготовленная запись | [TestAppointmentServiceLondon/GetAll/success](../usecase/appointment_test.go) |
| TC-08-03 | Записи пациента | Возвращён заданный PatientID | [TestAppointmentServiceLondon/GetByPatient/success](../usecase/appointment_test.go) |
| TC-08-04 | Ошибка поиска по ID | Исходная ошибка репозитория | [TestAppointmentServiceLondon/GetByID/appointment.FindByID](../usecase/appointment_test.go) |
| TC-08-05 | Ошибка списка | Исходная ошибка репозитория | [TestAppointmentServiceLondon/GetAll/appointment.FindAll](../usecase/appointment_test.go) |
| TC-08-06 | Ошибка списка пациента | Исходная ошибка репозитория | [TestAppointmentServiceLondon/GetByPatient/appointment.FindByPatient](../usecase/appointment_test.go) |

<a id="uc-09"></a>

## UC-09. Начало приёма

Врач начинает визит по записи CREATED и указывает жалобы. Визит становится IN_PROGRESS, а запись — COMPLETED.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-09-01 | Созданная запись и непустые жалобы | Визит сохранён; связанная запись сохранена со статусом COMPLETED | [TestVisitServiceLondon/Create/success](../usecase/visit_test.go) |
| TC-09-02 | Нет ID записи, пустые жалобы, запись CANCELLED или COMPLETED | ValidationError либо ErrInvalidStatus; нет новых сохранений, исходная запись не изменена | [TestVisitCreateErrors](../usecase/errors_test.go) |
| TC-09-03 | Ошибка обновления связанной записи | Сервис возвращает ошибку. Откат уже сохранённого визита этот тест не обещает | [TestVisitServiceLondon/Create/appointment.Update](../usecase/visit_test.go) |
| TC-09-04 | Завершение доменной записи в состояниях CREATED, CANCELLED и COMPLETED | CREATED переходит в COMPLETED; остальные возвращают ErrInvalidStatus и сохраняют прежний статус | [TestAppointmentCompleteTransitions](../domain/domain_test.go) |
| TC-09-05 | Недопустимый статус доменного визита | ErrValidation | [TestVisitValidate](../domain/domain_test.go) |

<a id="uc-10"></a>

## UC-10. Завершение приёма

Для завершения нужны диагноз и рекомендации. Повторное завершение запрещено.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-10-01 | Корректное завершение | Статус COMPLETED, диагноз и рекомендации переданы на сохранение | [TestVisitServiceLondon/Complete/success](../usecase/visit_test.go) |
| TC-10-02 | Визит уже завершён | ErrInvalidStatus; DTO пустой, визит не изменён и повторно не сохранён | [TestVisitCompleteErrors/already_completed](../usecase/errors_test.go) |
| TC-10-03 | Пустой диагноз | ValidationError поля Diagnosis; визит не изменён и не сохранён | [TestVisitCompleteErrors/empty_diagnosis](../usecase/errors_test.go) |
| TC-10-04 | Пустые рекомендации | ValidationError поля Recommendations; визит не изменён и не сохранён | [TestVisitCompleteErrors/empty_recommendations](../usecase/errors_test.go) |
| TC-10-05 | Ошибка сохранения завершённого визита | Исходная ошибка репозитория | [TestVisitServiceLondon/Complete/visit.Update](../usecase/visit_test.go) |
| TC-10-06 | Доменное завершение визита: корректные данные, повторное завершение, пустые диагноз или рекомендации | При успехе сохранены диагноз, рекомендации и COMPLETED; иначе ErrInvalidStatus или ErrValidation | [TestVisitCompleteTransitions](../domain/domain_test.go) |

<a id="uc-11"></a>

## UC-11. Просмотр истории приёмов

Визит можно получить по записи, а историю визитов — по пациенту.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-11-01 | Поиск визита по записи | Ожидаемый AppointmentID в DTO | [TestVisitServiceLondon/GetByAppointment/success](../usecase/visit_test.go) |
| TC-11-02 | История пациента | Один подготовленный визит с нужным AppointmentID | [TestVisitServiceLondon/GetByPatient/success](../usecase/visit_test.go) |
| TC-11-03 | Ошибка поиска визита | Исходная ошибка репозитория | [TestVisitServiceLondon/GetByAppointment/visit.FindByAppointment](../usecase/visit_test.go) |
| TC-11-04 | Ошибка чтения визита при сборке истории | Исходная ошибка возвращена вызывающему коду | [TestVisitServiceLondon/GetByPatient/visit.FindByAppointment](../usecase/visit_test.go) |

<a id="uc-12"></a>

## UC-12. Срочная заявка на свободное время

Если активная запись на время не найдена, создаются срочная заявка SCHEDULED и запись пациента.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-12-01 | Время свободно | Заявка SCHEDULED сохранена; создана запись нужного пациента | [TestEmergencyFreeSlot/success](../usecase/emergency_test.go) |
| TC-12-02 | Ошибка сохранения заявки | Исходная ошибка репозитория | [TestEmergencyFreeSlot/emergency.SaveRequest](../usecase/emergency_test.go) |
| TC-12-03 | Ошибка сохранения записи | Исходная ошибка репозитория; откат ранее сохранённой заявки не заявляется | [TestEmergencyFreeSlot/appointment.Save](../usecase/emergency_test.go) |
| TC-12-04 | Недопустимый статус срочной заявки | ErrValidation | [TestEmergencyRequestValidate](../domain/domain_test.go) |

<a id="uc-13"></a>

## UC-13. Срочная заявка на занятое время

Создаются заявка WAITING_RESPONSE и предложение переноса PENDING. Альтернатива ищется с шагом 30 минут в рабочее время.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-13-01 | Время занято, следующая альтернатива свободна | Сохранены WAITING_RESPONSE и PENDING; предложение через 30 минут | [TestEmergencyServiceLondon/Create/success](../usecase/emergency_test.go) |
| TC-13-02 | Поиск до открытия: 07:00 | Предложение на 09:00 того же дня | [TestEmergencyProposedDateBoundaries/before_open](../usecase/emergency_test.go) |
| TC-13-03 | Исходное время 16:30 | Предложение на 09:00 следующего дня | [TestEmergencyProposedDateBoundaries/closing](../usecase/emergency_test.go) |
| TC-13-04 | Все проверенные альтернативы заняты | ErrValidation; предложение не сохранено | [TestEmergencyProposedDateBoundaries/no_free_slot](../usecase/emergency_test.go) |
| TC-13-05 | Ошибка сохранения предложения | Исходная ошибка репозитория | [TestEmergencyServiceLondon/Create/emergency.SaveOffer](../usecase/emergency_test.go) |

<a id="uc-14"></a>

## UC-14. Ответ на предложение переноса

Ожидающее предложение можно принять или отклонить. Повторно обрабатывать принятое или отклонённое предложение нельзя.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-14-01 | Принять PENDING | Сохранены предложение ACCEPTED и заявка SCHEDULED | [TestEmergencyServiceLondon/AcceptOffer/success](../usecase/emergency_test.go) |
| TC-14-02 | Отклонить PENDING | Сохранены предложение DECLINED и заявка NEEDS_REVIEW | [TestEmergencyServiceLondon/DeclineOffer/success](../usecase/emergency_test.go) |
| TC-14-03 | Принять или отклонить уже ACCEPTED или DECLINED | ErrInvalidStatus; предложение и новая запись не сохранены | [TestEmergencyOfferRejection](../usecase/emergency_test.go) |
| TC-14-04 | Ошибка обновления принимаемого предложения | Исходная ошибка репозитория | [TestEmergencyServiceLondon/AcceptOffer/emergency.UpdateOffer](../usecase/emergency_test.go) |
| TC-14-05 | Ошибка обновления заявки при отклонении | Исходная ошибка репозитория | [TestEmergencyServiceLondon/DeclineOffer/emergency.UpdateRequest](../usecase/emergency_test.go) |
| TC-14-06 | Доменное принятие предложения PENDING, ACCEPTED или DECLINED | PENDING переходит в ACCEPTED; остальные возвращают ErrInvalidStatus и сохраняют статус | [TestRescheduleOfferAcceptTransitions](../domain/domain_test.go) |
| TC-14-07 | Доменное отклонение предложения PENDING, ACCEPTED или DECLINED | PENDING переходит в DECLINED; остальные возвращают ErrInvalidStatus и сохраняют статус | [TestRescheduleOfferDeclineTransitions](../domain/domain_test.go) |
| TC-14-08 | Недопустимый статус предложения переноса | ErrValidation | [TestRescheduleOfferValidate](../domain/domain_test.go) |

<a id="uc-15"></a>

## UC-15. Списки срочных заявок и предложений

Сервис предоставляет списки заявок и предложений для их обработки.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-15-01 | Получить заявки | Подготовленная заявка нужного пациента | [TestEmergencyServiceLondon/GetRequests/success](../usecase/emergency_test.go) |
| TC-15-02 | Получить предложения | Подготовленное предложение PENDING | [TestEmergencyServiceLondon/GetOffers/success](../usecase/emergency_test.go) |
| TC-15-03 | Ошибка чтения заявок | Исходная ошибка репозитория | [TestEmergencyServiceLondon/GetRequests/emergency.FindAllRequests](../usecase/emergency_test.go) |
| TC-15-04 | Ошибка чтения предложений | Исходная ошибка репозитория | [TestEmergencyServiceLondon/GetOffers/emergency.FindAllOffers](../usecase/emergency_test.go) |

<a id="uc-16"></a>

## UC-16. Операции хранилищ

Это технические проверки компонентов, на которых основаны пользовательские сценарии выше. Memory проверяется публичным чтением состояния, MongoDB — подготовленными ответами mock-транспорта.

| Тест-кейс | Условие | Ожидаемый результат | Тест и файл |
|---|---|---|---|
| TC-16-01 | Пациенты в памяти | Сохранение, изменение, удаление, поиск, пустой список и отсутствие записи | [TestPatientRepositoryContract](../storage/memory/repository_test.go) |
| TC-16-02 | Записи в памяти | Сохранение, изменение, удаление, выборки и занятость времени | [TestAppointmentRepositoryContract](../storage/memory/repository_test.go) |
| TC-16-03 | Слоты в памяти | Сохранение, изменение, удаление, выборки по врачу и свободных слотов | [TestScheduleRepositoryContract](../storage/memory/repository_test.go) |
| TC-16-04 | Визиты в памяти | Сохранение, изменение, удаление, поиск по ID и записи | [TestVisitRepositoryContract](../storage/memory/repository_test.go) |
| TC-16-05 | MongoDB: пациенты | Обработка успешных, пустых и ошибочных ответов | [TestPatientMongoRepository](../storage/mongodb/repository_test.go) |
| TC-16-06 | MongoDB: записи | Обработка успешных, пустых и ошибочных ответов, включая занятость | [TestAppointmentMongoRepository](../storage/mongodb/repository_test.go) |
| TC-16-07 | MongoDB: слоты | Обработка успешных, пустых и ошибочных ответов | [TestScheduleMongoRepository](../storage/mongodb/repository_test.go) |
| TC-16-08 | MongoDB: визиты | Обработка успешных, пустых и ошибочных ответов | [TestVisitMongoRepository](../storage/mongodb/repository_test.go) |
| TC-16-09 | MongoDB: срочные заявки | Обработка успешных, пустых и ошибочных ответов | [TestEmergencyRequestMongoRepository](../storage/mongodb/repository_test.go) |
| TC-16-10 | MongoDB: предложения | Обработка успешных, пустых и ошибочных ответов | [TestRescheduleOfferMongoRepository](../storage/mongodb/repository_test.go) |
| TC-16-11 | MongoDB возвращает ошибку команды при сохранении пациента | require.ErrorAs извлекает mongo.CommandError; код равен 123 | [TestPatientMongoRepository/Save/server_error](../storage/mongodb/repository_test.go) |

## Границы подтверждения

- Сценарии относятся к модульным тестам. HTTP, интерфейс пользователя, PostgreSQL, миграции и настоящие соединения с MongoDB здесь не проверяются.
- Отсутствие новых сохранений проверяется в тестах валидации и отклонения статусов. Возврат ошибки после нескольких записей не означает транзакционный откат.
- Проверки моков показывают обработку подготовленного ответа, а не работу MongoDB-сервера.
- Не заявляется полный перебор всех входных данных. В каждой строке указаны именно проверяемые случаи.
- Родительские Go-функции показаны в Allure отдельно как «Группы тестов»; самостоятельные сценарии распределены по UC.

Подготовка к показу и разбор шаблона проверки ошибок: [README](../README.md).
