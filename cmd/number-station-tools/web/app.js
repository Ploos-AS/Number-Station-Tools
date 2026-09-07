const stationForm = document.querySelector("#station-form");
const scheduleForm = document.querySelector("#schedule-form");
const observationForm = document.querySelector("#observation-form");
const recordingForm = document.querySelector("#recording-form");
const stationSelect = document.querySelector("#station-select");
const observationStationSelect = document.querySelector("#observation-station-select");
const observationScheduleSelect = document.querySelector("#observation-schedule-select");
const recordingObservationSelect = document.querySelector("#recording-observation-select");
const stationsList = document.querySelector("#stations");
const schedulesList = document.querySelector("#schedules");
const observationsList = document.querySelector("#observations");
const recordingsList = document.querySelector("#recordings");
const nowList = document.querySelector("#now-list");
const nextList = document.querySelector("#next-list");
const statusBox = document.querySelector("#status");
const observationStatus = document.querySelector("#observation-status");
const recordingStatus = document.querySelector("#recording-status");
const clockBox = document.querySelector("#clock");
const refreshButton = document.querySelector("#refresh");
const filterBox = document.querySelector("#schedule-filter");
const observationFilter = document.querySelector("#observation-filter");
const recordingFilter = document.querySelector("#recording-filter");
const stationCancel = document.querySelector("#station-cancel");
const scheduleCancel = document.querySelector("#schedule-cancel");
const observationCancel = document.querySelector("#observation-cancel");
const recordingCancel = document.querySelector("#recording-cancel");

let stations = [];
let schedules = [];
let observations = [];
let recordings = [];

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
function observationByID(id) { return observations.find(o => o.id === id); }
function observationName(id) {
  const o = observationByID(id);
  return o ? `${stationName(o.station_id)} · ${khz(o.frequency_hz)} · ${new Date(o.heard_at).toISOString()}` : id;
}
function scheduleName(id) {
  const s = schedules.find(x => x.id === id);
  return s ? `${stationName(s.station_id)} ${khz(s.frequency_hz)} ${s.start_utc} UTC` : id;
}
function weekdays(days) { return days.map(d => ["Mon","Tue","Wed","Thu","Fri","Sat","Sun"][d - 1] || d).join(" "); }
function khz(hz) { return (hz / 1000).toLocaleString(undefined, {maximumFractionDigits: 3}) + " kHz"; }
function utcInputValue(value) { return value ? new Date(value).toISOString().slice(0,16) : ""; }
function recordingCount(observationID) { return recordings.filter(r => r.observation_id === observationID).length; }
function occurrenceHTML(item) {
  const start = new Date(item.start).toISOString().slice(11,16);
  const end = new Date(item.end).toISOString().slice(11,16);
  return `<li><strong>${escapeHTML(item.station_name)}</strong><span>${khz(item.frequency_hz)} · ${escapeHTML(item.mode || "mode n/a")} · ${start}–${end} UTC</span><div class="actions"><button type="button" data-log-schedule="${escapeHTML(item.schedule_id)}">Log heard</button></div></li>`;
}

function stationOptions() {
  return stations.length ? stations.map(s => `<option value="${escapeHTML(s.id)}">${escapeHTML(s.name)}</option>`).join("") : '<option value="">Add a station first</option>';
}
function observationOptions() {
  return observations.length ? observations.map(o => `<option value="${escapeHTML(o.id)}">${escapeHTML(observationName(o.id))}</option>`).join("") : '<option value="">Add an observation first</option>';
}
function syncObservationSchedules() {
  const stationID = observationForm.elements.station_id.value;
  const linked = observationForm.elements.schedule_id.value;
  const options = schedules.filter(s => s.station_id === stationID);
  observationScheduleSelect.innerHTML = '<option value="">None / unscheduled</option>' + options.map(s => `<option value="${escapeHTML(s.id)}">${escapeHTML(scheduleName(s.id))}</option>`).join("");
  if (options.some(s => s.id === linked)) observationScheduleSelect.value = linked;
}

