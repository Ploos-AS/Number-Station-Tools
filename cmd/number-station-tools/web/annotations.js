(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  function seconds(ms) {
    return (Number(ms || 0) / 1000).toFixed(3).replace(/\.000$/, "");
  }

  function escapeHTML(value) {
    return String(value ?? "").replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#039;"})[c]);
  }

  async function request(path, options = {}) {
    const response = await fetch(path, {headers: {"Content-Type": "application/json"}, ...options});
    const text = await response.text();
    if (!response.ok) throw new Error(text.trim() || "Request failed");
    return text ? JSON.parse(text) : null;
  }

  async function attach(item, recordingID) {
    if (item.querySelector(".annotation-panel")) return;
    const actions = item.querySelector(".actions");
    const panel = document.createElement("div");
    panel.className = "annotation-panel";
    panel.innerHTML = `
      <div class="annotation-head"><strong>Bookmarks / annotations</strong><span class="annotation-status"></span></div>
      <form class="annotation-form">
        <div class="annotation-times">
          <label>Start ms <input name="start_ms" type="number" min="0" step="1" value="0" required></label>
          <button type="button" data-annotation-current="start">Start = current</button>
          <label>End ms <input name="end_ms" type="number" min="0" step="1" placeholder="optional interval end"></label>
          <button type="button" data-annotation-current="end">End = current</button>
        </div>
        <label>Label <input name="label" maxlength="120" required placeholder="Call-up, station ID, message starts…"></label>
        <label>Notes <textarea name="notes" maxlength="2000" placeholder="Optional local note"></textarea></label>
        <button type="submit">Save annotation</button>
      </form>
      <ul class="annotation-list"></ul>`;
    item.insertBefore(panel, actions || null);

    const form = panel.querySelector(".annotation-form");
    const status = panel.querySelector(".annotation-status");
    const annotationsList = panel.querySelector(".annotation-list");

    async function refreshAnnotations() {
      try {
        const annotations = await request(`/api/recordings/${encodeURIComponent(recordingID)}/annotations`);
        annotationsList.innerHTML = annotations.length ? annotations.map(annotation => {
          const interval = annotation.end_ms > annotation.start_ms;
          const when = interval ? `${seconds(annotation.start_ms)}–${seconds(annotation.end_ms)} s` : `${seconds(annotation.start_ms)} s`;
          return `<li data-annotation-id="${escapeHTML(annotation.id)}"><button type="button" data-annotation-seek="${Number(annotation.start_ms)}">${escapeHTML(annotation.label)}</button><span>${escapeHTML(when)}</span>${annotation.notes ? `<span>${escapeHTML(annotation.notes)}</span>` : ""}<button type="button" data-annotation-delete="${escapeHTML(annotation.id)}">Delete</button></li>`;
        }).join("") : "<li>No annotations yet.</li>";
        status.textContent = `${annotations.length} saved`;
      } catch (error) {
        status.textContent = error.message;
      }
    }

    panel.addEventListener("click", async event => {
      const target = event.target;
      if (target.dataset.annotationCurrent) {
        const audio = item.querySelector(".recording-player audio");
        if (!audio || !Number.isFinite(audio.currentTime)) return;
        const field = target.dataset.annotationCurrent === "end" ? "end_ms" : "start_ms";
        form.elements[field].value = String(Math.max(0, Math.round(audio.currentTime * 1000)));
      }
      if (target.dataset.annotationSeek) {
        const audio = item.querySelector(".recording-player audio");
        if (!audio) return;
        audio.currentTime = Number(target.dataset.annotationSeek) / 1000;
      }
      if (target.dataset.annotationDelete) {
        try {
          await request(`/api/recordings/${encodeURIComponent(recordingID)}/annotations/${encodeURIComponent(target.dataset.annotationDelete)}`, {method: "DELETE"});
          await refreshAnnotations();
        } catch (error) {
          status.textContent = error.message;
        }
      }
    });

    form.addEventListener("submit", async event => {
      event.preventDefault();
      const data = new FormData(form);
      const payload = {
        start_ms: Number(data.get("start_ms") || 0),
        end_ms: Number(data.get("end_ms") || 0),
        label: data.get("label"),
        notes: data.get("notes")
      };
      try {
        await request(`/api/recordings/${encodeURIComponent(recordingID)}/annotations`, {method: "POST", body: JSON.stringify(payload)});
        const currentStart = form.elements.start_ms.value;
        form.reset();
        form.elements.start_ms.value = currentStart || "0";
        await refreshAnnotations();
      } catch (error) {
        status.textContent = error.message;
      }
    });

    await refreshAnnotations();
  }

  function render() {
    list.querySelectorAll("li").forEach(item => {
      const button = item.querySelector("[data-edit-recording], [data-delete-recording]");
      if (!button) return;
      const id = button.dataset.editRecording || button.dataset.deleteRecording;
      attach(item, id);
    });
  }

  new MutationObserver(render).observe(list, {childList: true, subtree: true});
  render();
})();
