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

export namespace fs {
	
	export class CompressPlan {
	    output: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new CompressPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.command = source["command"];
	    }
	}
	export class Entry {
	    name: string;
	    path: string;
	    kind: string;
	    dir: boolean;
	    target?: string;
	    size: number;
	    mode: string;
	    perm: number;
	    modTime: number;
	    uid: number;
	    gid: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.dir = source["dir"];
	        this.target = source["target"];
	        this.size = source["size"];
	        this.mode = source["mode"];
	        this.perm = source["perm"];
	        this.modTime = source["modTime"];
	        this.uid = source["uid"];
	        this.gid = source["gid"];
	    }
	}
	export class File {
	    name: string;
	    path: string;
	    kind: string;
	    dir: boolean;
	    target?: string;
	    size: number;
	    mode: string;
	    perm: number;
	    modTime: number;
	    uid: number;
	    gid: number;
	    data: number[];
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.dir = source["dir"];
	        this.target = source["target"];
	        this.size = source["size"];
	        this.mode = source["mode"];
	        this.perm = source["perm"];
	        this.modTime = source["modTime"];
	        this.uid = source["uid"];
	        this.gid = source["gid"];
	        this.data = source["data"];
	    }
	}
	export class RemovePlan {
	    name: string;
	    path: string;
	    kind: string;
	    dir: boolean;
	    target?: string;
	    size: number;
	    mode: string;
	    perm: number;
	    modTime: number;
	    uid: number;
	    gid: number;
	    nonEmpty: boolean;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new RemovePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.dir = source["dir"];
	        this.target = source["target"];
	        this.size = source["size"];
	        this.mode = source["mode"];
	        this.perm = source["perm"];
	        this.modTime = source["modTime"];
	        this.uid = source["uid"];
	        this.gid = source["gid"];
	        this.nonEmpty = source["nonEmpty"];
	        this.command = source["command"];
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

export namespace transfer {
	
	export class Job {
	    id: string;
	    kind: string;
	    serverID: string;
	    name: string;
	    src: string;
	    dst: string;
	    dir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.serverID = source["serverID"];
	        this.name = source["name"];
	        this.src = source["src"];
	        this.dst = source["dst"];
	        this.dir = source["dir"];
	    }
	}
	export class UploadItem {
	    local: string;
	    name: string;
	    dir: boolean;
	    size: number;
	    exists: boolean;
	    existsDir: boolean;
	    replaceCommand?: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.local = source["local"];
	        this.name = source["name"];
	        this.dir = source["dir"];
	        this.size = source["size"];
	        this.exists = source["exists"];
	        this.existsDir = source["existsDir"];
	        this.replaceCommand = source["replaceCommand"];
	    }
	}
	export class UploadRequest {
	    local: string;
	    replace: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UploadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.local = source["local"];
	        this.replace = source["replace"];
	    }
	}

}

