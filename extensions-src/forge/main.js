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

        var mainClass = "cpw.mods.bootstraplauncher.BootstrapLauncher";
        // Fallback for very old Forge that still uses modlauncher
        // will be overwritten if versionInfo provides its own
        var jvmArgs = [];
        var gameArgs = [];

        var cp = [];
        for (var idx = 0; idx < ctx.classpath.length; idx++) {
            cp.push(ctx.classpath[idx]);
        }

        // 2. Try to fetch the Forge version JSON for library listing
        // Maven no longer publishes a standalone .json for modern Forge
        // (it lives inside the installer jar), so we try Maven first and
        // fall back to Prism's upstream mirror, then continue with defaults.
        var versionInfo = null;
        var jsonUrl = mavenUrl + basePath + ".json";
        var prismUrl = "https://raw.githubusercontent.com/PrismLauncher/meta-upstream/master/forge/version_manifests/" + fullVer + ".json";
        try {
            var jsonStr = Aether.http.get(jsonUrl);
            versionInfo = JSON.parse(jsonStr);
        } catch (mavenErr) {
            try {
                var prismStr = Aether.http.get(prismUrl);
                versionInfo = JSON.parse(prismStr);
            } catch (prismErr) {
                // No version JSON available – proceed with defaults, do not fatal
            }
        }

        if (versionInfo) {
            function collectArgs(args) {
                var out = [];
                if (!args) return out;
                function allowed(item) {
                    if (!item.rules || item.rules.length === 0) return true;
                    var result = false;
                    for (var r = 0; r < item.rules.length; r++) {
                        var rule = item.rules[r];
                        if (rule.features && Object.keys(rule.features).length > 0) { result = false; break; }
                        var platform = rule.os;
                        if (platform && platform.name && platform.name !== ctx.os) continue;
                        if (platform && platform.arch && platform.arch !== ctx.arch && !(platform.arch === "x86" && ctx.arch === "386")) continue;
                        result = rule.action === "allow";
                    }
                    return result;
                }
                for (var a = 0; a < args.length; a++) {
                    var item = args[a];
                    if (typeof item === "string") {
                        out.push(item);
                    } else if (item && allowed(item)) {
                        if (typeof item.value === "string") out.push(item.value);
                        else if (Array.isArray(item.value)) {
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
                            throw new Error("[Forge] Failed to download library " + lib.name + ": " + libErr);
                        }
                    }
                }
            }

            var mc = versionInfo.mainClass;
            if (typeof mc === "string") {
                mainClass = mc;
            } else if (mc && mc.client) {
                mainClass = mc.client;
            }
        }

        // 3. Download the Forge jar – modern Forge publishes -client, legacy only -universal
        var jarPath = null;
        var jarCandidates = [
            mavenUrl + basePath + "-client.jar",
            mavenUrl + basePath + "-universal.jar",
            mavenUrl + basePath + "-installer.jar",
            mavenUrl + basePath + ".jar",
            "https://files.minecraftforge.net/maven/" + basePath + "-client.jar",
            "https://files.minecraftforge.net/maven/" + basePath + "-universal.jar"
        ];
        var lastErr = null;
        for (var c = 0; c < jarCandidates.length; c++) {
            var jarUrl = jarCandidates[c];
            var jarRel = basePath + "-" + jarUrl.substring(jarUrl.lastIndexOf("-") + 1);
            // keep the filename segment as-is for download cache key
            var rel = basePath + jarUrl.substring(jarUrl.lastIndexOf("/"));
            try {
                jarPath = Aether.fs.download(jarUrl, rel.substring(1));
                break;
            } catch (e) {
                lastErr = e;
            }
        }
        if (!jarPath) {
            throw new Error("[Forge] Could not download Forge jar: " + lastErr);
        }
        cp.push(jarPath);

        ctx.mainClass = mainClass;
        ctx.classpath = cp;
        ctx.jvmArgs = jvmArgs;
        ctx.gameArgs = gameArgs;
        return ctx;
    }
});
