export namespace vm {
	
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
	    }
	}

}

