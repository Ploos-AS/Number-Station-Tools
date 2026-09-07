(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  function svgFor(rawValues, maxHz) {
    const values = window.NumberStationPreview?.bytes(rawValues) || [];
    if (!values.length || !maxHz) return "";
    const width = 512;
    const height = 72;
    const step = width / values.length;
    const bars = values.map((value, i) => {
      const amplitude = Math.max(1, Math.round((Number(value) / 255) * (height - 16)));
      const x = Math.round(i * step * 100) / 100;
      const y = height - amplitude - 12;
      const w = Math.max(1, Math.ceil(step));
      return `<rect x="${x}" y="${y}" width="${w}" height="${amplitude}" rx="0.5"></rect>`;
    }).join("");
    const label = maxHz >= 1000 ? `${(maxHz / 1000).toFixed(maxHz % 1000 ? 1 : 0)} kHz` : `${maxHz} Hz`;
    return `<div class="frequency-labels"><span>0 Hz</span><span>${label}</span></div><svg class="frequency-preview" viewBox="0 0 ${width} ${height}" role="img" aria-label="Frequency magnitude preview from zero to Nyquist" preserveAspectRatio="none">${bars}</svg>`;
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
      if (item.querySelector(".frequency-preview")) return;
      const button = item.querySelector("[data-edit-recording], [data-delete-recording]");
      if (!button) return;
      const id = button.dataset.editRecording || button.dataset.deleteRecording;
      const recording = byID.get(id);
      const markup = svgFor(recording?.frequency, recording?.frequency_max_hz);
      if (!markup) return;
      const actions = item.querySelector(".actions");
      const wrapper = document.createElement("div");
      wrapper.className = "frequency-wrap";
      wrapper.innerHTML = markup;
      item.insertBefore(wrapper, actions || null);
    });
  }

  new MutationObserver(render).observe(list, {childList: true, subtree: true});
  render();
})();
