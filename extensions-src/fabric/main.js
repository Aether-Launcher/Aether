Aether.launcher.registerModLoader({
    id: "fabric",
    name: "Fabric",
    description: "Lightweight and fast mod loader",
    onLaunch: function(ctx) {
        // Fetch the Fabric loader profile for this MC version
        var metaUrl = "https://meta.fabricmc.net/v2/versions/loader/" + ctx.mcVersion;
        var metaStr = Aether.http.get(metaUrl);
        var metaJson = JSON.parse(metaStr);

        if (!metaJson || metaJson.length === 0) {
            throw new Error("Fabric is not available for Minecraft " + ctx.mcVersion);
        }

        var entry = metaJson[0];
        var profile = entry.launcherMeta;

        // Download all required libraries (common + client)
        var allLibs = [];
        if (entry.loader && entry.loader.maven) {
            allLibs.push({
                name: entry.loader.maven,
                url: "https://maven.fabricmc.net/"
            });
        }
        if (entry.intermediary && entry.intermediary.maven) {
            allLibs.push({
                name: entry.intermediary.maven,
                url: "https://maven.fabricmc.net/"
            });
        }
        if (profile.libraries.common) {
            for (var i = 0; i < profile.libraries.common.length; i++) {
                allLibs.push(profile.libraries.common[i]);
            }
        }
        if (profile.libraries.client) {
            for (var i = 0; i < profile.libraries.client.length; i++) {
                allLibs.push(profile.libraries.client[i]);
            }
        }

        var cp = [];
        for (var idx = 0; idx < ctx.classpath.length; idx++) {
            cp.push(ctx.classpath[idx]);
        }

        for (var i = 0; i < allLibs.length; i++) {
            var lib = allLibs[i];
            var parts = lib.name.split(":"); // group:artifact:version
            var groupPath = parts[0].replace(/\./g, "/");
            var artifact = parts[1];
            var version = parts[2];
            var relPath = groupPath + "/" + artifact + "/" + version + "/" + artifact + "-" + version + ".jar";

            var baseUrl = lib.url || "https://maven.fabricmc.net/";
            if (baseUrl.charAt(baseUrl.length - 1) !== "/") {
                baseUrl += "/";
            }

            var localPath = Aether.fs.download(baseUrl + relPath, relPath);
            cp.push(localPath);
        }

        // Set the main class
        // In modern Fabric API, mainClass is either a plain string or { client: "...", server: "..." }
        var mc = profile.mainClass;
        if (typeof mc === "string") {
            ctx.mainClass = mc;
        } else if (mc && mc.client) {
            ctx.mainClass = mc.client;
        } else {
            ctx.mainClass = "net.fabricmc.loader.impl.launch.knot.KnotClient";
        }

        ctx.classpath = cp;
        return ctx;
    }
});
