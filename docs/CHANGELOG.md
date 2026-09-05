# Changelog

Changes to SaveForge 2.0 are recorded here. The 1.x changelog remains available
in the corresponding Git tags.

## [2.0.0]

Status: Unreleased. These notes describe the implemented 2.0 rewrite, not a
published or fully validated release. Outstanding verification is listed below.

### Application and editing workspaces

- Rebuilt the desktop application around Wails 2, React and TypeScript, with a
  shared application shell, English and Polish translations, and the Home,
  Character, Items, Equipment, World, Advanced and Tools workspaces.
- Implemented character profile and progression editing, appearance presets,
  and favorite presets through the new backend contracts.
- Integrated Inventory, Storage and the Item Database with item names and
  icons, grid/table views, quantity editing, batch operations, favorites and
  sorting. Available operations follow backend capabilities and Safety Profiles.
- Integrated equipment presentation and slot-compatible candidates with
  backend-validated equipment mutations.
- Implemented World progress views and supported unlock/lock operations, plus
  Network Tuning controls and presets in Advanced.

### Save sessions and change review

- Introduced shared mutation receipts, revision checks, structured errors and
  session-change notifications across the frontend/backend boundary.
- Added Review Changes, operation history, Undo/Redo, selective reversion,
  risk presentation and validation before saving.
- Implemented Save and Save As with backups before overwriting existing targets,
  backup retention, Recent Files and crash-recovery journals.
- Added configurable backup naming with backend validation and preview.

### Tools and deployment

- Added Settings, build-template management, Deployment, Save Manager and
  About & Updates, including explicit update checks and approved project links.
- Integrated save validation and supported repair plans into Tools.
- Added local and SSH/SFTP deployment targets, file transfer, target backup
  management, host-key trust confirmation and configurable game-status commands.
  Deployment reports blocked operations and uncertain replacement outcomes.
- Added instance-wide Debug Mode, disabled at each launch, and a filtered
  diagnostic console that reads incrementally while expanded.
- Added bounded local JSONL diagnostic logs and diagnostic report export with
  restricted event fields. Debug Mode does not modify save revisions or history.

### Architecture and build configuration

- Replaced the 1.x architecture and public backend API with the 2.0 GameCatalog,
  SaveEngine and endpoint layers. This is a breaking API change, not an
  API-compatible update to 1.x.
- Added generated Wails bindings, endpoint documentation, the local Scalar API
  explorer and a read-only GameCatalog viewer.
- Adapted build and packaging configuration for macOS ARM64, Windows AMD64 and
  Linux AMD64, using pnpm with a frozen lockfile and the Makefile release version.
  Configured packages are a macOS app ZIP, a Windows executable ZIP and a Linux
  AppImage ZIP; configuration alone does not establish platform compatibility.

### Outstanding verification and limitations

- The adapted workflow and its packaged artifacts have not yet been verified on
  the target runners. Windows/Linux desktop compatibility is not confirmed.
- Native Wails UI verification of Debug Mode and the diagnostic console remains
  pending.
- Additional native PC/PS4 save validation and the final third-party code, data
  and icon attribution audit remain open. These notes do not certify all save
  formats or in-game behavior.
- Super marchant remains a placeholder; character fields without a confirmed
  backend contract remain deferred.
