# TODO

The current SaveForge 2.0 write path is being reimplemented from the behaviour
shared by SaveForge 1.5.8 and 1.6.8. The following native validation and product
decisions are deliberately deferred and do not block that implementation:

- Validate a PC quantity change through `WriteSave`, a fresh reload and a game
  cold start.
- Validate empty and inactive PC slots, including whether a checksum that starts
  as sixteen zero bytes must remain the native empty-slot marker or may be
  recalculated as both legacy versions did.
- Reconcile `docs/sl2-binary-format-spec.md`, which describes an MD5 prefix for
  PC UserData11, with both legacy writers and the current 2.0 container layout,
  which store UserData11 without that prefix.
- Validate a PS4 quantity change through `WriteSave`, a fresh reload and a game
  start when console testing is resumed.
- Confirm on a game-accepted output that `CSPlayerGameDataHash` remains
  untouched.
- Extend the in-memory WriteSave reload from the currently mutable
  Inventory/Storage/GaItem surface to the remaining 1.5.8/1.6.8 write-ahead
  checks, including the full offset chain and stat sanity, when SaveForge 2.0
  has shared readers for those invariants.
- Test encrypted PC output after `LoadSave` gains AES support.
- Decide and test automatic backup and retention for an existing write target.
- Test atomic replacement of an existing target on Windows.
- Test two sessions writing to the same target.
- Test detection of an externally changed target if that protection is added.
