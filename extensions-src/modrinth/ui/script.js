/**
 * Modrinth Browser – UI script
 * Supports: Mods + Modpacks tabs, paginated search, mod install, modpack install.
 */

// ────────────────────────────────────────────────────────────────────────────
// IPC bridge (postMessage ↔ parent sandbox)
// ────────────────────────────────────────────────────────────────────────────
let _reqId = 0;
const _pending = {};

function sendMessage(payload, timeoutMs) {
  return new Promise((resolve, reject) => {
    const id = ++_reqId;
    payload.requestId = id;
    const timer = setTimeout(() => {
      delete _pending[id];
      reject(new Error("IPC timeout after " + (timeoutMs || 15000) + "ms"));
    }, timeoutMs || 15000);
    _pending[id] = { resolve, reject, timer };
    window.parent.postMessage(payload, "*");
  });
}

window.addEventListener("message", (e) => {
  const msg = e.data;
  if (!msg || msg.requestId == null) return;
  const pend = _pending[msg.requestId];
  if (!pend) return;
  clearTimeout(pend.timer);
  delete _pending[msg.requestId];
  if (msg.success === false || msg.error) {
    pend.reject(new Error(msg.error || "Unknown error"));
  } else {
    pend.resolve(msg);
  }
});

// ────────────────────────────────────────────────────────────────────────────
// State
// ────────────────────────────────────────────────────────────────────────────
const PAGE_SIZE = 20;
let currentType = "mod";  // "mod" | "modpack"
let currentPage = 1;
let totalHits = 0;
let lastQuery = "";
let searchTimer = null;
let instances = [];

// ────────────────────────────────────────────────────────────────────────────
// DOM references
// ────────────────────────────────────────────────────────────────────────────
const searchInput     = document.getElementById("searchInput");
const searchBtn       = document.getElementById("searchBtn");
const resultsContainer = document.getElementById("resultsContainer");
const pagination      = document.getElementById("pagination");
const prevBtn         = document.getElementById("prevBtn");
const nextBtn         = document.getElementById("nextBtn");
const pageInfo        = document.getElementById("pageInfo");

// Tab buttons
const tabMods     = document.getElementById("tabMods");
const tabModpacks = document.getElementById("tabModpacks");

// Mod modal
const installModal    = document.getElementById("installModal");
const modalClose      = document.getElementById("modalClose");
const modalCancel     = document.getElementById("modalCancel");
const modalInstall    = document.getElementById("modalInstall");
const modalInstallText = document.getElementById("modalInstallText");
const modalIcon       = document.getElementById("modalIcon");
const modalName       = document.getElementById("modalName");
const modalAuthor     = document.getElementById("modalAuthor");
const versionSelect   = document.getElementById("versionSelect");
const instanceSelect  = document.getElementById("instanceSelect");
const installStatus   = document.getElementById("installStatus");

// Modpack modal
const packModal           = document.getElementById("packModal");
const packModalClose      = document.getElementById("packModalClose");
const packModalCancel     = document.getElementById("packModalCancel");
const packModalInstall    = document.getElementById("packModalInstall");
const packModalInstallText = document.getElementById("packModalInstallText");
const packModalIcon       = document.getElementById("packModalIcon");
const packModalName       = document.getElementById("packModalName");
const packModalAuthor     = document.getElementById("packModalAuthor");
const packVersionSelect   = document.getElementById("packVersionSelect");
const packNameInput       = document.getElementById("packNameInput");
const packInstallStatus   = document.getElementById("packInstallStatus");

// ────────────────────────────────────────────────────────────────────────────
// Tab switching
// ────────────────────────────────────────────────────────────────────────────
function switchTab(type) {
  currentType = type;
  currentPage = 1;
  totalHits = 0;

  tabMods.classList.toggle("active", type === "mod");
  tabModpacks.classList.toggle("active", type === "modpack");

  const q = searchInput.value.trim();
  if (q.length >= 2) {
    search(q);
  } else {
    showPlaceholder(type === "modpack" ? "Search for modpacks above" : "Search for mods above");
  }
}

tabMods.addEventListener("click", () => switchTab("mod"));
tabModpacks.addEventListener("click", () => switchTab("modpack"));

