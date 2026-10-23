package gui

import "html/template"

func parseTemplates() *template.Template {
	return template.Must(template.New("gui").Parse(commonTemplate + patientsTemplate + appointmentsTemplate + scheduleTemplate + visitsTemplate + emergenciesTemplate))
}

const commonTemplate = `
{{define "head"}}
<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - {{.AppName}}</title>
  <style>
    :root {
      --bg: #f4f7fb;
      --panel: #ffffff;
      --panel-soft: #f8fafc;
      --text: #172033;
      --muted: #667085;
      --line: #e1e7ef;
      --accent: #12847a;
      --accent-dark: #0a665f;
      --accent-soft: #e7f6f3;
      --ink: #173c43;
      --indigo: #4f63d7;
      --amber: #d99116;
      --danger: #d9483b;
      --danger-soft: #fff1ee;
      --shadow: 0 12px 30px rgba(27, 50, 72, .08);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font-family: Inter, "Segoe UI", Arial, sans-serif;
      font-size: 14px;
      line-height: 1.5;
    }
    a { color: inherit; text-decoration: none; }
    .shell { max-width: 1500px; margin: 0 auto; padding: 24px; }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 20px;
      min-height: 72px;
      margin-bottom: 18px;
      padding: 12px 14px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: rgba(255,255,255,.94);
      box-shadow: var(--shadow);
    }
    .brand {
      display: inline-flex;
      align-items: center;
      gap: 12px;
      font-weight: 800;
    }
    .brand-mark {
      display: grid;
      place-items: center;
      width: 42px;
      height: 42px;
      border-radius: 8px;
      background: var(--accent);
      color: #fff;
      font-weight: 900;
      box-shadow: 0 10px 22px rgba(18, 132, 122, .22);
    }
    .brand small {
      display: block;
      margin-top: 1px;
      color: var(--muted);
      font-weight: 700;
    }
    .nav { display: flex; gap: 6px; flex-wrap: wrap; justify-content: flex-end; }
    .nav a {
      border: 1px solid transparent;
      border-radius: 8px;
      padding: 10px 13px;
      color: #52606f;
      font-weight: 800;
    }
    .nav a:hover { background: var(--panel-soft); border-color: var(--line); }
    .nav a.active { color: #fff; background: var(--ink); border-color: var(--ink); }
    h1 { font-size: 28px; margin: 0; line-height: 1.12; letter-spacing: 0; }
    h2 { font-size: 15px; margin: 0 0 16px; letter-spacing: 0; }
    .page-hero {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: 22px;
      align-items: center;
      margin: 0 0 18px;
      padding: 24px 28px;
      border-radius: 8px;
      background: var(--ink);
      color: #fff;
      box-shadow: var(--shadow);
    }
    .eyebrow {
      margin: 0 0 8px;
      color: #9fe3d9;
      font-weight: 900;
      letter-spacing: 0;
    }
    .lead { max-width: 680px; margin: 10px 0 0; color: #d7e6e8; }
    .metrics { display: grid; grid-template-columns: repeat(2, 136px); gap: 12px; }
    .metric {
      min-height: 82px;
      padding: 14px;
      border: 1px solid rgba(255,255,255,.18);
      border-radius: 8px;
      background: rgba(255,255,255,.1);
    }
    .metric strong { display: block; font-size: 28px; line-height: 1.05; color: #fff; }
    .metric span { display: block; margin-top: 8px; color: #d7e6e8; font-weight: 800; }
    .layout { display: grid; grid-template-columns: minmax(0, 1fr) 420px; gap: 18px; align-items: start; }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 18px;
      box-shadow: var(--shadow);
    }
    .panel h2 { display: flex; align-items: center; justify-content: space-between; }
    .stack { display: grid; gap: 14px; }
    .message { border-radius: 8px; padding: 12px 14px; margin: 0 0 16px; border: 1px solid var(--line); background: var(--panel); font-weight: 800; }
    .message.error { border-color: #f1b2a8; color: var(--danger); background: var(--danger-soft); }
    .message.notice { border-color: #addfd8; color: var(--accent-dark); background: var(--accent-soft); }
    form { display: grid; gap: 12px; }
    label { display: grid; gap: 6px; color: #52606f; font-weight: 700; }
    input, select {
      width: 100%;
      border: 1px solid #d7e0ea;
      border-radius: 8px;
      padding: 11px 12px;
      background: #fff;
      color: var(--text);
      font: inherit;
      box-shadow: inset 0 1px 0 rgba(23,32,51,.02);
    }
    input:focus, select:focus {
      outline: 3px solid rgba(18, 132, 122, .14);
      border-color: var(--accent);
    }
    button {
      border: 1px solid var(--accent);
      background: var(--accent);
      color: #fff;
      border-radius: 8px;
      padding: 11px 13px;
      font: inherit;
      font-weight: 800;
      cursor: pointer;
      box-shadow: 0 8px 18px rgba(18, 132, 122, .18);
    }
    button:hover { background: var(--accent-dark); border-color: var(--accent-dark); }
    button.secondary { color: var(--ink); background: #eef3ff; border-color: #d9e0ff; box-shadow: none; }
    button.secondary:hover { color: #fff; background: var(--indigo); border-color: var(--indigo); }
    button.danger { background: var(--danger); border-color: var(--danger); box-shadow: none; }
    button.danger:hover { background: #b73528; border-color: #b73528; }
    .actions { display: flex; gap: 12px; flex-wrap: wrap; align-items: end; margin-bottom: 14px; }
    .actions label { min-width: 220px; flex: 1; }
    .search-row {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      align-items: end;
      margin-bottom: 14px;
    }
    .search-row label { min-width: 260px; flex: 1; }
    .clear-link {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-height: 44px;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 10px 13px;
      color: var(--muted);
      background: var(--panel-soft);
      font-weight: 800;
    }
    .clear-link:hover { color: var(--ink); border-color: #c8d3df; }
    table { width: 100%; border-collapse: separate; border-spacing: 0; background: var(--panel); }
    th, td { padding: 14px 13px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: middle; }
    th { color: #667085; font-weight: 900; background: #f8fafc; }
    th:first-child { border-top-left-radius: 8px; }
    th:last-child { border-top-right-radius: 8px; }
    tbody tr:hover td { background: #fbfcff; }
    tbody tr:last-child td { border-bottom: 0; }
    .empty {
      color: var(--muted);
      padding: 30px 24px;
      border: 1px dashed #cbd6e2;
      border-radius: 8px;
      background: var(--panel-soft);
      text-align: center;
      font-weight: 800;
    }
    .id { font-family: Consolas, "SFMono-Regular", monospace; word-break: break-all; color: #405060; }
    .badge { display: inline-flex; align-items: center; border-radius: 999px; padding: 5px 10px; border: 1px solid var(--line); color: var(--muted); background: var(--panel-soft); font-weight: 900; white-space: nowrap; }
    .badge.accent { color: var(--accent-dark); border-color: #addfd8; background: var(--accent-soft); }
    .badge.danger { color: var(--danger); border-color: #f1b2a8; background: var(--danger-soft); }
    .badge.muted { color: #596579; border-color: #d8dee8; background: #f3f5f8; }
    .compact { max-width: 190px; }
    .person { display: flex; align-items: center; gap: 11px; }
    .avatar {
      flex: 0 0 auto;
      display: grid;
      place-items: center;
      width: 34px;
      height: 34px;
      border-radius: 8px;
      background: #eef3ff;
      color: var(--indigo);
      font-weight: 900;
    }
    .primary-text { display: block; font-weight: 900; color: var(--text); }
    .subline { display: block; color: var(--muted); margin-top: 4px; }
    .micro-id { display: block; max-width: 220px; margin-top: 4px; font-size: 12px; color: #7a8797; word-break: break-all; }
    .doctor-list { display: grid; gap: 10px; }
    .doctor-card {
      display: grid;
      gap: 5px;
      padding: 12px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: var(--panel-soft);
    }
    .doctor-card strong { color: var(--text); }
    .doctor-card span { color: var(--muted); font-weight: 700; }
    .doctor-card .id { font-size: 12px; }
    .table-wrap { overflow-x: auto; border: 1px solid var(--line); border-radius: 8px; }
    .table-wrap table { min-width: 860px; }
    @media (max-width: 980px) {
      .shell { padding: 14px; }
      .topbar { align-items: flex-start; flex-direction: column; }
      .nav { justify-content: flex-start; }
      .page-hero { grid-template-columns: 1fr; padding: 20px; }
      .metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .layout { grid-template-columns: 1fr; }
      table { display: table; overflow: visible; }
    }
  </style>
</head>
<body>
<div class="shell">
{{end}}

{{define "nav"}}
  <header class="topbar">
    <a class="brand" href="/patients">
      <span class="brand-mark">V</span>
      <span>{{.AppName}}<small>medical workspace</small></span>
    </a>
    <nav class="nav">
      <a class="{{if eq .Active "patients"}}active{{end}}" href="/patients">Пациенты</a>
      <a class="{{if eq .Active "appointments"}}active{{end}}" href="/appointments">Записи</a>
      <a class="{{if eq .Active "schedule"}}active{{end}}" href="/schedule">Расписание</a>
      <a class="{{if eq .Active "visits"}}active{{end}}" href="/visits">Приёмы</a>
      <a class="{{if eq .Active "emergencies"}}active{{end}}" href="/emergencies">Срочные</a>
    </nav>
  </header>
{{end}}

{{define "flash"}}
  {{if .Notice}}<div class="message notice">{{.Notice}}</div>{{end}}
  {{if .Error}}<div class="message error">{{.Error}}</div>{{end}}
{{end}}

{{define "footer"}}
</div>
</body>
</html>
{{end}}
`

