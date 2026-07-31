/*
Endpoint: GetEquippedSpells
EndpointID: get_equipped_spells
Purpose: Zwraca zaklęcia w slotach pamięci oraz wykorzystaną i dostępną pojemność Memory Slots.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument: Spell.
Input variables: characterID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetEquippedSpellsEndpointID is the stable backend identifier of GetEquippedSpells.
const GetEquippedSpellsEndpointID = "get_equipped_spells"

// GetEquippedSpellsDefinition describes the public getter contract.
var GetEquippedSpellsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetEquippedSpells",
	ID:                         GetEquippedSpellsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Spell",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca zaklęcia w slotach pamięci oraz wykorzystaną i dostępną pojemność Memory Slots.",
})
