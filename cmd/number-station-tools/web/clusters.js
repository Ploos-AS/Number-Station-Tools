(() => {
  const list = document.querySelector("#recording-clusters");
  const status = document.querySelector("#cluster-status");
  const refresh = document.querySelector("#cluster-refresh");
  const threshold = document.querySelector("#cluster-threshold");
  if (!list || !status || !refresh || !threshold) return;

  const classifications = [
    ["same transmission", "Same transmission"],
    ["same station", "Same station"],
    ["false positive", "False positive"],
    ["duplicate capture", "Duplicate capture"],
  ];

  async function saveReview(cluster, select, notes, message) {
    message.textContent = "Saving review…";
    const value = Number(threshold.value) || 98;
    try {
      const response = await fetch(`/api/recording-clusters/${encodeURIComponent(cluster.id)}/review?threshold=${encodeURIComponent(value)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ classification: select.value, notes: notes.value }),
      });
      if (!response.ok) throw new Error(await response.text());
      message.textContent = "Review saved locally.";
      await loadClusters();
    } catch (error) {
      message.textContent = `Review save failed: ${error.message}`;
    }
  }

  async function clearReview(cluster, message) {
    message.textContent = "Clearing review…";
    try {
      const response = await fetch(`/api/recording-clusters/${encodeURIComponent(cluster.id)}/review`, { method: "DELETE" });
      if (!response.ok) throw new Error(await response.text());
      await loadClusters();
    } catch (error) {
      message.textContent = `Review clear failed: ${error.message}`;
    }
  }

  function addReviewControls(item, cluster) {
    const form = document.createElement("div");
    form.className = "cluster-review";

    const select = document.createElement("select");
    for (const [value, label] of classifications) {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = label;
      select.appendChild(option);
    }
    select.value = cluster.review?.classification || "same transmission";

    const notes = document.createElement("textarea");
    notes.rows = 2;
    notes.maxLength = 2000;
    notes.placeholder = "Review notes…";
    notes.value = cluster.review?.notes || "";

    const actions = document.createElement("div");
    actions.className = "actions";
    const save = document.createElement("button");
    save.type = "button";
    save.textContent = cluster.review ? "Update review" : "Save review";
    const clear = document.createElement("button");
    clear.type = "button";
    clear.textContent = "Clear review";
    clear.hidden = !cluster.review;
    const message = document.createElement("span");
    message.className = "status";

    save.addEventListener("click", () => saveReview(cluster, select, notes, message));
    clear.addEventListener("click", () => clearReview(cluster, message));
    actions.append(save, clear, message);
    form.append(select, notes, actions);
    item.appendChild(form);
  }

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
        const reviewLabel = cluster.review ? ` · reviewed: ${cluster.review.classification}` : " · unreviewed";
        item.innerHTML = `<strong>${label}</strong> · ${cluster.members.length} recordings · minimum pair score ${cluster.minimum_score.toFixed(2)}%${reviewLabel}`;
        const detail = document.createElement("div");
        detail.className = "status";
        detail.textContent = cluster.members.map(member => member.path).join(" | ");
        item.appendChild(detail);
        if (cluster.review?.reviewed_at) {
          const reviewed = document.createElement("div");
          reviewed.className = "status";
          reviewed.textContent = `Reviewed ${cluster.review.reviewed_at}`;
          item.appendChild(reviewed);
        }
        addReviewControls(item, cluster);
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
