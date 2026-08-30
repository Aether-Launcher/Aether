Aether.launcher.registerModLoader({
    id: "quilt",
    name: "Quilt",
    description: "Modern, community-driven mod loader with Fabric mod support",
    onLaunch: function(ctx) {
        // Fetch the Quilt loader metadata for this MC version (v3 API, same
        // shape as Fabric's launcherMeta). The array is NOT sorted; the newest
        // loader is selected by semantic version.
        var metaUrl = "https://meta.quiltmc.org/v3/versions/loader/" + ctx.mcVersion;
        var metaStr = Aether.http.get(metaUrl);
        var metaJson = JSON.parse(metaStr);

        if (!metaJson || metaJson.length === 0) {
            throw new Error("Quilt is not available for Minecraft " + ctx.mcVersion);
        }

        // Compare loader versions like "0.30.1-beta.2" / "0.24.0".
        function parseVersion(v) {
            var m = /^(\d+(?:\.\d+)*)(?:-(.+))?$/.exec(v);
            if (!m) return null;
            var base = m[1].split(".").map(function(x) { return parseInt(x, 10); });
            var pre = m[2] ? m[2].split(".") : null;
            return { base: base, pre: pre };
        }
        function compareVersions(a, b) {
            var pa = parseVersion(a), pb = parseVersion(b);
            if (!pa) return 0;
            if (!pb) return 1;
            var n = Math.max(pa.base.length, pb.base.length);
            for (var i = 0; i < n; i++) {
                var x = pa.base[i] || 0, y = pb.base[i] || 0;
                if (x !== y) return x - y;
            }
            if (pa.pre && pb.pre) {
                var m = Math.max(pa.pre.length, pb.pre.length);
                for (var j = 0; j < m; j++) {
                    var xp = pa.pre[j], yp = pb.pre[j];
                    if (xp === undefined) return -1;
                    if (yp === undefined) return 1;
                    if (xp === yp) continue;
                    var xn = parseInt(xp, 10), yn = parseInt(yp, 10);
                    if (!isNaN(xn) && !isNaN(yn)) return xn - yn;
                    return xp < yp ? -1 : 1;
                }
                return 0;
            }
            if (pa.pre) return -1;
            if (pb.pre) return 1;
            return 0;
        }

        metaJson.sort(function(a, b) {
            return compareVersions(b.loader.version, a.loader.version);
        });

        var entry = metaJson[0];
        if (!entry.loader || !entry.launcherMeta) {
            throw new Error("Quilt metadata is missing for Minecraft " + ctx.mcVersion);
        }

        // Download all required libraries: the loader + hashed intermediary
        // jars from Quilt's maven, plus the loader's own dependencies.
        var quiltMaven = "https://maven.quiltmc.org/repository/release/";
        var allLibs = [];
        if (entry.loader.maven) {
            allLibs.push({ name: entry.loader.maven, url: quiltMaven });
        }
        if (entry.hashed && entry.hashed.maven) {
            allLibs.push({ name: entry.hashed.maven, url: quiltMaven });
        }

        var profile = entry.launcherMeta;
        if (profile.libraries && profile.libraries.common) {
            for (var i = 0; i < profile.libraries.common.length; i++) {
                allLibs.push(profile.libraries.common[i]);
            }
        }
        if (profile.libraries && profile.libraries.client) {
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

            var baseUrl = lib.url || quiltMaven;
            if (baseUrl.charAt(baseUrl.length - 1) !== "/") {
                baseUrl += "/";
            }

            var localPath = Aether.fs.download(baseUrl + relPath, relPath);
            cp.push(localPath);
        }

        // Set the main class from the launcher metadata
        var mc = profile.mainClass;
        if (typeof mc === "string") {
            ctx.mainClass = mc;
        } else if (mc && mc.client) {
            ctx.mainClass = mc.client;
        } else {
            ctx.mainClass = "org.quiltmc.loader.impl.launch.knot.QuiltKnotClient";
        }

        ctx.classpath = cp;
        return ctx;
    }
});