// ────────────────────────────────────────────────────────────────────────────
// Pagination
// ────────────────────────────────────────────────────────────────────────────
function totalPages() {
  return Math.max(1, Math.ceil(totalHits / PAGE_SIZE));
}

function updatePagination() {
  const total = totalPages();
  const hidden = totalHits <= PAGE_SIZE;
  pagination.classList.toggle("hidden", hidden);
  if (!hidden) {
    prevBtn.disabled = currentPage <= 1;
    nextBtn.disabled = currentPage >= total;
    pageInfo.textContent = `Page ${currentPage} of ${total}`;
  }
}

prevBtn.addEventListener("click", () => {
  if (currentPage > 1) {
    currentPage--;
    search(searchInput.value.trim());
  }
});

nextBtn.addEventListener("click", () => {
  if (currentPage < totalPages()) {
    currentPage++;
    search(searchInput.value.trim());
  }
});

// ────────────────────────────────────────────────────────────────────────────
// Search
// ────────────────────────────────────────────────────────────────────────────
searchInput.addEventListener("input", () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    currentPage = 1;
    search(searchInput.value.trim());
  }, 400);
});

searchBtn.addEventListener("click", () => {
  currentPage = 1;
  search(searchInput.value.trim());
});

searchInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    currentPage = 1;
    search(searchInput.value.trim());
  }
});

async function search(query) {
  lastQuery = query;
  if (!query) {
    showPlaceholder(currentType === "modpack" ? "Search for modpacks above" : "Search for mods above");
    pagination.classList.add("hidden");
    return;
  }

  showLoading();
  const offset = (currentPage - 1) * PAGE_SIZE;

  // Build facets for project type
  const facets = `[["project_type:${currentType}"]]`;
  const url = `https://api.modrinth.com/v2/search?query=${encodeURIComponent(query)}&facets=${encodeURIComponent(facets)}&limit=${PAGE_SIZE}&offset=${offset}`;

  try {
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();

    totalHits = data.total_hits || 0;
    updatePagination();

    if (!data.hits || data.hits.length === 0) {
      showEmpty(currentType === "modpack" ? "No modpacks found" : "No mods found");
      return;
    }
    renderResults(data.hits);
  } catch (err) {
    showError("Search failed: " + err.message);
  }
}

// ────────────────────────────────────────────────────────────────────────────
// Rendering
// ────────────────────────────────────────────────────────────────────────────
function renderResults(hits) {
  resultsContainer.innerHTML = "";
  hits.forEach((hit) => {
    const card = document.createElement("div");
    card.className = "mod-card";
    card.dataset.id = hit.project_id;
    card.dataset.slug = hit.slug;
    card.dataset.type = hit.project_type;

    const isModpack = hit.project_type === "modpack";
    const label = hit.latest_version || "";
    const downloads = formatNum(hit.downloads);
    const follows = formatNum(hit.follows);

    card.innerHTML = `
      <div class="card-top">
        <img
          class="card-icon"
          src="${hit.icon_url || ""}"
          alt="${escHtml(hit.title)}"
          onerror="this.style.display='none';this.nextElementSibling.style.display='flex'"
        />
        <div class="card-icon-fallback" style="display:none">${escHtml(hit.title[0] || "?")}</div>
        <div class="card-info">
          <h3 class="card-title">${escHtml(hit.title)}</h3>
          <p class="card-author">by ${escHtml(hit.author)}</p>
        </div>
      </div>
      <p class="card-desc">${escHtml(hit.description || "")}</p>
      <div class="card-meta">
        ${label ? `<span class="card-tag">${escHtml(label)}</span>` : ""}
        <span class="card-stat">⬇ ${downloads}</span>
        <span class="card-stat">♥ ${follows}</span>
        ${isModpack ? `<span class="card-tag pack-tag">Modpack</span>` : ""}
      </div>
      <button class="btn-card-install" data-id="${hit.project_id}" data-type="${hit.project_type}">
        ${isModpack ? "Install Pack" : "Install"}
      </button>
    `;

    const btn = card.querySelector(".btn-card-install");
    btn.addEventListener("click", (ev) => {
      ev.stopPropagation();
      if (hit.project_type === "modpack") {
        openPackModal(hit);
      } else {
        openModModal(hit);
      }
    });

    resultsContainer.appendChild(card);
  });
}

