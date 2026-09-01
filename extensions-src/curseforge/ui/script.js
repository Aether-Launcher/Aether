// ── Custom Select (replaces native <select>) ──────────────────────────────

function initCustomSelect(containerEl) {
    const trigger = containerEl.querySelector('.custom-select-trigger');
    const optionsEl = containerEl.querySelector('.custom-select-options');
    const textEl = containerEl.querySelector('.custom-select-text');
    const chevron = containerEl.querySelector('.custom-select-chevron');

    function close() {
        optionsEl.classList.add('hidden');
        trigger.classList.remove('open');
        chevron.classList.remove('open');
    }

    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        const isOpening = optionsEl.classList.contains('hidden');

        document.querySelectorAll('.custom-select-options').forEach(o => {
            if (o !== optionsEl) o.classList.add('hidden');
        });

        document.querySelectorAll('.custom-select-trigger').forEach(t => {
            if (t !== trigger) t.classList.remove('open');
        });

        document.querySelectorAll('.custom-select-chevron').forEach(c => {
            if (c !== chevron) c.classList.remove('open');
        });

        if (isOpening) {
            optionsEl.classList.remove('hidden');
            trigger.classList.add('open');
            chevron.classList.add('open');
        } else {
            close();
        }
    });

    document.addEventListener('click', () => close(), { once: false });

    return {
        container: containerEl,
        trigger,
        optionsEl,
        textEl,

        setOptions(options) {
            optionsEl.innerHTML = options.map((opt, i) =>
                `<div class="custom-select-option" data-index="${i}">${escapeHtml(opt.label)}</div>`
            ).join('');

            containerEl._optionValues = options.map(o => o.value);
            containerEl._selectedIndex = -1;

            optionsEl.querySelectorAll('.custom-select-option').forEach(el => {
                el.addEventListener('click', () => {
                    const idx = parseInt(el.dataset.index, 10);

                    containerEl._selectedIndex = idx;
                    textEl.textContent = options[idx].label;
                    textEl.classList.add('selected');

                    optionsEl
                        .querySelectorAll('.custom-select-option')
                        .forEach(o => o.classList.remove('selected'));

                    el.classList.add('selected');
                    close();

                    containerEl.dispatchEvent(
                        new CustomEvent('change', {
                            detail: {
                                value: options[idx].value,
                                index: idx
                            }
                        })
                    );
                });
            });

            if (options.length > 0) {
                containerEl._selectedIndex = 0;
                textEl.textContent = options[0].label;
                textEl.classList.add('selected');

                optionsEl
                    .querySelector('.custom-select-option')
                    ?.classList.add('selected');
            } else {
                textEl.textContent = 'No options';
                textEl.classList.remove('selected');
            }
        },

        getValue() {
            if (
                containerEl._selectedIndex >= 0 &&
                containerEl._optionValues
            ) {
                return containerEl._optionValues[
                    containerEl._selectedIndex
                ];
            }

            return null;
        },

        close
    };
}


// ── CurseForge API ───────────────────────────────────────────────────────

const CURSEFORGE_PROXY = 'https://curseforge-proxy.cribest7890.workers.dev';

let launcherToken = 'f2269b54edcf1439afa4675c23f8d3aefb3828ef622e3c9741171501cf5e1bfe';

async function loadLauncherToken() {
    if (!launcherToken) {
        throw new Error('Launcher token is empty.');
    }
    return launcherToken;
}

async function curseForgeFetch(path, options = {}) {
    if (!launcherToken) {
        await loadLauncherToken();
    }

    const headers = new Headers(options.headers || {});
    headers.set('X-Launcher-Token', launcherToken);
    headers.set('Accept', 'application/json');

    return fetch(`${CURSEFORGE_PROXY}${path}`, {
        ...options,
        headers
    });
}

async function curseForgeJson(path, options = {}) {
    const res = await curseForgeFetch(path, options);

    if (!res.ok) {
        let message = `CurseForge API Error ${res.status}`;
        try {
            const errorData = await res.json();
            if (errorData?.error) message = errorData.error;
            else if (errorData?.message) message = errorData.message;
        } catch (_) {}

        throw new Error(message);
    }

    return res.json();
}


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

let currentMod = null;
let currentVersions = [];
let currentInstances = [];

// Content Category State
let currentClassId = '6'; // 6=Mods, 4471=Modpacks, 12=Resource Packs, 6552=Shaders

// Pagination state
const PAGE_SIZE = 20;
let currentQuery = '';
let currentPage = 0;
let hasNextPage = false;


