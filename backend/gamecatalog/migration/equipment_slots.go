package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

type weaponEquipmentFlags struct {
	right bool
	left  bool
	both  bool
	arrow bool
	bolt  bool
}

func readWeaponEquipmentFlags(row ParameterRow) (weaponEquipmentFlags, error) {
	right, err := regulationBool(row, "rightHandEquipable")
	if err != nil {
		return weaponEquipmentFlags{}, err
	}
	left, err := regulationBool(row, "leftHandEquipable")
	if err != nil {
		return weaponEquipmentFlags{}, err
	}
	both, err := regulationBool(row, "bothHandEquipable")
	if err != nil {
		return weaponEquipmentFlags{}, err
	}
	arrow, err := regulationBool(row, "arrowSlotEquipable")
	if err != nil {
		return weaponEquipmentFlags{}, err
	}
	bolt, err := regulationBool(row, "boltSlotEquipable")
	if err != nil {
		return weaponEquipmentFlags{}, err
	}
	return weaponEquipmentFlags{
		right: right,
		left:  left,
		both:  both,
		arrow: arrow,
		bolt:  bolt,
	}, nil
}

func (flags weaponEquipmentFlags) slots() []schema.EquipmentSlot {
	slots := make([]schema.EquipmentSlot, 0, 4)
	if flags.left {
		slots = append(slots, schema.EquipmentSlotLeftHand)
	}
	if flags.right {
		slots = append(slots, schema.EquipmentSlotRightHand)
	}
	if flags.arrow {
		slots = append(slots, schema.EquipmentSlotArrow)
	}
	if flags.bolt {
		slots = append(slots, schema.EquipmentSlotBolt)
	}
	return slots
}

type armorEquipmentFlags struct {
	head bool
	body bool
	arms bool
	legs bool
}

func readArmorEquipmentFlags(row ParameterRow) (armorEquipmentFlags, error) {
	head, err := regulationBool(row, "headEquip")
	if err != nil {
		return armorEquipmentFlags{}, err
	}
	body, err := regulationBool(row, "bodyEquip")
	if err != nil {
		return armorEquipmentFlags{}, err
	}
	arms, err := regulationBool(row, "armEquip")
	if err != nil {
		return armorEquipmentFlags{}, err
	}
	legs, err := regulationBool(row, "legEquip")
	if err != nil {
		return armorEquipmentFlags{}, err
	}
	return armorEquipmentFlags{
		head: head,
		body: body,
		arms: arms,
		legs: legs,
	}, nil
}

func (flags armorEquipmentFlags) slots() []schema.EquipmentSlot {
	slots := make([]schema.EquipmentSlot, 0, 4)
	if flags.head {
		slots = append(slots, schema.EquipmentSlotHead)
	}
	if flags.body {
		slots = append(slots, schema.EquipmentSlotChest)
	}
	if flags.arms {
		slots = append(slots, schema.EquipmentSlotArms)
	}
	if flags.legs {
		slots = append(slots, schema.EquipmentSlotLegs)
	}
	return slots
}
