// viewer.js - thin adapter over the vendored skin3d library
// (ui/vendor/skin3d.bundle.js, MIT). All Minecraft model math (UVs, slim
// arms, overlay layers, cape box) lives upstream; this file only preserves
// the SelectorViewer interface used by app.js.
(function () {
  var viewers = {};

  function dims(el) {
    return { w: el.clientWidth || 300, h: el.clientHeight || 300 };
  }

  function reportError(el, err) {
    // Surface remote-texture failures on the main preview status line.
    if (el && el.id === "main-3d") {
      var s = document.getElementById("status-line");
      if (s) s.textContent = "Preview failed to load (" + err + ") — try Download PNG instead";
    }
  }

  function loadSkinInto(st, source, slim) {
    st.skin = source;
    st.slim = !!slim;
    try {
      var p = st.render.loadSkin(source, { model: slim ? "slim" : "default" });
      if (p && typeof p.catch === "function") {
        p.catch(function (err) { reportError(st.el, (err && err.message) || err); });
      }
    } catch (err) {
      reportError(st.el, (err && err.message) || err);
    }
  }

  function ensure(id) {
    if (viewers[id]) return viewers[id];
    var el = document.getElementById(id);
    if (!el) return null;
    // Hidden containers (e.g. the cape modal before first open) report zero
    // size, which would break camera setup — caller retries via init().
    if (!el.clientWidth || !el.clientHeight) return null;
    var d = dims(el);
    // skin3d accepts a container, but WebGLRenderer needs a real canvas —
    // create one explicitly so behavior never depends on that ambiguity.
    el.style.position = "relative";
    var cv = document.createElement("canvas");
    cv.style.display = "block";
    cv.style.width = "100%";
    cv.style.height = "100%";
    el.appendChild(cv);
    var render = new window.skin3d.Render({
      canvas: cv,
      width: d.w,
      height: d.h,
      enableControls: true,
      autoRotate: true
    });
    // NOTE: Render's constructor ignores options.autoRotate (field defaults
    // false), so set it explicitly — otherwise the first toggle appears dead.
    render.autoRotate = true;
    // Gentle idle animation (breathing/sway) on top of auto-rotate.
    try {
      render.animation = new window.skin3d.IdleAnimation(render.playerObject);
    } catch (e) { /* static preview is fine */ }
    // Drag-rotate + wheel-zoom like Modrinth, but never lose the model.
    render.controls.enablePan = false;
    var st = { render: render, el: el, skin: null, slim: false, capeUrl: null, rotate: true };
    viewers[id] = st;
    el.dataset.viewerInit = "1";
    window.addEventListener("resize", function () {
      if (!viewers[id]) return;
      render.width = el.clientWidth || d.w;
      render.height = el.clientHeight || d.h;
    });
    return st;
  }

  window.SelectorViewer = {
    init: function (id) { ensure(id); },
    loadUrl: function (id, url, slim) {
      var st = ensure(id);
      if (!st || !url) return;
      loadSkinInto(st, url, slim);
    },
    loadDataUrl: function (id, dataUrl, slim) {
      var st = ensure(id);
      if (!st || !dataUrl) return;
      loadSkinInto(st, dataUrl, slim);
    },
    setSlim: function (id, slim) {
      var st = ensure(id) || viewers[id];
      if (!st) {
        // No viewer yet (hidden modal): remember for first load.
        return;
      }
      st.slim = !!slim;
      if (st.skin) loadSkinInto(st, st.skin, slim);
    },
    isSlim: function (id) {
      return !!(viewers[id] && viewers[id].slim);
    },
    setRotate: function (id, on) {
      var st = viewers[id];
      if (!st) return;
      st.rotate = !!on;
      st.render.autoRotate = !!on;
    },
    loadCapeUrl: function (id, url) {
      var st = ensure(id);
      if (!st) return;
      st.capeUrl = url || null;
      try {
        var p = url ? st.render.loadCape(url) : st.render.loadCape(null);
        if (p && typeof p.catch === "function") {
          p.catch(function () { /* cape texture failed; skin preview stays */ });
        }
      } catch (e) { /* keep skin preview usable */ }
    },
    clearCape: function (id) {
      var st = viewers[id];
      if (!st) return;
      st.capeUrl = null;
      try { st.render.loadCape(null); } catch (e) { /* keep skin preview usable */ }
    },
    hasCape: function (id) {
      return !!(viewers[id] && viewers[id].capeUrl);
    },
    resetView: function (id) {
      var st = viewers[id];
      if (!st) return;
      try {
        if (typeof st.render.resetModelRotation === "function") st.render.resetModelRotation();
        if (typeof st.render.resetCameraPose === "function") st.render.resetCameraPose();
      } catch (e) { /* cosmetic only */ }
    }
  };
})();
