// My Servers UI — talks to main.js over the sandbox IPC bridge.
// All untrusted text (server names, MOTDs, world names) is rendered with
// textContent only — never innerHTML.

var pending = {};
var reqCounter = 0;

function sendMessage(payload, timeoutMs) {
  var ms = timeoutMs || 15000;
  return new Promise(function (resolve, reject) {
    var id = ++reqCounter;
    payload.requestId = id;
    pending[id] = { resolve: resolve, reject: reject };
    // NOTE: targetOrigin MUST be "*" here. window.location is the *iframe's*
    // own origin (http://127.0.0.1:port), but window.parent is the Wails
    // webview (wails://...) — a computed origin never matches and the
    // message is silently dropped. Correlation via requestId is the actual
    // security boundary.
    window.parent.postMessage(payload, "*");
    setTimeout(function () {
      if (pending[id]) {
        delete pending[id];
        reject(new Error("Request timed out"));
      }
    }, ms);
  });
}

window.addEventListener("message", function (e) {
  var msg = e.data;
  // Backend responses carry no marker (ExtensionView forwards payloads
  // as-is), so only the requestId correlation applies here.
  if (!msg || msg.requestId == null) return;
  var p = pending[msg.requestId];
  if (!p) return;
  delete pending[msg.requestId];
  if (msg.error || msg.success === false) p.reject(new Error(msg.error || "failed"));
  else p.resolve(msg);
});

var instanceSelect = document.getElementById("instanceSelect");
var statusEl = document.getElementById("status");
var serversEl = document.getElementById("servers");
var worldsEl = document.getElementById("worlds");

function setStatus(text, kind) {
  statusEl.textContent = text;
  statusEl.className = "status" + (kind ? " " + kind : "");
}

function el(tag, cls, text) {
  var n = document.createElement(tag);
  if (cls) n.className = cls;
  if (text != null) n.textContent = text;
  return n;
}

var GAME_MODES = { 0: "Survival", 1: "Creative", 2: "Adventure", 3: "Spectator" };

function currentInstanceId() {
  return instanceSelect.value;
}

function refresh() {
  var id = currentInstanceId();
  if (!id) return;
  setStatus("");
  loadServers(id);
  loadWorlds(id);
}

function loadServers(id) {
  serversEl.innerHTML = "";
  serversEl.appendChild(el("div", "empty", "Pinging servers…"));
  sendMessage({ type: "get_servers", instanceId: id }, 30000).then(function (res) {
    serversEl.innerHTML = "";
    var list = res.servers || [];
    if (!list.length) {
      serversEl.appendChild(el("div", "empty", "No saved servers. Add some in-game first — they live in this instance's servers.dat."));
      return;
    }
    list.forEach(function (s) {
      var row = el("div", "row");
      var dot = el("span", "dot" + (s.online ? " online" : ""));
      row.appendChild(dot);
      var info = el("div", "info");
      info.appendChild(el("div", "name", s.name || s.ip));
      var meta = s.online
        ? (s.motd || "") + "  •  " + (s.playersOnline || 0) + "/" + (s.playersMax || "?") + "  •  " + (s.latencyMs != null ? s.latencyMs + "ms" : "")
        : (s.ip + "  •  offline");
      info.appendChild(el("div", "meta", meta));
      row.appendChild(info);
      var btn = el("button", "play", "Play");
      btn.addEventListener("click", function () { playServer(id, s); });
      row.appendChild(btn);
      serversEl.appendChild(row);
    });
  }).catch(function (err) {
    serversEl.innerHTML = "";
    serversEl.appendChild(el("div", "empty", "Couldn't load servers: " + err.message));
  });
}

function loadWorlds(id) {
  worldsEl.innerHTML = "";
  worldsEl.appendChild(el("div", "empty", "Reading worlds…"));
  sendMessage({ type: "get_worlds", instanceId: id }).then(function (res) {
    worldsEl.innerHTML = "";
    var list = res.worlds || [];
    if (!list.length) {
      worldsEl.appendChild(el("div", "empty", "No singleplayer worlds in this instance yet."));
      return;
    }
    list.forEach(function (w) {
      var row = el("div", "row");
      row.appendChild(el("span", "dot online"));
      var info = el("div", "info");
      info.appendChild(el("div", "name", w.name || w.id));
      var bits = [];
      if (w.gameMode != null && GAME_MODES[w.gameMode]) bits.push(GAME_MODES[w.gameMode]);
      if (w.lastPlayed) {
        try { bits.push(new Date(w.lastPlayed).toLocaleString()); } catch (e) { /* ignore bad timestamps */ }
      }
      info.appendChild(el("div", "meta", bits.join("  •  ") || w.id));
      row.appendChild(info);
      var btn = el("button", "play", "Play");
      btn.addEventListener("click", function () { playWorld(id, w); });
      row.appendChild(btn);
      worldsEl.appendChild(row);
    });
  }).catch(function (err) {
    worldsEl.innerHTML = "";
    worldsEl.appendChild(el("div", "empty", "Couldn't load worlds: " + err.message));
  });
}

function playServer(instanceId, server) {
  setStatus("Launching " + (server.name || server.ip) + "…");
  sendMessage({ type: "play_server", instanceId: instanceId, ip: server.ip, host: server.host, port: server.port }, 120000).then(function () {
    setStatus("Game launching — connecting to " + (server.name || server.ip) + ".", "ok");
  }).catch(function (err) {
    if (err && err.message === "Request timed out") {
      setStatus("Launch is still starting in the background (first launches download Java). Check the game window.", "ok");
    } else {
      setStatus("Launch failed: " + err.message, "error");
    }
  });
}

function playWorld(instanceId, world) {
  setStatus("Launching " + (world.name || world.id) + "…");
  sendMessage({ type: "play_world", instanceId: instanceId, world: world.id }, 120000).then(function () {
    setStatus("Game launching — loading " + (world.name || world.id) + ".", "ok");
  }).catch(function (err) {
    if (err && err.message === "Request timed out") {
      setStatus("Launch is still starting in the background (first launches download Java). Check the game window.", "ok");
    } else {
      setStatus("Launch failed: " + err.message, "error");
    }
  });
}

instanceSelect.addEventListener("change", refresh);

sendMessage({ type: "get_instances" }).then(function (res) {
  var list = res.instances || [];
  instanceSelect.innerHTML = "";
  if (!list.length) {
    setStatus("No instances yet — create one on the Instances page first.", "error");
    return;
  }
  list.forEach(function (inst) {
    var opt = document.createElement("option");
    opt.value = inst.id;
    opt.textContent = inst.name + "  (" + inst.version + ")";
    instanceSelect.appendChild(opt);
  });
  refresh();
}).catch(function (err) {
  setStatus("Couldn't list instances: " + err.message, "error");
});