const patientsTemplate = `
{{define "patients"}}
{{template "head" .}}
{{template "nav" .}}
<section class="page-hero">
  <div>
    <p class="eyebrow">Пациентская база</p>
    <h1>Пациенты</h1>
    <p class="lead">Регистрация, поиск и контроль карточек пациентов в одном рабочем окне.</p>
  </div>
  <div class="metrics">
    <div class="metric"><strong>{{len .Patients}}</strong><span>карточек</span></div>
    <div class="metric"><strong>{{len .Doctors}}</strong><span>врачей</span></div>
  </div>
</section>
{{template "flash" .}}
<div class="layout">
  <section class="panel">
    <h2>Список пациентов</h2>
    <form class="search-row" method="get" action="/patients">
      <label>Поиск пациента
        <input name="q" value="{{.PatientSearch}}" placeholder="ФИО, телефон, email или ID">
      </label>
      <button class="secondary" type="submit">Найти</button>
      <a class="clear-link" href="/patients">Сбросить</a>
    </form>
    {{if .Patients}}
    <div class="table-wrap">
    <table>
      <thead>
        <tr><th>ФИО</th><th>Контакты</th><th>Дата рождения</th><th>ID</th><th></th></tr>
      </thead>
      <tbody>
        {{range .Patients}}
        <tr>
          <td><span class="person"><span class="avatar">+</span><span><span class="primary-text">{{.FullName}}</span><span class="subline">создан: {{.CreatedAt}}</span></span></span></td>
          <td><span class="primary-text">{{.Phone}}</span><span class="subline">{{.Email}}</span></td>
          <td>{{.BirthDate}}</td>
          <td class="id compact">{{.ID}}</td>
          <td>
            <form method="post" action="/patients/delete">
              <input type="hidden" name="id" value="{{.ID}}">
              <button class="danger" type="submit">Удалить</button>
            </form>
          </td>
        </tr>
        {{end}}
      </tbody>
    </table>
    </div>
    {{else}}
    <div class="empty">Пациенты не найдены</div>
    {{end}}
  </section>
  <aside class="stack">
    <section class="panel">
      <h2>Регистрация пациента</h2>
      <form method="post" action="/patients/register">
        <label>ФИО <input name="full-name" required></label>
        <label>Телефон <input name="phone" required placeholder="+79001234567"></label>
        <label>Email <input name="email" type="email" required></label>
        <label>Дата рождения <input name="birth-date" type="date" required></label>
        <button type="submit">Зарегистрировать</button>
      </form>
    </section>
    <section class="panel">
      <h2>Быстрый поиск</h2>
      <form method="get" action="/patients">
        <label>Пациент <input name="q" value="{{.PatientSearch}}" placeholder="Смирнова, +7900 или email"></label>
        <button class="secondary" type="submit">Найти</button>
      </form>
    </section>
  </aside>
</div>
{{template "footer" .}}
{{end}}
`

