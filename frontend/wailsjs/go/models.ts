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

export namespace inventory {
	
	export class InventoryRecord {
	    ownedItemID: string;
	    kind: string;
	    key: string;
	    gameID: number;
	    containerSection: string;
	    physicalIndex: number;
	    gaItemHandle: number;
	    quantity: number;
	    acquisitionIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new InventoryRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ownedItemID = source["ownedItemID"];
	        this.kind = source["kind"];
	        this.key = source["key"];
	        this.gameID = source["gameID"];
	        this.containerSection = source["containerSection"];
	        this.physicalIndex = source["physicalIndex"];
	        this.gaItemHandle = source["gaItemHandle"];
	        this.quantity = source["quantity"];
	        this.acquisitionIndex = source["acquisitionIndex"];
	    }
	}
	export class GetInventoryResult {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    records: InventoryRecord[];
	    total: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new GetInventoryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.records = this.convertValues(source["records"], InventoryRecord);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
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
	export class StorageRecord {
	    ownedItemID: string;
	    kind: string;
	    key: string;
	    gameID: number;
	    containerSection: string;
	    physicalIndex: number;
	    gaItemHandle: number;
	    quantity: number;
	    acquisitionIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new StorageRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ownedItemID = source["ownedItemID"];
	        this.kind = source["kind"];
	        this.key = source["key"];
	        this.gameID = source["gameID"];
	        this.containerSection = source["containerSection"];
	        this.physicalIndex = source["physicalIndex"];
	        this.gaItemHandle = source["gaItemHandle"];
	        this.quantity = source["quantity"];
	        this.acquisitionIndex = source["acquisitionIndex"];
	    }
	}
	export class GetStorageResult {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    records: StorageRecord[];
	    total: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new GetStorageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.records = this.convertValues(source["records"], StorageRecord);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
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

