(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  const annotationTypes = ["call-up", "station ID", "message", "tone", "noise", "fade", "other"];

  function seconds(ms) {
    return (Number(ms || 0) / 1000).toFixed(3).replace(/\.000$/, "");
  }

  function escapeHTML(value) {
    return String(value ?? "").replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#039;"})[c]);
  }

  function typeSlug(value) {
    return String(value || "other").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "other";
  }

  function normalizedType(annotation) {
    return annotation?.type || "other";
  }

  async function request(path, options = {}) {
    const response = await fetch(path, {headers: {"Content-Type": "application/json"}, ...options});
    const text = await response.text();
    if (!response.ok) throw new Error(text.trim() || "Request failed");
    return text ? JSON.parse(text) : null;
  }

  function annotationTitle(annotation) {
    const interval = annotation.end_ms > annotation.start_ms;
    const when = interval
      ? `${seconds(annotation.start_ms)}–${seconds(annotation.end_ms)} s`
      : `${seconds(annotation.start_ms)} s`;
    const base = `${normalizedType(annotation)} · ${annotation.label} · ${when}`;
    return annotation.notes ? `${base} · ${annotation.notes}` : base;
  }

  function renderMarkers(item, recording, annotations) {
    item.querySelectorAll(".annotation-overlay").forEach(node => node.remove());
    const durationMS = Number(recording?.duration_ms || 0);
    if (!(durationMS > 0) || !annotations.length) return;

    item.querySelectorAll(".waveform-wrap.playback-track, .spectrogram-wrap.playback-track").forEach(track => {
      const overlay = document.createElement("div");
      overlay.className = "annotation-overlay";
      overlay.setAttribute("aria-label", "Saved recording annotations");

      annotations.forEach(annotation => {
        const start = Math.max(0, Math.min(durationMS, Number(annotation.start_ms || 0)));
        const end = Math.max(0, Math.min(durationMS, Number(annotation.end_ms || 0)));
        const startPct = start / durationMS * 100;
        const interval = end > start;
        const marker = document.createElement("button");
        marker.type = "button";
        marker.className = `${interval ? "annotation-marker annotation-interval" : "annotation-marker annotation-point"} annotation-type-${typeSlug(normalizedType(annotation))}`;
        marker.dataset.annotationMarker = String(annotation.start_ms || 0);
        marker.dataset.annotationType = normalizedType(annotation);
        marker.title = annotationTitle(annotation);
        marker.setAttribute("aria-label", `Seek to ${annotationTitle(annotation)}`);
        marker.style.left = `${startPct.toFixed(3)}%`;
        if (interval) marker.style.width = `${Math.max(0.35, (end - start) / durationMS * 100).toFixed(3)}%`;
        const label = document.createElement("span");
        label.textContent = `${normalizedType(annotation)}: ${annotation.label}`;
        marker.append(label);
        overlay.append(marker);
      });
      track.append(overlay);
    });
  }

  async function attach(item, recording) {
    if (item.querySelector(".annotation-panel")) return;
    const recordingID = recording.id;
    const actions = item.querySelector(".actions");
    const panel = document.createElement("div");
    panel.className = "annotation-panel";
    panel.innerHTML = `
      <div class="annotation-head"><strong>Bookmarks / annotations</strong><span class="annotation-status"></span></div>
      <form class="annotation-form">
        <input type="hidden" name="annotation_id">
        <div class="annotation-times">
          <label>Start ms <input name="start_ms" type="number" min="0" step="1" value="0" required></label>
          <button type="button" data-annotation-current="start">Start = current</button>
          <label>End ms <input name="end_ms" type="number" min="0" step="1" placeholder="optional interval end"></label>
          <button type="button" data-annotation-current="end">End = current</button>
        </div>
        <label>Type <select name="type">${annotationTypes.map(type => `<option value="${escapeHTML(type)}">${escapeHTML(type)}</option>`).join("")}</select></label>
        <label>Label <input name="label" maxlength="120" required placeholder="Call-up, station ID, message starts…"></label>
        <label>Notes <textarea name="notes" maxlength="2000" placeholder="Optional local note"></textarea></label>
        <div class="annotation-form-actions"><button type="submit" data-annotation-save>Save annotation</button><button type="button" data-annotation-cancel hidden>Cancel edit</button></div>
      </form>
      <div class="annotation-filter-row"><label>Show type <select data-annotation-filter><option value="">All types</option>${annotationTypes.map(type => `<option value="${escapeHTML(type)}">${escapeHTML(type)}</option>`).join("")}</select></label></div>
      <p class="annotation-help">Right-click a waveform/spectrogram position to save an Other bookmark immediately. With a preview focused, press B to bookmark the current playback position.</p>
      <ul class="annotation-list"></ul>`;
    item.insertBefore(panel, actions || null);

    const form = panel.querySelector(".annotation-form");
    const status = panel.querySelector(".annotation-status");
    const annotationsList = panel.querySelector(".annotation-list");
    const saveButton = panel.querySelector("[data-annotation-save]");
    const cancelButton = panel.querySelector("[data-annotation-cancel]");
    const filter = panel.querySelector("[data-annotation-filter]");
    let annotations = [];

    function resetForm(startMS = 0) {
      form.reset();
      form.elements.annotation_id.value = "";
      form.elements.start_ms.value = String(Math.max(0, Math.round(startMS)));
      form.elements.type.value = "other";
      saveButton.textContent = "Save annotation";
      cancelButton.hidden = true;
    }

    function editAnnotation(annotation) {
      form.elements.annotation_id.value = annotation.id;
      form.elements.start_ms.value = String(annotation.start_ms || 0);
      form.elements.end_ms.value = annotation.end_ms > annotation.start_ms ? String(annotation.end_ms) : "";
      form.elements.type.value = normalizedType(annotation);
      form.elements.label.value = annotation.label || "";
      form.elements.notes.value = annotation.notes || "";
      saveButton.textContent = "Update annotation";
      cancelButton.hidden = false;
      form.elements.label.focus();
    }

    function visibleAnnotations() {
      const selected = filter.value;
      return selected ? annotations.filter(annotation => normalizedType(annotation) === selected) : annotations;
    }

    function renderAnnotationViews() {
      const visible = visibleAnnotations();
      annotationsList.innerHTML = visible.length ? visible.map(annotation => {
        const interval = annotation.end_ms > annotation.start_ms;
        const when = interval ? `${seconds(annotation.start_ms)}–${seconds(annotation.end_ms)} s` : `${seconds(annotation.start_ms)} s`;
        return `<li data-annotation-id="${escapeHTML(annotation.id)}" data-annotation-type="${escapeHTML(normalizedType(annotation))}"><button type="button" data-annotation-seek="${Number(annotation.start_ms)}">${escapeHTML(annotation.label)}</button><span class="annotation-type-badge annotation-type-${typeSlug(normalizedType(annotation))}">${escapeHTML(normalizedType(annotation))}</span><span>${escapeHTML(when)}</span>${annotation.notes ? `<span>${escapeHTML(annotation.notes)}</span>` : ""}<div class="annotation-row-actions"><button type="button" data-annotation-edit="${escapeHTML(annotation.id)}">Edit</button><button type="button" data-annotation-delete="${escapeHTML(annotation.id)}">Delete</button></div></li>`;
      }).join("") : "<li>No annotations for this filter.</li>";
      status.textContent = filter.value ? `${visible.length} of ${annotations.length} shown` : `${annotations.length} saved`;
      renderMarkers(item, recording, visible);
    }

    async function refreshAnnotations() {
      try {
        annotations = await request(`/api/recordings/${encodeURIComponent(recordingID)}/annotations`);
        renderAnnotationViews();
      } catch (error) {
        status.textContent = error.message;
      }
    }

    async function createBookmarkAt(startMS) {
      const bounded = Math.max(0, Math.min(Number(recording.duration_ms || startMS), Math.round(startMS)));
      try {
        await request(`/api/recordings/${encodeURIComponent(recordingID)}/annotations`, {
          method: "POST",
          body: JSON.stringify({start_ms: bounded, end_ms: 0, type: "other", label: "Bookmark", notes: ""})
        });
        status.textContent = `Bookmark saved at ${seconds(bounded)} s`;
        await refreshAnnotations();
      } catch (error) {
        status.textContent = error.message;
      }
    }

    item.addEventListener("click", event => {
      const marker = event.target.closest?.("[data-annotation-marker]");
      if (!marker) return;
      const audio = item.querySelector(".recording-player audio");
      if (!audio) return;
      audio.currentTime = Number(marker.dataset.annotationMarker) / 1000;
    });

    item.addEventListener("contextmenu", event => {
      const track = event.target.closest?.(".waveform-wrap.playback-track, .spectrogram-wrap.playback-track");
      if (!track || event.target.closest?.(".annotation-marker")) return;
      const durationMS = Number(recording.duration_ms || 0);
      if (!(durationMS > 0)) return;
      const rect = track.getBoundingClientRect();
      if (!(rect.width > 0)) return;
      event.preventDefault();
      const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width));
      createBookmarkAt(ratio * durationMS);
    });

    item.addEventListener("keydown", event => {
      if (event.key.toLowerCase() !== "b" || !event.target.closest?.(".interactive-seek")) return;
      const audio = item.querySelector(".recording-player audio");
      if (!audio || !Number.isFinite(audio.currentTime)) return;
      event.preventDefault();
      createBookmarkAt(audio.currentTime * 1000);
    });

    filter.addEventListener("change", renderAnnotationViews);

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
        if (audio) audio.currentTime = Number(target.dataset.annotationSeek) / 1000;
      }
      if (target.dataset.annotationEdit) {
        const annotation = annotations.find(candidate => candidate.id === target.dataset.annotationEdit);
        if (annotation) editAnnotation(annotation);
      }
      if (target.dataset.annotationCancel !== undefined) resetForm();
      if (target.dataset.annotationDelete) {
        try {
          await request(`/api/recordings/${encodeURIComponent(recordingID)}/annotations/${encodeURIComponent(target.dataset.annotationDelete)}`, {method: "DELETE"});
          if (form.elements.annotation_id.value === target.dataset.annotationDelete) resetForm();
          await refreshAnnotations();
        } catch (error) {
          status.textContent = error.message;
        }
      }
    });

    form.addEventListener("submit", async event => {
      event.preventDefault();
      const data = new FormData(form);
      const annotationID = String(data.get("annotation_id") || "");
      const payload = {
        start_ms: Number(data.get("start_ms") || 0),
        end_ms: Number(data.get("end_ms") || 0),
        type: data.get("type") || "other",
        label: data.get("label"),
        notes: data.get("notes")
      };
      const path = annotationID
        ? `/api/recordings/${encodeURIComponent(recordingID)}/annotations/${encodeURIComponent(annotationID)}`
        : `/api/recordings/${encodeURIComponent(recordingID)}/annotations`;
      try {
        await request(path, {method: annotationID ? "PUT" : "POST", body: JSON.stringify(payload)});
        resetForm(payload.start_ms);
        await refreshAnnotations();
      } catch (error) {
        status.textContent = error.message;
      }
    });

    resetForm();
    await refreshAnnotations();
  }

  async function render() {
    let recordings;
    try {
      const response = await fetch("/api/recordings");
      if (!response.ok) return;
      recordings = await response.json();
    } catch {
      return;
    }
    const byID = new Map(recordings.map(recording => [recording.id, recording]));
    list.querySelectorAll("li").forEach(item => {
      const button = item.querySelector("[data-edit-recording], [data-delete-recording]");
      if (!button) return;
      const id = button.dataset.editRecording || button.dataset.deleteRecording;
      const recording = byID.get(id);
      if (recording) attach(item, recording);
    });
  }

  new MutationObserver(render).observe(list, {childList: true, subtree: true});
  render();
})();