const appointmentsTemplate = `
{{define "appointments"}}
{{template "head" .}}
{{template "nav" .}}
<section class="page-hero">
  <div>
    <p class="eyebrow">Журнал записи</p>
    <h1>Записи на приём</h1>
    <p class="lead">Создание, просмотр и отмена записей без перехода в техническую консоль.</p>
  </div>
  <div class="metrics">
    <div class="metric"><strong>{{len .Appointments}}</strong><span>записей</span></div>
    <div class="metric"><strong>{{len .Patients}}</strong><span>пациентов</span></div>
  </div>
</section>
{{template "flash" .}}
<div class="layout">
  <section class="panel">
    <h2>Текущие записи</h2>
    <form class="search-row" method="get" action="/appointments">
      <label>Пациент
        <input name="patient-q" value="{{.PatientSearch}}" placeholder="ФИО, email или ID">
      </label>
      <label>Врач
        <input name="doctor-q" value="{{.DoctorSearch}}" placeholder="ФИО, специальность или кабинет">
      </label>
      <button class="secondary" type="submit">Фильтровать</button>
      <a class="clear-link" href="/appointments">Сбросить</a>
    </form>
    {{if .Appointments}}
    <div class="table-wrap">
    <table>
      <thead>
        <tr><th>Дата</th><th>Статус</th><th>Пациент</th><th>Врач</th><th>ID</th><th></th></tr>
      </thead>
      <tbody>
        {{range .Appointments}}
        <tr>
          <td><span class="primary-text">{{.Date}}</span></td>
          <td><span class="badge {{.Badge}}">{{.Status}}</span></td>
          <td>
            <span class="primary-text">{{if .PatientName}}{{.PatientName}}{{else}}Пациент{{end}}</span>
            <span class="subline">{{.PatientEmail}}</span>
            <span class="micro-id">{{.PatientID}}</span>
          </td>
          <td>
            <span class="primary-text">{{if .DoctorName}}{{.DoctorName}}{{else}}Врач{{end}}</span>
            <span class="subline">{{.DoctorSpecialty}}</span>
            <span class="micro-id">{{.DoctorID}}</span>
          </td>
          <td class="id compact">{{.ID}}</td>
          <td>
            <form method="post" action="/appointments/cancel">
              <input type="hidden" name="id" value="{{.ID}}">
              <button class="danger" type="submit">Отменить</button>
            </form>
          </td>
        </tr>
        {{end}}
      </tbody>
    </table>
    </div>
    {{else}}
    <div class="empty">Записей нет</div>
    {{end}}
  </section>
  <aside class="stack">
    <section class="panel">
      <h2>Новая запись</h2>
      <form method="post" action="/appointments/create">
        <label>Пациент
          <select name="patient-id" required>
            {{range .Patients}}<option value="{{.ID}}">{{.FullName}} - {{.Email}}</option>{{end}}
          </select>
        </label>
        <label>Врач
          <select name="doctor-id" required>
            {{range .DoctorOptions}}<option value="{{.ID}}" {{if eq $.DoctorID .ID}}selected{{end}}>{{.FullName}} - {{.Specialty}}, каб. {{.Cabinet}}</option>{{end}}
          </select>
        </label>
        <label>Дата и время <input name="date-time" type="datetime-local" required value="{{.DateTime}}"></label>
        <button type="submit">Создать запись</button>
      </form>
    </section>
    <section class="panel">
      <h2>Врачи</h2>
      <form method="get" action="/appointments">
        <label>Поиск врача <input name="doctor-q" value="{{.DoctorSearch}}" placeholder="Кардиолог, Павлова или 202"></label>
        <button class="secondary" type="submit">Найти</button>
      </form>
      <div class="doctor-list">
        {{range .Doctors}}
        <div class="doctor-card">
          <strong>{{.FullName}}</strong>
          <span>{{.Specialty}} · кабинет {{.Cabinet}}</span>
          <span class="id">{{.ID}}</span>
        </div>
        {{end}}
      </div>
    </section>
  </aside>
</div>
{{template "footer" .}}
{{end}}
`

