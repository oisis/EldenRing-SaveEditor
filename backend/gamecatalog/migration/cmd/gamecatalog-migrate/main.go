package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/migration"
)

func main() {
	var regulationDirectory string
	var regulationParameterDirectory string
	var gameTextDirectory string
	var legacyIconDirectory string
	var outputDirectory string
	var gameVersion string
	var replace bool

	flag.StringVar(&regulationDirectory, "regulation-csv", "", "directory containing the extracted regulation CSV files")
	flag.StringVar(&regulationParameterDirectory, "regulation-params", "", "directory containing extracted raw regulation parameter files")
	flag.StringVar(&gameTextDirectory, "game-text", "", "directory containing the extracted English FMG files and JSON")
	flag.StringVar(&legacyIconDirectory, "legacy-icons", "frontend/public", "legacy public asset root")
	flag.StringVar(&outputDirectory, "output", "", "GameCatalog output directory")
	flag.StringVar(&gameVersion, "game-version", "unidentified-regulation-build", "game version label for the supplied regulation dump")
	flag.BoolVar(&replace, "replace", false, "replace the managed catalog.json, items, and item-icons output")
	flag.Parse()

	if err := run(
		regulationDirectory,
		regulationParameterDirectory,
		gameTextDirectory,
		legacyIconDirectory,
		outputDirectory,
		gameVersion,
		replace,
	); err != nil {
		fmt.Fprintln(os.Stderr, "gamecatalog-migrate:", err)
		os.Exit(1)
	}
}

func run(
	regulationDirectory string,
	regulationParameterDirectory string,
	gameTextDirectory string,
	legacyIconDirectory string,
	outputDirectory string,
	gameVersion string,
	replace bool,
) error {
	regulation, err := migration.ReadRegulationCSVDirectory(regulationDirectory)
	if err != nil {
		return err
	}
	regulationParams, err := migration.ReadRegulationParameterDirectory(
		regulationParameterDirectory,
	)
	if err != nil {
		return err
	}
	gameText, err := migration.ReadGameTextDirectory(gameTextDirectory)
	if err != nil {
		return err
	}
	catalog, err := migration.Generate(migration.GenerateOptions{
		Regulation:          regulation,
		RegulationParams:    regulationParams,
		GameText:            gameText,
		LegacyIconDirectory: legacyIconDirectory,
		GameVersion:         gameVersion,
	})
	if err != nil {
		return err
	}
	if err := migration.WriteCatalog(catalog, migration.WriteOptions{
		OutputDirectory:     outputDirectory,
		LegacyIconDirectory: legacyIconDirectory,
		Replace:             replace,
	}); err != nil {
		return err
	}
	fmt.Printf(
		"generated %d documents, %d shared icon files, data version %s\n",
		len(catalog.Resources),
		len(catalog.IconSources),
		catalog.Manifest.DataVersion,
	)
	return nil
}