// ── Elements ──────────────────────────────────────────────────────────────

const searchInput    = document.getElementById('searchInput');
const resultsDiv     = document.getElementById('resultsContainer');
const modal          = document.getElementById('installModal');
const modalModName   = document.getElementById('modalModName');
const modalModAuthor = document.getElementById('modalModAuthor');
const modalModIcon   = document.getElementById('modalModIcon');
const versionSelect  = initCustomSelect(document.getElementById('versionSelect'));
const instanceSelect = initCustomSelect(document.getElementById('instanceSelect'));
const installBtn     = document.getElementById('installBtn');
const installBtnText = document.getElementById('installBtnText');
const cancelBtn      = document.getElementById('cancelBtn');
const modalClose     = document.getElementById('modalClose');
const installStatus  = document.getElementById('installStatus');

// Modpack Modal Elements
const packModal          = document.getElementById('packModal');
const packModalName   = document.getElementById('packModalName');
const packModalAuthor = document.getElementById('packModalAuthor');
const packModalIcon   = document.getElementById('packModalIcon');
const packVersionSelect  = initCustomSelect(document.getElementById('packVersionSelect'));
const packNameInput      = document.getElementById('packNameInput');
const packInstallBtn     = document.getElementById('packInstallBtn');
const packInstallBtnText = document.getElementById('packInstallBtnText');
const packCancelBtn      = document.getElementById('packCancelBtn');
const packModalClose     = document.getElementById('packModalClose');
const packInstallStatus  = document.getElementById('packInstallStatus');

const paginationBar  = document.getElementById('paginationBar');
const prevPageBtn    = document.getElementById('prevPageBtn');
const nextPageBtn    = document.getElementById('nextPageBtn');
const pageIndicator  = document.getElementById('pageIndicator');


// ── Tab Event Handlers ────────────────────────────────────────────────────

document.querySelectorAll('.tab').forEach(tab => {
    tab.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        currentClassId = tab.dataset.classid || '6';
        currentPage = 0;
        search(searchInput.value, 0);
    });
});


// ── CurseForge helpers ────────────────────────────────────────────────────

const MOD_LOADER_TYPES = {
    forge: 1,
    cauldron: 2,
    liteloader: 3,
    fabric: 4,
    quilt: 5,
    neoforge: 6
};

function getLoaderType(loader) {
    if (!loader) return null;
    return MOD_LOADER_TYPES[loader.toLowerCase()] ?? null;
}

function normalizeCurseForgeFile(file) {
    return {
        id: file.id,
        version_number: file.displayName || file.fileName || `File ${file.id}`,
        name: file.displayName || file.fileName || '',
        game_versions: file.gameVersions || [],
        loaders: file.modLoader ? [getLoaderName(file.modLoader)] : [],
        files: [{
            filename: file.fileName,
            url: file.downloadUrl,
            primary: true
        }],
        _curseforge: file
    };
}

function getLoaderName(type) {
    switch (type) {
        case 1: return 'forge';
        case 2: return 'cauldron';
        case 3: return 'liteloader';
        case 4: return 'fabric';
        case 5: return 'quilt';
        case 6: return 'neoforge';
        default: return '';
    }
}


// ── Version Filtering ─────────────────────────────────────────────────────

function updateVersionDropdown(inst) {
    if (!inst) {
        versionSelect.setOptions([{ label: 'No instance selected', value: '' }]);
        return;
    }

    const loaderLower = inst.loader.toLowerCase();
    const instVer = inst.version;

    let filtered = currentVersions.filter(v => {
        const gameMatch = v.game_versions && v.game_versions.includes(instVer);
        if (!gameMatch) return false;

        if (loaderLower === 'vanilla') {
            const hasModLoader = (v.loaders || []).some(l => ['fabric', 'forge', 'neoforge', 'quilt'].includes(l));
            return !hasModLoader;
        }

        const vLoaders = (v.loaders || []).map(l => l.toLowerCase());
        return vLoaders.includes(loaderLower) || vLoaders.length === 0;
    });

    let warning = '';
    if (filtered.length === 0) {
        filtered = currentVersions.filter(v => v.game_versions?.includes(instVer));
        if (filtered.length > 0) {
            warning = ' (Loader mismatch)';
        } else {
            filtered = currentVersions;
            warning = ' (Incompatible version)';
        }
    }

    versionSelect.setOptions(
        filtered.map(v => {
            const origIdx = currentVersions.indexOf(v);
            const compatLabel = v.game_versions?.includes(instVer) ? '' : '⚠️ ';
            return {
                label: `${compatLabel}${v.version_number} — ${v.name} (${(v.game_versions || []).slice(0, 3).join(', ')})${warning}`,
                value: String(origIdx)
            };
        })
    );
}

