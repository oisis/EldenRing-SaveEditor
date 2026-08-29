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

export namespace saveengine {
	
	export class CharacterProfile {
	    saveSessionID: string;
	    characterID: number;
	    active: boolean;
	    name: string;
	    level: number;
	    startingClassID: number;
	    gender: number;
	    secondsPlayed: number;
	
	    static createFrom(source: any = {}) {
	        return new CharacterProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.name = source["name"];
	        this.level = source["level"];
	        this.startingClassID = source["startingClassID"];
	        this.gender = source["gender"];
	        this.secondsPlayed = source["secondsPlayed"];
	    }
	}
	export class CharacterStats {
	    saveSessionID: string;
	    characterID: number;
	    active: boolean;
	    vigor: number;
	    mind: number;
	    endurance: number;
	    strength: number;
	    dexterity: number;
	    intelligence: number;
	    faith: number;
	    arcane: number;
	    level: number;
	    hp: number;
	    maxHP: number;
	    baseMaxHP: number;
	    fp: number;
	    maxFP: number;
	    baseMaxFP: number;
	    sp: number;
	    maxSP: number;
	    baseMaxSP: number;
	
	    static createFrom(source: any = {}) {
	        return new CharacterStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.vigor = source["vigor"];
	        this.mind = source["mind"];
	        this.endurance = source["endurance"];
	        this.strength = source["strength"];
	        this.dexterity = source["dexterity"];
	        this.intelligence = source["intelligence"];
	        this.faith = source["faith"];
	        this.arcane = source["arcane"];
	        this.level = source["level"];
	        this.hp = source["hp"];
	        this.maxHP = source["maxHP"];
	        this.baseMaxHP = source["baseMaxHP"];
	        this.fp = source["fp"];
	        this.maxFP = source["maxFP"];
	        this.baseMaxFP = source["baseMaxFP"];
	        this.sp = source["sp"];
	        this.maxSP = source["maxSP"];
	        this.baseMaxSP = source["baseMaxSP"];
	    }
	}
	export class CharacterSummary {
	    characterID: number;
	    active: boolean;
	    name: string;
	    level: number;
	
	    static createFrom(source: any = {}) {
	        return new CharacterSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.name = source["name"];
	        this.level = source["level"];
	    }
	}
	export class SaveCharacters {
	    saveSessionID: string;
	    characters: CharacterSummary[];
	
	    static createFrom(source: any = {}) {
	        return new SaveCharacters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.characters = this.convertValues(source["characters"], CharacterSummary);
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
	export class SessionInfo {
	    saveSessionID: string;
	    platform: string;
	    format: string;
	    unsavedChanges: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.platform = source["platform"];
	        this.format = source["format"];
	        this.unsavedChanges = source["unsavedChanges"];
	    }
	}

}

