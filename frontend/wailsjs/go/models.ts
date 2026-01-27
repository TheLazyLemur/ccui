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
	    ID: string;
	    Name: string;
	    CreatedAt: string;
	    ModeID: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.CreatedAt = source["CreatedAt"];
	        this.ModeID = source["ModeID"];
	    }
	}

}

