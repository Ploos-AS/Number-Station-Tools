(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  function formatTime(seconds) {
    if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
    const whole = Math.floor(seconds);
    return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, "0")}`;
  }

  function durationFor(audio, recording) {
    if (Number.isFinite(audio.duration) && audio.duration > 0) return audio.duration;
    return Number(recording?.duration_ms || 0) / 1000;
  }

  function attach(item, recording) {
    if (!recording?.managed || item.querySelector(".loop-controls")) return;
    const player = item.querySelector(".recording-player");
    const audio = player?.querySelector("audio");
    if (!audio) return;

    let start = null;
    let end = null;
    let enabled = false;

    const controls = document.createElement("div");
    controls.className = "loop-controls";
    controls.innerHTML = '<button type="button" data-loop-start>Set A</button><button type="button" data-loop-end>Set B</button><button type="button" data-loop-toggle disabled aria-pressed="false">Loop off</button><button type="button" data-loop-clear disabled>Clear</button><span class="loop-status">A — · B —</span>';
    player.insertAdjacentElement("afterend", controls);

    const startButton = controls.querySelector("[data-loop-start]");
    const endButton = controls.querySelector("[data-loop-end]");
    const toggleButton = controls.querySelector("[data-loop-toggle]");
    const clearButton = controls.querySelector("[data-loop-clear]");
    const status = controls.querySelector(".loop-status");

    function syncVisuals() {
      const duration = durationFor(audio, recording);
      const startPct = duration > 0 && start !== null ? Math.max(0, Math.min(100, start / duration * 100)) : 0;
      const endPct = duration > 0 && end !== null ? Math.max(0, Math.min(100, end / duration * 100)) : 100;
      item.style.setProperty("--loop-start", `${startPct.toFixed(3)}%`);
      item.style.setProperty("--loop-end", `${endPct.toFixed(3)}%`);
      item.classList.toggle("loop-region-active", start !== null && end !== null && end > start);
    }

    function syncControls() {
      const valid = start !== null && end !== null && end > start;
      if (!valid) enabled = false;
      toggleButton.disabled = !valid;
      clearButton.disabled = start === null && end === null;
      toggleButton.textContent = enabled && valid ? "Loop on" : "Loop off";
      toggleButton.setAttribute("aria-pressed", enabled && valid ? "true" : "false");
      status.textContent = `A ${start === null ? "—" : formatTime(start)} · B ${end === null ? "—" : formatTime(end)}`;
      syncVisuals();
    }

    startButton.addEventListener("click", () => {
      start = audio.currentTime;
      if (end !== null && end <= start) end = null;
      enabled = false;
      syncControls();
    });

    endButton.addEventListener("click", () => {
      end = audio.currentTime;
      if (start !== null && end <= start) start = null;
      enabled = false;
      syncControls();
    });

    toggleButton.addEventListener("click", () => {
      if (start === null || end === null || end <= start) return;
      enabled = !enabled;
      if (enabled && (audio.currentTime < start || audio.currentTime >= end)) audio.currentTime = start;
      syncControls();
    });

    clearButton.addEventListener("click", () => {
      start = null;
      end = null;
      enabled = false;
      syncControls();
    });

    audio.addEventListener("timeupdate", () => {
      if (!enabled || start === null || end === null || end <= start) return;
      if (audio.currentTime >= end || audio.currentTime < start) audio.currentTime = start;
    });

    item.addEventListener("numberstation:seek", event => {
      const seconds = Number(event.detail?.seconds);
      if (!Number.isFinite(seconds)) return;
      if (event.detail?.setLoopStart) start = seconds;
      if (event.detail?.setLoopEnd) end = seconds;
      syncControls();
    });

    syncControls();
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
      attach(item, byID.get(id));
    });
  }

  new MutationObserver(render).observe(list, {childList: true, subtree: true});
  render();
})();
