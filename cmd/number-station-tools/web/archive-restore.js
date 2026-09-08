(() => {
  const form = document.querySelector("#recording-archive-import-form");
  const input = document.querySelector("#recording-archive-import-file");
  const status = document.querySelector("#recording-archive-import-status");
  if (!form || !input || !status) return;

  form.addEventListener("submit", async event => {
    event.preventDefault();
    const file = input.files?.[0];
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
      input.value = "";
      window.dispatchEvent(new CustomEvent("numberstation:archive-restored", {detail: result}));
    } catch (error) {
      status.textContent = error.message;
    }
  });
})();
