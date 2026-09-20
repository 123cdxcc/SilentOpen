export namespace process {
	
	export class Listener {
	    ip: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new Listener(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.port = source["port"];
	    }
	}
	export class ProcessInfo {
	    pid: number;
	    ppid?: number;
	    parentName: string;
	    name: string;
	    project: string;
	    cwd: string;
	    listeners: Listener[];
	    command: string;
	    startedAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.ppid = source["ppid"];
	        this.parentName = source["parentName"];
	        this.name = source["name"];
	        this.project = source["project"];
	        this.cwd = source["cwd"];
	        this.listeners = this.convertValues(source["listeners"], Listener);
	        this.command = source["command"];
	        this.startedAt = source["startedAt"];
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
	export class ProcessSnapshot {
	    processes: ProcessInfo[];
	    collectedAt: number;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProcessSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.processes = this.convertValues(source["processes"], ProcessInfo);
	        this.collectedAt = source["collectedAt"];
	        this.warnings = source["warnings"];
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
	
	export class Result {
	    currentVersion: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	    skipped: boolean;
	    notes: string;
	    pageUrl: string;
	    assetName: string;
	    checkedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.skipped = source["skipped"];
	        this.notes = source["notes"];
	        this.pageUrl = source["pageUrl"];
	        this.assetName = source["assetName"];
	        this.checkedAt = source["checkedAt"];
	    }
	}

}

