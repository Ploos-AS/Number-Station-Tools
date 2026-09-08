(() => {
  const form = document.querySelector("#annotation-search-form");
  const results = document.querySelector("#annotation-search-results");
  const status = document.querySelector("#annotation-search-status");
  const recordings = document.querySelector("#recordings");
  if (!form || !results || !status || !recordings) return;

  function escapeHTML(value) {
    return String(value ?? "").replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#039;"})[c]);
  }

  function seconds(ms) {
    return (Number(ms || 0) / 1000).toFixed(3).replace(/\.000$/, "");
  }

  function findRecordingItem(recordingID) {
    return Array.from(recordings.querySelectorAll("li")).find(item => {
      const button = item.querySelector("[data-edit-recording], [data-delete-recording]");
      return button && (button.dataset.editRecording || button.dataset.deleteRecording) === recordingID;
    });
  }

  function jumpToHit(recordingID, startMS) {
    const item = findRecordingItem(recordingID);
    if (!item) return;
    item.scrollIntoView({behavior: "smooth", block: "center"});
    item.classList.add("annotation-search-target");
    window.setTimeout(() => item.classList.remove("annotation-search-target"), 1600);
    const audio = item.querySelector(".recording-player audio");
    if (audio) audio.currentTime = Number(startMS || 0) / 1000;
  }

  async function search() {
    const data = new FormData(form);
    const params = new URLSearchParams();
    const q = String(data.get("q") || "").trim();
    const type = String(data.get("type") || "").trim();
    if (q) params.set("q", q);
    if (type) params.set("type", type);
    params.set("limit", "100");
    status.textContent = "Searching…";
    try {
      const response = await fetch(`/api/annotations?${params}`);
      const text = await response.text();
      if (!response.ok) throw new Error(text.trim() || "Search failed");
      const hits = JSON.parse(text);
      results.innerHTML = hits.length ? hits.map(hit => {
        const a = hit.annotation;
        const when = a.end_ms > a.start_ms ? `${seconds(a.start_ms)}–${seconds(a.end_ms)} s` : `${seconds(a.start_ms)} s`;
        return `<li><button type="button" data-annotation-search-recording="${escapeHTML(hit.recording_id)}" data-annotation-search-start="${Number(a.start_ms || 0)}">${escapeHTML(a.label)}</button><span>${escapeHTML(a.type || "other")} · ${escapeHTML(when)} · ${escapeHTML(hit.path || hit.recording_id)}</span>${a.notes ? `<span>${escapeHTML(a.notes)}</span>` : ""}</li>`;
      }).join("") : "<li>No matching annotations.</li>";
      status.textContent = `${hits.length} match${hits.length === 1 ? "" : "es"}`;
    } catch (error) {
      status.textContent = error.message;
      results.innerHTML = "";
    }
  }

  form.addEventListener("submit", event => { event.preventDefault(); search(); });
  form.addEventListener("input", () => search());
  results.addEventListener("click", event => {
    const button = event.target.closest?.("[data-annotation-search-recording]");
    if (!button) return;
    jumpToHit(button.dataset.annotationSearchRecording, button.dataset.annotationSearchStart);
  });
  window.addEventListener("numberstation:annotations-imported", search);
  search();
})();
