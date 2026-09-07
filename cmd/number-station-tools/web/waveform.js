(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  function svgFor(rawValues) {
    const values = window.NumberStationPreview?.bytes(rawValues) || [];
    if (!values.length) return "";
    const width = 512;
    const height = 64;
    const mid = height / 2;
    const step = width / values.length;
    const bars = values.map((value, i) => {
      const amplitude = Math.max(1, Math.round((Number(value) / 255) * (height - 4)));
      const x = Math.round(i * step * 100) / 100;
      const y = Math.round((mid - amplitude / 2) * 100) / 100;
      const w = Math.max(1, Math.ceil(step));
      return `<rect x="${x}" y="${y}" width="${w}" height="${amplitude}" rx="0.5" fill="currentColor"></rect>`;
    }).join("");
    return `<svg class="waveform-preview" viewBox="0 0 ${width} ${height}" role="img" aria-label="Audio amplitude preview" preserveAspectRatio="none">${bars}</svg>`;
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
      if (item.querySelector(".waveform-preview")) return;
      const button = item.querySelector("[data-edit-recording], [data-delete-recording]");
      if (!button) return;
      const id = button.dataset.editRecording || button.dataset.deleteRecording;
      const recording = byID.get(id);
      const markup = svgFor(recording?.waveform);
      if (!markup) return;
      const actions = item.querySelector(".actions");
      const wrapper = document.createElement("div");
      wrapper.className = "waveform-wrap playback-track";
      wrapper.innerHTML = markup;
      item.insertBefore(wrapper, actions || null);
    });
  }

  new MutationObserver(render).observe(list, {childList: true, subtree: true});
  render();
})();
