// Screenshot Manager Extension Backend
// Runs in the secure Goja Sandbox

Aether.ui.registerSidebarPage({
  id: "screenshot-manager",
  label: "Screenshots",
  icon: "icon.png",
  url: "ui/index.html",
});

Aether.ui.onMessage(function (msg) {
  // ── get_instances ──────────────────────────────────────────────────────
  if (msg.type === "get_instances") {
    try {
      var instances = Aether.instances.list();
      Aether.ui.postMessage({
        type: "instances_result",
        requestId: msg.requestId,
        instances: instances,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "instances_result",
        requestId: msg.requestId,
        instances: [],
        error: String(e),
      });
    }
    return;
  }

  // ── get_screenshots ────────────────────────────────────────────────────
  if (msg.type === "get_screenshots") {
    try {
      var screenshots = Aether.instances.listScreenshots(msg.instanceId);
      Aether.ui.postMessage({
        type: "screenshots_result",
        requestId: msg.requestId,
        screenshots: screenshots,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "screenshots_result",
        requestId: msg.requestId,
        screenshots: [],
        error: String(e),
      });
    }
    return;
  }

  // ── delete_screenshot ──────────────────────────────────────────────────
  if (msg.type === "delete_screenshot") {
    try {
      Aether.instances.deleteScreenshot(msg.instanceId, msg.fileName);
      Aether.ui.postMessage({
        type: "delete_result",
        requestId: msg.requestId,
        success: true,
        fileName: msg.fileName,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "delete_result",
        requestId: msg.requestId,
        success: false,
        error: String(e),
      });
    }
    return;
  }

  // ── open_screenshot ────────────────────────────────────────────────────
  if (msg.type === "open_screenshot") {
    try {
      Aether.instances.openScreenshot(msg.instanceId, msg.fileName);
      Aether.ui.postMessage({
        type: "open_result",
        requestId: msg.requestId,
        success: true,
      });
    } catch (e) {
      Aether.ui.postMessage({
        type: "open_result",
        requestId: msg.requestId,
        success: false,
        error: String(e),
      });
    }
    return;
  }
});