const scheduleTemplate = `
{{define "schedule"}}
{{template "head" .}}
{{template "nav" .}}
<section class="page-hero">
  <div>
    <p class="eyebrow">Календарь врача</p>
    <h1>Расписание</h1>
    <p class="lead">Слоты, кабинеты и свободное время на выбранную дату без перегруженного календаря.</p>
  </div>
  <div class="metrics">
    <div class="metric"><strong>{{len .Slots}}</strong><span>слотов</span></div>
    <div class="metric"><strong>{{len .FreeSlots}}</strong><span>свободно</span></div>
  </div>
</section>
{{template "flash" .}}
<div class="layout">
  <section class="stack">
    <div class="panel">
      <h2>Слоты врача</h2>
      <form class="actions" method="get" action="/schedule">
        <label>Врач
          <select name="doctor-id" required>
            {{range .DoctorOptions}}<option value="{{.ID}}" {{if eq $.DoctorID .ID}}selected{{end}}>{{.FullName}} - {{.Specialty}}, каб. {{.Cabinet}}</option>{{end}}
          </select>
        </label>
        <label>Поиск врача <input name="doctor-q" value="{{.DoctorSearch}}" placeholder="ФИО, специальность или кабинет"></label>
        <label>Дата <input name="date" type="date" value="{{.Date}}"></label>
        <button class="secondary" type="submit">Показать</button>
        <a class="clear-link" href="/schedule">Сбросить</a>
      </form>
      {{if .Slots}}
      <div class="table-wrap">
      <table>
        <thead>
          <tr><th>Дата</th><th>Время</th><th>Кабинет</th><th>Статус</th><th>ID</th><th></th></tr>
        </thead>
        <tbody>
          {{range .Slots}}
          <tr>
            <td>{{.Date}}</td>
            <td><span class="primary-text">{{.StartTime}} - {{.EndTime}}</span></td>
            <td><span class="badge">{{.Cabinet}}</span><span class="subline">{{.DoctorName}}</span></td>
            <td><span class="badge {{.Badge}}">{{if .IsBooked}}Занят{{else}}Свободен{{end}}</span></td>
            <td class="id compact">{{.ID}}</td>
            <td>
              <form method="post" action="/schedule/delete">
                <input type="hidden" name="id" value="{{.ID}}">
                <button class="danger" type="submit">Удалить</button>
              </form>
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
      </div>
      {{else}}
      <div class="empty">Для выбранного врача слотов нет</div>
      {{end}}
    </div>
    <div class="panel">
      <h2>Свободные слоты на дату</h2>
      {{if .FreeSlots}}
      <div class="table-wrap">
      <table>
        <thead><tr><th>Время</th><th>Кабинет</th><th>ID</th></tr></thead>
        <tbody>
          {{range .FreeSlots}}<tr><td><span class="primary-text">{{.StartTime}} - {{.EndTime}}</span></td><td><span class="badge">{{.Cabinet}}</span></td><td class="id">{{.ID}}</td></tr>{{end}}
        </tbody>
      </table>
      </div>
      {{else}}
      <div class="empty">Свободные слоты не найдены</div>
      {{end}}
    </div>
  </section>
  <aside class="stack">
    <section class="panel">
      <h2>Добавить слот</h2>
      <form method="post" action="/schedule/create">
        <label>Врач
          <select name="doctor-id" required>
            {{range .DoctorOptions}}<option value="{{.ID}}" {{if eq $.DoctorID .ID}}selected{{end}}>{{.FullName}} - {{.Specialty}}, каб. {{.Cabinet}}</option>{{end}}
          </select>
        </label>
        <label>Дата <input name="date" type="date" required value="{{.Date}}"></label>
        <label>Начало <input name="start" type="time" required value="09:00"></label>
        <label>Окончание <input name="end" type="time" required value="09:30"></label>
        <label>Кабинет <input name="cabinet" required value="101"></label>
        <button type="submit">Добавить слот</button>
      </form>
    </section>
    <section class="panel">
      <h2>Врачи</h2>
      <div class="doctor-list">
        {{range .Doctors}}
        <a class="doctor-card" href="/schedule?doctor-id={{.ID}}&date={{$.Date}}">
          <strong>{{.FullName}}</strong>
          <span>{{.Specialty}} · кабинет {{.Cabinet}}</span>
          <span class="id">{{.ID}}</span>
        </a>
        {{end}}
      </div>
    </section>
  </aside>
</div>
{{template "footer" .}}
{{end}}
`

