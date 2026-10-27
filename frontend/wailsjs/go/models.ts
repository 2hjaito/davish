export namespace app {
	
	export class ServerInput {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    user: string;
	    auth: string;
	    keyPath?: string;
	    color?: string;
	    password: string;
	    passphrase: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.auth = source["auth"];
	        this.keyPath = source["keyPath"];
	        this.color = source["color"];
	        this.password = source["password"];
	        this.passphrase = source["passphrase"];
	    }
	}
	export class ServerView {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    user: string;
	    auth: string;
	    keyPath?: string;
	    color?: string;
	    hasPassword: boolean;
	    hasPassphrase: boolean;
	    state: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.auth = source["auth"];
	        this.keyPath = source["keyPath"];
	        this.color = source["color"];
	        this.hasPassword = source["hasPassword"];
	        this.hasPassphrase = source["hasPassphrase"];
	        this.state = source["state"];
	    }
	}

}

export namespace sysmon {
	
	export class Process {
	    pid: number;
	    ppid: number;
	    user: string;
	    name: string;
	    command: string;
	    state: string;
	    cpu: number;
	    mem: number;
	    memPct: number;
	    threads: number;
	    age: number;
	
	    static createFrom(source: any = {}) {
	        return new Process(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.ppid = source["ppid"];
	        this.user = source["user"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.state = source["state"];
	        this.cpu = source["cpu"];
	        this.mem = source["mem"];
	        this.memPct = source["memPct"];
	        this.threads = source["threads"];
	        this.age = source["age"];
	    }
	}
	export class ProcessList {
	    total: number;
	    procs: Process[];
	
	    static createFrom(source: any = {}) {
	        return new ProcessList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.procs = this.convertValues(source["procs"], Process);
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
	export class SystemInfo {
	    hostname: string;
	    os: string;
	    kernel: string;
	    arch: string;
	    cpuModel: string;
	    virt: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.os = source["os"];
	        this.kernel = source["kernel"];
	        this.arch = source["arch"];
	        this.cpuModel = source["cpuModel"];
	        this.virt = source["virt"];
	    }
	}

}