function renderStations() {
  stationsList.innerHTML = stations.length ? stations.map(s => `<li><strong>${escapeHTML(s.name)}</strong><span>${escapeHTML((s.aliases || []).join(", "))}</span><div class="actions"><button type="button" data-edit-station="${escapeHTML(s.id)}">Edit</button><button type="button" data-delete-station="${escapeHTML(s.id)}">Delete</button></div></li>`).join("") : "<li>No stations yet.</li>";
}
function renderSchedules() {
  const q = filterBox.value.trim().toLowerCase();
  const filtered = schedules.filter(item => !q || [stationName(item.station_id), item.frequency_hz, item.mode, item.notes, item.start_utc, item.end_utc, weekdays(item.weekdays)].join(" ").toLowerCase().includes(q));
  schedulesList.innerHTML = filtered.length ? filtered.map(item => `<li><strong>${escapeHTML(stationName(item.station_id))}</strong><span>${khz(item.frequency_hz)} · ${escapeHTML(item.start_utc)}–${escapeHTML(item.end_utc)} UTC · ${weekdays(item.weekdays)} · ${escapeHTML(item.mode || "mode n/a")}</span><div class="actions"><button type="button" data-log-schedule="${escapeHTML(item.id)}">Log heard</button><button type="button" data-edit-schedule="${escapeHTML(item.id)}">Edit</button><button type="button" data-delete-schedule="${escapeHTML(item.id)}">Delete</button></div></li>`).join("") : "<li>No matching schedules.</li>";
}
function renderObservations() {
  const q = observationFilter.value.trim().toLowerCase();
  const filtered = observations.filter(item => !q || [stationName(item.station_id), item.frequency_hz, item.mode, item.signal, item.message, item.notes, item.schedule_id ? scheduleName(item.schedule_id) : "unscheduled", item.heard_at].join(" ").toLowerCase().includes(q));
  observationsList.innerHTML = filtered.length ? filtered.map(item => `<li><strong>${escapeHTML(stationName(item.station_id))}</strong><span>${khz(item.frequency_hz)} · ${escapeHTML(item.mode || "mode n/a")} · ${escapeHTML(item.signal || "signal n/a")} · ${escapeHTML(new Date(item.heard_at).toISOString())}</span>${item.schedule_id ? `<span>Schedule: ${escapeHTML(scheduleName(item.schedule_id))}</span>` : ""}${item.message ? `<span>Message: ${escapeHTML(item.message)}</span>` : ""}${item.notes ? `<span>Notes: ${escapeHTML(item.notes)}</span>` : ""}<span>${recordingCount(item.id)} recording metadata entr${recordingCount(item.id) === 1 ? "y" : "ies"}</span><div class="actions"><button type="button" data-add-recording="${escapeHTML(item.id)}">Add recording</button><button type="button" data-edit-observation="${escapeHTML(item.id)}">Edit</button><button type="button" data-delete-observation="${escapeHTML(item.id)}">Delete</button></div></li>`).join("") : "<li>No matching observations.</li>";
}
function renderRecordings() {
  const q = recordingFilter.value.trim().toLowerCase();
  const filtered = recordings.filter(item => !q || [observationName(item.observation_id), item.path, item.format, item.sha256, item.notes, item.size_bytes, item.duration_ms, item.sample_rate_hz, item.channels].join(" ").toLowerCase().includes(q));
  recordingsList.innerHTML = filtered.length ? filtered.map(item => `<li><strong>${escapeHTML(item.path)}</strong><span>${escapeHTML(item.format.toUpperCase())} · ${item.duration_ms || 0} ms · ${item.sample_rate_hz || 0} Hz · ${item.channels || 0} ch · ${item.size_bytes || 0} bytes</span><span>Observation: ${escapeHTML(observationName(item.observation_id))}</span>${item.sha256 ? `<span>SHA-256: ${escapeHTML(item.sha256)}</span>` : ""}${item.notes ? `<span>Notes: ${escapeHTML(item.notes)}</span>` : ""}<div class="actions"><button type="button" data-edit-recording="${escapeHTML(item.id)}">Edit</button><button type="button" data-delete-recording="${escapeHTML(item.id)}">Delete</button></div></li>`).join("") : "<li>No matching recording metadata.</li>";
}

