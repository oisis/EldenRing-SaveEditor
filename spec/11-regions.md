# 11 — Regions (Odblokowane Regiony)

> **Zakres**: Lista odblokowanych regionów świata — SEKCJA ZMIENNEJ DŁUGOŚCI.

---

## Opis ogólny

Regions to lista ID regionów, które postać odblokowała (odwiedziła). Wpływa na fast travel i wyświetlanie mapy.

**UWAGA**: Sekcja ma zmienną długość! Rozmiar zależy od postępu gracza w eksploracji.

---

## Struktura

```
┌─────────────────────────────────┐
│ Count (u32)                      │  4 bytes
├─────────────────────────────────┤
│ Region IDs: count × u32          │  count × 4 bytes
└─────────────────────────────────┘
Total: 4 + count × 4 bytes
```

---

## Region IDs

Region ID to u32 identyfikujący obszar w grze. Przykłady wymagają weryfikacji z Cheat Engine table / modding tools.

---

## Implikacje dla edycji

- **Zmiana count przesuwa WSZYSTKIE kolejne sekcje** (Torrent, Blood Stain, Event Flags, etc.)
- Dodanie regionu: zwiększ count + dopisz nowy ID na końcu listy
- Usunięcie regionu: zmniejsz count + kompaktuj listę + przesuń resztę
- Duplikaty: nieznane czy są dozwolone — bezpieczniej unikać

---

## Źródła

- er-save-manager: `parser/world.py` — klasa `Regions` (linie 92-117)
- er-save-manager: `parser/user_data_x.py` linia 127: `unlocked_regions: Regions`
- ER-Save-Editor (Rust): `src/save/common/save_slot.rs` — UnknownList z length prefix
