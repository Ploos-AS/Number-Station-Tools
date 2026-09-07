(() => {
  const list = document.querySelector("#recordings");
  if (!list) return;

  async function showSimilar(item, id) {
    let box = item.querySelector(".similar-results");
    if (!box) {
      box = document.createElement("div");
      box.className = "similar-results status";
      item.appendChild(box);
    }
    box.textContent = "Finding similar recordings…";
    try {
      const response = await fetch(`/api/recordings/${encodeURIComponent(id)}/similar?limit=5`);
      if (!response.ok) throw new Error(await response.text());
      const results = await response.json();
      box.textContent = results.length
        ? results.map(result => `${result.score.toFixed(2)}% · ${result.path}`).join(" | ")
        : "No comparable fingerprints yet.";
    } catch (error) {
      box.textContent = `Similarity lookup failed: ${error.message}`;
    }
  }

  function attach() {
    list.querySelectorAll("li").forEach(item => {
      if (item.querySelector("[data-find-similar]")) return;
      const anchor = item.querySelector("[data-edit-recording], [data-delete-recording]");
      if (!anchor) return;
      const id = anchor.dataset.editRecording || anchor.dataset.deleteRecording;
      const actions = item.querySelector(".actions") || item;
      const button = document.createElement("button");
      button.type = "button";
      button.dataset.findSimilar = id;
      button.textContent = "Find similar";
      button.addEventListener("click", () => showSimilar(item, id));
      actions.appendChild(button);
    });
  }

  new MutationObserver(attach).observe(list, { childList: true, subtree: true });
  attach();
})();
