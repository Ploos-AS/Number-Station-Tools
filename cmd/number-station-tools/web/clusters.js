(() => {
  const list = document.querySelector("#recording-clusters");
  const status = document.querySelector("#cluster-status");
  const refresh = document.querySelector("#cluster-refresh");
  const threshold = document.querySelector("#cluster-threshold");
  if (!list || !status || !refresh || !threshold) return;

  async function loadClusters() {
    const value = Number(threshold.value) || 98;
    status.textContent = "Scanning fingerprints…";
    list.replaceChildren();
    try {
      const response = await fetch(`/api/recording-clusters?threshold=${encodeURIComponent(value)}`);
      if (!response.ok) throw new Error(await response.text());
      const clusters = await response.json();
      status.textContent = clusters.length
        ? `${clusters.length} cluster${clusters.length === 1 ? "" : "s"} found at ≥ ${value}% similarity.`
        : `No clusters found at ≥ ${value}% similarity.`;
      for (const cluster of clusters) {
        const item = document.createElement("li");
        const label = cluster.exact_duplicate ? "Exact file duplicate" : "Repeated / very similar signal";
        item.innerHTML = `<strong>${label}</strong> · ${cluster.members.length} recordings · minimum pair score ${cluster.minimum_score.toFixed(2)}%`;
        const detail = document.createElement("div");
        detail.className = "status";
        detail.textContent = cluster.members.map(member => member.path).join(" | ");
        item.appendChild(detail);
        list.appendChild(item);
      }
    } catch (error) {
      status.textContent = `Cluster scan failed: ${error.message}`;
    }
  }

  refresh.addEventListener("click", loadClusters);
  threshold.addEventListener("change", loadClusters);
  loadClusters();
})();