async function refresh() {
  try {
    const [stationData, scheduleData, observationData, recordingData, nowNext] = await Promise.all([api("/api/stations"), api("/api/schedules"), api("/api/observations"), api("/api/recordings"), api("/api/now-next")]);
    stations = stationData; schedules = scheduleData; observations = observationData; recordings = recordingData;
    stationSelect.innerHTML = stationOptions();
    observationStationSelect.innerHTML = stationOptions();
    recordingObservationSelect.innerHTML = observationOptions();
    syncObservationSchedules();
    renderStations(); renderSchedules(); renderObservations(); renderRecordings();
    nowList.innerHTML = nowNext.now.length ? nowNext.now.map(occurrenceHTML).join("") : "<li>Nothing scheduled right now.</li>";
    nextList.innerHTML = nowNext.next.length ? nowNext.next.map(occurrenceHTML).join("") : "<li>No upcoming schedule in the next week.</li>";
    clockBox.textContent = "Schedule evaluated at " + new Date(nowNext.at).toISOString().replace(".000Z", "Z");
    statusBox.textContent = `${stations.length} stations · ${schedules.length} schedules · local data`;
    observationStatus.textContent = `${observations.length} observations · newest first`;
    recordingStatus.textContent = `${recordings.length} recording metadata entries · audio files not uploaded`;
  } catch (error) { statusBox.textContent = error.message; observationStatus.textContent = error.message; recordingStatus.textContent = error.message; clockBox.textContent = error.message; }
}

function resetStationForm() { stationForm.reset(); stationForm.elements.id.value = ""; document.querySelector("#station-form-title").textContent = "Add station"; }
function resetScheduleForm() { scheduleForm.reset(); scheduleForm.elements.id.value = ""; document.querySelector("#schedule-form-title").textContent = "Add schedule"; }
function resetObservationForm() { observationForm.reset(); observationForm.elements.id.value = ""; document.querySelector("#observation-form-title").textContent = "Log observation"; syncObservationSchedules(); }
function resetRecordingForm() { recordingForm.reset(); recordingForm.elements.id.value = ""; document.querySelector("#recording-form-title").textContent = "Add recording metadata"; }
function fillObservationFromSchedule(scheduleID) {
  const s = schedules.find(x => x.id === scheduleID); if (!s) return;
  resetObservationForm(); observationForm.elements.station_id.value = s.station_id; syncObservationSchedules(); observationForm.elements.schedule_id.value = s.id; observationForm.elements.frequency_hz.value = s.frequency_hz; observationForm.elements.mode.value = s.mode || ""; observationForm.elements.heard_at.value = utcInputValue(new Date()); document.querySelector("#observation-form-title").textContent = "Log heard schedule"; observationForm.scrollIntoView({behavior:"smooth", block:"start"});
}
function fillRecordingFromObservation(observationID) {
  resetRecordingForm(); recordingForm.elements.observation_id.value = observationID; document.querySelector("#recording-form-title").textContent = "Add recording for observation"; recordingForm.scrollIntoView({behavior:"smooth", block:"start"});
}

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
observationForm.addEventListener("submit", async event => {
  event.preventDefault(); const f = new FormData(observationForm); const id = f.get("id");
  const heard = f.get("heard_at") ? new Date(String(f.get("heard_at")) + "Z").toISOString() : null;
  const payload = {station_id:f.get("station_id"), schedule_id:f.get("schedule_id"), heard_at:heard, frequency_hz:Number(f.get("frequency_hz")), mode:f.get("mode"), signal:f.get("signal"), message:f.get("message"), notes:f.get("notes")};
  try { await api(id ? `/api/observations/${id}` : "/api/observations", {method:id ? "PUT" : "POST", body:JSON.stringify(payload)}); resetObservationForm(); await refresh(); } catch (error) { observationStatus.textContent = error.message; }
});
recordingForm.addEventListener("submit", async event => {
  event.preventDefault(); const f = new FormData(recordingForm); const id = f.get("id");
  const payload = {observation_id:f.get("observation_id"), path:f.get("path"), format:f.get("format"), size_bytes:Number(f.get("size_bytes") || 0), duration_ms:Number(f.get("duration_ms") || 0), sample_rate_hz:Number(f.get("sample_rate_hz") || 0), channels:Number(f.get("channels") || 0), sha256:f.get("sha256"), notes:f.get("notes")};
  try { await api(id ? `/api/recordings/${id}` : "/api/recordings", {method:id ? "PUT" : "POST", body:JSON.stringify(payload)}); resetRecordingForm(); await refresh(); } catch (error) { recordingStatus.textContent = error.message; }
});