const visitsTemplate = `
{{define "visits"}}
{{template "head" .}}
{{template "nav" .}}
<section class="page-hero">
  <div>
    <p class="eyebrow">Медицинские приёмы</p>
    <h1>Приёмы</h1>
    <p class="lead">Запуск и завершение визита с фиксацией жалоб, диагноза и рекомендаций.</p>
  </div>
  <div class="metrics">
    <div class="metric"><strong>{{len .Visits}}</strong><span>визитов</span></div>
    <div class="metric"><strong>{{len .Appointments}}</strong><span>записей</span></div>
  </div>
</section>
{{template "flash" .}}
<div class="layout">
  <section class="panel">
    <h2>Приёмы пациента</h2>
    <form class="actions" method="get" action="/visits">
      <label>ID пациента <input name="patient-id" required value="{{.PatientID}}"></label>
      <button class="secondary" type="submit">Показать</button>
    </form>
    {{if .Visits}}
    <div class="table-wrap">
    <table>
      <thead>
        <tr><th>Статус</th><th>Жалобы</th><th>Диагноз</th><th>Рекомендации</th><th>ID</th></tr>
      </thead>
      <tbody>
        {{range .Visits}}
        <tr>
          <td><span class="badge {{.Badge}}">{{.Status}}</span></td>
          <td>{{.Complaints}}</td>
          <td>{{.Diagnosis}}</td>
          <td>{{.Recommendations}}</td>
          <td class="id compact">{{.ID}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    </div>
    {{else}}
    <div class="empty">Приёмы для пациента не найдены</div>
    {{end}}
  </section>
  <aside class="stack">
    <section class="panel">
      <h2>Начать приём</h2>
      <form method="post" action="/visits/start">
        <label>Запись
          <select name="appointment-id" required>
            {{range .Appointments}}<option value="{{.ID}}">{{.Date}} - {{.Status}}</option>{{end}}
          </select>
        </label>
        <label>Жалобы <input name="complaints" required></label>
        <button type="submit">Начать</button>
      </form>
    </section>
    <section class="panel">
      <h2>Завершить приём</h2>
      <form method="post" action="/visits/complete">
        <label>ID приёма <input name="visit-id" required></label>
        <label>Диагноз <input name="diagnosis" required></label>
        <label>Рекомендации <input name="recommendations" required></label>
        <button type="submit">Завершить</button>
      </form>
    </section>
  </aside>
</div>
{{template "footer" .}}
{{end}}
`

