(() => {
  const form = document.querySelector("#annotation-import-form");
  const input = document.querySelector("#annotation-import-file");
  const status = document.querySelector("#annotation-import-status");
  if (!form || !input || !status) return;

  form.addEventListener("submit", async event => {
    event.preventDefault();
    const file = input.files?.[0];
    if (!file) {
      status.textContent = "Choose a JSON export first.";
      return;
    }
    if (file.size > 8 * 1024 * 1024) {
      status.textContent = "Import file exceeds 8 MiB.";
      return;
    }
    status.textContent = "Importing…";
    try {
      const text = await file.text();
      JSON.parse(text);
      const response = await fetch("/api/annotations/import", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: text
      });
      const body = await response.text();
      if (!response.ok) throw new Error(body.trim() || "Import failed");
      const result = JSON.parse(body);
      status.textContent = `${result.imported} imported · ${result.duplicates} duplicates · ${result.unmatched} unmatched`;
      input.value = "";
      window.dispatchEvent(new CustomEvent("numberstation:annotations-imported", {detail: result}));
    } catch (error) {
      status.textContent = error.message;
    }
  });
})();
