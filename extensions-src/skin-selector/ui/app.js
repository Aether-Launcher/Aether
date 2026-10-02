// app.js - Skin Selector frontend, Modrinth-style single page.
(function () {
  var reqId = 0;
  var pending = {};
  var account = null;
  var mySkins = [];
  var myCapes = [];
  var selected = null; // {kind:'account'|'gallery'|'default'|'upload', id, url, variant, name, dataUrl?}
  var activeCapeId = null;
  var modalCape = null; // cape id | 'none' | null
  var thumbCache = {};

  function $(id) { return document.getElementById(id); }
  function notice(msg, ok) {
    var n = $("notice");
    n.textContent = msg;
    n.classList.remove("hidden");
    n.classList.toggle("ok", !!ok);
    clearTimeout(n._t);
    n._t = setTimeout(function () { n.classList.add("hidden"); }, 4500);
  }
  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }

  function send(type, extra) {
    return new Promise(function (resolve, reject) {
      var id = ++reqId;
      pending[id] = { resolve: resolve, reject: reject };
      window.parent.postMessage(Object.assign({ type: type, reqId: String(id) }, extra || {}), "*");
      setTimeout(function () {
        if (pending[id]) { delete pending[id]; reject(new Error("Request timed out")); }
      }, 30000);
    });
  }

  window.addEventListener("message", function (e) {
    var msg = e.data;
    if (!msg) return;
    if (msg.reqId != null && pending[msg.reqId]) {
      var p = pending[msg.reqId];
      delete pending[msg.reqId];
      p.resolve(msg);
      return;
    }
    handlePush(msg);
  });

  function handlePush(msg) {
    if (!msg || !msg.type) return;
    if (msg.type === "gallery_result" && !msg.error && !msg.skinUrl) return;
    if (msg.type === "account_result" && !msg.error && !msg.account) return;
    if (msg.type === "my_skins_result" && !msg.error && !msg.skins) return;
    if (msg.type === "my_capes_result" && !msg.error && !msg.capes) return;
    if (msg.type === "upload_result" && !msg.error && !msg.skin) return;
    if (msg.type === "export_result" && !msg.error && !msg.path) return;
    if (msg.type === "gallery_result") onGalleryResult(msg);
    else if (msg.type === "account_result") onAccount(msg);
    else if (msg.type === "my_skins_result") onMySkins(msg);
    else if (msg.type === "my_capes_result") onMyCapes(msg);
    else if (msg.type === "upload_result") onUpload(msg);
    else if (msg.type === "cape_result") onCapeOp(msg);
  }

  function modelVariant() {
    var el = document.querySelector('input[name="model"]:checked');
    return el ? el.value : "classic";
  }
  // Mojang reports "CLASSIC"/"SLIM" (uppercase); the gallery backend reports
  // "slim"/"default". Normalize everything to "slim" | "classic" so a slim
  // account skin is never previewed/uploaded with the wrong (classic) arm
  // width, which visibly stretches the arms and sleeves.
  function normVariant(v) {
    v = String(v == null ? "" : v).toLowerCase();
    return v === "slim" ? "slim" : "classic";
  }
  // The model radio is the authority for preview AND upload. It is synced
  // from a skin's own variant only at selection time (never during preview),
  // otherwise it snaps the user's click straight back.
  function syncRadio(variant) {
    var r = document.querySelector('input[name="model"][value="' + normVariant(variant) + '"]');
    if (r) r.checked = true;
  }

  // ---------- viewers ----------
  // Main viewer only; the cape modal viewer is created lazily on first open
  // because its hidden container reports zero size at boot.
  SelectorViewer.init("main-3d");

  $("rotate-btn").addEventListener("click", function () {
    var on = this.classList.toggle("active");
    SelectorViewer.setRotate("main-3d", on);
  });
  document.querySelectorAll('input[name="model"]').forEach(function (r) {
    r.addEventListener("change", function () {
      if (selected) previewSelected();
    });
  });

  function previewSelected() {
    if (!selected) return;
    var slim = modelVariant() === "slim";
    if (selected.dataUrl) SelectorViewer.loadDataUrl("main-3d", selected.dataUrl, slim);
    else SelectorViewer.loadUrl("main-3d", selected.url, slim);
    $("preview-name").textContent = selected.name || (account && account.username) || "Skin";
    refreshApplyRow();
  }

  function refreshApplyRow() {
    var apply = $("apply-btn"), dl = $("download-btn");
    if (!selected || !selected.url && !selected.dataUrl) { apply.disabled = true; dl.disabled = true; return; }
    dl.disabled = false;
    if (selected.kind === "account" && selected.active) {
      apply.disabled = true; apply.textContent = "In use";
    } else {
      apply.disabled = false; apply.textContent = "Apply skin";
    }
  }

  // ---------- paper-doll thumbnails ----------
  // Paper-doll thumbnails: composite the base layer AND the overlay layer
  // (hat at 40,8 over head at 8,8; jacket at 20,32 over torso at 20,20) so
  // skins carrying their design on the 2nd layer (TV screens, hats, etc.)
  // render correctly on cards instead of showing the blank base layer.
  function thumbForSkin(url) {
    if (thumbCache[url]) return Promise.resolve(thumbCache[url]);
    return new Promise(function (resolve) {
      var img = new Image();
      img.crossOrigin = "Anonymous";
      img.onload = function () {
        try {
          var c = document.createElement("canvas");
          c.width = 48; c.height = 64;
          var x = c.getContext("2d");
          x.imageSmoothingEnabled = false;
          // head front 8x8 -> top, body front 8x12 below
          var sw = img.naturalWidth || 64, sh = img.naturalHeight || 64;
          if (sw === 64 && sh === 32) {
            // legacy layout has no overlay layer
            x.drawImage(img, 8, 8, 8, 8, 8, 0, 32, 32);
            x.drawImage(img, 20, 20, 8, 12, 12, 32, 24, 32);
          } else {
            x.drawImage(img, 8, 8, 8, 8, 8, 0, 32, 32);
            x.drawImage(img, 40, 8, 8, 8, 8, 0, 32, 32);
            x.drawImage(img, 20, 20, 8, 12, 12, 32, 24, 32);
            x.drawImage(img, 20, 32, 8, 12, 12, 32, 24, 32);
          }
          var d = c.toDataURL();
          thumbCache[url] = d;
          resolve(d);
        } catch (e) { resolve(url); }
      };
      img.onerror = function () { resolve(url); };
      img.src = url;
    });
  }

  // Cape thumbnails: crop the front face region (1,1,10,16) of the 64x32
  // cape texture instead of squeezing the whole sheet.
  var capeThumbCache = {};
  function capeThumb(url) {
    if (capeThumbCache[url]) return Promise.resolve(capeThumbCache[url]);
    return new Promise(function (resolve) {
      var img = new Image();
      img.crossOrigin = "Anonymous";
      img.onload = function () {
        try {
          var c = document.createElement("canvas");
          c.width = 20; c.height = 32;
          var x = c.getContext("2d");
          x.imageSmoothingEnabled = false;
          x.drawImage(img, 1, 1, 10, 16, 0, 0, 20, 32);
          var d = c.toDataURL();
          capeThumbCache[url] = d;
          resolve(d);
        } catch (e) { resolve(url); }
      };
      img.onerror = function () { resolve(url); };
      img.src = url;
    });
  }
  // ---------- gallery search ----------
  $("search-btn").addEventListener("click", doSearch);
  $("username-input").addEventListener("keydown", function (e) { if (e.key === "Enter") doSearch(); });
  function doSearch() {
    var name = $("username-input").value.trim();
    if (!name) return;
    $("status-line").textContent = "Searching Mojang + NameMC…";
    send("gallery_lookup", { username: name }).then(handlePush).catch(function (err) {
      $("status-line").textContent = "Error: " + err.message;
    });
  }
  function onGalleryResult(msg) {
    if (msg.error) { $("status-line").textContent = msg.error; return; }
    selected = { kind: "gallery", name: msg.username || "Skin", url: msg.skinUrl, variant: (msg.modelType === "slim" ? "slim" : modelVariant()) };
    if (msg.modelType) syncRadio(msg.modelType);
    previewSelected();
    $("status-line").textContent = (msg.username || "") + (msg.capeUrl ? " • cape found on profile" : "") + " • via " + (msg.source || "public db");
    renderSavedGrid();
  }

  // ---------- account ----------
  $("acct-refresh").addEventListener("click", function () { boot(true); });
  function onAccount(msg) {
    if (msg.error) { $("acct-text").textContent = "Account: unavailable"; notice(msg.error); return; }
    account = msg.account;
    if (!account || !account.signedIn) {
      $("acct-text").textContent = "Not signed in — gallery only";
      return;
    }
    $("acct-text").textContent = account.username + " • " + account.type;
    $("preview-name").textContent = account.username;
  }

  function onMySkins(msg) {
    if (msg.error) { notice(msg.error); return; }
    mySkins = msg.skins || [];
    renderSavedGrid();
    var active = mySkins.filter(function (s) { return String(s.state).toUpperCase() === "ACTIVE"; })[0] || mySkins[0];
    if (active && !selected) {
      selected = { kind: "account", id: active.id, name: (account && account.username) || "My skin", url: active.url, variant: active.variant || "classic", active: String(active.state).toUpperCase() === "ACTIVE" };
      syncRadio(selected.variant);
      previewSelected();
    } else if (selected && selected.kind === "account") {
      var still = mySkins.filter(function (s) { return s.id === selected.id; })[0];
      if (still) { selected.url = still.url; selected.active = String(still.state).toUpperCase() === "ACTIVE"; previewSelected(); }
    }
  }

  function renderSavedGrid() {
    var grid = $("saved-grid");
    grid.innerHTML = "";
    // Add-a-skin card
    var add = document.createElement("button");
    add.className = "skin-card add";
    add.innerHTML = "<span class='plus'>+</span><span>Add a skin</span>";
    add.addEventListener("click", function () { $("skin-file").click(); });
    grid.appendChild(add);
    if (!mySkins.length) {
      var empty = document.createElement("div");
      empty.className = "grid-note";
      empty.textContent = account && account.type === "microsoft" ? "Loading your skins…" : "Sign in with Microsoft to see saved skins.";
      grid.appendChild(empty);
      return;
    }
    mySkins.forEach(function (sk) {
      var isActive = String(sk.state).toUpperCase() === "ACTIVE";
      var card = document.createElement("button");
      card.className = "skin-card" + (isActive ? " selected" : "") + (selected && selected.id === sk.id ? " focused" : "");
      var img = document.createElement("img");
      img.alt = "skin";
      thumbForSkin(sk.url).then(function (t) { img.src = t; });
      img.src = sk.url;
      var label = document.createElement("span");
      label.textContent = isActive ? "Active" : (sk.variant || "");
      card.appendChild(img); card.appendChild(label);
      card.addEventListener("click", function () {
        selected = { kind: "account", id: sk.id, name: (account && account.username) || "My skin", url: sk.url, variant: sk.variant || "classic", active: isActive };
        syncRadio(selected.variant);
        previewSelected(); renderSavedGrid();
      });
      grid.appendChild(card);
    });
  }

  $("skin-file").addEventListener("change", function () {
    var f = this.files[0];
    if (!f) return;
    var fr = new FileReader();
    fr.onload = function () {
      selected = { kind: "upload", name: f.name.replace(/\.png$/i, ""), url: fr.result, dataUrl: fr.result, variant: modelVariant() };
      previewSelected();
      $("status-line").textContent = "Uploading… (confirm in launcher)";
      send("upload_skin", { data: String(fr.result).split(",")[1], variant: modelVariant() }).then(handlePush)
        .catch(function (err) { notice(err.message); });
    };
    fr.readAsDataURL(f);
    this.value = "";
  });

  function onUpload(msg) {
    if (msg.error) { notice(msg.error); $("status-line").textContent = msg.error; return; }
    notice("Skin applied to your account", true);
    $("status-line").textContent = "Skin applied ✓";
    send("my_skins", {}).then(handlePush).catch(function () {});
  }

  $("apply-btn").addEventListener("click", function () {
    if (!selected) return;
    if (!account || !account.signedIn || account.type !== "microsoft") { notice("Sign in with Microsoft to apply skins."); return; }
    $("status-line").textContent = "Applying skin… (confirm in launcher)";
    if (selected.kind === "gallery" || selected.kind === "account") {
      send("apply_url", { url: selected.url, variant: modelVariant() }).then(handlePush)
        .catch(function (err) { notice(err.message); });
    } else {
      var b64 = String(selected.dataUrl).split(",")[1];
      send("upload_skin", { data: b64, variant: modelVariant() }).then(handlePush)
        .catch(function (err) { notice(err.message); });
    }
  });

  $("download-btn").addEventListener("click", function () {
    if (!selected) return;
    var src = selected.dataUrl || selected.url;
    if (!src) return;
    function exportB64(b64, name) {
      send("export_skin", { data: b64, filename: name + ".png" }).then(function (res) {
        if (res.error) notice(res.error); else notice("Saved: " + (res.path || "skin.png"), true);
      }).catch(function (err) { notice(err.message); });
    }
    if (selected.dataUrl) { exportB64(String(src).split(",")[1], selected.name || "skin"); return; }
    fetch(src).then(function (r) { return r.blob(); }).then(function (blob) {
      var fr = new FileReader();
      fr.onload = function () { exportB64(String(fr.result).split(",")[1], selected.name || "skin"); };
      fr.readAsDataURL(blob);
    }).catch(function (err) { notice("Download failed: " + err.message); });
  });

  // ---------- capes ----------
  function onMyCapes(msg) {
    if (msg.error) { notice(msg.error); return; }
    myCapes = msg.capes || [];
    activeCapeId = null;
    myCapes.forEach(function (c) { if (String(c.state).toUpperCase() === "ACTIVE") activeCapeId = c.id; });
    if (activeCapeId) {
      var ac = myCapes.filter(function (c) { return c.id === activeCapeId; })[0];
      if (ac) SelectorViewer.loadCapeUrl("main-3d", ac.url);
    } else {
      SelectorViewer.clearCape("main-3d");
    }
  }

  $("change-cape-btn").addEventListener("click", openCapeModal);
  $("cape-close").addEventListener("click", closeCapeModal);
  $("cape-cancel").addEventListener("click", function () {
    // restore main viewer cape, discard pending pick
    if (activeCapeId) {
      var ac = myCapes.filter(function (c) { return c.id === activeCapeId; })[0];
      if (ac) SelectorViewer.loadCapeUrl("main-3d", ac.url); else SelectorViewer.clearCape("main-3d");
    } else SelectorViewer.clearCape("main-3d");
    closeCapeModal();
  });

  function openCapeModal() {
    if (!selected) { notice("Pick or search a skin first."); return; }
    // Unhide first so the lazily created viewer measures a real size.
    $("cape-modal").classList.remove("hidden");
    SelectorViewer.init("cape-3d");
    // mirror current skin into modal viewer
    var modalSlim = normVariant(selected.variant || modelVariant()) === "slim";
    if (selected.dataUrl) SelectorViewer.loadDataUrl("cape-3d", selected.dataUrl, modalSlim);
    else SelectorViewer.loadUrl("cape-3d", selected.url, modalSlim);
    SelectorViewer.resetView("cape-3d");
    if (activeCapeId) {
      var ac = myCapes.filter(function (c) { return c.id === activeCapeId; })[0];
      if (ac) SelectorViewer.loadCapeUrl("cape-3d", ac.url);
    } else SelectorViewer.clearCape("cape-3d");
    modalCape = activeCapeId || "none";
    renderCapeList();
  }
  function closeCapeModal() { $("cape-modal").classList.add("hidden"); }

  function renderCapeList() {
    var list = $("cape-list");
    list.innerHTML = "";
    document.querySelector('.cape-opt[data-cape="none"]').classList.toggle("picked", modalCape === "none" || !modalCape);
    if (!myCapes.length) {
      var d = document.createElement("div");
      d.className = "grid-note";
      d.textContent = "No capes on this account yet.";
      list.appendChild(d);
      return;
    }
    myCapes.forEach(function (cp) {
      var b = document.createElement("button");
      b.className = "cape-opt" + (modalCape === cp.id ? " picked" : "");
      b.title = cp.alias || cp.id;
      var img = document.createElement("img");
      img.alt = cp.alias || "cape";
      capeThumb(cp.url).then(function (t) { img.src = t; });
      img.src = cp.url;
      var nm = document.createElement("span");
      nm.textContent = cp.alias || String(cp.id).slice(0, 8);
      b.appendChild(img); b.appendChild(nm);
      b.addEventListener("click", function () {
        modalCape = cp.id;
        SelectorViewer.loadCapeUrl("cape-3d", cp.url);
        renderCapeList();
      });
      list.appendChild(b);
    });
  }
  document.querySelector('.cape-opt[data-cape="none"]').addEventListener("click", function () {
    modalCape = "none";
    SelectorViewer.clearCape("cape-3d");
    renderCapeList();
  });

  $("cape-select").addEventListener("click", function () {
    closeCapeModal();
    if (modalCape === "none" || !modalCape) {
      send("hide_cape", {}).then(function (msg) {
        if (msg.error) { notice(msg.error); return; }
        activeCapeId = null;
        SelectorViewer.clearCape("main-3d");
        notice("Cape hidden", true);
        send("my_capes", {}).then(handlePush).catch(function () {});
      }).catch(function (err) { notice(err.message); });
      return;
    }
    var cp = myCapes.filter(function (c) { return c.id === modalCape; })[0];
    send("equip_cape", { capeId: modalCape }).then(function (msg) {
      if (msg.error) { notice(msg.error); return; }
      activeCapeId = modalCape;
      if (cp) SelectorViewer.loadCapeUrl("main-3d", cp.url);
      notice("Cape equipped: " + (cp && (cp.alias || cp.id) || ""), true);
      send("my_capes", {}).then(handlePush).catch(function () {});
    }).catch(function (err) { notice(err.message); });
  });

  function onCapeOp(msg) {
    if (msg.error) { notice(msg.error); return; }
    notice("Cape updated", true);
    send("my_capes", {}).then(handlePush).catch(function () {});
  }

  // ---------- boot ----------
  function boot(isRefresh) {
    send("get_account", {}).then(handlePush).catch(function () {
      $("acct-text").textContent = "Account unknown (offline?)";
    }).then(function () {
      send("my_skins", {}).then(handlePush).catch(function () {});
      send("my_capes", {}).then(handlePush).catch(function () {});
    });
  }
  boot(false);
})();
