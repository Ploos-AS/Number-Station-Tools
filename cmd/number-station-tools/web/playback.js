(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  function formatTime(seconds) {
    if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
    const whole = Math.floor(seconds);
    return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, "0")}`;
  }

  function durationFor(audio, recording) {
    return Number.isFinite(audio.duration) && audio.duration > 0
      ? audio.duration
      : (recording.duration_ms || 0) / 1000;
  }

  function seekFromPointer(event, track, audio, recording) {
    const duration = durationFor(audio, recording);
    if (!(duration > 0)) return;
    const rect = track.getBoundingClientRect();
    if (!(rect.width > 0)) return;
    const ratio = Math.min(1, Math.max(0, (event.clientX - rect.left) / rect.width));
    audio.currentTime = ratio * duration;
  }

  function makeTrackSeekable(track, audio, recording) {
    if (!track || track.dataset.seekable === "true") return;
    track.dataset.seekable = "true";
    track.classList.add("interactive-seek");
    track.tabIndex = 0;
    track.setAttribute("role", "slider");
    track.setAttribute("aria-label", "Recording position");
    track.setAttribute("aria-valuemin", "0");
    track.setAttribute("aria-valuemax", "100");
    track.setAttribute("aria-valuenow", "0");

    let dragging = false;
    track.addEventListener("pointerdown", event => {
      dragging = true;
      track.setPointerCapture?.(event.pointerId);
      seekFromPointer(event, track, audio, recording);
    });
    track.addEventListener("pointermove", event => {
      if (dragging) seekFromPointer(event, track, audio, recording);
    });
    const stopDragging = event => {
      if (!dragging) return;
      dragging = false;
      track.releasePointerCapture?.(event.pointerId);
    };
    track.addEventListener("pointerup", stopDragging);
    track.addEventListener("pointercancel", stopDragging);

    track.addEventListener("keydown", event => {
      const duration = durationFor(audio, recording);
      if (!(duration > 0)) return;
      const step = event.shiftKey ? 10 : 1;
      if (event.key === "ArrowLeft") {
        event.preventDefault();
        audio.currentTime = Math.max(0, audio.currentTime - step);
      } else if (event.key === "ArrowRight") {
        event.preventDefault();
        audio.currentTime = Math.min(duration, audio.currentTime + step);
      } else if (event.key === "Home") {
        event.preventDefault();
        audio.currentTime = 0;
      } else if (event.key === "End") {
        event.preventDefault();
        audio.currentTime = duration;
      }
    });
  }

  function attachSeeking(item, audio, recording) {
    item.querySelectorAll(".waveform-preview, .spectrogram-preview").forEach(track => {
      makeTrackSeekable(track, audio, recording);
    });
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
      const duration = durationFor(audio, recording);
      const progress = duration > 0 ? Math.min(1, Math.max(0, audio.currentTime / duration)) : 0;
      item.style.setProperty("--playback-progress", `${(progress * 100).toFixed(3)}%`);
      status.textContent = `${formatTime(audio.currentTime)} / ${formatTime(duration)}`;
      item.querySelectorAll(".interactive-seek").forEach(track => {
        track.setAttribute("aria-valuenow", (progress * 100).toFixed(1));
        track.setAttribute("aria-valuetext", `${formatTime(audio.currentTime)} of ${formatTime(duration)}`);
      });
      attachSeeking(item, audio, recording);
    };
    audio.addEventListener("timeupdate", update);
    audio.addEventListener("loadedmetadata", update);
    audio.addEventListener("seeked", update);
    audio.addEventListener("ended", update);
    attachSeeking(item, audio, recording);
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
      attachPlayer(item, recording);
      const audio = item.querySelector(".recording-player audio");
      if (audio && recording) attachSeeking(item, audio, recording);
    });
  }

  new MutationObserver(render).observe(list, {childList: true, subtree: true});
  render();
})();
