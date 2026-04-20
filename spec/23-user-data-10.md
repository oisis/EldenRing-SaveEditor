# 23 — UserData10 (Profil Konta)

> **Zakres**: Sekcja wspólna dla wszystkich slotów — ProfileSummary, SteamID, active slots.

---

## Opis ogólny

UserData10 to sekcja po 10 slotach postaci. Zawiera:
- Informacje o koncie (Steam ID)
- Podsumowania 10 postaci (ProfileSummary) — wyświetlane w menu wyboru postaci
- Flagi aktywnych slotów

Rozmiar: 0x60000 bajtów (393,216 bytes).

Na PC: poprzedzone 16-bajtowym MD5 checksumem (jak sloty postaci).

---

## Layout

```
┌────────────────────────────────────────┐
│ [PC only] MD5 Checksum (16 bytes)       │
├────────────────────────────────────────┤
│ Steam ID (u64) — 8 bytes                │
├────────────────────────────────────────┤
│ ... (padding / unknown)                 │
├────────────────────────────────────────┤
│ Active Slots bitfield                   │  → offset zależy od platformy
├────────────────────────────────────────┤
│ ProfileSummary[0]                       │  0x100 bytes (256 bytes)
│ ProfileSummary[1]                       │
│ ... × 10                                │  → offset zależy od platformy
├────────────────────────────────────────┤
│ CSMenuSystemSaveLoad                    │  0x60000 bytes (menu system data)
└────────────────────────────────────────┘
```

---

## Offsety platform-specyficzne

| Pole | PC | PS4 |
|---|---|---|
| Active Slots | 0x1C (od początku UserData10) | 0x300 |
| ProfileSummary start | 0x26 | 0x30A |

---

## ProfileSummary (0x100 = 256 bytes per slot)

Podsumowanie postaci widoczne w menu wyboru:

| Offset | Typ | Opis |
|---|---|---|
| 0x00 | u16[16] | Character Name (UTF-16LE) |
| 0x20 | u32 | Level |
| ... | ... | (pozostałe pola do zbadania) |
| ... | u8[0x120] | Face Data (skrócone, 288 bytes) |

ProfileSummary MUSI być zsynchronizowane z danymi w slocie — inaczej menu pokazuje złe informacje.

---

## Active Slots

Bitfield lub tablica flag — wskazuje które sloty (0-9) mają aktywne postacie.

---

## CSMenuSystemSaveLoad (0x60000 bytes)

Duży blok danych systemu menu — ustawienia HUD, preferencje wyświetlania, quickslot konfiguracja na poziomie konta.

---

## Implikacje dla edycji

- **Steam ID**: musi odpowiadać Steam ID gracza na PC — inaczej save nie załaduje się
- **ProfileSummary**: po edycji imienia/levelu w slocie TRZEBA zaktualizować też tutaj
- **Active Slots**: po dodaniu/usunięciu postaci trzeba zaktualizować
- **MD5**: po modyfikacji UserData10 na PC — przeliczyć checksum
- **Konwersja platform**: offsety Active Slots i ProfileSummary są RÓŻNE — błędny offset = uszkodzony save

---

## Źródła

- er-save-manager: `parser/user_data_10.py` — klasa `UserData10`
- er-save-manager: `parser/save.py` linie 209-228
- Steam Guide: https://steamcommunity.com/sharedfiles/filedetails/?id=2797241037
