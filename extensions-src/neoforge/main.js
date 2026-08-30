Aether.launcher.registerModLoader({
    id: "neoforge",
    name: "NeoForge",
    description: "Modern, community-maintained fork of Forge",
    onLaunch: function(ctx) {
        var mcVersion = ctx.mcVersion;

        // 1. Fetch NeoForge Maven API to get the version list
        var apiUrl = "https://maven.neoforged.net/api/maven/versions/releases/net/neoforged/neoforge";
        var apiStr = Aether.http.get(apiUrl);
        var data = JSON.parse(apiStr);
        var versions = data.versions || [];

        // Build version prefix matching Minecraft version
        var parts = mcVersion.split(".");
        var major = parts[0];
        var prefix = "";
        if (major === "1") {
            var minor = parts[1] || "0";
            var patch = parts[2] || "0";
            prefix = minor + "." + patch + ".";
        } else {
            var minor = parts[1] || "0";
            var patch = parts[2] || "0";
            prefix = major + "." + minor + "." + patch + ".";
        }

        var compatible = versions.filter(function(v) {
            return v.indexOf(prefix) === 0;
        });

        if (compatible.length === 0) {
            // Fallback: try major.minor prefix
            var altPrefix = major === "1" ? (parts[1] || "0") + "." : major + "." + (parts[1] || "0") + ".";
            compatible = versions.filter(function(v) {
                return v.indexOf(altPrefix) === 0;
            });
        }

        if (compatible.length === 0) {
            throw new Error("NeoForge is not available for Minecraft " + mcVersion);
        }

        // Select the latest compatible version
        var neoforgeVer = compatible[compatible.length - 1];

        var mavenUrl = "https://maven.neoforged.net/releases/";
        var basePath = "net/neoforged/neoforge/" + neoforgeVer + "/neoforge-" + neoforgeVer;
        var mainClass = "cpw.mods.bootstraplauncher.BootstrapLauncher";
        var jvmArgs = [];
        var gameArgs = [];

        var cp = [];
        for (var idx = 0; idx < ctx.classpath.length; idx++) {
            cp.push(ctx.classpath[idx]);
        }

        // 2. Fetch the NeoForge version JSON for library listing
        // Maven no longer publishes a standalone .json for NeoForge; Prism's
        // upstream mirror does. Tries Maven first, then Prism, then continues
        // with defaults to avoid fatal error on 404 (the 1.21.1 bug).
        var versionInfo = null;
        var jsonUrl = mavenUrl + basePath + ".json";
        var prismUrl = "https://raw.githubusercontent.com/PrismLauncher/meta-upstream/master/neoforge/version_manifests/" + neoforgeVer + ".json";
        try {
            var jsonStr = Aether.http.get(jsonUrl);
            versionInfo = JSON.parse(jsonStr);
        } catch (mavenErr) {
            try {
                var prismStr = Aether.http.get(prismUrl);
                versionInfo = JSON.parse(prismStr);
            } catch (prismErr) {
                // No version JSON available – proceed with main jar only (non-fatal)
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
            for (var j = 0; j < libs.length; j++) {
                var lib = libs[j];
                if (lib.name && lib.name.indexOf("net.neoforged:neoforge:") === -1) {
                    if (lib.downloads && lib.downloads.artifact && lib.downloads.artifact.url) {
                        try {
                            var localPath = Aether.fs.download(lib.downloads.artifact.url, lib.downloads.artifact.path);
                            cp.push(localPath);
                        } catch (libErr) {
                            throw new Error("[NeoForge] Failed to download library " + lib.name + ": " + libErr);
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

        // 3. Download the NeoForge jar – modern NeoForge only publishes universal/installer
        var jarPath = null;
        var jarCandidates = [
            mavenUrl + basePath + "-universal.jar",
            mavenUrl + basePath + "-installer.jar",
            mavenUrl + basePath + "-client.jar",
            mavenUrl + basePath + ".jar"
        ];
        var lastErr = null;
        for (var c = 0; c < jarCandidates.length; c++) {
            var jarUrl = jarCandidates[c];
            var rel = jarUrl.substring(jarUrl.indexOf("net/neoforged"));
            try {
                jarPath = Aether.fs.download(jarUrl, rel);
                break;
            } catch (e) {
                lastErr = e;
            }
        }
        if (!jarPath) {
            throw new Error("[NeoForge] Could not download NeoForge jar: " + lastErr);
        }
        cp.push(jarPath);

        ctx.mainClass = mainClass;
        ctx.classpath = cp;
        ctx.jvmArgs = jvmArgs;
        ctx.gameArgs = gameArgs;
        return ctx;
    }
});
