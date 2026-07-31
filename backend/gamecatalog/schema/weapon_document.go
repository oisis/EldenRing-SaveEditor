package schema

type WeaponData struct {
	SourceRowID            Fact[uint32]              `json:"sourceRowID"`
	IconID                 Fact[uint32]              `json:"iconID"`
	WeaponTypeID           Fact[uint16]              `json:"weaponTypeID"`
	SortID                 Fact[uint32]              `json:"sortID"`
	SortGroupID            Fact[uint8]               `json:"sortGroupID"`
	ReinforceTypeID        Fact[int32]               `json:"reinforceTypeID"`
	GemMountType           Fact[uint8]               `json:"gemMountType"`
	Weight                 Fact[float64]             `json:"weight"`
	AttackPhysical         Fact[int32]               `json:"attackPhysical"`
	AttackMagic            Fact[int32]               `json:"attackMagic"`
	AttackFire             Fact[int32]               `json:"attackFire"`
	AttackLightning        Fact[int32]               `json:"attackLightning"`
	AttackHoly             Fact[int32]               `json:"attackHoly"`
	AttackStamina          Fact[int32]               `json:"attackStamina"`
	GuardPhysical          Fact[int32]               `json:"guardPhysical"`
	GuardMagic             Fact[int32]               `json:"guardMagic"`
	GuardFire              Fact[int32]               `json:"guardFire"`
	GuardLightning         Fact[int32]               `json:"guardLightning"`
	GuardHoly              Fact[int32]               `json:"guardHoly"`
	GuardBoost             Fact[int32]               `json:"guardBoost"`
	RequiredStrength       Fact[int32]               `json:"requiredStrength"`
	RequiredDexterity      Fact[int32]               `json:"requiredDexterity"`
	RequiredIntelligence   Fact[int32]               `json:"requiredIntelligence"`
	RequiredFaith          Fact[int32]               `json:"requiredFaith"`
	RequiredArcane         Fact[int32]               `json:"requiredArcane"`
	ScalingStrengthRaw     Fact[float64]             `json:"scalingStrengthRaw"`
	ScalingDexterityRaw    Fact[float64]             `json:"scalingDexterityRaw"`
	ScalingIntelligenceRaw Fact[float64]             `json:"scalingIntelligenceRaw"`
	ScalingFaithRaw        Fact[float64]             `json:"scalingFaithRaw"`
	ScalingArcaneRaw       Fact[float64]             `json:"scalingArcaneRaw"`
	Critical               Fact[int32]               `json:"critical"`
	StatusPoison           Fact[int32]               `json:"statusPoison"`
	StatusBleed            Fact[int32]               `json:"statusBleed"`
	StatusFrost            Fact[int32]               `json:"statusFrost"`
	StatusSleep            Fact[int32]               `json:"statusSleep"`
	StatusMadness          Fact[int32]               `json:"statusMadness"`
	StatusScarletRot       Fact[int32]               `json:"statusScarletRot"`
	PassiveEffects         []WeaponPassiveEffectData `json:"passiveEffects"`
	DefaultAshOfWarID      Fact[int32]               `json:"defaultAshOfWarID"`
	SwordArtsName          Fact[string]              `json:"swordArtsName"`
	IsInfusable            Fact[bool]                `json:"isInfusable"`
	IsSomber               Fact[bool]                `json:"isSomber"`
	MaxUpgrade             Fact[int32]               `json:"maxUpgrade"`
	Warnings               Fact[[]string]            `json:"warnings"`
	RightHandEquipable     Fact[bool]                `json:"rightHandEquipable"`
	LeftHandEquipable      Fact[bool]                `json:"leftHandEquipable"`
	BothHandEquipable      Fact[bool]                `json:"bothHandEquipable"`
	ArrowSlotEquipable     Fact[bool]                `json:"arrowSlotEquipable"`
	BoltSlotEquipable      Fact[bool]                `json:"boltSlotEquipable"`
}

type WeaponPassiveEffectData struct {
	Kind       Fact[string] `json:"kind"`
	Source     Fact[string] `json:"source"`
	SpEffectID Fact[int32]  `json:"spEffectID"`
	Label      Fact[string] `json:"label"`
	Value      Fact[int32]  `json:"value"`
	Known      Fact[bool]   `json:"known"`
}
