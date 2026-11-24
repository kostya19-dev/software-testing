// Сценарий нагрузки ЛР4 для k6: виртуальный пациент регистрируется, входит в профиль, смотрит
// расписание, записывается к врачу, приходит на приём или отменяет запись и смотрит историю
// визитов, между шагами думает 0,3–1 с. Профиль числа VU (ramp, stress, stable) берётся из
// scenarios.json, как в генераторе ЛР4: SCENARIO — имя сценария, TIME_SCALE — масштаб времени.
//
// Каждый запрос помечается тегами op (операция) и invalid (намеренно ошибочная операция),
// ожидаемый код ответа задаётся через responseCallback и попадает в тег expected_response.
// После прогона `lab6 after-load` переводит CSV k6 в формат отчёта ЛР4 (load/requests.csv.gz).

import http from 'k6/http';
import { sleep } from 'k6';
import exec from 'k6/execution';

const catalog = JSON.parse(open('/config/scenarios.json'));
const scenarioName = __ENV.SCENARIO;
const scale = parseFloat(__ENV.TIME_SCALE || '1');
const plan = catalog.scenarios[scenarioName];
if (!plan) {
  throw new Error(`неизвестный сценарий ${scenarioName}, есть: ${Object.keys(catalog.scenarios).join(', ')}`);
}

const api = __ENV.API_URL || 'http://api:8080';
const errorRate = plan.error_rate || 0;

const seededDoctor = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';
const unknownId = '00000000-0000-4000-8000-0000000dead0';
const slotLengthMs = 30 * 60 * 1000;
const cancelShare = 0.3;
const visitsPerPatient = 5;

// --- профиль нагрузки -------------------------------------------------------

function scaledSeconds(seconds) {
  return Math.max(1, Math.floor(seconds * scale + 0.5));
}

function phases(s) {
  const out = [];
  const add = (seconds, vus) => out.push({ seconds: scaledSeconds(seconds), vus });
  switch (s.kind) {
    case 'steps':
      for (const level of s.levels) {
        add(s.ramp_s, level);
        add(s.hold_s, level);
      }
      break;
    case 'spike':
      add(s.ramp_s, s.baseline_vus);
      add(s.warmup_s, s.baseline_vus);
      add(s.ramp_s, s.peak_vus);
      add(s.overload_s, s.peak_vus);
      add(s.ramp_s, s.baseline_vus);
      add(s.recovery_s, s.baseline_vus);
      break;
    case 'constant': {
      const ramp = Math.min(s.ramp_s || 0, s.duration_s);
      const warmup = Math.min(s.warmup_s || 0, s.duration_s - ramp);
      if (ramp > 0) {
        add(ramp, s.vus);
      }
      add(warmup, s.vus);
      add(s.duration_s - ramp - warmup, s.vus);
      break;
    }
    default:
      throw new Error(`неизвестный вид сценария ${s.kind}`);
  }
  return out;
}

function stages(s) {
  const out = [];
  if (s.kind === 'constant' && !s.ramp_s) {
    out.push({ duration: '0s', target: s.vus });
  }
  for (const phase of phases(s)) {
    out.push({ duration: `${phase.seconds}s`, target: phase.vus });
  }
  return out;
}

export const options = {
  scenarios: {
    [scenarioName]: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: stages(plan),
      // Как в генераторе ЛР4: снятый VU не начинает новых шагов, а прогон не продолжается после профиля.
      gracefulRampDown: '1s',
      gracefulStop: '1s',
    },
  },
  // Тела ответов нужны только там, где из них берётся ID созданной сущности.
  discardResponseBodies: true,
  systemTags: ['status', 'expected_response'],
  summaryTrendStats: ['avg', 'p(50)', 'p(75)', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

// --- HTTP ------------------------------------------------------------------

function send(op, method, path, body, expect, { invalid = false, wantBody = false } = {}) {
  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { op, invalid: invalid ? '1' : '0' },
    responseCallback: http.expectedStatuses(expect),
    timeout: '10s',
  };
  if (wantBody) {
    params.responseType = 'text';
  }
  return http.request(method, api + path, body === undefined ? null : JSON.stringify(body), params);
}

function succeeded(op, method, path, body, expect) {
  return send(op, method, path, body, expect).status === expect;
}

function created(op, method, path, body) {
  const response = send(op, method, path, body, 201, { wantBody: true });
  if (response.status !== 201) {
    return '';
  }
  try {
    const entity = JSON.parse(response.body);
    return entity.ID || entity.id || '';
  } catch (e) {
    return '';
  }
}

function wrong(op, method, path, body, expect) {
  send(op, method, path, body, expect, { invalid: true });
}

// --- пациент: состояние одного VU живёт между итерациями ---------------------

const firstDay = new Date(Date.UTC(new Date().getUTCFullYear(), new Date().getUTCMonth(), new Date().getUTCDate() + 1));
let patientId = '';
let visitsLeft = 0;
let slot = 0;

function doctorId() {
  return 'd0c70000-0000-4000-8000-' + String(exec.vu.idInTest).padStart(12, '0');
}

