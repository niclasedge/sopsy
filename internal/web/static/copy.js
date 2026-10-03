// The only script: copies the text of the element a data-copy button names.
document.addEventListener('click', (event) => {
  const button = event.target.closest('button[data-copy]');
  if (!button) return;
  const text = document.getElementById(button.dataset.copy).textContent.trim();
  navigator.clipboard.writeText(text).then(() => { button.textContent = 'Copied'; });
});
