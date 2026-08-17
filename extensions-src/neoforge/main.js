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
        var mainClass = "cpw.mods.modlauncher.Launcher";
        var jvmArgs = [];
        var gameArgs = [];

        var cp = [];
        for (var idx = 0; idx < ctx.classpath.length; idx++) {
            cp.push(ctx.classpath[idx]);
        }

        // 2. Fetch the NeoForge version JSON for library listing
        var jsonUrl = mavenUrl + basePath + ".json";
        try {
            var jsonStr = Aether.http.get(jsonUrl);
            var versionInfo = JSON.parse(jsonStr);

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
            for (var j = 0; j < libs.length; j++) {
                var lib = libs[j];
                if (lib.name && lib.name.indexOf("net.neoforged:neoforge:") === -1) {
                    if (lib.downloads && lib.downloads.artifact && lib.downloads.artifact.url) {
                        try {
                            var localPath = Aether.fs.download(lib.downloads.artifact.url, lib.downloads.artifact.path);
                            cp.push(localPath);
                        } catch (libErr) {
                            // Log the failure but continue — a missing optional lib shouldn't abort the launch
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
        } catch (jsonErr) {
            // Version JSON not available — proceed with main jar only
            throw new Error("[NeoForge] Could not fetch version JSON: " + jsonErr);
        }

        // 3. Download the NeoForge client jar, fall back to the bundled main jar
        var jarPath;
        var jarUrl = mavenUrl + basePath + "-client.jar";
        try {
            jarPath = Aether.fs.download(jarUrl, basePath + "-client.jar");
        } catch (e) {
            // Fallback for versions that bundle the client into the main jar
            jarUrl = mavenUrl + basePath + ".jar";
            jarPath = Aether.fs.download(jarUrl, basePath + ".jar");
        }
        cp.push(jarPath);

        ctx.mainClass = mainClass;
        ctx.classpath = cp;
        ctx.jvmArgs = jvmArgs;
        ctx.gameArgs = gameArgs;
        return ctx;
    }
});