const emergenciesTemplate = `
{{define "emergencies"}}
{{template "head" .}}
{{template "nav" .}}
<section class="page-hero">
  <div>
    <p class="eyebrow">Вертикальный срез</p>
    <h1>Срочные записи</h1>
    <p class="lead">Создание срочной заявки, предложение переноса занятого слота и фиксация решения пациента.</p>
  </div>
  <div class="metrics">
    <div class="metric"><strong>{{len .EmergencyRequests}}</strong><span>заявок</span></div>
    <div class="metric"><strong>{{len .RescheduleOffers}}</strong><span>переносов</span></div>
  </div>
</section>
{{template "flash" .}}
<div class="layout">
  <section class="stack">
    <div class="panel">
      <h2>Срочные заявки</h2>
      {{if .EmergencyRequests}}
      <div class="table-wrap">
      <table>
        <thead>
          <tr><th>Время</th><th>Статус</th><th>Пациент</th><th>Врач</th><th>Причина</th><th>ID</th></tr>
        </thead>
        <tbody>
          {{range .EmergencyRequests}}
          <tr>
            <td><span class="primary-text">{{.RequestedDate}}</span><span class="subline">создана: {{.CreatedAt}}</span></td>
            <td><span class="badge {{.Badge}}">{{.Status}}</span></td>
            <td>
              <span class="primary-text">{{if .PatientName}}{{.PatientName}}{{else}}Пациент{{end}}</span>
              <span class="subline">{{.PatientEmail}}</span>
              <span class="micro-id">{{.PatientID}}</span>
            </td>
            <td>
              <span class="primary-text">{{if .DoctorName}}{{.DoctorName}}{{else}}Врач{{end}}</span>
              <span class="subline">{{.DoctorSpecialty}}</span>
              <span class="micro-id">{{.DoctorID}}</span>
            </td>
            <td>{{.Reason}}</td>
            <td class="id compact">{{.ID}}</td>
          </tr>
          {{end}}
        </tbody>
      </table>
      </div>
      {{else}}
      <div class="empty">Срочных заявок пока нет</div>
      {{end}}
    </div>

    <div class="panel">
      <h2>Предложения переноса</h2>
      {{if .RescheduleOffers}}
      <div class="table-wrap">
      <table>
        <thead>
          <tr><th>Статус</th><th>Пациент</th><th>Врач</th><th>Было</th><th>Предложено</th><th></th></tr>
        </thead>
        <tbody>
          {{range .RescheduleOffers}}
          <tr>
            <td><span class="badge {{.Badge}}">{{.Status}}</span><span class="micro-id">{{.ID}}</span></td>
            <td>
              <span class="primary-text">{{if .PatientName}}{{.PatientName}}{{else}}Пациент{{end}}</span>
              <span class="subline">{{.PatientEmail}}</span>
            </td>
            <td>
              <span class="primary-text">{{if .DoctorName}}{{.DoctorName}}{{else}}Врач{{end}}</span>
              <span class="subline">{{.DoctorSpecialty}}</span>
            </td>
            <td>{{.OldDate}}</td>
            <td>{{.ProposedDate}}</td>
            <td>
              {{if eq .Status "PENDING"}}
              <form method="post" action="/emergencies/offers/accept">
                <input type="hidden" name="id" value="{{.ID}}">
                <button type="submit">Принять</button>
              </form>
              <form method="post" action="/emergencies/offers/decline">
                <input type="hidden" name="id" value="{{.ID}}">
                <button class="danger" type="submit">Отклонить</button>
              </form>
              {{else}}
              <span class="badge {{.Badge}}">{{.Status}}</span>
              {{end}}
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
      </div>
      {{else}}
      <div class="empty">Предложений переноса пока нет</div>
      {{end}}
    </div>
  </section>

  <aside class="stack">
    <section class="panel">
      <h2>Новая срочная запись</h2>
      <form method="post" action="/emergencies/create">
        <label>Пациент
          <select name="patient-id" required>
            {{range .Patients}}<option value="{{.ID}}">{{.FullName}} - {{.Email}}</option>{{end}}
          </select>
        </label>
        <label>Врач
          <select name="doctor-id" required>
            {{range .DoctorOptions}}<option value="{{.ID}}" {{if eq $.DoctorID .ID}}selected{{end}}>{{.FullName}} - {{.Specialty}}, каб. {{.Cabinet}}</option>{{end}}
          </select>
        </label>
        <label>Желаемое время <input name="requested-date" type="datetime-local" required value="{{.DateTime}}"></label>
        <label>Причина <input name="reason" required placeholder="Острая боль, травма, экстренный осмотр"></label>
        <button type="submit">Создать срочную заявку</button>
      </form>
    </section>
    <section class="panel">
      <h2>Контроль слота</h2>
      <div class="doctor-list">
        <div class="doctor-card">
          <strong>Занятый слот для демо</strong>
          <span>Павлова Елена Сергеевна · завтра 09:00</span>
          <span>Существующая запись занимает время, срочная заявка формирует перенос.</span>
        </div>
        <div class="doctor-card">
          <strong>Свободный слот</strong>
          <span>При отсутствии конфликта срочный пациент получает запись сразу.</span>
        </div>
      </div>
    </section>
  </aside>
</div>
{{template "footer" .}}
{{end}}
`
