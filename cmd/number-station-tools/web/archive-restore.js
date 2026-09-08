(() => {
  const form = document.querySelector("#recording-archive-import-form");
  const input = document.querySelector("#recording-archive-import-file");
  const status = document.querySelector("#recording-archive-import-status");
  const planButton = document.querySelector("#recording-archive-plan");
  const results = document.querySelector("#recording-archive-plan-results");
  if (!form || !input || !status || !planButton || !results) return;

  const selectedFile = () => input.files?.[0] || null;

  const renderPlan = plan => {
    results.replaceChildren();
    for (const entry of plan.entries || []) {
      const li = document.createElement("li");
      const reason = entry.reason ? ` · ${entry.reason}` : "";
      li.textContent = `${entry.action.toUpperCase()} · ${entry.recording_id} · ${entry.path} · annotations +${entry.annotations_import}/skip ${entry.annotations_skip}${reason}`;
      results.append(li);
    }
    status.textContent = `Plan: ${plan.import} import, ${plan.duplicate} duplicate, ${plan.conflict} conflict, ${plan.unmatched} unmatched.`;
  };

  planButton.addEventListener("click", async () => {
    const file = selectedFile();
    if (!file) return;
    status.textContent = "Planning restore…";
    results.replaceChildren();
    try {
      const response = await fetch("/api/recording-archive/plan", {
        method: "POST",
        headers: {"Content-Type": "application/gzip"},
        body: file,
      });
      const text = await response.text();
      if (!response.ok) throw new Error(text.trim() || "Restore planning failed");
      renderPlan(JSON.parse(text));
    } catch (error) {
      status.textContent = error.message;
    }
  });

  form.addEventListener("submit", async event => {
    event.preventDefault();
    const file = selectedFile();
    if (!file) return;
    status.textContent = "Verifying archive…";
    try {
      const response = await fetch("/api/recording-archive/import", {
        method: "POST",
        headers: {"Content-Type": "application/gzip"},
        body: file,
      });
      const text = await response.text();
      if (!response.ok) throw new Error(text.trim() || "Archive restore failed");
      const result = JSON.parse(text);
      status.textContent = `Imported ${result.imported_recordings} recordings and ${result.imported_annotations} annotations; ${result.duplicates} duplicates, ${result.conflicts} conflicts, ${result.unmatched} unmatched observations.`;
      results.replaceChildren();
      input.value = "";
      window.dispatchEvent(new CustomEvent("numberstation:archive-restored", {detail: result}));
    } catch (error) {
      status.textContent = error.message;
    }
  });
})();
