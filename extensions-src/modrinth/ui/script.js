// ── Helpers ───────────────────────────────────────────────────────────────

function isModrinthIconURL(url) {
    try {
        const u = new URL(url);
        return u.protocol === 'https:' && (u.hostname === 'cdn.modrinth.com' || u.hostname.endsWith('.modrinth.com'));
    } catch {
        return false;
    }
}

function escHTML(s) {
    return String(s).replace(/[&<>"']/g, m => ({
        '&': '&',
        '<': '<',
        '>': '>',
        '"': '"',
        "'": '''
    }[m]));
}

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
        // Close all other open dropdowns
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

    // Click outside closes all
    document.addEventListener('click', () => close(), { once: false });

return {
        container: containerEl,
        trigger,
        optionsEl,
        textEl,
        setOptions(options) {
            // Build options DOM safely — no innerHTML with user data
            const optionElements = options.map((opt, i) => {
                const div = document.createElement('div');
                div.className = 'custom-select-option';
                div.dataset.index = i;
                div.textContent = opt.label;
                div.title = opt.label;
                return div;
            });
            optionsEl.replaceChildren(...optionElements);

            // Store values as data attributes on the container
            containerEl._optionValues = options.map(o => o.value);
            containerEl._selectedIndex = -1;

            // Attach click handlers
            optionsEl.querySelectorAll('.custom-select-option').forEach(el => {
                el.addEventListener('click', () => {
                    const idx = parseInt(el.dataset.index, 10);
                    containerEl._selectedIndex = idx;
                    textEl.textContent = options[idx].label;
                    textEl.classList.add('selected');
                    // Highlight selected
                    optionsEl.querySelectorAll('.custom-select-option').forEach(o => o.classList.remove('selected'));
                    el.classList.add('selected');
                    close();
                    // Trigger change event
                    containerEl.dispatchEvent(new CustomEvent('change', { detail: { value: options[idx].value, index: idx } }));
                });
            });

            // Auto-select first
            if (options.length > 0) {
                containerEl._selectedIndex = 0;
                textEl.textContent = options[0].label;
                textEl.classList.add('selected');
                optionsEl.querySelector('.custom-select-option')?.classList.add('selected');
            } else {
                textEl.textContent = 'No options';
                textEl.classList.remove('selected');
            }
        },
        getValue() {
            if (containerEl._selectedIndex >= 0 && containerEl._optionValues) {
                return containerEl._optionValues[containerEl._selectedIndex];
            }
            return null;
        },
        close
    };
}

// ── IPC Bridge ────────────────────────────────────────────────────────────
const pending = {};
let reqCounter = 0;

function sendMessage(payload, timeoutMs) {
    const ms = timeoutMs || 15000;
    return new Promise((resolve) => {
        const id = ++reqCounter;
        payload.requestId = id;
        payload.__aether = true;
        pending[id] = resolve;
        // Use specific origin reference; fallback to window.parent which is the launcher
        const targetOrigin = typeof window !== 'undefined' && window.location
            ? window.location.protocol + '//' + window.location.hostname + (window.location.port ? ':' + window.location.port : '')
            : '';
        window.parent.postMessage(payload, targetOrigin);
        // Never leave the UI hanging: resolve with an error if the sandbox
        // does not answer in time.
        setTimeout(() => {
            if (pending[id]) {
                delete pending[id];
                resolve({ requestId: id, error: 'Request timed out' });
            }
        }, ms);
    });
}

window.addEventListener('message', (e) => {
    const msg = e.data;
    if (!msg || !msg.requestId) return;
    // Only accept messages from the launcher parent, and only our marked payloads
    if (e.source !== window.parent || msg.__aether !== true) return;
    const resolve = pending[msg.requestId];
    if (resolve) {
        delete pending[msg.requestId];
        resolve(msg);
    }
});

// ── State ─────────────────────────────────────────────────────────────────────
let currentMod = null;
let currentVersions = [];
let currentInstances = [];

// ── Elements ──────────────────────────────────────────────────────────────────
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

// ── Version Filtering ─────────────────────────────────────────────────────────
function updateVersionDropdown(inst) {
    if (!inst) {
        versionSelect.setOptions([{ label: 'No instance selected', value: '' }]);
        return;
    }

    const loaderLower = inst.loader.toLowerCase();
    const instVer = inst.version;

    // Filter versions by loader and game version
    let filtered = currentVersions.filter(v => {
        // Game version check
        const gameMatch = v.game_versions.includes(instVer);
        if (!gameMatch) return false;

        // Loader check
        const vLoaders = (v.loaders || []).map(l => l.toLowerCase());
        if (loaderLower === 'vanilla') {
            const hasModLoader = vLoaders.some(l => ['fabric', 'forge', 'neoforge', 'quilt'].includes(l));
            return !hasModLoader;
        } else {
            return vLoaders.includes(loaderLower) || vLoaders.length === 0;
        }
    });

    let warning = '';
    if (filtered.length === 0) {
        // Fallback 1: show versions matching game version only (with warning)
        filtered = currentVersions.filter(v => v.game_versions.includes(instVer));
        if (filtered.length > 0) {
            warning = ' (Loader mismatch)';
        } else {
            // Fallback 2: show all versions
            filtered = currentVersions;
            warning = ' (Incompatible version)';
        }
    }

    versionSelect.setOptions(filtered.map(v => {
        const origIdx = currentVersions.indexOf(v);
        const compatLabel = v.game_versions.includes(instVer) ? '' : '⚠️ ';
        // Avoid redundant name when it already contains the version_number
        const needsName = v.name && v.name !== v.version_number && !v.name.includes(v.version_number);
        const namePart = needsName ? ` — ${v.name}` : '';
        // Keep label concise: version + optional non-redundant name. Game version is already implied by filter.
        let label = `${compatLabel}${v.version_number}${namePart}${warning}`;
        // Hard truncate for UI (CSS ellipsis also applies, but keep DOM light)
        if (label.length > 52) label = label.slice(0, 49) + '...';
        return {
            label,
            value: String(origIdx)
        };
    }));
}

// Listen for instance selection changes to update version filtering
document.getElementById('instanceSelect').addEventListener('change', (e) => {
    const instId = e.detail.value;
    const inst = currentInstances.find(i => i.id === instId);
    if (inst) {
        updateVersionDropdown(inst);
    }
});

// ── Search ────────────────────────────────────────────────────────────────────
async function search(query) {
    if (!query.trim()) return;
    resultsDiv.innerHTML = '<div class="loading"><div class="spinner"></div><p>Searching Modrinth...</p></div>';

    try {
        const res = await fetch(`https://api.modrinth.com/v2/search?query=${encodeURIComponent(query)}&limit=20`);
        if (!res.ok) throw new Error('API Error ' + res.status);
        const data = await res.json();
        const modHits = data.hits.filter(hit => hit.project_type === 'mod');

        if (!modHits.length) {
            resultsDiv.innerHTML = '<div class="placeholder-wrap"><div class="placeholder-icon">&#128230;</div><p>No mods found.</p></div>';
            return;
        }

        resultsDiv.replaceChildren();

        if (!modHits.length) {
            const placeholder = document.createElement('div');
            placeholder.className = 'placeholder-wrap';
            const placeholderIcon = document.createElement('div');
            placeholderIcon.className = 'placeholder-icon';
            placeholderIcon.textContent = '♛';
            const placeholderText = document.createElement('p');
            placeholderText.textContent = 'No mods found.';
            placeholder.appendChild(placeholderIcon);
            placeholder.appendChild(placeholderText);
            resultsDiv.appendChild(placeholder);
            return;
        }

        const grid = document.createElement('div');
        grid.className = 'grid';

        modHits.forEach(hit => {
            const card = document.createElement('div');
            card.className = 'card';
            card.dataset.id = hit.project_id;
            card.dataset.title = encodeURIComponent(hit.title);
            card.dataset.author = encodeURIComponent(hit.author);
            card.dataset.icon = encodeURIComponent(hit.icon_url || '');

            // Title element (safe text)
            const titleEl = document.createElement('div');
            titleEl.className = 'card-title';
            titleEl.textContent = hit.title;

            // Author element (safe text)
            const authorEl = document.createElement('div');
            authorEl.className = 'card-author';
            authorEl.textContent = 'by ' + hit.author;

            // Icon
            const iconContainer = document.createElement('div');
            iconContainer.className = 'card-top';

            let iconElement;
            if (hit.icon_url && isModrinthIconURL(hit.icon_url)) {
                iconElement = document.createElement('img');
                iconElement.className = 'card-icon';
                iconElement.src = hit.icon_url;
                iconElement.alt = '';
            } else {
                const placeholder = document.createElement('div');
                placeholder.className = 'card-icon card-icon-placeholder';
                placeholder.textContent = hit.title ? hit.title.charAt(0) : '';
                iconElement = placeholder;
            }
            iconContainer.appendChild(iconElement);

            // Description (safe text, escaped)
            const descEl = document.createElement('p');
            descEl.className = 'card-desc';
            descEl.textContent = hit.description.replace(/[&<>"']/g, m => ({
                '&': '&',
                '<': '<',
                '>': '>',
                '"': '"',
                "'": '''
            }[m]));

            // Downloads
            const dl = hit.downloads >= 1000000
                ? (hit.downloads / 1000000).toFixed(1) + 'M'
                : hit.downloads >= 1000
                    ? (hit.downloads / 1000).toFixed(0) + 'K'
                    : hit.downloads;
            const downloadsSpan = document.createElement('span');
            downloadsSpan.className = 'card-downloads';
            downloadsSpan.textContent = '⬇ ' + dl;

            // Install button
            const installBtn = document.createElement('button');
            installBtn.className = 'btn-install-card';
            installBtn.dataset.id = hit.project_id;
            installBtn.textContent = 'Install';

            // Assemble card top
            const cardTop = document.createElement('div');
            cardTop.className = 'card-top';
            cardTop.appendChild(iconElement);

            // Assemble card
            card.appendChild(cardTop);
            card.appendChild(titleEl);
            card.appendChild(authorEl);
            card.appendChild(descEl);

            // Append footer with downloads and button
            const footer = document.createElement('div');
            footer.className = 'card-footer';
            footer.appendChild(downloadsSpan);
            footer.appendChild(installBtn);
            card.appendChild(footer);

            grid.appendChild(card);
        });

        resultsDiv.appendChild(grid);

        // Attach click handlers to Install buttons
        document.querySelectorAll('.btn-install-card').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                const card = btn.closest('.card');
                openInstallModal({
                    id: card.dataset.id,
                    title: decodeURIComponent(card.dataset.title),
                    author: decodeURIComponent(card.dataset.author),
                    icon: decodeURIComponent(card.dataset.icon)
                });
            });
        });

    } catch (err) {
        const errEl = document.createElement('div');
        errEl.className = 'placeholder-wrap';
        const errP = document.createElement('p');
        errP.textContent = 'Error: ' + err.message;
        errEl.appendChild(errP);
        resultsDiv.appendChild(errEl);
    }
}

