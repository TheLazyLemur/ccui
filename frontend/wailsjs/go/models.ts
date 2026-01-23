export namespace main {
	
	export class ReviewComment {
	    id: string;
	    type: string;
	    filePath?: string;
	    lineNumber?: number;
	    hunkIndex?: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new ReviewComment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.filePath = source["filePath"];
	        this.lineNumber = source["lineNumber"];
	        this.hunkIndex = source["hunkIndex"];
	        this.text = source["text"];
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

