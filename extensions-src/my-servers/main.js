// My Servers — official Aether extension.
// Lists an instance's saved servers (with live ping) and singleplayer
// worlds, and launches the game straight into either one.

Aether.ui.registerSidebarPage({
  id: "my-servers",
  label: "My Servers",
  icon: "icon.png",
  url: "ui/index.html",
});

function ok(msg, extra) {
  var payload = {
    type: msg.type + "_result",
    requestId: msg.requestId,
    success: true,
  };
  if (extra) {
    for (var k in extra) payload[k] = extra[k];
  }
  Aether.ui.postMessage(payload);
}

function fail(msg, err) {
  var message = err && err.message ? err.message : String(err);
  Aether.ui.postMessage({
    type: msg.type + "_result",
    requestId: msg.requestId,
    success: false,
    error: message,
  });
}

// Splits "host" / "host:port" as stored in servers.dat. Kept in sync with
// the launcher's own parsing; the backend revalidates before launching.
function splitHostPort(ip) {
  var raw = String(ip == null ? "" : ip);
  var host = raw;
  var port = 25565;
  var idx = raw.lastIndexOf(":");
  if (idx > 0 && idx < raw.length - 1) {
    var p = parseInt(raw.slice(idx + 1), 10);
    if (!isNaN(p) && p > 0 && p <= 65535 && raw.indexOf(":") === idx) {
      port = p;
      host = raw.slice(0, idx);
    }
  }
  return { host: host, port: port };
}

Aether.ui.onMessage(function (msg) {
  try {
    switch (msg.type) {
      case "get_instances": {
        var instances = Aether.instances.list();
        return ok(msg, { instances: instances });
      }
      case "get_servers": {
        // One bulk call: pings run concurrently in Go (bounded per-server
        // budget) and results are cached per servers.dat content, so page
        // revisits are instant. Never list()+ping() in a loop here — goja
        // is single-threaded and each dead server would stall the list.
        var rows = Aether.servers.listWithStatus(msg.instanceId, 3000);
        var out = rows.map(function (r) {
          return {
            name: r.name, ip: r.ip, hidden: !!r.hidden,
            online: !!r.online, host: r.host, port: r.port,
            motd: r.motd || "",
            playersOnline: r.playersOnline, playersMax: r.playersMax,
            version: r.version || "",
            latencyMs: r.latencyMs,
          };
        });
        return ok(msg, { servers: out });
      }
      case "get_worlds": {
        var worlds = Aether.instances.listWorlds(msg.instanceId);
        return ok(msg, { worlds: worlds });
      }
      case "play_server": {
        // Rows from listWithStatus already carry the parsed host/port; the
        // split is only a fallback for callers passing a raw address.
        var host = msg.host || splitHostPort(msg.ip).host;
        var port = msg.port || splitHostPort(msg.ip).port;
        Aether.instances.launchToServer(msg.instanceId, host, port);
        return ok(msg, {});
      }
      case "play_world": {
        Aether.instances.launchToWorld(msg.instanceId, msg.world);
        return ok(msg, {});
      }
      default:
        return {};
    }
  } catch (e) {
    return fail(msg, e);
  }
});
