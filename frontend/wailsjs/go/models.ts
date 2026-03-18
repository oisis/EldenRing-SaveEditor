export namespace backend {
	
	export class CharacterDetails {
	    slotIndex: number;
	    name: string;
	    level: number;
	    vigor: number;
	    mind: number;
	    endurance: number;
	    strength: number;
	    dexterity: number;
	    intelligence: number;
	    faith: number;
	    arcane: number;
	    souls: number;
	
	    static createFrom(source: any = {}) {
	        return new CharacterDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slotIndex = source["slotIndex"];
	        this.name = source["name"];
	        this.level = source["level"];
	        this.vigor = source["vigor"];
	        this.mind = source["mind"];
	        this.endurance = source["endurance"];
	        this.strength = source["strength"];
	        this.dexterity = source["dexterity"];
	        this.intelligence = source["intelligence"];
	        this.faith = source["faith"];
	        this.arcane = source["arcane"];
	        this.souls = source["souls"];
	    }
	}
	export class CharacterInfo {
	    slotIndex: number;
	    name: string;
	    level: number;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CharacterInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slotIndex = source["slotIndex"];
	        this.name = source["name"];
	        this.level = source["level"];
	        this.isActive = source["isActive"];
	    }
	}
	export class EventItem {
	    id: number;
	    name: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EventItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	    }
	}

}

