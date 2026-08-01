export namespace main {
	
	export class RequestParams {
	    Method: string;
	    URL: string;
	    Body: string;
	    Headers: Record<string, string>;
	    TimeoutMs: number;
	    SessionName: string;
	    SessionNew: boolean;
	    Variables: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RequestParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Method = source["Method"];
	        this.URL = source["URL"];
	        this.Body = source["Body"];
	        this.Headers = source["Headers"];
	        this.TimeoutMs = source["TimeoutMs"];
	        this.SessionName = source["SessionName"];
	        this.SessionNew = source["SessionNew"];
	        this.Variables = source["Variables"];
	    }
	}

}

export namespace response {
	
	export class ResponseData {
	    statusCode: number;
	    status: string;
	    headers: Record<string, string>;
	    body: string;
	    elapsedMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ResponseData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.statusCode = source["statusCode"];
	        this.status = source["status"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	        this.elapsedMs = source["elapsedMs"];
	    }
	}

}

