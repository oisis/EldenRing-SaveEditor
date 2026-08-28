export namespace application {
	
	export class SupportedSchema {
	    name: string;
	    minimumVersion: number;
	    currentVersion: number;
	
	    static createFrom(source: any = {}) {
	        return new SupportedSchema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.minimumVersion = source["minimumVersion"];
	        this.currentVersion = source["currentVersion"];
	    }
	}
	export class GetApplicationInfoResult {
	    applicationVersion: string;
	    supportedSchemas: SupportedSchema[];
	    capabilities: string[];
	
	    static createFrom(source: any = {}) {
	        return new GetApplicationInfoResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applicationVersion = source["applicationVersion"];
	        this.supportedSchemas = this.convertValues(source["supportedSchemas"], SupportedSchema);
	        this.capabilities = source["capabilities"];
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

