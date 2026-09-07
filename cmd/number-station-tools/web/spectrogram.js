(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  function renderGrid(recording) {
    const values = recording.spectrogram;
    const rows = Number(recording.spectrogram_frequency_bins || 0);
    const cols = Number(recording.spectrogram_time_bins || 0);
    if (!Array.isArray(values) || !rows || !cols || values.length !== rows * cols) return "";
    const cells = [];
    for (let f = rows - 1; f >= 0; f--) {
      for (let t = 0; t < cols; t++) {
        const value = Number(values[t * rows + f] || 0);
        const alpha = Math.max(0.04, value / 255);
        cells.push(`<i style="opacity:${alpha.toFixed(3)}"></i>`);
      }
    }
    const maxHz = Number(recording.spectrogram_max_hz || 0);
    const label = maxHz >= 1000 ? `${(maxHz / 1000).toFixed(maxHz % 1000 ? 1 : 0)} kHz` : `${maxHz} Hz`;
    return `<div class="spectrogram-labels"><span>${label}</span><span>time →</span><span>0 Hz</span></div><div class="spectrogram-preview" style="grid-template-columns:repeat(${cols},1fr)" role="img" aria-label="Mini spectrogram, time left to right and frequency bottom to top">${cells.join("")}</div>`;
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
      if (item.querySelector(".spectrogram-preview")) return;
      const button = item.querySelector("[data-edit-recording], [data-delete-recording]");
      if (!button) return;
      const id = button.dataset.editRecording || button.dataset.deleteRecording;
      const recording = byID.get(id);
      if (!recording?.spectrogram?.length) return;
      const actions = item.querySelector(".actions");
      const wrapper = document.createElement("div");
      wrapper.className = "spectrogram-wrap";
      wrapper.innerHTML = renderGrid(recording);
      item.insertBefore(wrapper, actions || null);
    });
  }

  const observer = new MutationObserver(render);
  observer.observe(list, {childList: true, subtree: true});
  render();
})();