// Debounce search
let debounceTimer;
searchInput.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => search(searchInput.value), 400);
});
searchInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { clearTimeout(debounceTimer); search(searchInput.value); }
});

// ── Install Modal ─────────────────────────────────────────────────────────────
async function openInstallModal(mod) {
    currentMod = mod;
    currentVersions = [];
    currentInstances = [];

    // Populate header
    modalModName.textContent = mod.title;
    modalModAuthor.textContent = 'by ' + mod.author;
    if (mod.icon && isModrinthIconURL(mod.icon)) {
        const img = document.createElement('img');
        img.src = mod.icon;
        img.alt = '';
        modalModIcon.innerHTML = ''; // clear first
        modalModIcon.appendChild(img);
    } else {
        modalModIcon.textContent = '';
        const letter = document.createElement('div');
        letter.className = 'icon-letter';
        letter.textContent = mod.title ? mod.title.charAt(0) : '';
        modalModIcon.appendChild(letter);
    }

    // Reset state
    versionSelect.setOptions([{ label: 'Loading versions...', value: '' }]);
    instanceSelect.setOptions([{ label: 'Loading instances...', value: '' }]);
    installStatus.classList.add('hidden');
    installStatus.textContent = '';
    installBtnText.textContent = 'Install';
    installBtn.disabled = false;
    modal.classList.remove('hidden');

    // Fetch versions and instances in parallel
    let versionsRes, instancesMsg;
    try {
        [versionsRes, instancesMsg] = await Promise.all([
            fetch(`https://api.modrinth.com/v2/project/${mod.id}/version`).then(r => r.json()),
            sendMessage({ type: 'get_instances' })
        ]);
    } catch (err) {
        instanceSelect.setOptions([{ label: 'Failed to load instances', value: '' }]);
        versionSelect.setOptions([{ label: 'Failed to load versions', value: '' }]);
        showStatus(`Failed to load: ${err.message}`, 'error');
        return;
    }

    // Store state
    currentVersions = versionsRes;
    currentInstances = instancesMsg.instances || [];

    if (instancesMsg.error) {
        instanceSelect.setOptions([{ label: 'Failed to load instances', value: '' }]);
        versionSelect.setOptions([{ label: 'No instances available', value: '' }]);
        showStatus(`Failed to load instances: ${instancesMsg.error}`, 'error');
        return;
    }

    // Populate instances dropdown
    instanceSelect.setOptions(currentInstances.length
        ? currentInstances.map(inst => ({
            label: `${inst.name} (${inst.version} • ${inst.loader})`,
            value: inst.id
          }))
        : [{ label: 'No instances found', value: '' }]
    );

    // Initial version list update based on the default selected instance
    if (currentInstances.length > 0) {
        updateVersionDropdown(currentInstances[0]);
    } else {
        versionSelect.setOptions([{ label: 'No instances available', value: '' }]);
    }
}

function closeModal() {
    modal.classList.add('hidden');
    currentMod = null;
}

modalClose.addEventListener('click', closeModal);
cancelBtn.addEventListener('click', closeModal);
modal.addEventListener('click', (e) => { if (e.target === modal) closeModal(); });

installBtn.addEventListener('click', async () => {
    const vIdxStr = versionSelect.getValue();
    const instanceId = instanceSelect.getValue();

    if (vIdxStr === null || !instanceId) {
        showStatus('Please select a version and instance.', 'error');
        return;
    }

    const vIdx = parseInt(vIdxStr, 10);
    const version = currentVersions[vIdx];
    // Pick the primary jar file
    const file = version.files.find(f => f.primary) || version.files[0];
    if (!file) {
        showStatus('No downloadable file found for this version.', 'error');
        return;
    }

    installBtnText.textContent = 'Installing...';
    installBtn.disabled = true;
    installStatus.classList.add('hidden');

    const result = await sendMessage({
        type: 'install_mod',
        instanceId,
        jarName: file.filename,
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

function showStatus(msg, type) {
    installStatus.textContent = msg;
    installStatus.className = `install-status ${type}`;
}

// Initial search
search('fabric');