function formatNum(n) {
  if (!n) return "0";
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
  if (n >= 1_000) return (n / 1_000).toFixed(1) + "K";
  return String(n);
}

function escHtml(s) {
  return String(s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

// ────────────────────────────────────────────────────────────────────────────
// State helpers
// ────────────────────────────────────────────────────────────────────────────
function showPlaceholder(msg) {
  resultsContainer.innerHTML = `
    <div class="placeholder">
      <div class="placeholder-icon">🔍</div>
      <p>${escHtml(msg)}</p>
    </div>`;
}

function showLoading() {
  resultsContainer.innerHTML = `
    <div class="placeholder">
      <div class="spinner"></div>
      <p>Searching…</p>
    </div>`;
}

function showEmpty(msg) {
  resultsContainer.innerHTML = `
    <div class="placeholder">
      <div class="placeholder-icon">📭</div>
      <p>${escHtml(msg)}</p>
    </div>`;
}

function showError(msg) {
  resultsContainer.innerHTML = `
    <div class="placeholder error">
      <div class="placeholder-icon">⚠️</div>
      <p>${escHtml(msg)}</p>
    </div>`;
}

// ────────────────────────────────────────────────────────────────────────────
// Mod install modal
// ────────────────────────────────────────────────────────────────────────────
let _modVersions = [];
let _modProjectId = null;

function openModModal(hit) {
  _modProjectId = hit.project_id;
  modalIcon.src = hit.icon_url || "";
  modalIcon.alt = hit.title;
  modalName.textContent = hit.title;
  modalAuthor.textContent = "by " + (hit.author || "");

  versionSelect.innerHTML = `<option disabled selected>Loading versions…</option>`;
  instanceSelect.innerHTML = `<option disabled selected>Loading instances…</option>`;
  installStatus.classList.add("hidden");
  installStatus.textContent = "";
  modalInstall.disabled = false;
  modalInstallText.textContent = "Install";
  installModal.classList.remove("hidden");

  // Fetch versions + instances in parallel
  Promise.all([
    fetch(`https://api.modrinth.com/v2/project/${hit.project_id}/version`).then((r) => r.json()),
    loadInstances(),
  ])
    .then(([versions]) => {
      _modVersions = versions;
      versionSelect.innerHTML = "";
      versions.forEach((v) => {
        const opt = document.createElement("option");
        opt.value = v.id;
        const loaders = (v.loaders || []).join(", ");
        const mc = (v.game_versions || []).slice(-1)[0] || "";
        opt.textContent = `${v.version_number}  [${mc}${loaders ? " · " + loaders : ""}]`;
        versionSelect.appendChild(opt);
      });
    })
    .catch(() => {
      versionSelect.innerHTML = `<option>Failed to load versions</option>`;
    });
}

async function loadInstances() {
  try {
    const res = await sendMessage({ type: "get_instances" });
    instances = res.instances || [];
    instanceSelect.innerHTML = "";
    if (!instances.length) {
      instanceSelect.innerHTML = `<option disabled selected>No instances found</option>`;
      return;
    }
    instances.forEach((inst) => {
      const opt = document.createElement("option");
      opt.value = inst.id;
      opt.textContent = `${inst.name} (${inst.version}${inst.loader !== "Vanilla" ? " · " + inst.loader : ""})`;
      instanceSelect.appendChild(opt);
    });
  } catch {
    instanceSelect.innerHTML = `<option disabled>Could not load instances</option>`;
  }
}

modalClose.addEventListener("click", closeModModal);
modalCancel.addEventListener("click", closeModModal);
installModal.addEventListener("click", (e) => {
  if (e.target === installModal) closeModModal();
});

function closeModModal() {
  installModal.classList.add("hidden");
}

modalInstall.addEventListener("click", async () => {
  const versionId = versionSelect.value;
  const instanceId = instanceSelect.value;
  if (!versionId || !instanceId) return;

  const version = _modVersions.find((v) => v.id === versionId);
  if (!version) return;

  const primaryFile = version.files.find((f) => f.primary) || version.files[0];
  if (!primaryFile) {
    showInstallStatus(installStatus, "error", "No download file found.");
    return;
  }

  modalInstall.disabled = true;
  modalInstallText.textContent = "Installing…";
  showInstallStatus(installStatus, "info", "Downloading mod…");

  try {
    await sendMessage(
      {
        type: "install_mod",
        instanceId,
        jarName: primaryFile.filename,
        downloadUrl: primaryFile.url,
      },
      60000
    );
    showInstallStatus(installStatus, "success", `✓ Installed ${primaryFile.filename}`);
    modalInstallText.textContent = "Installed ✓";
  } catch (err) {
    showInstallStatus(installStatus, "error", "Install failed: " + err.message);
    modalInstall.disabled = false;
    modalInstallText.textContent = "Install";
  }
});

// ────────────────────────────────────────────────────────────────────────────
// Modpack install modal
// ────────────────────────────────────────────────────────────────────────────
let _packVersions = [];

function openPackModal(hit) {
  packModalIcon.src = hit.icon_url || "";
  packModalIcon.alt = hit.title;
  packModalName.textContent = hit.title;
  packModalAuthor.textContent = "by " + (hit.author || "");
  packNameInput.value = hit.title;

  packVersionSelect.innerHTML = `<option disabled selected>Loading versions…</option>`;
  packInstallStatus.classList.add("hidden");
  packInstallStatus.textContent = "";
  packModalInstall.disabled = false;
  packModalInstallText.textContent = "Create & Install";
  packModal.classList.remove("hidden");

  fetch(`https://api.modrinth.com/v2/project/${hit.project_id}/version`)
    .then((r) => r.json())
    .then((versions) => {
      _packVersions = versions;
      packVersionSelect.innerHTML = "";
      versions.forEach((v) => {
        const opt = document.createElement("option");
        opt.value = v.id;
        const mc = (v.game_versions || []).slice(-1)[0] || "";
        const loaders = (v.loaders || []).join(", ");
        opt.textContent = `${v.version_number}  [${mc}${loaders ? " · " + loaders : ""}]`;
        packVersionSelect.appendChild(opt);
      });
    })
    .catch(() => {
      packVersionSelect.innerHTML = `<option>Failed to load versions</option>`;
    });
}

packModalClose.addEventListener("click", closePackModal);
packModalCancel.addEventListener("click", closePackModal);
packModal.addEventListener("click", (e) => {
  if (e.target === packModal) closePackModal();
});

function closePackModal() {
  packModal.classList.add("hidden");
}

packModalInstall.addEventListener("click", async () => {
  const versionId = packVersionSelect.value;
  if (!versionId) return;

  const version = _packVersions.find((v) => v.id === versionId);
  if (!version) return;

  // Find the .mrpack primary file
  const mrpackFile =
    version.files.find((f) => f.primary && f.filename.endsWith(".mrpack")) ||
    version.files.find((f) => f.filename.endsWith(".mrpack")) ||
    version.files[0];

  if (!mrpackFile) {
    showInstallStatus(packInstallStatus, "error", "No .mrpack file found for this version.");
    return;
  }

  const packName = packNameInput.value.trim() || "";
  packModalInstall.disabled = true;
  packModalInstallText.textContent = "Installing…";
  showInstallStatus(
    packInstallStatus,
    "info",
    "Downloading and installing modpack — this may take a few minutes…"
  );

  try {
    // 5 minute timeout for large modpacks
    const res = await sendMessage(
      {
        type: "install_modpack",
        packUrl: mrpackFile.url,
        packName,
      },
      5 * 60 * 1000
    );
    showInstallStatus(
      packInstallStatus,
      "success",
      `✓ Modpack installed! Instance "${res.packName || packName}" is ready.`
    );
    packModalInstallText.textContent = "Installed ✓";
  } catch (err) {
    showInstallStatus(packInstallStatus, "error", "Install failed: " + err.message);
    packModalInstall.disabled = false;
    packModalInstallText.textContent = "Create & Install";
  }
});

// ────────────────────────────────────────────────────────────────────────────
// Shared status helper
// ────────────────────────────────────────────────────────────────────────────
function showInstallStatus(el, type, msg) {
  el.className = "install-status " + type;
  el.textContent = msg;
  el.classList.remove("hidden");
}
