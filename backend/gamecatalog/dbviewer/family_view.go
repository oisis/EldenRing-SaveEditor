package dbviewer

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (server *Server) familyFacts(item *schema.ItemDocument) []factView {
	if item == nil {
		return nil
	}

	var familyData any
	switch item.Family.Value {
	case schema.ItemFamilyWeapon:
		familyData = item.Weapon
	case schema.ItemFamilyArmor:
		familyData = item.Armor
	case schema.ItemFamilyTalisman:
		familyData = item.Talisman
	case schema.ItemFamilyAshOfWar:
		familyData = item.AshOfWar
	case schema.ItemFamilySpell:
		familyData = item.Spell
	case schema.ItemFamilySpiritAsh:
		familyData = item.SpiritAsh
	case schema.ItemFamilyGoods:
		familyData = item.Goods
	case schema.ItemFamilyGesture:
		familyData = item.Gesture
	default:
		return nil
	}
	return server.readableFacts(familyData, "")
}

func (server *Server) variantFamilyFacts(
	family schema.ItemFamily,
	data schema.VariantDocumentData,
) []factView {
	switch family {
	case schema.ItemFamilyWeapon:
		return server.readableFacts(data.Weapon, "")
	case schema.ItemFamilySpiritAsh:
		return server.readableFacts(data.SpiritAsh, "")
	default:
		return nil
	}
}

func (server *Server) readableFacts(value any, prefix string) []factView {
	return server.attachFactSources(server.factViews(reflect.ValueOf(value), prefix, false))
}

func (server *Server) metadataFacts(
	value any,
	prefix string,
	family schema.ItemFamily,
) []factView {
	return server.attachFactSources(server.factViews(
		reflect.ValueOf(value),
		prefix,
		family == schema.ItemFamilySpiritAsh,
	))
}

func (server *Server) factViews(
	value reflect.Value,
	prefix string,
	allowNotApplicable bool,
) []factView {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	if fact, ok := reflectedFact(value, prefix, allowNotApplicable); ok {
		return []factView{fact}
	}

	var facts []factView
	switch value.Kind() {
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			fieldType := value.Type().Field(index)
			if fieldType.PkgPath != "" {
				continue
			}
			label := joinFactLabel(prefix, fieldLabel(fieldType))
			facts = append(facts, server.factViews(value.Field(index), label, allowNotApplicable)...)
		}
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			label := fmt.Sprintf("%s %d", prefix, index+1)
			facts = append(facts, server.factViews(value.Index(index), label, allowNotApplicable)...)
		}
	}
	return facts
}

func reflectedFact(
	value reflect.Value,
	label string,
	allowNotApplicable bool,
) (factView, bool) {
	if value.Kind() != reflect.Struct {
		return factView{}, false
	}
	known := value.FieldByName("Known")
	rawValue := value.FieldByName("Value")
	provenance := value.FieldByName("Provenance")
	if !known.IsValid() || known.Kind() != reflect.Bool ||
		!rawValue.IsValid() || !rawValue.CanInterface() ||
		!provenance.IsValid() || !provenance.CanInterface() {
		return factView{}, false
	}
	source, ok := provenance.Interface().(schema.Provenance)
	if !ok {
		return factView{}, false
	}
	displayValue := fmt.Sprint(rawValue.Interface())
	if label == "Compatibility mask" && rawValue.Kind() == reflect.Uint64 {
		displayValue = fmt.Sprintf("0x%X", rawValue.Uint())
	}
	return factView{
		Label:         label,
		Value:         displayValue,
		Known:         known.Bool(),
		NotApplicable: allowNotApplicable && !known.Bool() && source.MarksNotApplicable(),
		Source:        source.Source,
		Method:        source.Method,
	}, true
}

func (server *Server) attachFactSources(facts []factView) []factView {
	for index := range facts {
		facts[index].SourceLocation = server.sources[facts[index].Source].Location
		if facts[index].NotApplicable {
			facts[index].Value = "N/A"
		} else if !facts[index].Known {
			facts[index].Value = "Unknown"
		}
	}
	return facts
}

func fieldLabel(field reflect.StructField) string {
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "" || name == "-" {
		name = field.Name
	}
	if name == "attackPhysical" {
		return "Physical attack"
	}
	if strings.HasSuffix(name, "-sfv") {
		return humanizeIdentifier(
			strings.TrimSuffix(name, "-sfv"),
		) + " — SaveForge value"
	}
	return humanizeIdentifier(name)
}

func joinFactLabel(prefix string, label string) string {
	if prefix == "" {
		return label
	}
	return prefix + " / " + label
}

func humanizeIdentifier(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	var parts []string
	start := 0
	for index := 1; index < len(runes); index++ {
		current := runes[index]
		previous := runes[index-1]
		nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
		if unicode.IsUpper(current) && (unicode.IsLower(previous) || nextIsLower) {
			parts = append(parts, string(runes[start:index]))
			start = index
		}
	}
	parts = append(parts, string(runes[start:]))
	for index, part := range parts {
		if part != strings.ToUpper(part) {
			parts[index] = strings.ToLower(part)
		}
	}
	first := []rune(parts[0])
	if len(first) > 0 {
		first[0] = unicode.ToUpper(first[0])
		parts[0] = string(first)
	}
	return strings.Join(parts, " ")
}

type variantView struct {
	GameID       string
	GameIDPath   string
	Name         string
	IconURL      string
	Kind         string
	Affinity     string
	UpgradeLevel string
	SourceRowID  string
}

func (server *Server) variantViews(item *schema.ItemDocument) []variantView {
	variants := make([]variantView, 0, len(item.Variants))
	for _, variant := range item.Variants {
		gameID := formatGameID(variant.GameID.Value)
		variants = append(variants, variantView{
			GameID:       gameID,
			GameIDPath:   strings.TrimPrefix(gameID, "0x"),
			Name:         variantDisplayName(item.Presentation.CanonicalName.Value, variant),
			IconURL:      variantIconURL(item, variant),
			Kind:         knownText(variant.Kind.Known, string(variant.Kind.Value)),
			Affinity:     variantAffinity(item.Family.Value, variant),
			UpgradeLevel: knownUpgradeLevel(variant.UpgradeLevel),
			SourceRowID:  knownNumber(variant.SourceRowID.Known, variant.SourceRowID.Value),
		})
	}
	return variants
}

func variantAffinity(family schema.ItemFamily, variant schema.ItemVariant) string {
	if family == schema.ItemFamilySpiritAsh &&
		!variant.Affinity.Known && variant.Affinity.Provenance.MarksNotApplicable() {
		return "N/A"
	}
	return knownText(variant.Affinity.Known, string(variant.Affinity.Value))
}

func knownUpgradeLevel(level schema.Fact[uint8]) string {
	if !level.Known {
		return "Unknown"
	}
	return fmt.Sprintf("+%d", level.Value)
}

func knownNumber(known bool, value any) string {
	if !known {
		return "Unknown"
	}
	return fmt.Sprint(value)
}
