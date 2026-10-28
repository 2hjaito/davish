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

export namespace audit {
	
	export class Entry {
	    id: number;
	    time: number;
	    serverID: string;
	    app: string;
	    command: string;
	    exit: number;
	    durMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.serverID = source["serverID"];
	        this.app = source["app"];
	        this.command = source["command"];
	        this.exit = source["exit"];
	        this.durMs = source["durMs"];
	    }
	}

}

export namespace cron {
	
	export class Change {
	    op: string;
	    id: string;
	    schedule: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.op = source["op"];
	        this.id = source["id"];
	        this.schedule = source["schedule"];
	        this.command = source["command"];
	    }
	}
	export class Env {
	    source: string;
	    name: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Env(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.name = source["name"];
	        this.value = source["value"];
	    }
	}
	export class Schedule {
	    raw: string;
	    reboot: boolean;
	    minute: number[];
	    hour: number[];
	    dom: number[];
	    month: number[];
	    dow: number[];
	    domAny: boolean;
	    dowAny: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Schedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.raw = source["raw"];
	        this.reboot = source["reboot"];
	        this.minute = source["minute"];
	        this.hour = source["hour"];
	        this.dom = source["dom"];
	        this.month = source["month"];
	        this.dow = source["dow"];
	        this.domAny = source["domAny"];
	        this.dowAny = source["dowAny"];
	    }
	}
	export class Job {
	    id: string;
	    source: string;
	    line: number;
	    schedule: Schedule;
	    user?: string;
	    command: string;
	    enabled: boolean;
	    next: number[];
	    editable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.line = source["line"];
	        this.schedule = this.convertValues(source["schedule"], Schedule);
	        this.user = source["user"];
	        this.command = source["command"];
	        this.enabled = source["enabled"];
	        this.next = source["next"];
	        this.editable = source["editable"];
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
	export class Plan {
	    command: string;
	    before: string;
	    after: string;
	    backup: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.backup = source["backup"];
	        this.version = source["version"];
	    }
	}
	export class Preview {
	    schedule: Schedule;
	    next: number[];
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schedule = this.convertValues(source["schedule"], Schedule);
	        this.next = source["next"];
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
	export class Runs {
	    lines: string[];
	    restricted: boolean;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new Runs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = source["lines"];
	        this.restricted = source["restricted"];
	        this.sudo = source["sudo"];
	    }
	}
	
	export class Table {
	    jobs: Job[];
	    env: Env[];
	    zone: string;
	    unreadable: string[];
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new Table(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobs = this.convertValues(source["jobs"], Job);
	        this.env = this.convertValues(source["env"], Env);
	        this.zone = source["zone"];
	        this.unreadable = source["unreadable"];
	        this.version = source["version"];
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

export namespace docker {
	
	export class Container {
	    id: string;
	    name: string;
	    image: string;
	    state: string;
	    status: string;
	    ports: string;
	    command: string;
	    created: string;
	
	    static createFrom(source: any = {}) {
	        return new Container(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.image = source["image"];
	        this.state = source["state"];
	        this.status = source["status"];
	        this.ports = source["ports"];
	        this.command = source["command"];
	        this.created = source["created"];
	    }
	}
	export class Detail {
	    name: string;
	    image: string;
	    state: string;
	    exitCode: number;
	    error: string;
	    oomKilled: boolean;
	    health: string;
	    startedAt: string;
	    restarts: number;
	    policy: string;
	    ports: string[];
	    mounts: string[];
	    json: string;
	
	    static createFrom(source: any = {}) {
	        return new Detail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.image = source["image"];
	        this.state = source["state"];
	        this.exitCode = source["exitCode"];
	        this.error = source["error"];
	        this.oomKilled = source["oomKilled"];
	        this.health = source["health"];
	        this.startedAt = source["startedAt"];
	        this.restarts = source["restarts"];
	        this.policy = source["policy"];
	        this.ports = source["ports"];
	        this.mounts = source["mounts"];
	        this.json = source["json"];
	    }
	}
	export class LogLine {
	    time: string;
	    text: string;
	    stderr: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LogLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.text = source["text"];
	        this.stderr = source["stderr"];
	    }
	}
	export class Logs {
	    lines: LogLine[];
	
	    static createFrom(source: any = {}) {
	        return new Logs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = this.convertValues(source["lines"], LogLine);
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
	export class Plan {
	    name: string;
	    action: string;
	    command: string;
	    confirm: boolean;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.action = source["action"];
	        this.command = source["command"];
	        this.confirm = source["confirm"];
	        this.sudo = source["sudo"];
	    }
	}
	export class Stats {
	    id: string;
	    cpu: number;
	    memUsed: number;
	    memLimit: number;
	    pids: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cpu = source["cpu"];
	        this.memUsed = source["memUsed"];
	        this.memLimit = source["memLimit"];
	        this.pids = source["pids"];
	    }
	}

}

export namespace fs {
	
	export class Access {
	    writable: boolean;
	    sudoEdit: string;
	
	    static createFrom(source: any = {}) {
	        return new Access(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.writable = source["writable"];
	        this.sudoEdit = source["sudoEdit"];
	    }
	}
	export class CompressPlan {
	    output: string;
	    command: string;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new CompressPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.command = source["command"];
	        this.sudo = source["sudo"];
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
	export class Expect {
	    modTime: number;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Expect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modTime = source["modTime"];
	        this.size = source["size"];
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
	    sudo: string;
	
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
	        this.sudo = source["sudo"];
	    }
	}
	export class SaveOptions {
	    backup: boolean;
	    expect?: Expect;
	
	    static createFrom(source: any = {}) {
	        return new SaveOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backup = source["backup"];
	        this.expect = this.convertValues(source["expect"], Expect);
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
	export class SaveResult {
	    entry: Entry;
	    backup?: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = this.convertValues(source["entry"], Entry);
	        this.backup = source["backup"];
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

export namespace git {
	
	export class Branch {
	    name: string;
	    remote: boolean;
	    current: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Branch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.remote = source["remote"];
	        this.current = source["current"];
	    }
	}
	export class Change {
	    path: string;
	    origPath: string;
	    index: string;
	    worktree: string;
	    conflict: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.origPath = source["origPath"];
	        this.index = source["index"];
	        this.worktree = source["worktree"];
	        this.conflict = source["conflict"];
	    }
	}
	export class Ref {
	    name: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new Ref(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	    }
	}
	export class Commit {
	    hash: string;
	    short: string;
	    author: string;
	    time: number;
	    subject: string;
	    parents: string[];
	    refs: Ref[];
	
	    static createFrom(source: any = {}) {
	        return new Commit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.short = source["short"];
	        this.author = source["author"];
	        this.time = source["time"];
	        this.subject = source["subject"];
	        this.parents = source["parents"];
	        this.refs = this.convertValues(source["refs"], Ref);
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
	export class Diff {
	    text: string;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Diff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.truncated = source["truncated"];
	    }
	}
	export class DiffRequest {
	    dir: string;
	    commit: string;
	    file: string;
	    untracked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiffRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dir = source["dir"];
	        this.commit = source["commit"];
	        this.file = source["file"];
	        this.untracked = source["untracked"];
	    }
	}
	export class Plan {
	    command: string;
	    dirty: number;
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.dirty = source["dirty"];
	    }
	}
	
	export class Repo {
	    path: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Repo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	    }
	}
	export class Result {
	    output: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}
	export class Status {
	    branch: string;
	    upstream: string;
	    ahead: number;
	    behind: number;
	    last?: Commit;
	    changes: Change[];
	    truncated: boolean;
	    install: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branch = source["branch"];
	        this.upstream = source["upstream"];
	        this.ahead = source["ahead"];
	        this.behind = source["behind"];
	        this.last = this.convertValues(source["last"], Commit);
	        this.changes = this.convertValues(source["changes"], Change);
	        this.truncated = source["truncated"];
	        this.install = source["install"];
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

export namespace lock {
	
	export class State {
	    enabled: boolean;
	    locked: boolean;
	    biometric: boolean;
	    retryAt: number;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.locked = source["locked"];
	        this.biometric = source["biometric"];
	        this.retryAt = source["retryAt"];
	    }
	}

}

export namespace logs {
	
	export class Source {
	    kind: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.target = source["target"];
	    }
	}
	export class Started {
	    streamID: string;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new Started(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.streamID = source["streamID"];
	        this.sudo = source["sudo"];
	    }
	}

}

export namespace nginx {
	
	export class Cert {
	    path: string;
	    notAfter: number;
	    names: string[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Cert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.notAfter = source["notAfter"];
	        this.names = source["names"];
	        this.error = source["error"];
	    }
	}
	export class Issue {
	    level: string;
	    message: string;
	    file?: string;
	    line?: number;
	
	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.message = source["message"];
	        this.file = source["file"];
	        this.line = source["line"];
	    }
	}
	export class TestResult {
	    ok: boolean;
	    needsRoot: boolean;
	    issues: Issue[];
	    output: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.needsRoot = source["needsRoot"];
	        this.issues = this.convertValues(source["issues"], Issue);
	        this.output = source["output"];
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
	export class ReloadResult {
	    test: TestResult;
	    reloaded: boolean;
	    needsRoot: boolean;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new ReloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.test = this.convertValues(source["test"], TestResult);
	        this.reloaded = source["reloaded"];
	        this.needsRoot = source["needsRoot"];
	        this.sudo = source["sudo"];
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
	export class RemovePlan {
	    path: string;
	    name: string;
	    enabled: boolean;
	    certs: string[];
	    withCerts: boolean;
	    sharedCerts: string[];
	    command: string;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new RemovePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.certs = source["certs"];
	        this.withCerts = source["withCerts"];
	        this.sharedCerts = source["sharedCerts"];
	        this.command = source["command"];
	        this.sudo = source["sudo"];
	    }
	}
	export class Server {
	    names: string[];
	    listen: string[];
	    root?: string;
	    ssl: boolean;
	    certs: string[];
	    proxyPass: string[];
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.names = source["names"];
	        this.listen = source["listen"];
	        this.root = source["root"];
	        this.ssl = source["ssl"];
	        this.certs = source["certs"];
	        this.proxyPass = source["proxyPass"];
	        this.line = source["line"];
	    }
	}
	export class Site {
	    name: string;
	    path: string;
	    kind: string;
	    enabled: boolean;
	    toggleable: boolean;
	    servers: Server[];
	    upstreams: string[];
	    certs: Cert[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Site(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.enabled = source["enabled"];
	        this.toggleable = source["toggleable"];
	        this.servers = this.convertValues(source["servers"], Server);
	        this.upstreams = source["upstreams"];
	        this.certs = this.convertValues(source["certs"], Cert);
	        this.error = source["error"];
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
	
	export class TogglePlan {
	    name: string;
	    enable: boolean;
	    command: string;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new TogglePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.enable = source["enable"];
	        this.command = source["command"];
	        this.sudo = source["sudo"];
	    }
	}

}

export namespace packages {
	
	export class App {
	    id: string;
	    label: string;
	    kind: string;
	    version: string;
	    path: string;
	    configs: string[];
	    units: string[];
	    versions: string[];
	
	    static createFrom(source: any = {}) {
	        return new App(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.configs = source["configs"];
	        this.units = source["units"];
	        this.versions = source["versions"];
	    }
	}
	export class Update {
	    name: string;
	    current: string;
	    available: string;
	    security: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Update(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.current = source["current"];
	        this.available = source["available"];
	        this.security = source["security"];
	    }
	}
	export class Package {
	    name: string;
	    version: string;
	    summary: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Package(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.summary = source["summary"];
	        this.size = source["size"];
	    }
	}
	export class Inventory {
	    manager: string;
	    installed: Package[];
	    updates: Update[];
	    listsAge: number;
	
	    static createFrom(source: any = {}) {
	        return new Inventory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manager = source["manager"];
	        this.installed = this.convertValues(source["installed"], Package);
	        this.updates = this.convertValues(source["updates"], Update);
	        this.listsAge = source["listsAge"];
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
	export class NpmGlobal {
	    name: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new NpmGlobal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	    }
	}
	

}

export namespace sshkeys {
	
	export class AuthorizedKey {
	    type: string;
	    fingerprint: string;
	    comment: string;
	    options: string[];
	    login: boolean;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthorizedKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.fingerprint = source["fingerprint"];
	        this.comment = source["comment"];
	        this.options = source["options"];
	        this.login = source["login"];
	        this.text = source["text"];
	    }
	}
	export class Authorized {
	    path: string;
	    keys: AuthorizedKey[];
	
	    static createFrom(source: any = {}) {
	        return new Authorized(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.keys = this.convertValues(source["keys"], AuthorizedKey);
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
	
	export class CreatePlan {
	    name: string;
	    path: string;
	    comment: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new CreatePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.comment = source["comment"];
	        this.command = source["command"];
	    }
	}
	export class GitTestPlan {
	    host: string;
	    key: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new GitTestPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.key = source["key"];
	        this.command = source["command"];
	    }
	}
	export class GitTestResult {
	    outcome: string;
	    account?: string;
	    output: string;
	
	    static createFrom(source: any = {}) {
	        return new GitTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outcome = source["outcome"];
	        this.account = source["account"];
	        this.output = source["output"];
	    }
	}
	export class KeyPlan {
	    fingerprint: string;
	    comment: string;
	    login: boolean;
	    backup?: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fingerprint = source["fingerprint"];
	        this.comment = source["comment"];
	        this.login = source["login"];
	        this.backup = source["backup"];
	        this.command = source["command"];
	    }
	}
	export class LocalKey {
	    name: string;
	    path: string;
	    type: string;
	    fingerprint: string;
	    comment: string;
	    public: string;
	    encrypted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LocalKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.fingerprint = source["fingerprint"];
	        this.comment = source["comment"];
	        this.public = source["public"];
	        this.encrypted = source["encrypted"];
	    }
	}
	export class ServerKey {
	    name: string;
	    path: string;
	    type: string;
	    fingerprint: string;
	    comment: string;
	    public: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.fingerprint = source["fingerprint"];
	        this.comment = source["comment"];
	        this.public = source["public"];
	    }
	}

}

export namespace svc {
	
	export class Journal {
	    lines: string[];
	    restricted: boolean;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new Journal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = source["lines"];
	        this.restricted = source["restricted"];
	        this.sudo = source["sudo"];
	    }
	}
	export class Plan {
	    unit: string;
	    action: string;
	    command: string;
	    confirm: boolean;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unit = source["unit"];
	        this.action = source["action"];
	        this.command = source["command"];
	        this.confirm = source["confirm"];
	        this.sudo = source["sudo"];
	    }
	}
	export class RemovePlan {
	    unit: string;
	    mode: string;
	    file?: string;
	    triggers: string[];
	    command: string;
	    sudo: string;
	
	    static createFrom(source: any = {}) {
	        return new RemovePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unit = source["unit"];
	        this.mode = source["mode"];
	        this.file = source["file"];
	        this.triggers = source["triggers"];
	        this.command = source["command"];
	        this.sudo = source["sudo"];
	    }
	}
	export class Status {
	    unit: string;
	    description: string;
	    load: string;
	    active: string;
	    sub: string;
	    enabled: string;
	    activeFor: number;
	    mainPID: number;
	    memory: number;
	    restarts: number;
	    result: string;
	    file: string;
	    triggeredBy: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unit = source["unit"];
	        this.description = source["description"];
	        this.load = source["load"];
	        this.active = source["active"];
	        this.sub = source["sub"];
	        this.enabled = source["enabled"];
	        this.activeFor = source["activeFor"];
	        this.mainPID = source["mainPID"];
	        this.memory = source["memory"];
	        this.restarts = source["restarts"];
	        this.result = source["result"];
	        this.file = source["file"];
	        this.triggeredBy = source["triggeredBy"];
	    }
	}
	export class Unit {
	    name: string;
	    load: string;
	    active: string;
	    sub: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Unit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.load = source["load"];
	        this.active = source["active"];
	        this.sub = source["sub"];
	        this.description = source["description"];
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
	    fromServerID?: string;
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
	        this.fromServerID = source["fromServerID"];
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

