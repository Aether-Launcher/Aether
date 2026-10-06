// main.js - Skin Selector backend (Goja sandbox).
// Public DB: Mojang official first, NameMC fallback. Account skins/capes via Go.

Aether.ui.registerSidebarPage({
    id: "skin-selector",
    label: "Skin Selector",
    url: "ui/index.html"
});

function mojangProfile(username) {
    var profileRes = Aether.http.get("https://api.mojang.com/users/profiles/minecraft/" + encodeURIComponent(username));
    var profile = JSON.parse(profileRes);
    if (!profile || !profile.id) throw new Error("Player not found (Mojang)");
    var uuid = profile.id;
    var sessionRes = Aether.http.get("https://sessionserver.mojang.com/session/minecraft/profile/" + encodeURIComponent(uuid));
    var session = JSON.parse(sessionRes);
    var skinUrl = "";
    var capeUrl = "";
    var modelType = "default";
    if (session && session.properties) {
        for (var i = 0; i < session.properties.length; i++) {
            if (session.properties[i].name === "textures") {
                // Goja has no atob; decode via Go-side? Textures value is base64 JSON.
                // Fall back to NameMC path below if decode unavailable.
                try {
                    var decoded = decodeBase64Json(session.properties[i].value);
                    if (decoded && decoded.textures) {
                        if (decoded.textures.SKIN) {
                            skinUrl = decoded.textures.SKIN.url;
                            if (decoded.textures.SKIN.metadata && decoded.textures.SKIN.metadata.model === "slim") modelType = "slim";
                        }
                        if (decoded.textures.CAPE) capeUrl = decoded.textures.CAPE.url;
                    }
                } catch (e) { /* fall through to NameMC */ }
            }
        }
    }
    return { uuid: uuid, username: profile.name || username, skinUrl: skinUrl, capeUrl: capeUrl, modelType: modelType, source: "mojang" };
}

// Minimal base64 JSON decoder in pure JS (Goja-safe, no Node Buffer/atob).
function decodeBase64Json(b64) {
    var chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";
    var output = "";
    var i = 0;
    b64 = String(b64).replace(/[^A-Za-z0-9\+\/\=]/g, "");
    while (i < b64.length) {
        var enc1 = chars.indexOf(b64.charAt(i++));
        var enc2 = chars.indexOf(b64.charAt(i++));
        var enc3 = chars.indexOf(b64.charAt(i++));
        var enc4 = chars.indexOf(b64.charAt(i++));
        var chr1 = (enc1 << 2) | (enc2 >> 4);
        var chr2 = ((enc2 & 15) << 4) | (enc3 >> 2);
        var chr3 = ((enc3 & 3) << 6) | enc4;
        output += String.fromCharCode(chr1);
        if (enc3 !== 64) output += String.fromCharCode(chr2);
        if (enc4 !== 64) output += String.fromCharCode(chr3);
    }
    // decode UTF-8 bytes to string
    try {
        var utf = decodeURIComponent(escape(output));
        return JSON.parse(utf);
    } catch (e) {
        return JSON.parse(output);
    }
}

function namemcProfile(username) {
    var resp = Aether.http.get("https://api.namemc.com/v2/profile/" + encodeURIComponent(username));
    var data = JSON.parse(resp);
    if (!data || !data.id) throw new Error("Player not found (NameMC)");
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
    return { uuid: data.id, username: data.name, skinUrl: skinUrl, capeUrl: capeUrl, modelType: modelType, source: "namemc" };
}

Aether.ui.onMessage(function (msg) {
    // NOTE: request/response uses the RETURN value only (it is emitted as
    // extension:message with the same reqId). Do NOT also call
    // Aether.ui.postMessage here — that delivers a duplicate message and the
    // empty one clobbers the real data in the UI.
    if (msg.type === "gallery_lookup") {
        var lastErr = "";
        try {
            var mojang = mojangProfile(msg.username);
            if (mojang.skinUrl) {
                return { type: "gallery_result", reqId: msg.reqId, uuid: mojang.uuid, username: mojang.username, skinUrl: mojang.skinUrl, capeUrl: mojang.capeUrl, modelType: mojang.modelType, source: mojang.source };
            }
            lastErr = "Mojang returned no skin";
        } catch (e) { lastErr = e.toString(); }
        try {
            var nm = namemcProfile(msg.username);
            return { type: "gallery_result", reqId: msg.reqId, uuid: nm.uuid, username: nm.username, skinUrl: nm.skinUrl, capeUrl: nm.capeUrl, modelType: nm.modelType, source: nm.source };
        } catch (e2) {
            return { type: "gallery_result", reqId: msg.reqId, error: "Not found. Mojang: " + lastErr + " | NameMC: " + e2.toString() };
        }
    }

    if (msg.type === "get_account") {
        try {
            var acc = Aether.account.getActive();
            return { type: "account_result", reqId: msg.reqId, account: acc };
        } catch (e) {
            return { type: "account_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    if (msg.type === "my_skins") {
        try {
            var skins = Aether.skins.listMine();
            return { type: "my_skins_result", reqId: msg.reqId, skins: skins };
        } catch (e) {
            return { type: "my_skins_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    if (msg.type === "my_capes") {
        try {
            var capes = Aether.capes.listMine();
            return { type: "my_capes_result", reqId: msg.reqId, capes: capes };
        } catch (e) {
            return { type: "my_capes_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    if (msg.type === "upload_skin") {
        try {
            var skin = Aether.skins.upload(msg.data, msg.variant || "classic");
            return { type: "upload_result", reqId: msg.reqId, skin: skin };
        } catch (e) {
            return { type: "upload_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    if (msg.type === "apply_url") {
        try {
            var applied = Aether.skins.applyUrl(msg.url, msg.variant || "classic");
            return { type: "upload_result", reqId: msg.reqId, skin: applied };
        } catch (e) {
            return { type: "upload_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    if (msg.type === "equip_cape") {
        try {
            Aether.capes.equip(msg.capeId);
            return { type: "cape_result", reqId: msg.reqId, ok: true };
        } catch (e) {
            return { type: "cape_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    if (msg.type === "hide_cape") {
        try {
            Aether.capes.hide();
            return { type: "cape_result", reqId: msg.reqId, ok: true };
        } catch (e) {
            return { type: "cape_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    if (msg.type === "export_skin") {
        try {
            var path = Aether.skins.export(msg.data, msg.filename || "skin.png");
            return { type: "export_result", reqId: msg.reqId, path: path };
        } catch (e) {
            return { type: "export_result", reqId: msg.reqId, error: e.toString() };
        }
    }

    return {};
});
