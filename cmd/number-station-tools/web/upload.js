const audioUploadForm = document.querySelector("#audio-upload-form");
const audioObservationSelect = document.querySelector("#audio-observation-select");
const audioUploadStatus = document.querySelector("#audio-upload-status");

async function refreshAudioObservationOptions() {
  try {
    const response = await fetch("/api/observations");
    if (!response.ok) throw new Error("Could not load observations");
    const items = await response.json();
    audioObservationSelect.innerHTML = items.length
      ? items.map(o => `<option value="${escapeHTML(o.id)}">${escapeHTML(observationName(o.id))}</option>`).join("")
      : '<option value="">Add an observation first</option>';
  } catch (error) {
    audioUploadStatus.textContent = error.message;
  }
}

audioUploadForm.addEventListener("submit", async event => {
  event.preventDefault();
  audioUploadStatus.textContent = "Uploading…";
  try {
    const response = await fetch("/api/audio", {
      method: "POST",
      body: new FormData(audioUploadForm)
    });
    const text = await response.text();
    if (!response.ok) throw new Error(text.trim() || "Upload failed");
    const rec = JSON.parse(text);
    audioUploadStatus.textContent = `Stored ${rec.original_name || rec.path} · ${rec.size_bytes} bytes · SHA-256 ${rec.sha256}`;
    audioUploadForm.reset();
    await refresh();
    await refreshAudioObservationOptions();
  } catch (error) {
    audioUploadStatus.textContent = error.message;
  }
});

refreshAudioObservationOptions();
