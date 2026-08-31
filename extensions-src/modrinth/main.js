// Modrinth Extension – backend sandbox script
// Handles IPC messages forwarded from the sidebar iframe UI.

Aether.ui.registerSidebarPage({
  id: "modrinth",
  label: "Modrinth",
  icon: "icon.png",
  url: "ui/index.html",
});

Aether.ui.onMessage(function (msg) {
  // ── get_instances ──────────────────────────────────────────────────────
  if (msg.type === "get_instances") {
    var instances = Aether.instances.list();
    Aether.ui.postMessage({
      type: "instances_result",
      requestId: msg.requestId,
      instances: instances,
    });
    return;
  }

  // ── install_mod ────────────────────────────────────────────────────────
  if (msg.type === "install_mod") {
    try {
      var path = Aether.instances.installMod(
        msg.instanceId,
        msg.jarName,
        msg.downloadUrl
      );
      Aether.ui.postMessage({
        type: "install_result",
        requestId: msg.requestId,
        success: true,
        jarName: msg.jarName,
        path: path,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "install_result",
        requestId: msg.requestId,
        success: false,
        error: String(e),
      });
    }
    return;
  }

  // ── install_modpack ────────────────────────────────────────────────────
  // Downloads the .mrpack, creates a new instance, and installs all files.
  // This call blocks until the entire pack is installed.
  if (msg.type === "install_modpack") {
    try {
      var instanceId = Aether.instances.installModpack(
        msg.packUrl,
        msg.packName
      );
      Aether.ui.postMessage({
        type: "install_modpack_result",
        requestId: msg.requestId,
        success: true,
        instanceId: instanceId,
        packName: msg.packName,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "install_modpack_result",
        requestId: msg.requestId,
        success: false,
        error: String(e),
      });
    }
    return;
  }
});
