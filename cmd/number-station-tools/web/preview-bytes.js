(() => {
  function bytes(value) {
    if (Array.isArray(value)) return value.map(Number);
    if (typeof value !== "string" || value === "") return [];
    try {
      const binary = atob(value);
      return Array.from(binary, ch => ch.charCodeAt(0));
    } catch {
      return [];
    }
  }
  window.NumberStationPreview = {bytes};
})();
