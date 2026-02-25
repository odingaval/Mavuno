import { triggerSync } from './sync.js';

// Example: user clicks "Upload" while offline
document.getElementById('uploadBtn').addEventListener('click', async () => {
  // Here you would normally save to IndexedDB (skipped for now)
  console.log('Pretend we queued a new item');

  // Trigger SW sync
  await triggerSync();
});

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js')
    .then(() => console.log('Service Worker registered'))
    .catch(err => console.error('SW registration failed:', err));
}
