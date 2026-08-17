Aether.launcher.registerModLoader({
    id: "forge",
    name: "Forge",
    description: "The original Minecraft mod loader",
    onLaunch: function(ctx) {
        var mcVersion = ctx.mcVersion;

        // 1. Fetch promotions to get the recommended Forge version
        var promosStr = Aether.http.get("https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json");
        var promos = JSON.parse(promosStr);
        var forgeVer = promos.promos[mcVersion + "-recommended"] || promos.promos[mcVersion + "-latest"];
        if (!forgeVer) {
            throw new Error("Forge is not available for Minecraft " + mcVersion);
        }

        var fullVer = mcVersion + "-" + forgeVer;
        var mavenUrl = "https://maven.minecraftforge.net/";
        var basePath = "net/minecraftforge/forge/" + fullVer + "/forge-" + fullVer;

        // 2. Determine which jar to download (client for modern, universal for legacy)
        var jarType = "client";
        var mainClass = "cpw.mods.modlauncher.Launcher";
        var jvmArgs = [];
        var gameArgs = [];

        var cp = [];
        for (var idx = 0; idx < ctx.classpath.length; idx++) {
            cp.push(ctx.classpath[idx]);
        }

        // 3. Try to fetch the Forge version JSON for library listing
        var jsonUrl = mavenUrl + basePath + ".json";
        try {
            var jsonStr = Aether.http.get(jsonUrl);
            var versionInfo = JSON.parse(jsonStr);

            // Download all libraries from the version JSON
            function collectArgs(args) {
                var out = [];
                if (!args) return out;
                for (var a = 0; a < args.length; a++) {
                    var item = args[a];
                    if (typeof item === "string") {
                        out.push(item);
                    } else if (item && item.value) {
                        if (typeof item.value === "string") {
                            out.push(item.value);
                        } else if (Array.isArray(item.value)) {
                            for (var av = 0; av < item.value.length; av++) out.push(item.value[av]);
                        }
                    }
                }
                return out;
            }

            if (versionInfo.arguments) {
                jvmArgs = collectArgs(versionInfo.arguments.jvm);
                gameArgs = collectArgs(versionInfo.arguments.game);
            }

            var libs = versionInfo.libraries || [];
            for (var i = 0; i < libs.length; i++) {
                var lib = libs[i];
                if (lib.name && lib.name.indexOf("net.minecraftforge:forge:") === -1) {
                    if (lib.downloads && lib.downloads.artifact && lib.downloads.artifact.url) {
                        try {
                            var localPath = Aether.fs.download(lib.downloads.artifact.url, lib.downloads.artifact.path);
                            cp.push(localPath);
                        } catch (libErr) {
                            // Log the failure but continue — a missing optional lib shouldn't abort the launch
                            throw new Error("[Forge] Failed to download library " + lib.name + ": " + libErr);
                        }
                    }
                }
            }

            // Use mainClass from version JSON
            var mc = versionInfo.mainClass;
            if (typeof mc === "string") {
                mainClass = mc;
            } else if (mc && mc.client) {
                mainClass = mc.client;
            }
        } catch (jsonErr) {
            // Version JSON not available for this Forge build — proceed with client jar only
            throw new Error("[Forge] Could not fetch version JSON: " + jsonErr);
        }

        // 4. Download the Forge client jar, fall back to universal for legacy versions
        var jarPath;
        var jarUrl = mavenUrl + basePath + "-" + jarType + ".jar";
        try {
            jarPath = Aether.fs.download(jarUrl, basePath + "-" + jarType + ".jar");
        } catch (e) {
            // Fallback to universal jar for older Forge versions
            jarUrl = mavenUrl + basePath + "-universal.jar";
            jarPath = Aether.fs.download(jarUrl, basePath + "-universal.jar");
        }
        cp.push(jarPath);

        ctx.mainClass = mainClass;
        ctx.classpath = cp;
        ctx.jvmArgs = jvmArgs;
        ctx.gameArgs = gameArgs;
        return ctx;
    }
});
