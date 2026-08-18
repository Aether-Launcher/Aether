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

