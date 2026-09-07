const stationForm = document.querySelector("#station-form");
const scheduleForm = document.querySelector("#schedule-form");
const stationSelect = document.querySelector("#station-select");
const stationsList = document.querySelector("#stations");
const schedulesList = document.querySelector("#schedules");
const nowList = document.querySelector("#now-list");
const nextList = document.querySelector("#next-list");
const statusBox = document.querySelector("#status");
const clockBox = document.querySelector("#clock");
const refreshButton = document.querySelector("#refresh");
const filterBox = document.querySelector("#schedule-filter");
const stationCancel = document.querySelector("#station-cancel");
const scheduleCancel = document.querySelector("#schedule-cancel");

let stations = [];
let schedules = [];

async function api(path, options = {}) {
  const response = await fetch(path, {headers: {"Content-Type": "application/json"}, ...options});
  const text = await response.text();
  let body;
  try { body = text ? JSON.parse(text) : null; } catch { body = text; }
  if (!response.ok) throw new Error(typeof body === "string" ? body.trim() : body?.error || "Request failed");
  return body;
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#039;"})[c]);
}
function stationName(id) { return stations.find(s => s.id === id)?.name || id; }
function weekdays(days) { return days.map(d => ["Mon","Tue","Wed","Thu","Fri","Sat","Sun"][d - 1] || d).join(" "); }
function khz(hz) { return (hz / 1000).toLocaleString(undefined, {maximumFractionDigits: 3}) + " kHz"; }
function occurrenceHTML(item) {
  const start = new Date(item.start).toISOString().slice(11,16);
  const end = new Date(item.end).toISOString().slice(11,16);
  return `<li><strong>${escapeHTML(item.station_name)}</strong><span>${khz(item.frequency_hz)} · ${escapeHTML(item.mode || "mode n/a")} · ${start}–${end} UTC</span></li>`;
}

function renderStations() {
  stationsList.innerHTML = stations.length ? stations.map(s => `<li><strong>${escapeHTML(s.name)}</strong><span>${escapeHTML((s.aliases || []).join(", "))}</span><div class="actions"><button type="button" data-edit-station="${escapeHTML(s.id)}">Edit</button><button type="button" data-delete-station="${escapeHTML(s.id)}">Delete</button></div></li>`).join("") : "<li>No stations yet.</li>";
}

function renderSchedules() {
  const q = filterBox.value.trim().toLowerCase();
  const filtered = schedules.filter(item => !q || [stationName(item.station_id), item.frequency_hz, item.mode, item.notes, item.start_utc, item.end_utc, weekdays(item.weekdays)].join(" ").toLowerCase().includes(q));
  schedulesList.innerHTML = filtered.length ? filtered.map(item => `<li><strong>${escapeHTML(stationName(item.station_id))}</strong><span>${khz(item.frequency_hz)} · ${escapeHTML(item.start_utc)}–${escapeHTML(item.end_utc)} UTC · ${weekdays(item.weekdays)} · ${escapeHTML(item.mode || "mode n/a")}</span><div class="actions"><button type="button" data-edit-schedule="${escapeHTML(item.id)}">Edit</button><button type="button" data-delete-schedule="${escapeHTML(item.id)}">Delete</button></div></li>`).join("") : "<li>No matching schedules.</li>";
}

async function refresh() {
  try {
    const [stationData, scheduleData, nowNext] = await Promise.all([api("/api/stations"), api("/api/schedules"), api("/api/now-next")]);
    stations = stationData;
    schedules = scheduleData;
    stationSelect.innerHTML = stations.length ? stations.map(s => `<option value="${escapeHTML(s.id)}">${escapeHTML(s.name)}</option>`).join("") : '<option value="">Add a station first</option>';
    renderStations();
    renderSchedules();
    nowList.innerHTML = nowNext.now.length ? nowNext.now.map(occurrenceHTML).join("") : "<li>Nothing scheduled right now.</li>";
    nextList.innerHTML = nowNext.next.length ? nowNext.next.map(occurrenceHTML).join("") : "<li>No upcoming schedule in the next week.</li>";
    clockBox.textContent = "Schedule evaluated at " + new Date(nowNext.at).toISOString().replace(".000Z", "Z");
    statusBox.textContent = `${stations.length} stations · ${schedules.length} schedules · local data`;
  } catch (error) { statusBox.textContent = error.message; clockBox.textContent = error.message; }
}

