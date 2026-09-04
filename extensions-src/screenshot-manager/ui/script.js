// ── IPC Bridge ────────────────────────────────────────────────────────────

const pending = {};
let reqCounter = 0;

function sendMessage(payload) {
  return new Promise((resolve) => {
    const id = ++reqCounter;
    payload.requestId = id;
    pending[id] = resolve;
    window.parent.postMessage(payload, '*');
  });
}

window.addEventListener('message', (e) => {
  const msg = e.data;
  if (!msg || !msg.requestId) return;
  const resolve = pending[msg.requestId];
  if (resolve) {
    delete pending[msg.requestId];
    resolve(msg);
  }
});


// ── State ─────────────────────────────────────────────────────────────────

let instances = [];
let selectedInstanceId = '';
let screenshots = [];
let currentLightboxIndex = -1;


// ── DOM Elements ──────────────────────────────────────────────────────────

const instanceSelect   = document.getElementById('instanceSelect');
const refreshBtn       = document.getElementById('refreshBtn');
const countBadge       = document.getElementById('countBadge');
const galleryContainer = document.getElementById('galleryContainer');

// Lightbox
const lightboxModal    = document.getElementById('lightboxModal');
const lightboxImg      = document.getElementById('lightboxImg');
const lightboxTitle    = document.getElementById('lightboxTitle');
const lightboxMeta     = document.getElementById('lightboxMeta');
const prevImgBtn       = document.getElementById('prevImgBtn');
const nextImgBtn       = document.getElementById('nextImgBtn');
const openExternalBtn  = document.getElementById('openExternalBtn');
const copyBtn          = document.getElementById('copyBtn');
const deleteBtn        = document.getElementById('deleteBtn');
const closeLightboxBtn = document.getElementById('closeLightboxBtn');


// ── Formatters ────────────────────────────────────────────────────────────

function formatSize(bytes) {
  if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB';
  return bytes + ' B';
}

function formatDate(isoStr) {
  if (!isoStr) return '';
  try {
    const d = new Date(isoStr);
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return isoStr;
  }
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}


// ── Load Instances ────────────────────────────────────────────────────────

async function loadInstances() {
  try {
    const res = await sendMessage({ type: 'get_instances' });
    instances = res.instances || [];

    if (instances.length === 0) {
      instanceSelect.innerHTML = '<option value="">No instances found</option>';
      galleryContainer.innerHTML = `
        <div class="placeholder-wrap">
          <div class="placeholder-icon">📦</div>
          <p>No Minecraft instances created yet.</p>
        </div>
      `;
      return;
    }

    instanceSelect.innerHTML = instances
      .map(inst => `<option value="${escapeHtml(inst.id)}">${escapeHtml(inst.name)} (${inst.version})</option>`)
      .join('');

    selectedInstanceId = instances[0].id;
    loadScreenshots();
  } catch (err) {
    console.error('Failed to load instances:', err);
  }
}

instanceSelect.addEventListener('change', () => {
  selectedInstanceId = instanceSelect.value;
  loadScreenshots();
});

refreshBtn.addEventListener('click', () => {
  loadScreenshots();
});


// ── Load Screenshots ──────────────────────────────────────────────────────

