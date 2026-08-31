// CurseForge Browser Extension Backend
// Runs in the secure Goja Sandbox

Aether.ui.registerSidebarPage({
    id: "curseforge",
    label: "CurseForge",
    url: "ui/index.html"
});


// ── Handle IPC messages from the UI iframe ────────────────────────────────

Aether.ui.onMessage(function(msg) {

    // ── Get instances ─────────────────────────────────────────────────────

    if (msg.type === "get_instances") {

        try {
            var instances = Aether.instances.list();

            Aether.ui.postMessage({
                type: "instances_result",
                requestId: msg.requestId,
                instances: instances
            });

        } catch (e) {

            Aether.ui.postMessage({
                type: "instances_result",
                requestId: msg.requestId,
                instances: [],
                error: String(e)
            });
        }

        return;
    }


    // ── Install CurseForge mod ────────────────────────────────────────────

    if (msg.type === "install_mod") {

        try {

            if (!msg.instanceId) {
                throw new Error("Missing instanceId");
            }

            if (!msg.jarName) {
                throw new Error("Missing jarName");
            }

            if (!msg.downloadUrl) {
                throw new Error("Missing downloadUrl");
            }


            /*
             * msg.downloadUrl comes directly from CurseForge.
             *
             * Example:
             *
             * https://edge.forgecdn.net/files/...
             *
             * Aether handles the actual download and installation.
             */

            Aether.instances.installMod(
                msg.instanceId,
                msg.jarName,
                msg.downloadUrl
            );


            Aether.ui.postMessage({
                type: "install_result",
                requestId: msg.requestId,
                success: true,
                jarName: msg.jarName
            });

        } catch (e) {

            Aether.ui.postMessage({
                type: "install_result",
                requestId: msg.requestId,
                success: false,
                error: String(e)
            });
        }

        return;
    }
});