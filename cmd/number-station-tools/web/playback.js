(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  function formatTime(seconds) {
    if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
    const whole = Math.floor(seconds);
    return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, "0")}`;
  }

  function attachPlayer(item, recording) {
    if (!recording?.managed || item.querySelector(".recording-player")) return;
    const actions = item.querySelector(".actions");
    const wrap = document.createElement("div");
    wrap.className = "recording-player";
    const audio = document.createElement("audio");
    audio.controls = true;
    audio.preload = "metadata";
    audio.src = `/api/recordings/${encodeURIComponent(recording.id)}/stream`;
    const status = document.createElement("span");
    status.className = "playback-status";
    status.textContent = `0:00 / ${formatTime((recording.duration_ms || 0) / 1000)}`;
    wrap.append(audio, status);
    item.insertBefore(wrap, actions || null);

    const update = () => {
      const duration = Number.isFinite(audio.duration) && audio.duration > 0 ? audio.duration : (recording.duration_ms || 0) / 1000;
      const progress = duration > 0 ? Math.min(1, Math.max(0, audio.currentTime / duration)) : 0;
      item.style.setProperty("--playback-progress", `${(progress * 100).toFixed(3)}%`);
      status.textContent = `${formatTime(audio.currentTime)} / ${formatTime(duration)}`;
    };
    audio.addEventListener("timeupdate", update);
    audio.addEventListener("loadedmetadata", update);
    audio.addEventListener("seeked", update);
    audio.addEventListener("ended", update);
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
      attachPlayer(item, byID.get(id));
    });
  }

  new MutationObserver(render).observe(list, {childList: true, subtree: true});
  render();
})();