async function loadScreenshots() {
  if (!selectedInstanceId) return;

  galleryContainer.innerHTML = `
    <div class="loading">
      <div class="spinner"></div>
      <p>Scanning screenshots...</p>
    </div>
  `;

  try {
    const res = await sendMessage({
      type: 'get_screenshots',
      instanceId: selectedInstanceId
    });

    screenshots = (res.screenshots || []).sort((a, b) => (b.modTime || '').localeCompare(a.modTime || ''));
    countBadge.textContent = `${screenshots.length} screenshot${screenshots.length === 1 ? '' : 's'}`;

    if (screenshots.length === 0) {
      galleryContainer.innerHTML = `
        <div class="placeholder-wrap">
          <div class="placeholder-icon">📷</div>
          <p>No screenshots taken in this instance yet.</p>
          <p style="font-size:12px; color:#666;">Press F2 in Minecraft to capture screenshots!</p>
        </div>
      `;
      return;
    }

    galleryContainer.innerHTML = `
      <div class="grid">
        ${screenshots.map((item, idx) => `
          <div class="screenshot-card" data-index="${idx}">
            <div class="thumb-wrap">
              <img class="thumb-img" src="${item.url}" alt="${escapeHtml(item.name)}" loading="lazy" />
            </div>
            <div class="card-details">
              <div class="card-filename">${escapeHtml(item.name)}</div>
              <div class="card-meta">
                <span>${formatSize(item.size)}</span>
                <span>${formatDate(item.modTime)}</span>
              </div>
            </div>
          </div>
        `).join('')}
      </div>
    `;

    document.querySelectorAll('.screenshot-card').forEach(card => {
      card.addEventListener('click', () => {
        const idx = parseInt(card.dataset.index, 10);
        openLightbox(idx);
      });
    });

  } catch (err) {
    console.error('Failed to load screenshots:', err);
    galleryContainer.innerHTML = `
      <div class="placeholder-wrap">
        <p>Error loading screenshots: ${escapeHtml(err.message)}</p>
      </div>
    `;
  }
}


// ── Lightbox Modal ────────────────────────────────────────────────────────

function openLightbox(index) {
  if (index < 0 || index >= screenshots.length) return;
  currentLightboxIndex = index;
  const item = screenshots[index];

  lightboxImg.src = item.url;
  lightboxTitle.textContent = item.name;
  lightboxMeta.textContent = `${formatSize(item.size)} · ${formatDate(item.modTime)}`;

  prevImgBtn.disabled = index === 0;
  nextImgBtn.disabled = index === screenshots.length - 1;

  lightboxModal.classList.remove('hidden');
}

function closeLightbox() {
  lightboxModal.classList.add('hidden');
  lightboxImg.src = '';
  currentLightboxIndex = -1;
}

closeLightboxBtn.addEventListener('click', closeLightbox);
lightboxModal.addEventListener('click', (e) => {
  if (e.target === lightboxModal) closeLightbox();
});

prevImgBtn.addEventListener('click', () => {
  if (currentLightboxIndex > 0) openLightbox(currentLightboxIndex - 1);
});

nextImgBtn.addEventListener('click', () => {
  if (currentLightboxIndex < screenshots.length - 1) openLightbox(currentLightboxIndex + 1);
});

window.addEventListener('keydown', (e) => {
  if (lightboxModal.classList.contains('hidden')) return;
  if (e.key === 'ArrowLeft' && currentLightboxIndex > 0) openLightbox(currentLightboxIndex - 1);
  if (e.key === 'ArrowRight' && currentLightboxIndex < screenshots.length - 1) openLightbox(currentLightboxIndex + 1);
  if (e.key === 'Escape') closeLightbox();
});

// Actions
openExternalBtn.addEventListener('click', async () => {
  if (currentLightboxIndex < 0) return;
  const item = screenshots[currentLightboxIndex];
  await sendMessage({
    type: 'open_screenshot',
    instanceId: selectedInstanceId,
    fileName: item.name
  });
});

copyBtn.addEventListener('click', async () => {
  if (currentLightboxIndex < 0) return;
  const item = screenshots[currentLightboxIndex];
  try {
    const response = await fetch(item.url);
    const blob = await response.blob();
    await navigator.clipboard.write([
      new ClipboardItem({ [blob.type]: blob })
    ]);
    copyBtn.textContent = '✓ Copied!';
    setTimeout(() => { copyBtn.textContent = '📋 Copy'; }, 2000);
  } catch (err) {
    console.error('Clipboard copy failed:', err);
  }
});

deleteBtn.addEventListener('click', async () => {
  if (currentLightboxIndex < 0) return;
  const item = screenshots[currentLightboxIndex];
  const confirmed = confirm(`Are you sure you want to delete "${item.name}"?`);
  if (!confirmed) return;

  const res = await sendMessage({
    type: 'delete_screenshot',
    instanceId: selectedInstanceId,
    fileName: item.name
  });

  if (res.success) {
    closeLightbox();
    loadScreenshots();
  } else {
    alert(`Failed to delete screenshot: ${res.error}`);
  }
});


// ── Initialize ────────────────────────────────────────────────────────────

loadInstances();
