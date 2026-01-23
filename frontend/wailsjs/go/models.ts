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

}

