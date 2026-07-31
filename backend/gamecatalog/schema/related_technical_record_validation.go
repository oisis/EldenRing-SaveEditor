package schema

import "fmt"

func validateRelatedTechnicalRecords(
	name string,
	records []RelatedTechnicalRecord,
	ownerGameID uint32,
	sources map[SourceID]struct{},
) error {
	seen := make(map[uint32]struct{}, len(records))
	for index, record := range records {
		recordName := fmt.Sprintf("%s[%d]", name, index)
		if err := validateFact(recordName+".kind", record.Kind, sources); err != nil {
			return err
		}
		if !record.Kind.Known || record.Kind.Value != TechnicalRecordAppearanceState {
			return fmt.Errorf("%s.kind must be known and supported", recordName)
		}
		if err := validateFact(recordName+".gameID", record.GameID, sources); err != nil {
			return err
		}
		if !record.GameID.Known || record.GameID.Value == 0 {
			return fmt.Errorf("%s.gameID must be known and greater than zero", recordName)
		}
		if ownerGameID != 0 && record.GameID.Value == ownerGameID {
			return fmt.Errorf("%s cannot use the owning item game ID", recordName)
		}
		if _, exists := seen[record.GameID.Value]; exists {
			return fmt.Errorf("%s: duplicate technical game ID 0x%08X", recordName, record.GameID.Value)
		}
		seen[record.GameID.Value] = struct{}{}
		if err := validateDescriptionRecord(record.Description, sources); err != nil {
			return fmt.Errorf("%s.description: %w", recordName, err)
		}
		if err := validateFact(recordName+".gameMaxInventory", record.GameMaxInventory, sources); err != nil {
			return err
		}
		if err := validateFact(recordName+".gameMaxStorage", record.GameMaxStorage, sources); err != nil {
			return err
		}
		if len(record.SourceRecords) == 0 {
			return fmt.Errorf("%s.sourceRecords must not be empty", recordName)
		}
		if err := validateParameterRecords(recordName+".sourceRecords", record.SourceRecords, sources); err != nil {
			return err
		}
		subject := record
		subject.SourceRecords = nil
		if err := validateRegulationProvenanceCoverage(
			recordName,
			subject,
			record.SourceRecords,
		); err != nil {
			return err
		}
	}
	return nil
}
