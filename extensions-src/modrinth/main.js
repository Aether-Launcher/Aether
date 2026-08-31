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

  // ── install_resourcepack ───────────────────────────────────────────────
  if (msg.type === "install_resourcepack") {
    try {
      var resPath = Aether.instances.installResourcePack(
        msg.instanceId,
        msg.fileName,
        msg.downloadUrl
      );
      Aether.ui.postMessage({
        type: "install_resourcepack_result",
        requestId: msg.requestId,
        success: true,
        fileName: msg.fileName,
        path: resPath,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "install_resourcepack_result",
        requestId: msg.requestId,
        success: false,
        error: String(e),
      });
    }
    return;
  }

  // ── install_shaderpack ─────────────────────────────────────────────────
  if (msg.type === "install_shaderpack") {
    try {
      var shaderPath = Aether.instances.installShaderPack(
        msg.instanceId,
        msg.fileName,
        msg.downloadUrl
      );
      Aether.ui.postMessage({
        type: "install_shaderpack_result",
        requestId: msg.requestId,
        success: true,
        fileName: msg.fileName,
        path: shaderPath,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "install_shaderpack_result",
        requestId: msg.requestId,
        success: false,
        error: String(e),
      });
    }
    return;
  }
});
