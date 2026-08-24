// Discord Rich Presence – auto-start official extension
// Shows Idle in launcher, then Playing <instance> / 1.21.1 • fabric with elapsed timer

// Idle presence on load
try {
    Aether.discord.setActivity({
        details: "Idle in Launcher",
        state: "Aether",
        largeImageKey: "aether-logo",
        largeText: "Aether Launcher"
    });
} catch (e) {}

// Track running instance for timer
var runningStart = null;
var runningInstanceId = null;

function loaderSmallKey(loader) {
    var l = (loader || "vanilla").toLowerCase();
    if (l.indexOf("fabric") !== -1) return "fabric";
    if (l.indexOf("forge") !== -1) return "forge";
    if (l.indexOf("neoforge") !== -1) return "neoforge";
    if (l.indexOf("quilt") !== -1) return "quilt";
    return "";
}

function ensureDetails(name) {
    if (!name || name.length < 2) return "Playing Minecraft";
    return name;
}

Aether.events.on("instance:state", function(evt) {
    try {
        var id = evt.id;
        var state = evt.state;

        if (state === "Running") {
            var instances = [];
            try { instances = Aether.instances.list(); } catch (e) {}
            var inst = null;
            for (var i = 0; i < instances.length; i++) {
                if (instances[i].id === id) { inst = instances[i]; break; }
            }
            var name = inst ? inst.name : "Minecraft";
            var ver = inst ? inst.version : "";
            var loader = inst ? inst.loader : "vanilla";
            var stateText = ver ? ver + " \u2022 " + loader : loader;

            runningStart = Date.now();
            runningInstanceId = id;

            Aether.discord.setActivity({
                details: ensureDetails(name),
                state: stateText.length < 2 ? "Playing Minecraft" : stateText,
                largeImageKey: "grass-block",
                largeText: name,
                smallImageKey: loaderSmallKey(loader),
                smallText: loader,
                startTimestamp: runningStart
            });
        } else if (state === "Stopped" || state === "Crashed") {
            if (runningInstanceId === id || !runningInstanceId) {
                runningStart = null;
                runningInstanceId = null;
                Aether.discord.setActivity({
                    details: "Idle in Launcher",
                    state: "Aether",
                    largeImageKey: "aether-logo",
                    largeText: "Aether Launcher"
                });
            }
        }
    } catch (e) {}
});