document.getElementById('instanceSelect').addEventListener('change', (e) => {
    const instId = e.detail.value;
    const inst = currentInstances.find(i => i.id === instId);
    if (inst) updateVersionDropdown(inst);
});


// ── Search & Browse ───────────────────────────────────────────────────────

async function search(query = '', page = 0) {
    currentQuery = query.trim();
    currentPage = page;

    resultsDiv.innerHTML = `
        <div class="loading">
            <div class="spinner"></div>
            <p>${currentQuery ? 'Searching CurseForge...' : 'Loading popular content...'}</p>
        </div>
    `;

    paginationBar.classList.add('hidden');

    try {
        const params = new URLSearchParams({
            gameId: '432',
            classId: currentClassId,
            pageSize: String(PAGE_SIZE),
            index: String(page * PAGE_SIZE)
        });

        if (currentQuery) {
            params.set('searchFilter', currentQuery);
            params.set('sortField', '2'); // Popularity
            params.set('sortOrder', 'desc');
        } else {
            params.set('sortField', '6'); // TotalDownloads
            params.set('sortOrder', 'desc');
        }

        const data = await curseForgeJson(`/v1/mods/search?${params.toString()}`);
        const mods = data.data || [];
        const pagination = data.pagination || {};

        if (!mods.length) {
            resultsDiv.innerHTML = `
                <div class="placeholder-wrap">
                    <div class="placeholder-icon">📦</div>
                    <p>No content found.</p>
                </div>
            `;
            return;
        }

        const resultCount = pagination.resultCount ?? mods.length;
        const totalCount = pagination.totalCount ?? (page * PAGE_SIZE + resultCount);
        hasNextPage = (page * PAGE_SIZE + resultCount) < totalCount;

        updatePaginationBar();

        resultsDiv.innerHTML =
            '<div class="grid">' +
            mods.map(mod => {
                const icon = mod.logo?.url
                    ? `<img src="${mod.logo.url}" class="card-icon" alt="" />`
                    : `<div class="card-icon card-icon-placeholder">${escapeHtml((mod.name || '?').charAt(0))}</div>`;

                const dl = mod.downloadCount >= 1000000
                    ? (mod.downloadCount / 1000000).toFixed(1) + 'M'
                    : mod.downloadCount >= 1000
                        ? (mod.downloadCount / 1000).toFixed(0) + 'K'
                        : mod.downloadCount;

                const title = mod.name || 'Unknown';
                const author = mod.authors?.[0]?.name || 'Unknown';
                const description = mod.summary || 'No description available.';
                const iconUrl = mod.logo?.url || '';

                let badge = '';
                let btnLabel = 'Install';
                if (currentClassId === '4471') {
                    badge = '<span class="card-tag pack-tag">Modpack</span>';
                    btnLabel = 'Install Pack';
                } else if (currentClassId === '12') {
                    badge = '<span class="card-tag respack-tag">Resource Pack</span>';
                } else if (currentClassId === '6552') {
                    badge = '<span class="card-tag shader-tag">Shader</span>';
                }

                return `
                    <div
                        class="card"
                        data-id="${mod.id}"
                        data-title="${encodeURIComponent(title)}"
                        data-author="${encodeURIComponent(author)}"
                        data-icon="${encodeURIComponent(iconUrl)}"
                    >
                        <div class="card-top">
                            ${icon}
                            <div class="card-info">
                                <div class="card-title">${escapeHtml(title)}</div>
                                <div class="card-author">by ${escapeHtml(author)}</div>
                            </div>
                        </div>

                        <p class="card-desc">${escapeHtml(description)}</p>

                        <div class="card-footer">
                            <span class="card-downloads">
                                ⬇ ${dl} ${badge}
                            </span>
                            <button class="btn-install-card" data-id="${mod.id}">
                                ${btnLabel}
                            </button>
                        </div>
                    </div>
                `;
            }).join('') +
            '</div>';

        // Attach install handlers
        document.querySelectorAll('.btn-install-card').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                const card = btn.closest('.card');
                const hit = {
                    id: card.dataset.id,
                    title: decodeURIComponent(card.dataset.title),
                    author: decodeURIComponent(card.dataset.author),
                    icon: decodeURIComponent(card.dataset.icon)
                };

                if (currentClassId === '4471') {
                    openPackModal(hit);
                } else {
                    openInstallModal(hit);
                }
            });
        });

    } catch (err) {
        console.error('CurseForge search error:', err);
        resultsDiv.innerHTML = `
            <div class="placeholder-wrap">
                <p>Error: ${escapeHtml(err.message)}</p>
            </div>
        `;
    }
}


