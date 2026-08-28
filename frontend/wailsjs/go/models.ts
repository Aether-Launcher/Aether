export namespace auth {
	
	export class Account {
	    id: string;
	    type: string;
	    username: string;
	    accessToken?: string;
	    refreshToken?: string;
	    expiresAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.username = source["username"];
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.expiresAt = source["expiresAt"];
	    }
	}

}

export namespace extensions {
	
	export class Extension {
	    id: string;
	    name: string;
	    version: string;
	    author: string;
	    description: string;
	    status: string;
	    memory: string;
	    cpu: string;
	    trust: string;
	    iconUrl?: string;
	    reloading: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Extension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.memory = source["memory"];
	        this.cpu = source["cpu"];
	        this.trust = source["trust"];
	        this.iconUrl = source["iconUrl"];
	        this.reloading = source["reloading"];
	    }
	}
	export class ExtensionUpdate {
	    id: string;
	    name: string;
	    currentVersion: string;
	    newVersion: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtensionUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.currentVersion = source["currentVersion"];
	        this.newVersion = source["newVersion"];
	        this.url = source["url"];
	    }
	}

}

export namespace instance {
	
	export class Instance {
	    id: string;
	    name: string;
	    version: string;
	    loader: string;
	    memory: string;
	    lastPlayed: string;
	    installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.loader = source["loader"];
	        this.memory = source["memory"];
	        this.lastPlayed = source["lastPlayed"];
	        this.installed = source["installed"];
	    }
	}

}

export namespace main {
	
	export class JavaRuntimeStatus {
	    version: number;
	    installed: boolean;
	    path: string;
	    isSystem: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JavaRuntimeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.installed = source["installed"];
	        this.path = source["path"];
	        this.isSystem = source["isSystem"];
	    }
	}
	export class ModLoaderInfo {
	    id: string;
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ModLoaderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}

}

export namespace mojang {
	
	export class ServiceStatus {
	    name: string;
	    host: string;
	    reachable: boolean;
	    latencyMs: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.reachable = source["reachable"];
	        this.latencyMs = source["latencyMs"];
	        this.error = source["error"];
	    }
	}
	export class ConnectivityStatus {
	    overall: string;
	    // Go type: time
	    checkedAt: any;
	    services: ServiceStatus[];
	
	    static createFrom(source: any = {}) {
	        return new ConnectivityStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overall = source["overall"];
	        this.checkedAt = this.convertValues(source["checkedAt"], null);
	        this.services = this.convertValues(source["services"], ServiceStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace settings {
	
	export class GlobalSettings {
	    defaultMemory: string;
	    closeOnLaunch: boolean;
	    developerMode: boolean;
	    disableExtensions: boolean;
	    garbageCollector?: string;
	    customJvmArgs?: string;
	    autoCheckUpdates: boolean;
	    includeBetaUpdates: boolean;
	    activeTheme?: string;
	
	    static createFrom(source: any = {}) {
	        return new GlobalSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultMemory = source["defaultMemory"];
	        this.closeOnLaunch = source["closeOnLaunch"];
	        this.developerMode = source["developerMode"];
	        this.disableExtensions = source["disableExtensions"];
	        this.garbageCollector = source["garbageCollector"];
	        this.customJvmArgs = source["customJvmArgs"];
	        this.autoCheckUpdates = source["autoCheckUpdates"];
	        this.includeBetaUpdates = source["includeBetaUpdates"];
	        this.activeTheme = source["activeTheme"];
	    }
	}

}

export namespace theme {
	
	export class Info {
	    id: string;
	    name: string;
	    version: string;
	    author?: string;
	    description?: string;
	    iconUrl?: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.description = source["description"];
	        this.iconUrl = source["iconUrl"];
	        this.active = source["active"];
	    }
	}
	export class Manifest {
	    id: string;
	    name: string;
	    version: string;
	    author?: string;
	    description?: string;
	    icon?: string;
	    css?: string;
	    overwrite?: string;
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	        this.css = source["css"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class InstallResult {
	    Manifest: Manifest;
	    Warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Manifest = this.convertValues(source["Manifest"], Manifest);
	        this.Warnings = source["Warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace update {
	
	export class Info {
	    version: string;
	    assetName: string;
	    downloadUrl: string;
	    releaseNotes: string;
	    isPrerelease: boolean;
	    installerOnly?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.assetName = source["assetName"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseNotes = source["releaseNotes"];
	        this.isPrerelease = source["isPrerelease"];
	        this.installerOnly = source["installerOnly"];
	    }
	}

}