document.addEventListener("click", async event => {
  const t = event.target;
  if (t.dataset.logSchedule) fillObservationFromSchedule(t.dataset.logSchedule);
  if (t.dataset.addRecording) fillRecordingFromObservation(t.dataset.addRecording);
  if (t.dataset.editStation) { const s=stations.find(x=>x.id===t.dataset.editStation); stationForm.elements.id.value=s.id; stationForm.elements.name.value=s.name; stationForm.elements.aliases.value=(s.aliases||[]).join(", "); stationForm.elements.languages.value=(s.languages||[]).join(", "); stationForm.elements.notes.value=s.notes||""; document.querySelector("#station-form-title").textContent="Edit station"; }
  if (t.dataset.editSchedule) { const s=schedules.find(x=>x.id===t.dataset.editSchedule); scheduleForm.elements.id.value=s.id; scheduleForm.elements.station_id.value=s.station_id; scheduleForm.elements.frequency_hz.value=s.frequency_hz; scheduleForm.elements.start_utc.value=s.start_utc; scheduleForm.elements.end_utc.value=s.end_utc; scheduleForm.elements.weekdays.value=s.weekdays.join(","); scheduleForm.elements.mode.value=s.mode||""; scheduleForm.elements.notes.value=s.notes||""; document.querySelector("#schedule-form-title").textContent="Edit schedule"; }
  if (t.dataset.editObservation) { const o=observations.find(x=>x.id===t.dataset.editObservation); observationForm.elements.id.value=o.id; observationForm.elements.station_id.value=o.station_id; syncObservationSchedules(); observationForm.elements.schedule_id.value=o.schedule_id||""; observationForm.elements.heard_at.value=utcInputValue(o.heard_at); observationForm.elements.frequency_hz.value=o.frequency_hz; observationForm.elements.mode.value=o.mode||""; observationForm.elements.signal.value=o.signal||""; observationForm.elements.message.value=o.message||""; observationForm.elements.notes.value=o.notes||""; document.querySelector("#observation-form-title").textContent="Edit observation"; }
  if (t.dataset.editRecording) { const r=recordings.find(x=>x.id===t.dataset.editRecording); recordingForm.elements.id.value=r.id; recordingForm.elements.observation_id.value=r.observation_id; recordingForm.elements.path.value=r.path; recordingForm.elements.format.value=r.format; recordingForm.elements.size_bytes.value=r.size_bytes||0; recordingForm.elements.duration_ms.value=r.duration_ms||0; recordingForm.elements.sample_rate_hz.value=r.sample_rate_hz||0; recordingForm.elements.channels.value=r.channels||0; recordingForm.elements.sha256.value=r.sha256||""; recordingForm.elements.notes.value=r.notes||""; document.querySelector("#recording-form-title").textContent="Edit recording metadata"; }
  if (t.dataset.deleteRecording && confirm("Delete this recording metadata entry? The referenced audio file is not touched.")) { try { await api(`/api/recordings/${t.dataset.deleteRecording}`, {method:"DELETE"}); await refresh(); } catch(error){ recordingStatus.textContent=error.message; } }
  if (t.dataset.deleteObservation && confirm("Delete this observation? Recording metadata must be removed first.")) { try { await api(`/api/observations/${t.dataset.deleteObservation}`, {method:"DELETE"}); await refresh(); } catch(error){ observationStatus.textContent=error.message; } }
  if (t.dataset.deleteSchedule && confirm("Delete this schedule? Linked observations must be removed or unlinked first.")) { try { await api(`/api/schedules/${t.dataset.deleteSchedule}`, {method:"DELETE"}); await refresh(); } catch(error){ statusBox.textContent=error.message; } }
  if (t.dataset.deleteStation && confirm("Delete this station? This is blocked while schedules or observations reference it.")) { try { await api(`/api/stations/${t.dataset.deleteStation}`, {method:"DELETE"}); await refresh(); } catch(error){ statusBox.textContent=error.message; } }
});

observationStationSelect.addEventListener("change", syncObservationSchedules);
stationCancel.addEventListener("click", resetStationForm);
scheduleCancel.addEventListener("click", resetScheduleForm);
observationCancel.addEventListener("click", resetObservationForm);
recordingCancel.addEventListener("click", resetRecordingForm);
filterBox.addEventListener("input", renderSchedules);
observationFilter.addEventListener("input", renderObservations);
recordingFilter.addEventListener("input", renderRecordings);
refreshButton.addEventListener("click", refresh);
refresh();
setInterval(refresh, 60000);
