export namespace data {
	
	export class ArmorStats {
	    Weight: number;
	    Physical: number;
	    Strike: number;
	    Slash: number;
	    Pierce: number;
	    Magic: number;
	    Fire: number;
	    Lightning: number;
	    Holy: number;
	    Immunity: number;
	    Robustness: number;
	    Focus: number;
	    Vitality: number;
	    Poise: number;
	
	    static createFrom(source: any = {}) {
	        return new ArmorStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Weight = source["Weight"];
	        this.Physical = source["Physical"];
	        this.Strike = source["Strike"];
	        this.Slash = source["Slash"];
	        this.Pierce = source["Pierce"];
	        this.Magic = source["Magic"];
	        this.Fire = source["Fire"];
	        this.Lightning = source["Lightning"];
	        this.Holy = source["Holy"];
	        this.Immunity = source["Immunity"];
	        this.Robustness = source["Robustness"];
	        this.Focus = source["Focus"];
	        this.Vitality = source["Vitality"];
	        this.Poise = source["Poise"];
	    }
	}
	export class SpellStats {
	    FPCost: number;
	    Slots: number;
	    ReqInt: number;
	    ReqFai: number;
	    ReqArc: number;
	
	    static createFrom(source: any = {}) {
	        return new SpellStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FPCost = source["FPCost"];
	        this.Slots = source["Slots"];
	        this.ReqInt = source["ReqInt"];
	        this.ReqFai = source["ReqFai"];
	        this.ReqArc = source["ReqArc"];
	    }
	}
	export class WeaponStats {
	    Weight: number;
	    PhysDamage: number;
	    MagDamage: number;
	    FireDamage: number;
	    LitDamage: number;
	    HolyDamage: number;
	    ScaleStr: number;
	    ScaleDex: number;
	    ScaleInt: number;
	    ScaleFai: number;
	    ReqStr: number;
	    ReqDex: number;
	    ReqInt: number;
	    ReqFai: number;
	    ReqArc: number;
	
	    static createFrom(source: any = {}) {
	        return new WeaponStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Weight = source["Weight"];
	        this.PhysDamage = source["PhysDamage"];
	        this.MagDamage = source["MagDamage"];
	        this.FireDamage = source["FireDamage"];
	        this.LitDamage = source["LitDamage"];
	        this.HolyDamage = source["HolyDamage"];
	        this.ScaleStr = source["ScaleStr"];
	        this.ScaleDex = source["ScaleDex"];
	        this.ScaleInt = source["ScaleInt"];
	        this.ScaleFai = source["ScaleFai"];
	        this.ReqStr = source["ReqStr"];
	        this.ReqDex = source["ReqDex"];
	        this.ReqInt = source["ReqInt"];
	        this.ReqFai = source["ReqFai"];
	        this.ReqArc = source["ReqArc"];
	    }
	}

}

export namespace db {
	
	export class BossEntry {
	    id: number;
	    name: string;
	    region: string;
	    type: string;
	    remembrance: boolean;
	    defeated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BossEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.region = source["region"];
	        this.type = source["type"];
	        this.remembrance = source["remembrance"];
	        this.defeated = source["defeated"];
	    }
	}
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
	    description?: string;
	    weight?: number;
	    weapon?: data.WeaponStats;
	    armor?: data.ArmorStats;
	    spell?: data.SpellStats;
	
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
	        this.description = source["description"];
	        this.weight = source["weight"];
	        this.weapon = this.convertValues(source["weapon"], data.WeaponStats);
	        this.armor = this.convertValues(source["armor"], data.ArmorStats);
	        this.spell = this.convertValues(source["spell"], data.SpellStats);
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
	
	export class StatValidationResult {
	    valid: boolean;
	    errors: string[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new StatValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.errors = source["errors"];
	        this.warnings = source["warnings"];
	    }
	}
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
	    class: number;
	    className: string;
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
	    statValidation?: StatValidationResult;
	    eventFlagsAvailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CharacterViewModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.level = source["level"];
	        this.souls = source["souls"];
	        this.class = source["class"];
	        this.className = source["className"];
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
	        this.statValidation = this.convertValues(source["statValidation"], StatValidationResult);
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

