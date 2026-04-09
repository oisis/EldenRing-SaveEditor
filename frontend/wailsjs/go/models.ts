export namespace db {
	
	export class GraceEntry {
	    id: number;
	    name: string;
	    region: string;
	    visited: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GraceEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.region = source["region"];
	        this.visited = source["visited"];
	    }
	}
	export class InfuseType {
	    name: string;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new InfuseType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.offset = source["offset"];
	    }
	}
	export class ItemEntry {
	    id: number;
	    name: string;
	    category: string;
	    maxInventory: number;
	    maxStorage: number;
	    maxUpgrade: number;
	    iconPath: string;
	    flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new ItemEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.maxInventory = source["maxInventory"];
	        this.maxStorage = source["maxStorage"];
	        this.maxUpgrade = source["maxUpgrade"];
	        this.iconPath = source["iconPath"];
	        this.flags = source["flags"];
	    }
	}

}

export namespace main {
	
	export class DiffEntry {
	    category: string;
	    action: string;
	    field: string;
	    oldValue: string;
	    newValue: string;
	
	    static createFrom(source: any = {}) {
	        return new DiffEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.action = source["action"];
	        this.field = source["field"];
	        this.oldValue = source["oldValue"];
	        this.newValue = source["newValue"];
	    }
	}
	export class SlotDiffSummary {
	    slotIndex: number;
	    charName: string;
	    changeCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SlotDiffSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slotIndex = source["slotIndex"];
	        this.charName = source["charName"];
	        this.changeCount = source["changeCount"];
	    }
	}

}

export namespace vm {
	
	export class ItemViewModel {
	    handle: number;
	    id: number;
	    name: string;
	    category: string;
	    subCategory: string;
	    quantity: number;
	    maxInventory: number;
	    maxStorage: number;
	    maxUpgrade: number;
	    currentUpgrade: number;
	    iconPath: string;
	    flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new ItemViewModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.handle = source["handle"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.subCategory = source["subCategory"];
	        this.quantity = source["quantity"];
	        this.maxInventory = source["maxInventory"];
	        this.maxStorage = source["maxStorage"];
	        this.maxUpgrade = source["maxUpgrade"];
	        this.currentUpgrade = source["currentUpgrade"];
	        this.iconPath = source["iconPath"];
	        this.flags = source["flags"];
	    }
	}
	export class CharacterViewModel {
	    name: string;
	    level: number;
	    souls: number;
	    vigor: number;
	    mind: number;
	    endurance: number;
	    strength: number;
	    dexterity: number;
	    intelligence: number;
	    faith: number;
	    arcane: number;
	    scadutreeBlessing: number;
	    shadowRealmBlessing: number;
	    inventory: ItemViewModel[];
	    storage: ItemViewModel[];
	    warnings: string[];
	    eventFlagsAvailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CharacterViewModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.level = source["level"];
	        this.souls = source["souls"];
	        this.vigor = source["vigor"];
	        this.mind = source["mind"];
	        this.endurance = source["endurance"];
	        this.strength = source["strength"];
	        this.dexterity = source["dexterity"];
	        this.intelligence = source["intelligence"];
	        this.faith = source["faith"];
	        this.arcane = source["arcane"];
	        this.scadutreeBlessing = source["scadutreeBlessing"];
	        this.shadowRealmBlessing = source["shadowRealmBlessing"];
	        this.inventory = this.convertValues(source["inventory"], ItemViewModel);
	        this.storage = this.convertValues(source["storage"], ItemViewModel);
	        this.warnings = source["warnings"];
	        this.eventFlagsAvailable = source["eventFlagsAvailable"];
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