// ── Pagination ────────────────────────────────────────────────────────────

function updatePaginationBar() {
    paginationBar.classList.remove('hidden');
    pageIndicator.textContent = `Page ${currentPage + 1}`;
    prevPageBtn.disabled = currentPage === 0;
    nextPageBtn.disabled = !hasNextPage;
}

prevPageBtn.addEventListener('click', () => {
    if (currentPage === 0) return;
    search(currentQuery, currentPage - 1);
    resultsDiv.scrollTop = 0;
});

nextPageBtn.addEventListener('click', () => {
    if (!hasNextPage) return;
    search(currentQuery, currentPage + 1);
    resultsDiv.scrollTop = 0;
});


// ── HTML escaping ─────────────────────────────────────────────────────────

function escapeHtml(value) {
    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#039;');
}


// ── Debounce search ───────────────────────────────────────────────────────

let debounceTimer;

searchInput.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
        search(searchInput.value, 0);
    }, 400);
});

searchInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        clearTimeout(debounceTimer);
        search(searchInput.value, 0);
    }
});


// ── Mod / Resource Pack / Shader Install Modal ────────────────────────────

async function openInstallModal(mod) {
    currentMod = mod;
    currentVersions = [];
    currentInstances = [];

    modalModName.textContent = mod.title;
    modalModAuthor.textContent = 'by ' + mod.author;

    if (mod.icon) {
        modalModIcon.innerHTML = `<img src="${mod.icon}" alt="" />`;
    } else {
        modalModIcon.innerHTML = `<div class="icon-letter">${escapeHtml(mod.title.charAt(0))}</div>`;
    }

    versionSelect.setOptions([{ label: 'Loading versions...', value: '' }]);
    instanceSelect.setOptions([{ label: 'Loading instances...', value: '' }]);

    installStatus.classList.add('hidden');
    installStatus.textContent = '';
    installBtnText.textContent = 'Install';
    installBtn.disabled = false;

    modal.classList.remove('hidden');

    try {
        const [versionsRes, instancesMsg] = await Promise.all([
            curseForgeJson(`/v1/mods/${encodeURIComponent(mod.id)}/files?pageSize=100`),
            sendMessage({ type: 'get_instances' })
        ]);

        const files = versionsRes.data || [];
        currentVersions = files
            .filter(file => file.downloadUrl && file.fileName)
            .map(normalizeCurseForgeFile);

        currentInstances = instancesMsg.instances || [];

        instanceSelect.setOptions(
            currentInstances.length
                ? currentInstances.map(inst => ({
                    label: `${inst.name} (${inst.version} • ${inst.loader})`,
                    value: inst.id
                }))
                : [{ label: 'No instances found', value: '' }]
        );

        if (currentInstances.length > 0) {
            updateVersionDropdown(currentInstances[0]);
        } else {
            versionSelect.setOptions([{ label: 'No instances available', value: '' }]);
        }

    } catch (err) {
        console.error('CurseForge versions error:', err);
        versionSelect.setOptions([{ label: 'Failed to load versions', value: '' }]);
        instanceSelect.setOptions([{ label: 'Unable to load instances', value: '' }]);
        showStatus(`Error: ${err.message}`, 'error');
    }
}

function closeModal() {
    modal.classList.add('hidden');
    currentMod = null;
}

modalClose.addEventListener('click', closeModal);
cancelBtn.addEventListener('click', closeModal);
modal.addEventListener('click', (e) => {
    if (e.target === modal) closeModal();
});

installBtn.addEventListener('click', async () => {
    const vIdxStr = versionSelect.getValue();
    const instanceId = instanceSelect.getValue();

    if (vIdxStr === null || !instanceId) {
        showStatus('Please select a version and instance.', 'error');
        return;
    }

    const vIdx = parseInt(vIdxStr, 10);
    const version = currentVersions[vIdx];

    if (!version) {
        showStatus('Invalid version selected.', 'error');
        return;
    }

    const file = version.files.find(f => f.primary) || version.files[0];
    if (!file || !file.url) {
        showStatus('No downloadable file found for this version.', 'error');
        return;
    }

    installBtnText.textContent = 'Installing...';
    installBtn.disabled = true;
    installStatus.classList.add('hidden');

    let ipcType = "install_mod";
    if (currentClassId === '12') ipcType = "install_resourcepack";
    else if (currentClassId === '6552') ipcType = "install_shaderpack";

    const result = await sendMessage({
        type: ipcType,
        instanceId,
        jarName: file.filename,
        fileName: file.filename,
        downloadUrl: file.url
    });

    if (result.success) {
        showStatus(`✓ ${file.filename} installed successfully!`, 'success');
        installBtnText.textContent = 'Done!';
    } else {
        showStatus(`✗ ${result.error}`, 'error');
        installBtnText.textContent = 'Install';
        installBtn.disabled = false;
    }
});


