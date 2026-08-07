package migration

type weaponStatsSeed struct {
	ItemID          uint32
	WepType         uint16
	SortGroupID     uint8
	ReinforceTypeID int32
	GemMountType    uint8
	Weight          float64

	AttackPhysical  int32
	AttackMagic     int32
	AttackFire      int32
	AttackLightning int32
	AttackHoly      int32
	AttackStamina   int32

	GuardPhysical  int32
	GuardMagic     int32
	GuardFire      int32
	GuardLightning int32
	GuardHoly      int32
	GuardBoost     int32

	StatReqStr int32
	StatReqDex int32
	StatReqInt int32
	StatReqFai int32
	StatReqArc int32

	Critical      int32
	ScalingStrRaw int32
	ScalingDexRaw int32
	ScalingIntRaw int32
	ScalingFaiRaw int32
	ScalingArcRaw int32

	PassiveEffects []passiveEffectSeed
	DefaultAoWID   int32
	IsInfusable    bool
	IsSomber       bool
	MaxUpgrade     int32
	SourceRowID    uint32
	Warnings       []string
}

type passiveEffectSeed struct {
	Kind       string
	Source     string
	SpEffectID int32
	Label      string
	Value      int32
	Known      bool
}
