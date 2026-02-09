export namespace automation {
	
	export class Automation {
	    id: string;
	    name: string;
	    prompt: string;
	    schedule: string;
	    projectDir: string;
	    backendType: string;
	    permissionLevel: string;
	    enabled: boolean;
	    useWorktree: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Automation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.prompt = source["prompt"];
	        this.schedule = source["schedule"];
	        this.projectDir = source["projectDir"];
	        this.backendType = source["backendType"];
	        this.permissionLevel = source["permissionLevel"];
	        this.enabled = source["enabled"];
	        this.useWorktree = source["useWorktree"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Run {
	    id: string;
	    automationId: string;
	    status: string;
	    startedAt: string;
	    completedAt?: string;
	    output?: string;
	    error?: string;
	    hasFindings: boolean;
	    read: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Run(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.automationId = source["automationId"];
	        this.status = source["status"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	        this.output = source["output"];
	        this.error = source["error"];
	        this.hasFindings = source["hasFindings"];
	        this.read = source["read"];
	    }
	}

}

export namespace backend {
	
	export class SessionMode {
	    id: string;
	    name: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionMode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}

}

export namespace main {
	
	export class ReviewComment {
	    ID: string;
	    Type: string;
	    FilePath: string;
	    Text: string;
	    LineNumber: number;
	    HunkIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new ReviewComment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Type = source["Type"];
	        this.FilePath = source["FilePath"];
	        this.Text = source["Text"];
	        this.LineNumber = source["LineNumber"];
	        this.HunkIndex = source["HunkIndex"];
	    }
	}
	export class SessionInfo {
	    id: string;
	    name: string;
	    createdAt: string;
	    modeId: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.createdAt = source["createdAt"];
	        this.modeId = source["modeId"];
	    }
	}

}

