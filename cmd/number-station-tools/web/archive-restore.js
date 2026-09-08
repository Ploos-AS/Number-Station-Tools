(() => {
  const form = document.querySelector("#recording-archive-import-form");
  const input = document.querySelector("#recording-archive-import-file");
  const status = document.querySelector("#recording-archive-import-status");
  const planButton = document.querySelector("#recording-archive-plan");
  const results = document.querySelector("#recording-archive-plan-results");
  if (!form || !input || !status || !planButton || !results) return;

  let selectedButton = document.querySelector("#recording-archive-restore-selected");
  if (!selectedButton) {
    selectedButton = document.createElement("button");
    selectedButton.type = "button";
    selectedButton.id = "recording-archive-restore-selected";
    selectedButton.textContent = "Restore selected";
    selectedButton.disabled = true;
    planButton.insertAdjacentElement("afterend", selectedButton);
  }

  const selectedFile = () => input.files?.[0] || null;

  const clearPlan = () => {
    results.replaceChildren();
    selectedButton.disabled = true;
  };

  const renderPlan = plan => {
    results.replaceChildren();
    let selectable = 0;
    for (const entry of plan.entries || []) {
      const li = document.createElement("li");
      const reason = entry.reason ? ` · ${entry.reason}` : "";
      if (entry.action === "import") {
        const label = document.createElement("label");
        const checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        checkbox.checked = true;
        checkbox.dataset.restoreRecordingId = entry.recording_id;
        label.append(checkbox, ` IMPORT · ${entry.recording_id} · ${entry.path} · annotations +${entry.annotations_import}/skip ${entry.annotations_skip}`);
        li.append(label);
        selectable++;
      } else {
        li.textContent = `${entry.action.toUpperCase()} · ${entry.recording_id} · ${entry.path} · annotations +${entry.annotations_import}/skip ${entry.annotations_skip}${reason}`;
      }
      results.append(li);
    }
    selectedButton.disabled = selectable === 0;
    status.textContent = `Plan: ${plan.import} import, ${plan.duplicate} duplicate, ${plan.conflict} conflict, ${plan.unmatched} unmatched.`;
  };

  input.addEventListener("change", clearPlan);

  planButton.addEventListener("click", async () => {
    const file = selectedFile();
    if (!file) return;
    status.textContent = "Planning restore…";
    clearPlan();
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

  selectedButton.addEventListener("click", async () => {
    const file = selectedFile();
    if (!file) return;
    const ids = [...results.querySelectorAll("[data-restore-recording-id]:checked")].map(node => node.dataset.restoreRecordingId);
    if (ids.length === 0) {
      status.textContent = "Select at least one import candidate from the restore plan.";
      return;
    }
    const query = new URLSearchParams();
    for (const id of ids) query.append("recording_id", id);
    status.textContent = `Restoring ${ids.length} selected recording${ids.length === 1 ? "" : "s"}…`;
    try {
      const response = await fetch(`/api/recording-archive/import-selected?${query}`, {
        method: "POST",
        headers: {"Content-Type": "application/gzip"},
        body: file,
      });
      const text = await response.text();
      if (!response.ok) throw new Error(text.trim() || "Selected archive restore failed");
      const result = JSON.parse(text);
      status.textContent = `Selected restore imported ${result.imported_recordings} recordings and ${result.imported_annotations} annotations; ${result.skipped_by_policy} archive entries skipped by policy.`;
      clearPlan();
      input.value = "";
      window.dispatchEvent(new CustomEvent("numberstation:archive-restored", {detail: result}));
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
      clearPlan();
      input.value = "";
      window.dispatchEvent(new CustomEvent("numberstation:archive-restored", {detail: result}));
    } catch (error) {
      status.textContent = error.message;
    }
  });
})();
