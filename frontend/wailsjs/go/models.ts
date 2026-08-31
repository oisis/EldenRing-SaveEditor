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

export namespace catalog {
	
	export class GetItemVariantsResult {
	    variants: schema.ItemVariant[];
	
	    static createFrom(source: any = {}) {
	        return new GetItemVariantsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.variants = this.convertValues(source["variants"], schema.ItemVariant);
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
	export class ResourcePresentationSummary {
	    kind: string;
	    key: string;
	    name: string;
	    iconPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ResourcePresentationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.key = source["key"];
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	    }
	}
	export class GetResourcePresentationSummariesResult {
	    resources: ResourcePresentationSummary[];
	
	    static createFrom(source: any = {}) {
	        return new GetResourcePresentationSummariesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resources = this.convertValues(source["resources"], ResourcePresentationSummary);
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
	export class GetResourceResult {
	    resource: schema.Resource;
	
	    static createFrom(source: any = {}) {
	        return new GetResourceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resource = this.convertValues(source["resource"], schema.Resource);
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
	export class GetResourcesEntry {
	    kind: string;
	    key: string;
	    family: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new GetResourcesEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.key = source["key"];
	        this.family = source["family"];
	        this.name = source["name"];
	    }
	}
	export class GetResourcesResult {
	    resources: GetResourcesEntry[];
	    total: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new GetResourcesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resources = this.convertValues(source["resources"], GetResourcesEntry);
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
	export class ResourcePresentationIdentity {
	    kind: string;
	    key: string;
	
	    static createFrom(source: any = {}) {
	        return new ResourcePresentationIdentity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.key = source["key"];
	    }
	}

}

export namespace diagnostics {
	
	export class SaveValidationIssue {
	    id: string;
	    code: string;
	    severity: string;
	    scope: string;
	    message: string;
	    ownedItemID: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveValidationIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.code = source["code"];
	        this.severity = source["severity"];
	        this.scope = source["scope"];
	        this.message = source["message"];
	        this.ownedItemID = source["ownedItemID"];
	    }
	}
	export class SaveValidationScopeCoverage {
	    scope: string;
	    checked: boolean;
	    reason: string;
	    recordsChecked: number;
	    unresolvedRecords: number;
	
	    static createFrom(source: any = {}) {
	        return new SaveValidationScopeCoverage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scope = source["scope"];
	        this.checked = source["checked"];
	        this.reason = source["reason"];
	        this.recordsChecked = source["recordsChecked"];
	        this.unresolvedRecords = source["unresolvedRecords"];
	    }
	}
	export class GetSaveValidationReportResult {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    coverage: SaveValidationScopeCoverage[];
	    issues: SaveValidationIssue[];
	    errorCount: number;
	    warningCount: number;
	
	    static createFrom(source: any = {}) {
	        return new GetSaveValidationReportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.coverage = this.convertValues(source["coverage"], SaveValidationScopeCoverage);
	        this.issues = this.convertValues(source["issues"], SaveValidationIssue);
	        this.errorCount = source["errorCount"];
	        this.warningCount = source["warningCount"];
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

export namespace equipment {
	
	export class EquippedSpellSlot {
	    rawMagicParamID: number;
	    resourceKey: string;
	    name: string;
	    memorySlots: number;
	
	    static createFrom(source: any = {}) {
	        return new EquippedSpellSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rawMagicParamID = source["rawMagicParamID"];
	        this.resourceKey = source["resourceKey"];
	        this.name = source["name"];
	        this.memorySlots = source["memorySlots"];
	    }
	}
	export class LoadoutSpellSlot {
	    state: string;
	    resource?: schema.ResourceRef;
	    name?: string;
	    iconPath?: string;
	    memorySlots?: number;
	
	    static createFrom(source: any = {}) {
	        return new LoadoutSpellSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.resource = this.convertValues(source["resource"], schema.ResourceRef);
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	        this.memorySlots = source["memorySlots"];
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
	export class LoadoutOwnedSlot {
	    slotType: string;
	    state: string;
	    ownedItemID?: string;
	    resource?: schema.ResourceRef;
	    name?: string;
	    iconPath?: string;
	    quantity?: number;
	
	    static createFrom(source: any = {}) {
	        return new LoadoutOwnedSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slotType = source["slotType"];
	        this.state = source["state"];
	        this.ownedItemID = source["ownedItemID"];
	        this.resource = this.convertValues(source["resource"], schema.ResourceRef);
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	        this.quantity = source["quantity"];
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
	export class LoadoutSlot {
	    slotType: string;
	    state: string;
	    resource?: schema.ResourceRef;
	    name?: string;
	    iconPath?: string;
	    rawValue: number;
	
	    static createFrom(source: any = {}) {
	        return new LoadoutSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slotType = source["slotType"];
	        this.state = source["state"];
	        this.resource = this.convertValues(source["resource"], schema.ResourceRef);
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	        this.rawValue = source["rawValue"];
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
	export class GetCharacterLoadoutResult {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    rightHand: LoadoutSlot[];
	    leftHand: LoadoutSlot[];
	    arrows: LoadoutSlot[];
	    bolts: LoadoutSlot[];
	    armor: LoadoutSlot[];
	    talismans: LoadoutSlot[];
	    quickItems: LoadoutOwnedSlot[];
	    pouch: LoadoutOwnedSlot[];
	    activeQuickItem: number;
	    physick: LoadoutSlot[];
	    spells: LoadoutSpellSlot[];
	    activeSpellIndex: number;
	    usedMemorySlots: number;
	    availableMemorySlots: number;
	    unlockedTalismanSlots: number;
	
	    static createFrom(source: any = {}) {
	        return new GetCharacterLoadoutResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.rightHand = this.convertValues(source["rightHand"], LoadoutSlot);
	        this.leftHand = this.convertValues(source["leftHand"], LoadoutSlot);
	        this.arrows = this.convertValues(source["arrows"], LoadoutSlot);
	        this.bolts = this.convertValues(source["bolts"], LoadoutSlot);
	        this.armor = this.convertValues(source["armor"], LoadoutSlot);
	        this.talismans = this.convertValues(source["talismans"], LoadoutSlot);
	        this.quickItems = this.convertValues(source["quickItems"], LoadoutOwnedSlot);
	        this.pouch = this.convertValues(source["pouch"], LoadoutOwnedSlot);
	        this.activeQuickItem = source["activeQuickItem"];
	        this.physick = this.convertValues(source["physick"], LoadoutSlot);
	        this.spells = this.convertValues(source["spells"], LoadoutSpellSlot);
	        this.activeSpellIndex = source["activeSpellIndex"];
	        this.usedMemorySlots = source["usedMemorySlots"];
	        this.availableMemorySlots = source["availableMemorySlots"];
	        this.unlockedTalismanSlots = source["unlockedTalismanSlots"];
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
	export class GetEquippedSpellsResult {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    spells: EquippedSpellSlot[];
	    usedMemorySlots: number;
	    availableMemorySlots: number;
	
	    static createFrom(source: any = {}) {
	        return new GetEquippedSpellsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.spells = this.convertValues(source["spells"], EquippedSpellSlot);
	        this.usedMemorySlots = source["usedMemorySlots"];
	        this.availableMemorySlots = source["availableMemorySlots"];
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
	
	export class CharacterEquipment {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    slots: number[];
	
	    static createFrom(source: any = {}) {
	        return new CharacterEquipment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.slots = source["slots"];
	    }
	}
	export class CharacterPhysickMixture {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    tears: number[];
	
	    static createFrom(source: any = {}) {
	        return new CharacterPhysickMixture(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.tears = source["tears"];
	    }
	}
	export class PouchItemSlot {
	    itemID: number;
	    equipIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new PouchItemSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemID = source["itemID"];
	        this.equipIndex = source["equipIndex"];
	    }
	}
	export class CharacterPouchItems {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    items: PouchItemSlot[];
	
	    static createFrom(source: any = {}) {
	        return new CharacterPouchItems(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.items = this.convertValues(source["items"], PouchItemSlot);
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
	export class CharacterProfile {
	    saveSessionID: string;
	    saveRevision: string;
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
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.name = source["name"];
	        this.level = source["level"];
	        this.startingClassID = source["startingClassID"];
	        this.gender = source["gender"];
	        this.secondsPlayed = source["secondsPlayed"];
	    }
	}
	export class QuickItemSlot {
	    itemID: number;
	    equipIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new QuickItemSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemID = source["itemID"];
	        this.equipIndex = source["equipIndex"];
	    }
	}
	export class CharacterQuickItems {
	    saveSessionID: string;
	    saveRevision: string;
	    characterID: number;
	    active: boolean;
	    items: QuickItemSlot[];
	    activeQuick: number;
	
	    static createFrom(source: any = {}) {
	        return new CharacterQuickItems(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
	        this.characterID = source["characterID"];
	        this.active = source["active"];
	        this.items = this.convertValues(source["items"], QuickItemSlot);
	        this.activeQuick = source["activeQuick"];
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
	export class CharacterStats {
	    saveSessionID: string;
	    saveRevision: string;
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
	        this.saveRevision = source["saveRevision"];
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
	    saveRevision: string;
	    characters: CharacterSummary[];
	
	    static createFrom(source: any = {}) {
	        return new SaveCharacters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.saveRevision = source["saveRevision"];
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
	    sourcePath: string;
	    sourceKind: string;
	    saveRevision: string;
	    unsavedChanges: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveSessionID = source["saveSessionID"];
	        this.platform = source["platform"];
	        this.format = source["format"];
	        this.sourcePath = source["sourcePath"];
	        this.sourceKind = source["sourceKind"];
	        this.saveRevision = source["saveRevision"];
	        this.unsavedChanges = source["unsavedChanges"];
	    }
	}

}

export namespace schema {
	
	export class Fact_bool_ {
	    known: boolean;
	    value: boolean;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_bool_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_float64_ {
	    known: boolean;
	    value: number;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_float64_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_uint8_ {
	    known: boolean;
	    value: number;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_uint8_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Provenance {
	    source: string;
	    method: string;
	    table?: string;
	    row?: string;
	    field?: string;
	
	    static createFrom(source: any = {}) {
	        return new Provenance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.method = source["method"];
	        this.table = source["table"];
	        this.row = source["row"];
	        this.field = source["field"];
	    }
	}
	export class Fact_uint32_ {
	    known: boolean;
	    value: number;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_uint32_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class ArmorData {
	    sourceRowID: Fact_uint32_;
	    iconIDMale: Fact_uint32_;
	    iconIDFemale: Fact_uint32_;
	    sortID: Fact_uint32_;
	    "sortID-sfv"?: Fact_uint32_;
	    sortGroupID: Fact_uint8_;
	    "sortGroupID-sfv"?: Fact_uint8_;
	    weight: Fact_float64_;
	    "weight-sfv"?: Fact_float64_;
	    physical: Fact_float64_;
	    "physical-sfv"?: Fact_float64_;
	    strike: Fact_float64_;
	    "strike-sfv"?: Fact_float64_;
	    slash: Fact_float64_;
	    "slash-sfv"?: Fact_float64_;
	    pierce: Fact_float64_;
	    "pierce-sfv"?: Fact_float64_;
	    magic: Fact_float64_;
	    "magic-sfv"?: Fact_float64_;
	    fire: Fact_float64_;
	    "fire-sfv"?: Fact_float64_;
	    lightning: Fact_float64_;
	    "lightning-sfv"?: Fact_float64_;
	    holy: Fact_float64_;
	    "holy-sfv"?: Fact_float64_;
	    immunity: Fact_uint32_;
	    "immunity-sfv"?: Fact_uint32_;
	    robustness: Fact_uint32_;
	    "robustness-sfv"?: Fact_uint32_;
	    focus: Fact_uint32_;
	    "focus-sfv"?: Fact_uint32_;
	    vitality: Fact_uint32_;
	    "vitality-sfv"?: Fact_uint32_;
	    poise: Fact_float64_;
	    "poise-sfv"?: Fact_float64_;
	    headEquipable: Fact_bool_;
	    bodyEquipable: Fact_bool_;
	    armEquipable: Fact_bool_;
	    legEquipable: Fact_bool_;
	
	    static createFrom(source: any = {}) {
	        return new ArmorData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.iconIDMale = this.convertValues(source["iconIDMale"], Fact_uint32_);
	        this.iconIDFemale = this.convertValues(source["iconIDFemale"], Fact_uint32_);
	        this.sortID = this.convertValues(source["sortID"], Fact_uint32_);
	        this["sortID-sfv"] = this.convertValues(source["sortID-sfv"], Fact_uint32_);
	        this.sortGroupID = this.convertValues(source["sortGroupID"], Fact_uint8_);
	        this["sortGroupID-sfv"] = this.convertValues(source["sortGroupID-sfv"], Fact_uint8_);
	        this.weight = this.convertValues(source["weight"], Fact_float64_);
	        this["weight-sfv"] = this.convertValues(source["weight-sfv"], Fact_float64_);
	        this.physical = this.convertValues(source["physical"], Fact_float64_);
	        this["physical-sfv"] = this.convertValues(source["physical-sfv"], Fact_float64_);
	        this.strike = this.convertValues(source["strike"], Fact_float64_);
	        this["strike-sfv"] = this.convertValues(source["strike-sfv"], Fact_float64_);
	        this.slash = this.convertValues(source["slash"], Fact_float64_);
	        this["slash-sfv"] = this.convertValues(source["slash-sfv"], Fact_float64_);
	        this.pierce = this.convertValues(source["pierce"], Fact_float64_);
	        this["pierce-sfv"] = this.convertValues(source["pierce-sfv"], Fact_float64_);
	        this.magic = this.convertValues(source["magic"], Fact_float64_);
	        this["magic-sfv"] = this.convertValues(source["magic-sfv"], Fact_float64_);
	        this.fire = this.convertValues(source["fire"], Fact_float64_);
	        this["fire-sfv"] = this.convertValues(source["fire-sfv"], Fact_float64_);
	        this.lightning = this.convertValues(source["lightning"], Fact_float64_);
	        this["lightning-sfv"] = this.convertValues(source["lightning-sfv"], Fact_float64_);
	        this.holy = this.convertValues(source["holy"], Fact_float64_);
	        this["holy-sfv"] = this.convertValues(source["holy-sfv"], Fact_float64_);
	        this.immunity = this.convertValues(source["immunity"], Fact_uint32_);
	        this["immunity-sfv"] = this.convertValues(source["immunity-sfv"], Fact_uint32_);
	        this.robustness = this.convertValues(source["robustness"], Fact_uint32_);
	        this["robustness-sfv"] = this.convertValues(source["robustness-sfv"], Fact_uint32_);
	        this.focus = this.convertValues(source["focus"], Fact_uint32_);
	        this["focus-sfv"] = this.convertValues(source["focus-sfv"], Fact_uint32_);
	        this.vitality = this.convertValues(source["vitality"], Fact_uint32_);
	        this["vitality-sfv"] = this.convertValues(source["vitality-sfv"], Fact_uint32_);
	        this.poise = this.convertValues(source["poise"], Fact_float64_);
	        this["poise-sfv"] = this.convertValues(source["poise-sfv"], Fact_float64_);
	        this.headEquipable = this.convertValues(source["headEquipable"], Fact_bool_);
	        this.bodyEquipable = this.convertValues(source["bodyEquipable"], Fact_bool_);
	        this.armEquipable = this.convertValues(source["armEquipable"], Fact_bool_);
	        this.legEquipable = this.convertValues(source["legEquipable"], Fact_bool_);
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
	export class Fact___string_ {
	    known: boolean;
	    value: string[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact___string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_uint64_ {
	    known: boolean;
	    value: number;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_uint64_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_string_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_int32_ {
	    known: boolean;
	    value: number;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_int32_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class AshOfWarData {
	    sourceRowID: Fact_uint32_;
	    iconID: Fact_uint32_;
	    sortID: Fact_uint32_;
	    "sortID-sfv"?: Fact_uint32_;
	    sortGroupID: Fact_uint8_;
	    "sortGroupID-sfv"?: Fact_uint8_;
	    swordArtsParamID: Fact_int32_;
	    swordArtsName: Fact_string_;
	    "swordArtsName-sfv"?: Fact_string_;
	    defaultAffinity: Fact_uint8_;
	    compatibilityMask: Fact_uint64_;
	    "compatibilityMask-sfv"?: Fact_uint64_;
	    compatibleClassNames: Fact___string_;
	
	    static createFrom(source: any = {}) {
	        return new AshOfWarData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.iconID = this.convertValues(source["iconID"], Fact_uint32_);
	        this.sortID = this.convertValues(source["sortID"], Fact_uint32_);
	        this["sortID-sfv"] = this.convertValues(source["sortID-sfv"], Fact_uint32_);
	        this.sortGroupID = this.convertValues(source["sortGroupID"], Fact_uint8_);
	        this["sortGroupID-sfv"] = this.convertValues(source["sortGroupID-sfv"], Fact_uint8_);
	        this.swordArtsParamID = this.convertValues(source["swordArtsParamID"], Fact_int32_);
	        this.swordArtsName = this.convertValues(source["swordArtsName"], Fact_string_);
	        this["swordArtsName-sfv"] = this.convertValues(source["swordArtsName-sfv"], Fact_string_);
	        this.defaultAffinity = this.convertValues(source["defaultAffinity"], Fact_uint8_);
	        this.compatibilityMask = this.convertValues(source["compatibilityMask"], Fact_uint64_);
	        this["compatibilityMask-sfv"] = this.convertValues(source["compatibilityMask-sfv"], Fact_uint64_);
	        this.compatibleClassNames = this.convertValues(source["compatibleClassNames"], Fact___string_);
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
	export class AshOfWarMountRules {
	    mode: string;
	    weaponType: string;
	    compatibilityBit: number;
	
	    static createFrom(source: any = {}) {
	        return new AshOfWarMountRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.weaponType = source["weaponType"];
	        this.compatibilityBit = source["compatibilityBit"];
	    }
	}
	export class BossDocument {
	    name: Fact_string_;
	    regionLabel: Fact_string_;
	    encounterType: Fact_string_;
	    remembrance: Fact_bool_;
	    defeatEventFlagID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new BossDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.regionLabel = this.convertValues(source["regionLabel"], Fact_string_);
	        this.encounterType = this.convertValues(source["encounterType"], Fact_string_);
	        this.remembrance = this.convertValues(source["remembrance"], Fact_bool_);
	        this.defeatEventFlagID = this.convertValues(source["defeatEventFlagID"], Fact_uint32_);
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
	export class Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_AshOfWarMountRules_ {
	    known: boolean;
	    enabled: boolean;
	    rules?: AshOfWarMountRules;
	    provenance: Provenance;
	    rulesEvidence?: Provenance[];
	
	    static createFrom(source: any = {}) {
	        return new Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_AshOfWarMountRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.enabled = source["enabled"];
	        this.rules = this.convertValues(source["rules"], AshOfWarMountRules);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	        this.rulesEvidence = this.convertValues(source["rulesEvidence"], Provenance);
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
	export class EquipmentRules {
	    allowedSlots: string[];
	
	    static createFrom(source: any = {}) {
	        return new EquipmentRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowedSlots = source["allowedSlots"];
	    }
	}
	export class Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_EquipmentRules_ {
	    known: boolean;
	    enabled: boolean;
	    rules?: EquipmentRules;
	    provenance: Provenance;
	    rulesEvidence?: Provenance[];
	
	    static createFrom(source: any = {}) {
	        return new Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_EquipmentRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.enabled = source["enabled"];
	        this.rules = this.convertValues(source["rules"], EquipmentRules);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	        this.rulesEvidence = this.convertValues(source["rulesEvidence"], Provenance);
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
	export class InfusionRules {
	    allowedAffinities: string[];
	
	    static createFrom(source: any = {}) {
	        return new InfusionRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowedAffinities = source["allowedAffinities"];
	    }
	}
	export class Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_InfusionRules_ {
	    known: boolean;
	    enabled: boolean;
	    rules?: InfusionRules;
	    provenance: Provenance;
	    rulesEvidence?: Provenance[];
	
	    static createFrom(source: any = {}) {
	        return new Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_InfusionRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.enabled = source["enabled"];
	        this.rules = this.convertValues(source["rules"], InfusionRules);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	        this.rulesEvidence = this.convertValues(source["rulesEvidence"], Provenance);
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
	export class StackRules {
	    maxPerStack: number;
	
	    static createFrom(source: any = {}) {
	        return new StackRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxPerStack = source["maxPerStack"];
	    }
	}
	export class Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_StackRules_ {
	    known: boolean;
	    enabled: boolean;
	    rules?: StackRules;
	    provenance: Provenance;
	    rulesEvidence?: Provenance[];
	
	    static createFrom(source: any = {}) {
	        return new Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_StackRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.enabled = source["enabled"];
	        this.rules = this.convertValues(source["rules"], StackRules);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	        this.rulesEvidence = this.convertValues(source["rulesEvidence"], Provenance);
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
	export class UpgradeRules {
	    model: string;
	    maxLevel: number;
	    "maxLevel-sfv"?: Fact_uint8_;
	
	    static createFrom(source: any = {}) {
	        return new UpgradeRules(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.maxLevel = source["maxLevel"];
	        this["maxLevel-sfv"] = this.convertValues(source["maxLevel-sfv"], Fact_uint8_);
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
	export class Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_UpgradeRules_ {
	    known: boolean;
	    enabled: boolean;
	    rules?: UpgradeRules;
	    provenance: Provenance;
	    rulesEvidence?: Provenance[];
	
	    static createFrom(source: any = {}) {
	        return new Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_UpgradeRules_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.enabled = source["enabled"];
	        this.rules = this.convertValues(source["rules"], UpgradeRules);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
	        this.rulesEvidence = this.convertValues(source["rulesEvidence"], Provenance);
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
	export class ClassDocument {
	    startingClassID: Fact_uint32_;
	    name: Fact_string_;
	    level: Fact_uint32_;
	    vigor: Fact_uint32_;
	    mind: Fact_uint32_;
	    endurance: Fact_uint32_;
	    strength: Fact_uint32_;
	    dexterity: Fact_uint32_;
	    intelligence: Fact_uint32_;
	    faith: Fact_uint32_;
	    arcane: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new ClassDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startingClassID = this.convertValues(source["startingClassID"], Fact_uint32_);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.level = this.convertValues(source["level"], Fact_uint32_);
	        this.vigor = this.convertValues(source["vigor"], Fact_uint32_);
	        this.mind = this.convertValues(source["mind"], Fact_uint32_);
	        this.endurance = this.convertValues(source["endurance"], Fact_uint32_);
	        this.strength = this.convertValues(source["strength"], Fact_uint32_);
	        this.dexterity = this.convertValues(source["dexterity"], Fact_uint32_);
	        this.intelligence = this.convertValues(source["intelligence"], Fact_uint32_);
	        this.faith = this.convertValues(source["faith"], Fact_uint32_);
	        this.arcane = this.convertValues(source["arcane"], Fact_uint32_);
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
	export class ColosseumDocument {
	    name: Fact_string_;
	    unlockEventFlagID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new ColosseumDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.unlockEventFlagID = this.convertValues(source["unlockEventFlagID"], Fact_uint32_);
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
	export class CuratedArmorStats {
	    weight: Fact_float64_;
	    physical: Fact_float64_;
	    strike: Fact_float64_;
	    slash: Fact_float64_;
	    pierce: Fact_float64_;
	    magic: Fact_float64_;
	    fire: Fact_float64_;
	    lightning: Fact_float64_;
	    holy: Fact_float64_;
	    immunity: Fact_uint32_;
	    robustness: Fact_uint32_;
	    focus: Fact_uint32_;
	    vitality: Fact_uint32_;
	    poise: Fact_float64_;
	
	    static createFrom(source: any = {}) {
	        return new CuratedArmorStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weight = this.convertValues(source["weight"], Fact_float64_);
	        this.physical = this.convertValues(source["physical"], Fact_float64_);
	        this.strike = this.convertValues(source["strike"], Fact_float64_);
	        this.slash = this.convertValues(source["slash"], Fact_float64_);
	        this.pierce = this.convertValues(source["pierce"], Fact_float64_);
	        this.magic = this.convertValues(source["magic"], Fact_float64_);
	        this.fire = this.convertValues(source["fire"], Fact_float64_);
	        this.lightning = this.convertValues(source["lightning"], Fact_float64_);
	        this.holy = this.convertValues(source["holy"], Fact_float64_);
	        this.immunity = this.convertValues(source["immunity"], Fact_uint32_);
	        this.robustness = this.convertValues(source["robustness"], Fact_uint32_);
	        this.focus = this.convertValues(source["focus"], Fact_uint32_);
	        this.vitality = this.convertValues(source["vitality"], Fact_uint32_);
	        this.poise = this.convertValues(source["poise"], Fact_float64_);
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
	export class CuratedSpellStats {
	    fpCost: Fact_uint32_;
	    memorySlots: Fact_uint32_;
	    requiredIntelligence: Fact_uint32_;
	    requiredFaith: Fact_uint32_;
	    requiredArcane: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new CuratedSpellStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fpCost = this.convertValues(source["fpCost"], Fact_uint32_);
	        this.memorySlots = this.convertValues(source["memorySlots"], Fact_uint32_);
	        this.requiredIntelligence = this.convertValues(source["requiredIntelligence"], Fact_uint32_);
	        this.requiredFaith = this.convertValues(source["requiredFaith"], Fact_uint32_);
	        this.requiredArcane = this.convertValues(source["requiredArcane"], Fact_uint32_);
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
	export class CuratedWeaponStats {
	    weight: Fact_float64_;
	    attackPhysical: Fact_uint32_;
	    attackMagic: Fact_uint32_;
	    attackFire: Fact_uint32_;
	    attackLightning: Fact_uint32_;
	    attackHoly: Fact_uint32_;
	    scalingStrength: Fact_uint32_;
	    scalingDexterity: Fact_uint32_;
	    scalingIntelligence: Fact_uint32_;
	    scalingFaith: Fact_uint32_;
	    requiredStrength: Fact_uint32_;
	    requiredDexterity: Fact_uint32_;
	    requiredIntelligence: Fact_uint32_;
	    requiredFaith: Fact_uint32_;
	    requiredArcane: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new CuratedWeaponStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weight = this.convertValues(source["weight"], Fact_float64_);
	        this.attackPhysical = this.convertValues(source["attackPhysical"], Fact_uint32_);
	        this.attackMagic = this.convertValues(source["attackMagic"], Fact_uint32_);
	        this.attackFire = this.convertValues(source["attackFire"], Fact_uint32_);
	        this.attackLightning = this.convertValues(source["attackLightning"], Fact_uint32_);
	        this.attackHoly = this.convertValues(source["attackHoly"], Fact_uint32_);
	        this.scalingStrength = this.convertValues(source["scalingStrength"], Fact_uint32_);
	        this.scalingDexterity = this.convertValues(source["scalingDexterity"], Fact_uint32_);
	        this.scalingIntelligence = this.convertValues(source["scalingIntelligence"], Fact_uint32_);
	        this.scalingFaith = this.convertValues(source["scalingFaith"], Fact_uint32_);
	        this.requiredStrength = this.convertValues(source["requiredStrength"], Fact_uint32_);
	        this.requiredDexterity = this.convertValues(source["requiredDexterity"], Fact_uint32_);
	        this.requiredIntelligence = this.convertValues(source["requiredIntelligence"], Fact_uint32_);
	        this.requiredFaith = this.convertValues(source["requiredFaith"], Fact_uint32_);
	        this.requiredArcane = this.convertValues(source["requiredArcane"], Fact_uint32_);
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
	export class EquipLoadModifier {
	    enduranceBonus: Fact_int32_;
	    "enduranceBonus-sfv"?: Fact_int32_;
	    equipLoadRate: Fact_float64_;
	    "equipLoadRate-sfv"?: Fact_float64_;
	
	    static createFrom(source: any = {}) {
	        return new EquipLoadModifier(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enduranceBonus = this.convertValues(source["enduranceBonus"], Fact_int32_);
	        this["enduranceBonus-sfv"] = this.convertValues(source["enduranceBonus-sfv"], Fact_int32_);
	        this.equipLoadRate = this.convertValues(source["equipLoadRate"], Fact_float64_);
	        this["equipLoadRate-sfv"] = this.convertValues(source["equipLoadRate-sfv"], Fact_float64_);
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
	
	
	export class Fact___uint32_ {
	    known: boolean;
	    value: number[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact___uint32_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	
	
	export class Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_Affinity_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_Affinity_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemFamily_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemFamily_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemVariantKind_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemVariantKind_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RecordMode_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RecordMode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedEventFlagKind_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedEventFlagKind_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedItemKind_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedItemKind_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_TechnicalRecordKind_ {
	    known: boolean;
	    value: string;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_TechnicalRecordKind_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	
	
	export class Fact_uint16_ {
	    known: boolean;
	    value: number;
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new Fact_uint16_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.known = source["known"];
	        this.value = source["value"];
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	
	
	
	export class ParameterField {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ParameterField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class ParameterRecord {
	    table: string;
	    rowID: number;
	    fields?: ParameterField[];
	    provenance: Provenance;
	
	    static createFrom(source: any = {}) {
	        return new ParameterRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.rowID = source["rowID"];
	        this.fields = this.convertValues(source["fields"], ParameterField);
	        this.provenance = this.convertValues(source["provenance"], Provenance);
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
	export class GestureSlotRecord {
	    slotID: Fact_uint32_;
	    itemID: Fact_uint32_;
	    name: Fact_string_;
	    category: Fact_string_;
	    flags: Fact___string_;
	    sourceRecords: ParameterRecord[];
	
	    static createFrom(source: any = {}) {
	        return new GestureSlotRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slotID = this.convertValues(source["slotID"], Fact_uint32_);
	        this.itemID = this.convertValues(source["itemID"], Fact_uint32_);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.category = this.convertValues(source["category"], Fact_string_);
	        this.flags = this.convertValues(source["flags"], Fact___string_);
	        this.sourceRecords = this.convertValues(source["sourceRecords"], ParameterRecord);
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
	export class GestureData {
	    goodsSourceRowID: Fact_uint32_;
	    iconID: Fact_uint32_;
	    slots: GestureSlotRecord[];
	
	    static createFrom(source: any = {}) {
	        return new GestureData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.goodsSourceRowID = this.convertValues(source["goodsSourceRowID"], Fact_uint32_);
	        this.iconID = this.convertValues(source["iconID"], Fact_uint32_);
	        this.slots = this.convertValues(source["slots"], GestureSlotRecord);
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
	
	export class GoodsData {
	    sourceRowID: Fact_uint32_;
	    iconID: Fact_uint32_;
	    sortID: Fact_uint32_;
	    "sortID-sfv"?: Fact_uint32_;
	    sortGroupID: Fact_uint8_;
	    "sortGroupID-sfv"?: Fact_uint8_;
	    goodsType: Fact_uint16_;
	    weight: Fact_float64_;
	    "weight-sfv"?: Fact_float64_;
	    maxQuantity: Fact_uint32_;
	    maxRepository: Fact_uint32_;
	    tutorialFlagID: Fact_uint32_;
	    isEquipable: Fact_bool_;
	    isConsumable: Fact_bool_;
	    isDiscardable: Fact_bool_;
	    isDepositable: Fact_bool_;
	    isDroppable: Fact_bool_;
	
	    static createFrom(source: any = {}) {
	        return new GoodsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.iconID = this.convertValues(source["iconID"], Fact_uint32_);
	        this.sortID = this.convertValues(source["sortID"], Fact_uint32_);
	        this["sortID-sfv"] = this.convertValues(source["sortID-sfv"], Fact_uint32_);
	        this.sortGroupID = this.convertValues(source["sortGroupID"], Fact_uint8_);
	        this["sortGroupID-sfv"] = this.convertValues(source["sortGroupID-sfv"], Fact_uint8_);
	        this.goodsType = this.convertValues(source["goodsType"], Fact_uint16_);
	        this.weight = this.convertValues(source["weight"], Fact_float64_);
	        this["weight-sfv"] = this.convertValues(source["weight-sfv"], Fact_float64_);
	        this.maxQuantity = this.convertValues(source["maxQuantity"], Fact_uint32_);
	        this.maxRepository = this.convertValues(source["maxRepository"], Fact_uint32_);
	        this.tutorialFlagID = this.convertValues(source["tutorialFlagID"], Fact_uint32_);
	        this.isEquipable = this.convertValues(source["isEquipable"], Fact_bool_);
	        this.isConsumable = this.convertValues(source["isConsumable"], Fact_bool_);
	        this.isDiscardable = this.convertValues(source["isDiscardable"], Fact_bool_);
	        this.isDepositable = this.convertValues(source["isDepositable"], Fact_bool_);
	        this.isDroppable = this.convertValues(source["isDroppable"], Fact_bool_);
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
	export class GraceDocument {
	    name: Fact_string_;
	    regionLabel: Fact_string_;
	    visitEventFlagID: Fact_uint32_;
	    bossArena: Fact_bool_;
	    dungeonType: Fact_string_;
	    doorEventFlagID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new GraceDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.regionLabel = this.convertValues(source["regionLabel"], Fact_string_);
	        this.visitEventFlagID = this.convertValues(source["visitEventFlagID"], Fact_uint32_);
	        this.bossArena = this.convertValues(source["bossArena"], Fact_bool_);
	        this.dungeonType = this.convertValues(source["dungeonType"], Fact_string_);
	        this.doorEventFlagID = this.convertValues(source["doorEventFlagID"], Fact_uint32_);
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
	
	export class ItemAcquisition {
	    requiredContainerID: Fact_uint32_;
	    isContainer: Fact_bool_;
	    containerPickupFlagIDs: Fact___uint32_;
	    containerVendorFlagIDs: Fact___uint32_;
	    bolsteringPickupFlagIDs: Fact___uint32_;
	    worldPickupFlagID: Fact_uint32_;
	    companionEventFlagIDs: Fact___uint32_;
	
	    static createFrom(source: any = {}) {
	        return new ItemAcquisition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requiredContainerID = this.convertValues(source["requiredContainerID"], Fact_uint32_);
	        this.isContainer = this.convertValues(source["isContainer"], Fact_bool_);
	        this.containerPickupFlagIDs = this.convertValues(source["containerPickupFlagIDs"], Fact___uint32_);
	        this.containerVendorFlagIDs = this.convertValues(source["containerVendorFlagIDs"], Fact___uint32_);
	        this.bolsteringPickupFlagIDs = this.convertValues(source["bolsteringPickupFlagIDs"], Fact___uint32_);
	        this.worldPickupFlagID = this.convertValues(source["worldPickupFlagID"], Fact_uint32_);
	        this.companionEventFlagIDs = this.convertValues(source["companionEventFlagIDs"], Fact___uint32_);
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
	export class ItemAlias {
	    gameID: Fact_uint32_;
	    sourceRecords: ParameterRecord[];
	
	    static createFrom(source: any = {}) {
	        return new ItemAlias(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameID = this.convertValues(source["gameID"], Fact_uint32_);
	        this.sourceRecords = this.convertValues(source["sourceRecords"], ParameterRecord);
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
	export class ItemCapabilities {
	    upgrade: Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_UpgradeRules_;
	    infusion: Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_InfusionRules_;
	    ashOfWarMount: Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_AshOfWarMountRules_;
	    stack: Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_StackRules_;
	    equipment: Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_EquipmentRules_;
	
	    static createFrom(source: any = {}) {
	        return new ItemCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.upgrade = this.convertValues(source["upgrade"], Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_UpgradeRules_);
	        this.infusion = this.convertValues(source["infusion"], Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_InfusionRules_);
	        this.ashOfWarMount = this.convertValues(source["ashOfWarMount"], Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_AshOfWarMountRules_);
	        this.stack = this.convertValues(source["stack"], Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_StackRules_);
	        this.equipment = this.convertValues(source["equipment"], Capability_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_EquipmentRules_);
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
	export class ItemDescriptionRecord {
	    description: Fact_string_;
	    location: Fact_string_;
	    weight: Fact_float64_;
	    weapon?: CuratedWeaponStats;
	    armor?: CuratedArmorStats;
	    spell?: CuratedSpellStats;
	
	    static createFrom(source: any = {}) {
	        return new ItemDescriptionRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = this.convertValues(source["description"], Fact_string_);
	        this.location = this.convertValues(source["location"], Fact_string_);
	        this.weight = this.convertValues(source["weight"], Fact_float64_);
	        this.weapon = this.convertValues(source["weapon"], CuratedWeaponStats);
	        this.armor = this.convertValues(source["armor"], CuratedArmorStats);
	        this.spell = this.convertValues(source["spell"], CuratedSpellStats);
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
	export class SpellData {
	    sourceRowID: Fact_uint32_;
	    iconID: Fact_uint32_;
	    sortID: Fact_uint32_;
	    "sortID-sfv"?: Fact_uint32_;
	    fpCost: Fact_uint32_;
	    "fpCost-sfv"?: Fact_uint32_;
	    staminaCost: Fact_uint32_;
	    memorySlots: Fact_uint8_;
	    "memorySlots-sfv"?: Fact_uint8_;
	    requiredIntelligence: Fact_uint32_;
	    "requiredIntelligence-sfv"?: Fact_uint32_;
	    requiredFaith: Fact_uint32_;
	    "requiredFaith-sfv"?: Fact_uint32_;
	    requiredArcane: Fact_uint32_;
	    "requiredArcane-sfv"?: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new SpellData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.iconID = this.convertValues(source["iconID"], Fact_uint32_);
	        this.sortID = this.convertValues(source["sortID"], Fact_uint32_);
	        this["sortID-sfv"] = this.convertValues(source["sortID-sfv"], Fact_uint32_);
	        this.fpCost = this.convertValues(source["fpCost"], Fact_uint32_);
	        this["fpCost-sfv"] = this.convertValues(source["fpCost-sfv"], Fact_uint32_);
	        this.staminaCost = this.convertValues(source["staminaCost"], Fact_uint32_);
	        this.memorySlots = this.convertValues(source["memorySlots"], Fact_uint8_);
	        this["memorySlots-sfv"] = this.convertValues(source["memorySlots-sfv"], Fact_uint8_);
	        this.requiredIntelligence = this.convertValues(source["requiredIntelligence"], Fact_uint32_);
	        this["requiredIntelligence-sfv"] = this.convertValues(source["requiredIntelligence-sfv"], Fact_uint32_);
	        this.requiredFaith = this.convertValues(source["requiredFaith"], Fact_uint32_);
	        this["requiredFaith-sfv"] = this.convertValues(source["requiredFaith-sfv"], Fact_uint32_);
	        this.requiredArcane = this.convertValues(source["requiredArcane"], Fact_uint32_);
	        this["requiredArcane-sfv"] = this.convertValues(source["requiredArcane-sfv"], Fact_uint32_);
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
	export class TalismanData {
	    sourceRowID: Fact_uint32_;
	    iconID: Fact_uint32_;
	    sortID: Fact_uint32_;
	    "sortID-sfv"?: Fact_uint32_;
	    sortGroupID: Fact_uint8_;
	    "sortGroupID-sfv"?: Fact_uint8_;
	    weight: Fact_float64_;
	    "weight-sfv"?: Fact_float64_;
	
	    static createFrom(source: any = {}) {
	        return new TalismanData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.iconID = this.convertValues(source["iconID"], Fact_uint32_);
	        this.sortID = this.convertValues(source["sortID"], Fact_uint32_);
	        this["sortID-sfv"] = this.convertValues(source["sortID-sfv"], Fact_uint32_);
	        this.sortGroupID = this.convertValues(source["sortGroupID"], Fact_uint8_);
	        this["sortGroupID-sfv"] = this.convertValues(source["sortGroupID-sfv"], Fact_uint8_);
	        this.weight = this.convertValues(source["weight"], Fact_float64_);
	        this["weight-sfv"] = this.convertValues(source["weight-sfv"], Fact_float64_);
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
	export class SpiritAshData {
	    sourceRowID: Fact_uint32_;
	    iconID: Fact_uint32_;
	    sortID: Fact_uint32_;
	    "sortID-sfv"?: Fact_uint32_;
	    sortGroupID: Fact_uint8_;
	    "sortGroupID-sfv"?: Fact_uint8_;
	    reinforceGoodsID: Fact_int32_;
	    reinforceMaterialID: Fact_int32_;
	    reinforcePrice: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new SpiritAshData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.iconID = this.convertValues(source["iconID"], Fact_uint32_);
	        this.sortID = this.convertValues(source["sortID"], Fact_uint32_);
	        this["sortID-sfv"] = this.convertValues(source["sortID-sfv"], Fact_uint32_);
	        this.sortGroupID = this.convertValues(source["sortGroupID"], Fact_uint8_);
	        this["sortGroupID-sfv"] = this.convertValues(source["sortGroupID-sfv"], Fact_uint8_);
	        this.reinforceGoodsID = this.convertValues(source["reinforceGoodsID"], Fact_int32_);
	        this.reinforceMaterialID = this.convertValues(source["reinforceMaterialID"], Fact_int32_);
	        this.reinforcePrice = this.convertValues(source["reinforcePrice"], Fact_uint32_);
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
	export class WeaponPassiveEffectData {
	    kind: Fact_string_;
	    source: Fact_string_;
	    spEffectID: Fact_int32_;
	    label: Fact_string_;
	    value: Fact_int32_;
	    known: Fact_bool_;
	
	    static createFrom(source: any = {}) {
	        return new WeaponPassiveEffectData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = this.convertValues(source["kind"], Fact_string_);
	        this.source = this.convertValues(source["source"], Fact_string_);
	        this.spEffectID = this.convertValues(source["spEffectID"], Fact_int32_);
	        this.label = this.convertValues(source["label"], Fact_string_);
	        this.value = this.convertValues(source["value"], Fact_int32_);
	        this.known = this.convertValues(source["known"], Fact_bool_);
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
	export class WeaponData {
	    sourceRowID: Fact_uint32_;
	    iconID: Fact_uint32_;
	    weaponTypeID: Fact_uint16_;
	    "weaponTypeID-sfv"?: Fact_uint16_;
	    sortID: Fact_uint32_;
	    "sortID-sfv"?: Fact_uint32_;
	    sortGroupID: Fact_uint8_;
	    "sortGroupID-sfv"?: Fact_uint8_;
	    reinforceTypeID: Fact_int32_;
	    "reinforceTypeID-sfv"?: Fact_int32_;
	    gemMountType: Fact_uint8_;
	    "gemMountType-sfv"?: Fact_uint8_;
	    weight: Fact_float64_;
	    "weight-sfv"?: Fact_float64_;
	    attackPhysical: Fact_int32_;
	    "attackPhysical-sfv"?: Fact_int32_;
	    attackMagic: Fact_int32_;
	    "attackMagic-sfv"?: Fact_int32_;
	    attackFire: Fact_int32_;
	    "attackFire-sfv"?: Fact_int32_;
	    attackLightning: Fact_int32_;
	    "attackLightning-sfv"?: Fact_int32_;
	    attackHoly: Fact_int32_;
	    "attackHoly-sfv"?: Fact_int32_;
	    attackStamina: Fact_int32_;
	    "attackStamina-sfv"?: Fact_int32_;
	    guardPhysical: Fact_int32_;
	    "guardPhysical-sfv"?: Fact_int32_;
	    guardMagic: Fact_int32_;
	    "guardMagic-sfv"?: Fact_int32_;
	    guardFire: Fact_int32_;
	    "guardFire-sfv"?: Fact_int32_;
	    guardLightning: Fact_int32_;
	    "guardLightning-sfv"?: Fact_int32_;
	    guardHoly: Fact_int32_;
	    "guardHoly-sfv"?: Fact_int32_;
	    guardBoost: Fact_int32_;
	    "guardBoost-sfv"?: Fact_int32_;
	    requiredStrength: Fact_int32_;
	    "requiredStrength-sfv"?: Fact_int32_;
	    requiredDexterity: Fact_int32_;
	    "requiredDexterity-sfv"?: Fact_int32_;
	    requiredIntelligence: Fact_int32_;
	    "requiredIntelligence-sfv"?: Fact_int32_;
	    requiredFaith: Fact_int32_;
	    "requiredFaith-sfv"?: Fact_int32_;
	    requiredArcane: Fact_int32_;
	    "requiredArcane-sfv"?: Fact_int32_;
	    scalingStrengthRaw: Fact_float64_;
	    scalingDexterityRaw: Fact_float64_;
	    scalingIntelligenceRaw: Fact_float64_;
	    scalingFaithRaw: Fact_float64_;
	    scalingArcaneRaw: Fact_float64_;
	    critical: Fact_int32_;
	    "critical-sfv"?: Fact_int32_;
	    passiveEffects: WeaponPassiveEffectData[];
	    defaultAshOfWarID: Fact_int32_;
	    "defaultAshOfWarID-sfv"?: Fact_int32_;
	    swordArtsName: Fact_string_;
	    "swordArtsName-sfv"?: Fact_string_;
	    isInfusable: Fact_bool_;
	    "isInfusable-sfv"?: Fact_bool_;
	    isSomber: Fact_bool_;
	    "isSomber-sfv"?: Fact_bool_;
	    maxUpgrade: Fact_int32_;
	    "maxUpgrade-sfv"?: Fact_int32_;
	    warnings: Fact___string_;
	    rightHandEquipable: Fact_bool_;
	    leftHandEquipable: Fact_bool_;
	    bothHandEquipable: Fact_bool_;
	    arrowSlotEquipable: Fact_bool_;
	    boltSlotEquipable: Fact_bool_;
	
	    static createFrom(source: any = {}) {
	        return new WeaponData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.iconID = this.convertValues(source["iconID"], Fact_uint32_);
	        this.weaponTypeID = this.convertValues(source["weaponTypeID"], Fact_uint16_);
	        this["weaponTypeID-sfv"] = this.convertValues(source["weaponTypeID-sfv"], Fact_uint16_);
	        this.sortID = this.convertValues(source["sortID"], Fact_uint32_);
	        this["sortID-sfv"] = this.convertValues(source["sortID-sfv"], Fact_uint32_);
	        this.sortGroupID = this.convertValues(source["sortGroupID"], Fact_uint8_);
	        this["sortGroupID-sfv"] = this.convertValues(source["sortGroupID-sfv"], Fact_uint8_);
	        this.reinforceTypeID = this.convertValues(source["reinforceTypeID"], Fact_int32_);
	        this["reinforceTypeID-sfv"] = this.convertValues(source["reinforceTypeID-sfv"], Fact_int32_);
	        this.gemMountType = this.convertValues(source["gemMountType"], Fact_uint8_);
	        this["gemMountType-sfv"] = this.convertValues(source["gemMountType-sfv"], Fact_uint8_);
	        this.weight = this.convertValues(source["weight"], Fact_float64_);
	        this["weight-sfv"] = this.convertValues(source["weight-sfv"], Fact_float64_);
	        this.attackPhysical = this.convertValues(source["attackPhysical"], Fact_int32_);
	        this["attackPhysical-sfv"] = this.convertValues(source["attackPhysical-sfv"], Fact_int32_);
	        this.attackMagic = this.convertValues(source["attackMagic"], Fact_int32_);
	        this["attackMagic-sfv"] = this.convertValues(source["attackMagic-sfv"], Fact_int32_);
	        this.attackFire = this.convertValues(source["attackFire"], Fact_int32_);
	        this["attackFire-sfv"] = this.convertValues(source["attackFire-sfv"], Fact_int32_);
	        this.attackLightning = this.convertValues(source["attackLightning"], Fact_int32_);
	        this["attackLightning-sfv"] = this.convertValues(source["attackLightning-sfv"], Fact_int32_);
	        this.attackHoly = this.convertValues(source["attackHoly"], Fact_int32_);
	        this["attackHoly-sfv"] = this.convertValues(source["attackHoly-sfv"], Fact_int32_);
	        this.attackStamina = this.convertValues(source["attackStamina"], Fact_int32_);
	        this["attackStamina-sfv"] = this.convertValues(source["attackStamina-sfv"], Fact_int32_);
	        this.guardPhysical = this.convertValues(source["guardPhysical"], Fact_int32_);
	        this["guardPhysical-sfv"] = this.convertValues(source["guardPhysical-sfv"], Fact_int32_);
	        this.guardMagic = this.convertValues(source["guardMagic"], Fact_int32_);
	        this["guardMagic-sfv"] = this.convertValues(source["guardMagic-sfv"], Fact_int32_);
	        this.guardFire = this.convertValues(source["guardFire"], Fact_int32_);
	        this["guardFire-sfv"] = this.convertValues(source["guardFire-sfv"], Fact_int32_);
	        this.guardLightning = this.convertValues(source["guardLightning"], Fact_int32_);
	        this["guardLightning-sfv"] = this.convertValues(source["guardLightning-sfv"], Fact_int32_);
	        this.guardHoly = this.convertValues(source["guardHoly"], Fact_int32_);
	        this["guardHoly-sfv"] = this.convertValues(source["guardHoly-sfv"], Fact_int32_);
	        this.guardBoost = this.convertValues(source["guardBoost"], Fact_int32_);
	        this["guardBoost-sfv"] = this.convertValues(source["guardBoost-sfv"], Fact_int32_);
	        this.requiredStrength = this.convertValues(source["requiredStrength"], Fact_int32_);
	        this["requiredStrength-sfv"] = this.convertValues(source["requiredStrength-sfv"], Fact_int32_);
	        this.requiredDexterity = this.convertValues(source["requiredDexterity"], Fact_int32_);
	        this["requiredDexterity-sfv"] = this.convertValues(source["requiredDexterity-sfv"], Fact_int32_);
	        this.requiredIntelligence = this.convertValues(source["requiredIntelligence"], Fact_int32_);
	        this["requiredIntelligence-sfv"] = this.convertValues(source["requiredIntelligence-sfv"], Fact_int32_);
	        this.requiredFaith = this.convertValues(source["requiredFaith"], Fact_int32_);
	        this["requiredFaith-sfv"] = this.convertValues(source["requiredFaith-sfv"], Fact_int32_);
	        this.requiredArcane = this.convertValues(source["requiredArcane"], Fact_int32_);
	        this["requiredArcane-sfv"] = this.convertValues(source["requiredArcane-sfv"], Fact_int32_);
	        this.scalingStrengthRaw = this.convertValues(source["scalingStrengthRaw"], Fact_float64_);
	        this.scalingDexterityRaw = this.convertValues(source["scalingDexterityRaw"], Fact_float64_);
	        this.scalingIntelligenceRaw = this.convertValues(source["scalingIntelligenceRaw"], Fact_float64_);
	        this.scalingFaithRaw = this.convertValues(source["scalingFaithRaw"], Fact_float64_);
	        this.scalingArcaneRaw = this.convertValues(source["scalingArcaneRaw"], Fact_float64_);
	        this.critical = this.convertValues(source["critical"], Fact_int32_);
	        this["critical-sfv"] = this.convertValues(source["critical-sfv"], Fact_int32_);
	        this.passiveEffects = this.convertValues(source["passiveEffects"], WeaponPassiveEffectData);
	        this.defaultAshOfWarID = this.convertValues(source["defaultAshOfWarID"], Fact_int32_);
	        this["defaultAshOfWarID-sfv"] = this.convertValues(source["defaultAshOfWarID-sfv"], Fact_int32_);
	        this.swordArtsName = this.convertValues(source["swordArtsName"], Fact_string_);
	        this["swordArtsName-sfv"] = this.convertValues(source["swordArtsName-sfv"], Fact_string_);
	        this.isInfusable = this.convertValues(source["isInfusable"], Fact_bool_);
	        this["isInfusable-sfv"] = this.convertValues(source["isInfusable-sfv"], Fact_bool_);
	        this.isSomber = this.convertValues(source["isSomber"], Fact_bool_);
	        this["isSomber-sfv"] = this.convertValues(source["isSomber-sfv"], Fact_bool_);
	        this.maxUpgrade = this.convertValues(source["maxUpgrade"], Fact_int32_);
	        this["maxUpgrade-sfv"] = this.convertValues(source["maxUpgrade-sfv"], Fact_int32_);
	        this.warnings = this.convertValues(source["warnings"], Fact___string_);
	        this.rightHandEquipable = this.convertValues(source["rightHandEquipable"], Fact_bool_);
	        this.leftHandEquipable = this.convertValues(source["leftHandEquipable"], Fact_bool_);
	        this.bothHandEquipable = this.convertValues(source["bothHandEquipable"], Fact_bool_);
	        this.arrowSlotEquipable = this.convertValues(source["arrowSlotEquipable"], Fact_bool_);
	        this.boltSlotEquipable = this.convertValues(source["boltSlotEquipable"], Fact_bool_);
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
	export class RelatedTechnicalRecord {
	    kind: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_TechnicalRecordKind_;
	    gameID: Fact_uint32_;
	    description: ItemDescriptionRecord;
	    gameMaxInventory: Fact_uint32_;
	    gameMaxStorage: Fact_uint32_;
	    sourceRecords: ParameterRecord[];
	
	    static createFrom(source: any = {}) {
	        return new RelatedTechnicalRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = this.convertValues(source["kind"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_TechnicalRecordKind_);
	        this.gameID = this.convertValues(source["gameID"], Fact_uint32_);
	        this.description = this.convertValues(source["description"], ItemDescriptionRecord);
	        this.gameMaxInventory = this.convertValues(source["gameMaxInventory"], Fact_uint32_);
	        this.gameMaxStorage = this.convertValues(source["gameMaxStorage"], Fact_uint32_);
	        this.sourceRecords = this.convertValues(source["sourceRecords"], ParameterRecord);
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
	export class ItemUnlock {
	    kind: Fact_string_;
	    eventFlagID: Fact_uint32_;
	    name: Fact_string_;
	    category: Fact_string_;
	
	    static createFrom(source: any = {}) {
	        return new ItemUnlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = this.convertValues(source["kind"], Fact_string_);
	        this.eventFlagID = this.convertValues(source["eventFlagID"], Fact_uint32_);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.category = this.convertValues(source["category"], Fact_string_);
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
	export class VariantDocumentData {
	    family: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemFamily_;
	    category: Fact_string_;
	    subcategory: Fact_string_;
	    presentation: ItemPresentation;
	    storage: ItemStorage;
	    capabilities: ItemCapabilities;
	    safety: ItemSafety;
	    acquisition: ItemAcquisition;
	    modifiers: ItemModifiers;
	    links: ItemLinks;
	    unlocks: ItemUnlock[];
	    relatedTechnicalRecords: RelatedTechnicalRecord[];
	    weapon?: WeaponData;
	    spiritAsh?: SpiritAshData;
	
	    static createFrom(source: any = {}) {
	        return new VariantDocumentData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.family = this.convertValues(source["family"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemFamily_);
	        this.category = this.convertValues(source["category"], Fact_string_);
	        this.subcategory = this.convertValues(source["subcategory"], Fact_string_);
	        this.presentation = this.convertValues(source["presentation"], ItemPresentation);
	        this.storage = this.convertValues(source["storage"], ItemStorage);
	        this.capabilities = this.convertValues(source["capabilities"], ItemCapabilities);
	        this.safety = this.convertValues(source["safety"], ItemSafety);
	        this.acquisition = this.convertValues(source["acquisition"], ItemAcquisition);
	        this.modifiers = this.convertValues(source["modifiers"], ItemModifiers);
	        this.links = this.convertValues(source["links"], ItemLinks);
	        this.unlocks = this.convertValues(source["unlocks"], ItemUnlock);
	        this.relatedTechnicalRecords = this.convertValues(source["relatedTechnicalRecords"], RelatedTechnicalRecord);
	        this.weapon = this.convertValues(source["weapon"], WeaponData);
	        this.spiritAsh = this.convertValues(source["spiritAsh"], SpiritAshData);
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
	export class ItemVariant {
	    gameID: Fact_uint32_;
	    kind: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemVariantKind_;
	    affinity: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_Affinity_;
	    upgradeLevel: Fact_uint8_;
	    sourceRowID: Fact_uint32_;
	    data: VariantDocumentData;
	    sourceRecords: ParameterRecord[];
	
	    static createFrom(source: any = {}) {
	        return new ItemVariant(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameID = this.convertValues(source["gameID"], Fact_uint32_);
	        this.kind = this.convertValues(source["kind"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemVariantKind_);
	        this.affinity = this.convertValues(source["affinity"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_Affinity_);
	        this.upgradeLevel = this.convertValues(source["upgradeLevel"], Fact_uint8_);
	        this.sourceRowID = this.convertValues(source["sourceRowID"], Fact_uint32_);
	        this.data = this.convertValues(source["data"], VariantDocumentData);
	        this.sourceRecords = this.convertValues(source["sourceRecords"], ParameterRecord);
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
	export class MapFragmentMetadata {
	    name: Fact_string_;
	    area: Fact_string_;
	    acquiredFlagID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new MapFragmentMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.area = this.convertValues(source["area"], Fact_string_);
	        this.acquiredFlagID = this.convertValues(source["acquiredFlagID"], Fact_uint32_);
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
	export class RelatedItem {
	    kind: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedItemKind_;
	    gameID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new RelatedItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = this.convertValues(source["kind"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedItemKind_);
	        this.gameID = this.convertValues(source["gameID"], Fact_uint32_);
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
	export class RelatedEventFlag {
	    kind: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedEventFlagKind_;
	    eventFlagID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new RelatedEventFlag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = this.convertValues(source["kind"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RelatedEventFlagKind_);
	        this.eventFlagID = this.convertValues(source["eventFlagID"], Fact_uint32_);
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
	export class ItemLinks {
	    aboutTutorialID: Fact_uint32_;
	    relatedEventFlags: RelatedEventFlag[];
	    relatedItems: RelatedItem[];
	    mapFragment?: MapFragmentMetadata;
	
	    static createFrom(source: any = {}) {
	        return new ItemLinks(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aboutTutorialID = this.convertValues(source["aboutTutorialID"], Fact_uint32_);
	        this.relatedEventFlags = this.convertValues(source["relatedEventFlags"], RelatedEventFlag);
	        this.relatedItems = this.convertValues(source["relatedItems"], RelatedItem);
	        this.mapFragment = this.convertValues(source["mapFragment"], MapFragmentMetadata);
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
	export class ItemModifiers {
	    equipLoad?: EquipLoadModifier;
	
	    static createFrom(source: any = {}) {
	        return new ItemModifiers(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.equipLoad = this.convertValues(source["equipLoad"], EquipLoadModifier);
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
	export class ItemSafety {
	    cutContent: Fact_bool_;
	    banRisk: Fact_bool_;
	    dlc: Fact_bool_;
	    noDatabase: Fact_bool_;
	    scalesWithNG: Fact_bool_;
	    preOrder: Fact_bool_;
	
	    static createFrom(source: any = {}) {
	        return new ItemSafety(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cutContent = this.convertValues(source["cutContent"], Fact_bool_);
	        this.banRisk = this.convertValues(source["banRisk"], Fact_bool_);
	        this.dlc = this.convertValues(source["dlc"], Fact_bool_);
	        this.noDatabase = this.convertValues(source["noDatabase"], Fact_bool_);
	        this.scalesWithNG = this.convertValues(source["scalesWithNG"], Fact_bool_);
	        this.preOrder = this.convertValues(source["preOrder"], Fact_bool_);
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
	export class ItemStorage {
	    recordMode: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RecordMode_;
	    maxInventory: Fact_uint32_;
	    safeModeMaxInventory?: Fact_uint32_;
	    "maxInventory-sfv"?: Fact_uint32_;
	    maxStorage: Fact_uint32_;
	    safeModeMaxStorage?: Fact_uint32_;
	    "maxStorage-sfv"?: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new ItemStorage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordMode = this.convertValues(source["recordMode"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_RecordMode_);
	        this.maxInventory = this.convertValues(source["maxInventory"], Fact_uint32_);
	        this.safeModeMaxInventory = this.convertValues(source["safeModeMaxInventory"], Fact_uint32_);
	        this["maxInventory-sfv"] = this.convertValues(source["maxInventory-sfv"], Fact_uint32_);
	        this.maxStorage = this.convertValues(source["maxStorage"], Fact_uint32_);
	        this.safeModeMaxStorage = this.convertValues(source["safeModeMaxStorage"], Fact_uint32_);
	        this["maxStorage-sfv"] = this.convertValues(source["maxStorage-sfv"], Fact_uint32_);
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
	export class ItemTextMetadata {
	    captionSource: Fact_string_;
	    descriptionSource: Fact_string_;
	    locationSource: Fact_string_;
	    dlcSource: Fact_string_;
	    notes: Fact_string_;
	
	    static createFrom(source: any = {}) {
	        return new ItemTextMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.captionSource = this.convertValues(source["captionSource"], Fact_string_);
	        this.descriptionSource = this.convertValues(source["descriptionSource"], Fact_string_);
	        this.locationSource = this.convertValues(source["locationSource"], Fact_string_);
	        this.dlcSource = this.convertValues(source["dlcSource"], Fact_string_);
	        this.notes = this.convertValues(source["notes"], Fact_string_);
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
	export class ItemPresentation {
	    name: Fact_string_;
	    caption: Fact_string_;
	    description: Fact_string_;
	    location: Fact_string_;
	    iconPath: Fact_string_;
	    textMetadata: ItemTextMetadata;
	
	    static createFrom(source: any = {}) {
	        return new ItemPresentation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.caption = this.convertValues(source["caption"], Fact_string_);
	        this.description = this.convertValues(source["description"], Fact_string_);
	        this.location = this.convertValues(source["location"], Fact_string_);
	        this.iconPath = this.convertValues(source["iconPath"], Fact_string_);
	        this.textMetadata = this.convertValues(source["textMetadata"], ItemTextMetadata);
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
	export class ItemDocument {
	    gameID: Fact_uint32_;
	    family: Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemFamily_;
	    category: Fact_string_;
	    subcategory: Fact_string_;
	    presentation: ItemPresentation;
	    storage: ItemStorage;
	    capabilities: ItemCapabilities;
	    safety: ItemSafety;
	    acquisition: ItemAcquisition;
	    modifiers: ItemModifiers;
	    links: ItemLinks;
	    variants: ItemVariant[];
	    aliases: ItemAlias[];
	    unlocks: ItemUnlock[];
	    relatedTechnicalRecords: RelatedTechnicalRecord[];
	    sourceRecords: ParameterRecord[];
	    weapon?: WeaponData;
	    armor?: ArmorData;
	    talisman?: TalismanData;
	    ashOfWar?: AshOfWarData;
	    spell?: SpellData;
	    spiritAsh?: SpiritAshData;
	    goods?: GoodsData;
	    gesture?: GestureData;
	
	    static createFrom(source: any = {}) {
	        return new ItemDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameID = this.convertValues(source["gameID"], Fact_uint32_);
	        this.family = this.convertValues(source["family"], Fact_github_com_oisis_EldenRing_SaveForge_backend_gamecatalog_schema_ItemFamily_);
	        this.category = this.convertValues(source["category"], Fact_string_);
	        this.subcategory = this.convertValues(source["subcategory"], Fact_string_);
	        this.presentation = this.convertValues(source["presentation"], ItemPresentation);
	        this.storage = this.convertValues(source["storage"], ItemStorage);
	        this.capabilities = this.convertValues(source["capabilities"], ItemCapabilities);
	        this.safety = this.convertValues(source["safety"], ItemSafety);
	        this.acquisition = this.convertValues(source["acquisition"], ItemAcquisition);
	        this.modifiers = this.convertValues(source["modifiers"], ItemModifiers);
	        this.links = this.convertValues(source["links"], ItemLinks);
	        this.variants = this.convertValues(source["variants"], ItemVariant);
	        this.aliases = this.convertValues(source["aliases"], ItemAlias);
	        this.unlocks = this.convertValues(source["unlocks"], ItemUnlock);
	        this.relatedTechnicalRecords = this.convertValues(source["relatedTechnicalRecords"], RelatedTechnicalRecord);
	        this.sourceRecords = this.convertValues(source["sourceRecords"], ParameterRecord);
	        this.weapon = this.convertValues(source["weapon"], WeaponData);
	        this.armor = this.convertValues(source["armor"], ArmorData);
	        this.talisman = this.convertValues(source["talisman"], TalismanData);
	        this.ashOfWar = this.convertValues(source["ashOfWar"], AshOfWarData);
	        this.spell = this.convertValues(source["spell"], SpellData);
	        this.spiritAsh = this.convertValues(source["spiritAsh"], SpiritAshData);
	        this.goods = this.convertValues(source["goods"], GoodsData);
	        this.gesture = this.convertValues(source["gesture"], GestureData);
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
	
	
	
	
	
	
	
	
	
	export class MapRegionDocument {
	    name: Fact_string_;
	    areaLabel: Fact_string_;
	    visibleEventFlagID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new MapRegionDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.areaLabel = this.convertValues(source["areaLabel"], Fact_string_);
	        this.visibleEventFlagID = this.convertValues(source["visibleEventFlagID"], Fact_uint32_);
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
	
	
	
	export class QuestFlag {
	    id: number;
	    value: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QuestFlag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.value = source["value"];
	    }
	}
	export class QuestStepDocument {
	    key: string;
	    description: Fact_string_;
	    location: Fact_string_;
	    flags: QuestFlag[];
	
	    static createFrom(source: any = {}) {
	        return new QuestStepDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.description = this.convertValues(source["description"], Fact_string_);
	        this.location = this.convertValues(source["location"], Fact_string_);
	        this.flags = this.convertValues(source["flags"], QuestFlag);
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
	export class QuestDocument {
	    name: Fact_string_;
	    steps: QuestStepDocument[];
	
	    static createFrom(source: any = {}) {
	        return new QuestDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.steps = this.convertValues(source["steps"], QuestStepDocument);
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
	
	
	export class RegionDocument {
	    regionID: Fact_uint32_;
	    name: Fact_string_;
	    area: Fact_string_;
	
	    static createFrom(source: any = {}) {
	        return new RegionDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.regionID = this.convertValues(source["regionID"], Fact_uint32_);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.area = this.convertValues(source["area"], Fact_string_);
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
	
	
	
	export class TutorialDocument {
	    tutorialID: Fact_uint32_;
	    title: Fact_string_;
	
	    static createFrom(source: any = {}) {
	        return new TutorialDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tutorialID = this.convertValues(source["tutorialID"], Fact_uint32_);
	        this.title = this.convertValues(source["title"], Fact_string_);
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
	export class SummoningPoolDocument {
	    name: Fact_string_;
	    regionLabel: Fact_string_;
	    activationEventFlagID: Fact_uint32_;
	
	    static createFrom(source: any = {}) {
	        return new SummoningPoolDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = this.convertValues(source["name"], Fact_string_);
	        this.regionLabel = this.convertValues(source["regionLabel"], Fact_string_);
	        this.activationEventFlagID = this.convertValues(source["activationEventFlagID"], Fact_uint32_);
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
	export class Resource {
	    key: string;
	    kind: string;
	    item?: ItemDocument;
	    colosseum?: ColosseumDocument;
	    region?: RegionDocument;
	    summoningPool?: SummoningPoolDocument;
	    grace?: GraceDocument;
	    boss?: BossDocument;
	    mapRegion?: MapRegionDocument;
	    tutorial?: TutorialDocument;
	    quest?: QuestDocument;
	    class?: ClassDocument;
	
	    static createFrom(source: any = {}) {
	        return new Resource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.kind = source["kind"];
	        this.item = this.convertValues(source["item"], ItemDocument);
	        this.colosseum = this.convertValues(source["colosseum"], ColosseumDocument);
	        this.region = this.convertValues(source["region"], RegionDocument);
	        this.summoningPool = this.convertValues(source["summoningPool"], SummoningPoolDocument);
	        this.grace = this.convertValues(source["grace"], GraceDocument);
	        this.boss = this.convertValues(source["boss"], BossDocument);
	        this.mapRegion = this.convertValues(source["mapRegion"], MapRegionDocument);
	        this.tutorial = this.convertValues(source["tutorial"], TutorialDocument);
	        this.quest = this.convertValues(source["quest"], QuestDocument);
	        this.class = this.convertValues(source["class"], ClassDocument);
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
	export class ResourceRef {
	    kind: string;
	    key: string;
	
	    static createFrom(source: any = {}) {
	        return new ResourceRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.key = source["key"];
	    }
	}
	
	
	
	
	
	
	
	
	

}

