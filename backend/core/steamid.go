package core

// SteamID flushing is handled by SaveFile.flushMetadata() in save_manager.go.
// The authoritative SteamID is stored in UserData10 (offset 0x04 after checksum).
// Per-slot SteamID exists at a dynamic offset within the sequential parsing chain
// (after BaseVersion, before PS5Activity) — NOT at SlotSize-8 (which is the hash region).