// ── Modpack Install Modal ─────────────────────────────────────────────────

let currentPackFiles = [];

async function openPackModal(mod) {
    currentMod = mod;
    currentPackFiles = [];

    packModalName.textContent = mod.title;
    packModalAuthor.textContent = 'by ' + mod.author;
    packNameInput.value = mod.title;

    if (mod.icon) {
        packModalIcon.innerHTML = `<img src="${mod.icon}" alt="" />`;
    } else {
        packModalIcon.innerHTML = `<div class="icon-letter">${escapeHtml(mod.title.charAt(0))}</div>`;
    }

    packVersionSelect.setOptions([{ label: 'Loading versions...', value: '' }]);
    packInstallStatus.classList.add('hidden');
    packInstallStatus.textContent = '';
    packInstallBtnText.textContent = 'Create & Install';
    packInstallBtn.disabled = false;

    packModal.classList.remove('hidden');

    try {
        const versionsRes = await curseForgeJson(`/v1/mods/${encodeURIComponent(mod.id)}/files?pageSize=50`);
        currentPackFiles = (versionsRes.data || []).filter(f => f.downloadUrl);

        if (currentPackFiles.length === 0) {
            packVersionSelect.setOptions([{ label: 'No versions found', value: '' }]);
            return;
        }

        packVersionSelect.setOptions(currentPackFiles.map((f, i) => ({
            label: `${f.displayName || f.fileName} (${(f.gameVersions || []).slice(0, 2).join(', ')})`,
            value: String(i)
        })));

    } catch (err) {
        console.error('CurseForge pack versions error:', err);
        packVersionSelect.setOptions([{ label: 'Failed to load versions', value: '' }]);
        showPackStatus(`Error: ${err.message}`, 'error');
    }
}

function closePackModal() {
    packModal.classList.add('hidden');
    currentMod = null;
}

packModalClose.addEventListener('click', closePackModal);
packCancelBtn.addEventListener('click', closePackModal);
packModal.addEventListener('click', (e) => {
    if (e.target === packModal) closePackModal();
});

packInstallBtn.addEventListener('click', async () => {
    const vIdxStr = packVersionSelect.getValue();
    if (vIdxStr === null) {
        showPackStatus('Please select a modpack version.', 'error');
        return;
    }

    const file = currentPackFiles[parseInt(vIdxStr, 10)];
    if (!file || !file.downloadUrl) {
        showPackStatus('Invalid version file.', 'error');
        return;
    }

    const packName = packNameInput.value.trim() || currentMod.title;

    packInstallBtnText.textContent = 'Creating...';
    packInstallBtn.disabled = true;
    showPackStatus('Downloading archive & creating instance in background...', 'info');

    const result = await sendMessage({
        type: 'install_modpack',
        packUrl: file.downloadUrl,
        packName: packName
    });

    if (result.success) {
        showPackStatus(`✓ Modpack "${packName}" created & installing in background!`, 'success');
        packInstallBtnText.textContent = 'Installed!';
        setTimeout(closePackModal, 2000);
    } else {
        showPackStatus(`✗ ${result.error}`, 'error');
        packInstallBtnText.textContent = 'Create & Install';
        packInstallBtn.disabled = false;
    }
});


// ── Status Helpers ────────────────────────────────────────────────────────

function showStatus(msg, type) {
    installStatus.textContent = msg;
    installStatus.className = `install-status ${type}`;
}

function showPackStatus(msg, type) {
    packInstallStatus.textContent = msg;
    packInstallStatus.className = `install-status ${type}`;
}


// ── Initialize ────────────────────────────────────────────────────────────

(async () => {
    try {
        await loadLauncherToken();
        // Load top popular content automatically on startup
        search('', 0);
    } catch (err) {
        console.error('CurseForge initialization error:', err);
        resultsDiv.innerHTML = `
            <div class="placeholder-wrap">
                <p>Unable to initialize CurseForge: ${escapeHtml(err.message)}</p>
            </div>
        `;
    }
})();