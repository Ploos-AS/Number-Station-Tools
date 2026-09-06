const stationForm = document.querySelector("#station-form");
const scheduleForm = document.querySelector("#schedule-form");
const stationSelect = document.querySelector("#station-select");
const schedulesList = document.querySelector("#schedules");
const nowList = document.querySelector("#now-list");
const nextList = document.querySelector("#next-list");
const statusBox = document.querySelector("#status");
const clockBox = document.querySelector("#clock");
const refreshButton = document.querySelector("#refresh");

let stations = [];

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: {"Content-Type": "application/json"},
    ...options
  });
  const text = await response.text();
  let body;
  try { body = JSON.parse(text); } catch { body = text; }
  if (!response.ok) throw new Error(typeof body === "string" ? body : body.error || "Request failed");
  return body;
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, c => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#039;"
  })[c]);
}

function stationName(id) {
  const found = stations.find(station => station.id === id);
  return found ? found.name : id;
}

function weekdays(days) {
  const names = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  return days.map(day => names[day - 1] || day).join(" ");
}

function khz(hz) {
  return (hz / 1000).toLocaleString(undefined, {maximumFractionDigits: 3}) + " kHz";
}

function occurrenceHTML(item) {
  const start = new Date(item.start).toISOString().slice(11, 16);
  const end = new Date(item.end).toISOString().slice(11, 16);
  return `<li><strong>${escapeHTML(item.station_name)}</strong><span>${khz(item.frequency_hz)} · ${escapeHTML(item.mode || "mode n/a")} · ${start}–${end} UTC</span></li>`;
}

async function refresh() {
  try {
    const [stationData, scheduleData, nowNext] = await Promise.all([
      api("/api/stations"), api("/api/schedules"), api("/api/now-next")
    ]);
    stations = stationData;
    stationSelect.innerHTML = stations.length
      ? stations.map(station => `<option value="${escapeHTML(station.id)}">${escapeHTML(station.name)}</option>`).join("")
      : '<option value="">Add a station first</option>';

    schedulesList.innerHTML = scheduleData.length
      ? scheduleData.map(item => `<li><strong>${escapeHTML(stationName(item.station_id))}</strong><span>${khz(item.frequency_hz)} · ${escapeHTML(item.start_utc)}–${escapeHTML(item.end_utc)} UTC · ${weekdays(item.weekdays)} · ${escapeHTML(item.mode || "mode n/a")}</span></li>`).join("")
      : "<li>No schedules yet.</li>";

    nowList.innerHTML = nowNext.now.length ? nowNext.now.map(occurrenceHTML).join("") : "<li>Nothing scheduled right now.</li>";
    nextList.innerHTML = nowNext.next.length ? nowNext.next.map(occurrenceHTML).join("") : "<li>No upcoming schedule in the next week.</li>";
    clockBox.textContent = "Schedule evaluated at " + new Date(nowNext.at).toISOString().replace(".000Z", "Z");
    statusBox.textContent = `${stations.length} stations · ${scheduleData.length} schedules · local data`;
  } catch (error) {
    statusBox.textContent = error.message;
    clockBox.textContent = error.message;
  }
}

stationForm.addEventListener("submit", async event => {
  event.preventDefault();
  const form = new FormData(stationForm);
  try {
    await api("/api/stations", {
      method: "POST",
      body: JSON.stringify({
        name: form.get("name"),
        aliases: String(form.get("aliases")).split(",").map(v => v.trim()).filter(Boolean),
        languages: String(form.get("languages")).split(",").map(v => v.trim()).filter(Boolean),
        notes: form.get("notes")
      })
    });
    stationForm.reset();
    await refresh();
  } catch (error) { statusBox.textContent = error.message; }
});

scheduleForm.addEventListener("submit", async event => {
  event.preventDefault();
  const form = new FormData(scheduleForm);
  try {
    await api("/api/schedules", {
      method: "POST",
      body: JSON.stringify({
        station_id: form.get("station_id"),
        frequency_hz: Number(form.get("frequency_hz")),
        start_utc: form.get("start_utc"),
        end_utc: form.get("end_utc"),
        weekdays: String(form.get("weekdays")).split(",").map(v => Number(v.trim())).filter(Number.isInteger),
        mode: form.get("mode"),
        notes: form.get("notes")
      })
    });
    scheduleForm.reset();
    await refresh();
  } catch (error) { statusBox.textContent = error.message; }
});

refreshButton.addEventListener("click", refresh);
refresh();
setInterval(refresh, 60000);
