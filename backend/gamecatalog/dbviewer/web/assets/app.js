document.addEventListener("click", async (event) => {
  const button = event.target.closest(".copy-button");
  if (!button) return;

  const targetID = button.dataset.copyTarget;
  const value = targetID
    ? document.getElementById(targetID)?.textContent
    : button.dataset.copy;
  if (value === undefined) return;

  try {
    await navigator.clipboard.writeText(value);
  } catch {
    button.textContent = "Copy failed";
    return;
  }
  const original = button.textContent;
  button.textContent = "Copied";
  window.setTimeout(() => {
    button.textContent = original;
  }, 1200);
});