function resetStationForm() { stationForm.reset(); stationForm.elements.id.value = ""; document.querySelector("#station-form-title").textContent = "Add station"; }
function resetScheduleForm() { scheduleForm.reset(); scheduleForm.elements.id.value = ""; document.querySelector("#schedule-form-title").textContent = "Add schedule"; }

stationForm.addEventListener("submit", async event => {
  event.preventDefault(); const f = new FormData(stationForm); const id = f.get("id");
  const payload = {name:f.get("name"), aliases:String(f.get("aliases")).split(",").map(v=>v.trim()).filter(Boolean), languages:String(f.get("languages")).split(",").map(v=>v.trim()).filter(Boolean), notes:f.get("notes")};
  try { await api(id ? `/api/stations/${id}` : "/api/stations", {method:id ? "PUT" : "POST", body:JSON.stringify(payload)}); resetStationForm(); await refresh(); } catch (error) { statusBox.textContent = error.message; }
});

scheduleForm.addEventListener("submit", async event => {
  event.preventDefault(); const f = new FormData(scheduleForm); const id = f.get("id");
  const payload = {station_id:f.get("station_id"), frequency_hz:Number(f.get("frequency_hz")), start_utc:f.get("start_utc"), end_utc:f.get("end_utc"), weekdays:String(f.get("weekdays")).split(",").map(v=>Number(v.trim())).filter(Number.isInteger), mode:f.get("mode"), notes:f.get("notes")};
  try { await api(id ? `/api/schedules/${id}` : "/api/schedules", {method:id ? "PUT" : "POST", body:JSON.stringify(payload)}); resetScheduleForm(); await refresh(); } catch (error) { statusBox.textContent = error.message; }
});

document.addEventListener("click", async event => {
  const eStation = event.target.dataset.editStation;
  const dStation = event.target.dataset.deleteStation;
  const eSchedule = event.target.dataset.editSchedule;
  const dSchedule = event.target.dataset.deleteSchedule;
  if (eStation) { const s = stations.find(x=>x.id===eStation); stationForm.elements.id.value=s.id; stationForm.elements.name.value=s.name; stationForm.elements.aliases.value=(s.aliases||[]).join(", "); stationForm.elements.languages.value=(s.languages||[]).join(", "); stationForm.elements.notes.value=s.notes||""; document.querySelector("#station-form-title").textContent="Edit station"; }
  if (eSchedule) { const s=schedules.find(x=>x.id===eSchedule); scheduleForm.elements.id.value=s.id; scheduleForm.elements.station_id.value=s.station_id; scheduleForm.elements.frequency_hz.value=s.frequency_hz; scheduleForm.elements.start_utc.value=s.start_utc; scheduleForm.elements.end_utc.value=s.end_utc; scheduleForm.elements.weekdays.value=s.weekdays.join(","); scheduleForm.elements.mode.value=s.mode||""; scheduleForm.elements.notes.value=s.notes||""; document.querySelector("#schedule-form-title").textContent="Edit schedule"; }
  if (dSchedule && confirm("Delete this schedule?")) { try { await api(`/api/schedules/${dSchedule}`, {method:"DELETE"}); await refresh(); } catch(error){ statusBox.textContent=error.message; } }
  if (dStation && confirm("Delete this station? This is blocked while schedules or observations reference it.")) { try { await api(`/api/stations/${dStation}`, {method:"DELETE"}); await refresh(); } catch(error){ statusBox.textContent=error.message; } }
});

stationCancel.addEventListener("click", resetStationForm);
scheduleCancel.addEventListener("click", resetScheduleForm);
filterBox.addEventListener("input", renderSchedules);
refreshButton.addEventListener("click", refresh);
refresh();
setInterval(refresh, 60000);