function between(min, max) {
  return min + Math.random() * (max - min);
}

function think() {
  sleep(between(0.3, 1.0));
}

function chance(probability) {
  return Math.random() < probability;
}

function randomText() {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let text = '';
  for (let i = 0; i < 26; i++) {
    text += alphabet[Math.floor(Math.random() * alphabet.length)];
  }
  return text;
}

function nextSlot() {
  const at = new Date(firstDay.getTime() + slot * slotLengthMs);
  slot++;
  return at.toISOString();
}

function day(date) {
  return date.toISOString().slice(0, 10);
}

const birthDate = '1990-01-01T00:00:00Z';

// Внештатные действия пользователя (ЛР4, п. 4): такие же запросы, но с ошибкой в данных.
const mistakes = {
  register: () => wrong('register', 'POST', '/api/patients',
    { full_name: 'Нагрузочный Пациент', phone: '+79001234567', email: 'не-адрес', birth_date: birthDate }, 400),
  login: () => wrong('login', 'GET', `/api/patients/${unknownId}`, undefined, 404),
  browse_schedule: () => wrong('browse_schedule', 'GET', `/api/schedule/free?doctor_id=${doctorId()}&date=`, undefined, 400),
  book_appointment: () => wrong('book_appointment', 'POST', '/api/appointments',
    { patient_id: patientId, doctor_id: doctorId(), date: new Date(Date.now() - 24 * 3600 * 1000).toISOString() }, 409),
  my_appointments: () => wrong('my_appointments', 'POST', `/api/appointments/${unknownId}/cancel`, undefined, 404),
  cancel_appointment: () => wrong('cancel_appointment', 'POST', `/api/appointments/${unknownId}/cancel`, undefined, 404),
  start_visit: () => wrong('start_visit', 'POST', '/api/visits', { appointment_id: unknownId, complaints: 'Ошибка ввода' }, 404),
  complete_visit: () => wrong('complete_visit', 'POST', `/api/visits/${unknownId}/complete`,
    { diagnosis: 'Ошибка ввода', recommendations: 'Нет' }, 404),
  visit_history: () => wrong('visit_history', 'GET', '/api/visits?patient_id=', undefined, 400),
};

function mistaken(op) {
  if (!chance(errorRate)) {
    return false;
  }
  mistakes[op]();
  return true;
}

function step(op, action) {
  return mistaken(op) ? false : action();
}

function create(op, action) {
  return mistaken(op) ? '' : action();
}

function signUp() {
  patientId = create('register', () => created('register', 'POST', '/api/patients',
    { full_name: 'Нагрузочный Пациент', phone: '+79001234567', email: `${randomText()}@load.test`, birth_date: birthDate }));
  if (!patientId) {
    return false;
  }
  visitsLeft = visitsPerPatient;
  think();
  return true;
}

function settle(appointmentId) {
  if (chance(cancelShare)) {
    step('cancel_appointment', () => succeeded('cancel_appointment', 'POST', `/api/appointments/${appointmentId}/cancel`, undefined, 200));
    return;
  }
  const visitId = create('start_visit', () => created('start_visit', 'POST', '/api/visits',
    { appointment_id: appointmentId, complaints: 'Головная боль' }));
  if (!visitId) {
    return;
  }
  think();
  step('complete_visit', () => succeeded('complete_visit', 'POST', `/api/visits/${visitId}/complete`,
    { diagnosis: 'ОРВИ', recommendations: 'Обильное питьё' }, 200));
}

function visitClinic() {
  step('login', () => succeeded('login', 'GET', `/api/patients/${patientId}`, undefined, 200));
  think();
  step('browse_schedule', () => succeeded('browse_schedule', 'GET',
    `/api/schedule/free?doctor_id=${seededDoctor}&date=${day(firstDay)}`, undefined, 200));
  think();
  const appointmentId = create('book_appointment', () => created('book_appointment', 'POST', '/api/appointments',
    { patient_id: patientId, doctor_id: doctorId(), date: nextSlot() }));
  think();
  step('my_appointments', () => succeeded('my_appointments', 'GET', `/api/appointments?patient_id=${patientId}`, undefined, 200));
  if (appointmentId) {
    think();
    settle(appointmentId);
  }
  think();
  step('visit_history', () => succeeded('visit_history', 'GET', `/api/visits?patient_id=${patientId}`, undefined, 200));
}

export function setup() {
  return { start: Date.now() };
}

export default function () {
  if (visitsLeft > 0 || signUp()) {
    visitClinic();
    visitsLeft--;
  }
  sleep(between(1, 2));
}

export function handleSummary(data) {
  const meta = { start_unix_ms: data.setup_data.start, scenario: scenarioName, scale };
  const requests = data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0;
  const failed = data.metrics.http_req_failed ? data.metrics.http_req_failed.values.passes : 0;
  return {
    '/results/meta.json': JSON.stringify(meta, null, 2),
    '/results/k6-summary.json': JSON.stringify(data, null, 2),
    stdout: `k6: сценарий ${scenarioName}, масштаб ${scale}: ${requests} запросов, неожиданных ответов ${failed}\n`,
  };
}
