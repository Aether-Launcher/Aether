Aether.ui.registerSidebarPage({
    id: "skin-maker",
    label: "Skins",
    url: "ui/index.html"
});

Aether.ui.onMessage(function (msg) {
    if (msg.type === "fetch_profile") {
        try {
            var resp = Aether.http.get("https://api.namemc.com/v2/profile/" + encodeURIComponent(msg.username));
            var data = JSON.parse(resp);
            if (!data || !data.id) {
                Aether.ui.postMessage({ type: "profile_result", reqId: msg.reqId, error: "Player not found or invalid response from NameMC API. Raw response: " + resp });
                return;
            }
            var skinUrl = "";
            var capeUrl = "";
            var modelType = "default";
            if (data.textures) {
                if (data.textures.skin) {
                    skinUrl = data.textures.skin.url;
                    if (data.textures.skin.model === "slim") modelType = "slim";
                }
                if (data.textures.cape) capeUrl = data.textures.cape.url;
            }
            Aether.ui.postMessage({
                type: "profile_result",
                reqId: msg.reqId,
                uuid: data.id,
                username: data.name,
                skinUrl: skinUrl,
                capeUrl: capeUrl,
                modelType: modelType
            });
        } catch (e) {
            Aether.ui.postMessage({ type: "profile_result", reqId: msg.reqId, error: "Failed to fetch profile: " + e.toString() });
        }
    }

    if (msg.type === "export_skin") {
        try {
            var path = Aether.skins.export(msg.data, msg.filename || "skin.png");
            Aether.ui.postMessage({ type: "export_result", reqId: msg.reqId, path: path });
        } catch (e) {
            Aether.ui.postMessage({ type: "export_result", reqId: msg.reqId, error: "Failed to export skin: " + e.toString() });
        }
    }
});
