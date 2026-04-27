# 35 — EAC & Server Validation (Mechanizm detekcji edycji save'a)

> **Zakres**: Co Easy Anti-Cheat (EAC) i serwery FromSoftware/Bandai Namco realnie sprawdzają przy synchronizacji online Elden Ring. Lista znanych wektorów detekcji + pre-flight checklist który edytor mógłby/powinien egzekwować przed dopuszczeniem save'a do trybu online.

> **Status**: ⚠️ **Mieszany** — większość twierdzeń to community-derived reverse engineering, nie oficjalna dokumentacja. Każdy claim oznaczony tagiem confidence (`[OFFICIAL]` / `[REVERSE-ENGINEERED]` / `[COMMUNITY]` / `[SPECULATION]`).

> **Cel praktyczny**: dostarczyć autorowi UI / backendu listę checków deterministycznych (możliwych do zaimplementowania bez dostępu do serwera) — komplementarnych do istniejącego `spec/32-ban-risk-system.md`. §32 mówi *jak* edukować użytkownika; ten plik mówi *co* warto zwalidować przed wgraniem save'a.

---

## Filozofia dokumentu

FromSoftware **nigdy nie opublikował** listy kryteriów detekcji. Wszystko poniżej jest złożeniem:
- oficjalnych komunikatów Bandai Namco i Epic Games (etykietowane `[OFFICIAL]`),
- reverse-engineeringu z otwartoźródłowych edytorów (`[REVERSE-ENGINEERED]`),
- raportów community z r/Eldenring, Steam Discussions, ResetEra, FearLess Cheat Engine forum (`[COMMUNITY]`),
- hipotez modderskich bez potwierdzenia (`[SPECULATION]`).

**Zasada pisania UI tekstów ostrzeżeń** (kontynuacja §32): nie twierdzimy oficjalnie że "FromSoftware zbanuje za X". Używamy *"community-reported"*, *"reported bans on r/Eldenring"*, *"detection rules whose exact mechanism is not publicly documented"*.

---

## Confidence Classification System (Severity × Confidence)

Dwa ortogonalne wymiary opisują każdą regułę detekcji:

1. **Severity** — jak groźna konsekwencja jeśli check się odpali (Tier 0/1/2 z `spec/32-ban-risk-system.md`).
2. **Confidence** — jak pewni jesteśmy że check w ogóle istnieje. **Trzystopniowy umbrella tier** dla UI; szczegółowe source types (`OFFICIAL` / `RE` / `COMMUNITY` / `SPECULATION`) zachowane w prozie tego dokumentu.

### Confidence tiery

| Tier | Nazwa | Definicja | Mapowanie z source types | UI kolor |
|---|---|---|---|---|
| **C** | **Confirmed** | Vendor-published lub niezależnie reverse-engineered z binarki/protokołu — technicznie weryfikowalne | `[OFFICIAL]`, `[REVERSE-ENGINEERED]` | czerwony |
| **R** | **Reported** | Wiele niezależnych community sources, spójny pattern, brak technical proof | `[COMMUNITY]`, `[COMMUNITY-strong]`, `[COMMUNITY-weak]`, `[COMMUNITY-fuzzy]` | pomarańczowy |
| **S** | **Speculated** | Single source / theoretical model / heuristic / brak public reference | `[SPECULATION]`, `[unverified]`, `[no public data]` | szary |

**Dlaczego NIE "official/unofficial/community":**
- "Official" sugerowałoby że FromSoft potwierdził regułę. **Żaden ban check w ER nie jest officially confirmed.** Czytelnik zobaczyłby "official" jako gold standard — mylące.
- "Reverse-engineered" daje techniczną pewność bez vendor fiat (np. cheater pool był RE z DS3, nie z ER docs). To zasługuje na **Confirmed**, nie na osobną "unofficial".

### Master classification table

Każda znana reguła z konfidencją + severity. Aktualizować przy nowych raportach.

| Reguła | Severity | Confidence | Notatki |
|---|---|---|---|
| Cut content item ID w inventory (`pavel`, debug items) | Tier 2 | **Confirmed** (RE — paramdex, Malcolm Reynolds 2022 case) | Najczystszy sygnał. Receiver flagged jeśli pickup z drop. |
| MD5 per-slot recompute required by game loader | n/d (integrity) | **Confirmed** (RE — wszystkie edytory robią to identycznie) | Bez tego save się nie ładuje. |
| Steam Family Sharing nie pokrywa EAC games | n/d (testbed) | **Confirmed** (Official — Steam Family Sharing FAQ) | Implication: alt-account z osobnym zakupem to jedyny safe testbed. |
| 180-day suspension istnieje | Tier 2 (consequence) | **Confirmed** (Official — Bandai support) | Reapply przy pierwszym online login po wygaśnięciu jeśli offending data zostały. |
| "Inappropriate activity detected" = kick, nie ban | n/d | **Confirmed** (Official — Bandai support) | Częste false-positives: Linux/Proton, mods, TDP changes, cache mismatch. |
| EAC user-mode w base ER (kernel w Nightreign) | n/d | **Confirmed** (Official) | Implication: edycje przy zamkniętym Steam są niewidoczne dla EAC. |
| Cheater pool / matchmaking segregation | Tier 2 | **Reported** (multi-source ER + RE z DS3) | Czy soft-ban i 180-day to dwa stany czy jeden — `[no public data]`. |
| Runes > 999,999,999 auto-flag | Tier 2 | **Reported** | Konsystentne raporty, brak RE proof |
| Atrybut > 99 ustawiony bezpośrednio (vs level-up) | Tier 2 | **Reported** (multi-source) | Najczystszy community signal: *"directly altering attributes will result in a ban; leveling normally won't"* |
| Level > 713 | Tier 2 | **Reported** | Konsekwencja stat overflow |
| Talisman pouch > 3 sloty | Tier 2 | **Reported** (single-strong) | Listed w starszych edytor warnings, mało recent confirmed cases |
| Quantity > MaxInventory | Tier 2 | **Reported** | Niejednoznacznie — niektórzy banują, niektórzy nie |
| Spirit ash upgrade > +10 | Tier 2 | **Reported** | IDs poza +10 nie istnieją w regulation params |
| Picking up illegal item from invader | Tier 2 | **Reported** (multi-source, Malcolm Reynolds + Bandai response) | Receiver flagged. Realne user-protection concern. |
| Pre-order item bez entitlement | Tier 2 | **Reported** | Server-side entitlement check |
| DLC item bez DLC ownership | Tier 2 | **Reported** | Same mechanism as pre-order |
| EAC checksum encrypted check w startup | n/d | **Reported** (single-source, FearLess thread) | Roboczy model, nie potwierdzone przez FromSoft/Epic |
| Bulk grace unlock | Tier 1 | **Reported** (weak) | Frequent caution, brak clear ban-report cluster |
| Map reveal full | Tier 1 | **Reported** (weak) | Spoiler concern dominuje nad ban concern |
| Quest step skip | Tier 1 | **Reported** (weak) | Większe zagrożenie: psucie questline niż ban |
| Wave ban cadence (nightly/weekly) | n/d | **Speculated** | Folklore — żaden FromSoft document nie podaje cadence |
| 8h delay vs months delay | n/d | **Speculated** | Multiple anecdotes, no signal/timing analysis |
| Stat consistency check by serwer (HP vs VIG) | Tier 2 | **Speculated** | Plauzybilne, zero confirmed reports |
| 7 dni offline po edycji | n/d | **Speculated** (heuristic) | NIE published rule |
| Save backup/restore własnego save = ban | n/d | **Speculated** | Single fuzzy case (Elden Beast rollback months later) |
| Soft-ban i 180-day to dwa różne stany | n/d | **Speculated** | Może być jeden stan z różnymi surface |
| "Mongo" item trigger | n/d | **Speculated** (unverified) | Brak public reference. Discord-internal nickname / typo "Mohg" / nie istnieje |

**Konwencja używania**: w tekście dokumentu zachowujemy szczegółowy source tag w nawiasach kwadratowych (`[OFFICIAL]`, `[REVERSE-ENGINEERED]`, `[COMMUNITY-strong]` itp.) dla precyzji. W UI / `RISK_INFO` dictionary używamy jednolitej trójki **Confirmed / Reported / Speculated**.

### Composite display w UI

```
[Tier 2 · Confirmed]    Cut content item detected: Pavel placeholder talisman
[Tier 2 · Reported]     Runes above 999,999,999 — community-reported as instant flag
[Tier 1 · Reported]     Bulk grace unlock — weak signal but cited in ban discussions
[Tier 1 · Speculated]   HP/FP/SP inconsistent with attributes — theoretical detection
```

**Reguła UI**: nigdy nie ukrywaj confidence. Jeśli issue ma Confidence=Speculated, etykieta MUSI to surface'ować — inaczej dajemy false sense of certainty co do reguły.

---

## 1. Easy Anti-Cheat (EAC) — postura w Elden Ring

### Vendor i tryb działania

| Aspekt | Wartość | Confidence |
|---|---|---|
| Wydawca | Epic Games (przejęte od Kamu w 2018) | `[OFFICIAL]` |
| Tryb działania (base ER 1.0–1.16) | **User-mode** — `EasyAntiCheat.exe` obok `eldenring.exe` | `[OFFICIAL]` |
| Tryb działania (Nightreign, maj 2025) | **Kernel-mode driver** — inny produkt, poza zakresem tego edytora | `[OFFICIAL]` |
| Platforma | Windows tylko; Linux via Proton wymaga toggle Steam → "Enable Easy Anti-Cheat for Linux" | `[OFFICIAL]` |
| Konkurencja | NIE jest BattlEye / Ricochet — częsty mit w starszych Reddit threads | `[OFFICIAL]` |

**Implikacja dla edytora**: bazujemy na user-mode EAC. Edytujemy plik save *poza* działającym procesem gry (Steam zamknięty, gra niewłączona). EAC nie ma wtedy żadnej widoczności na nasze edycje — to dobra wiadomość. Zła: EAC sprawdza save **przy starcie procesu** i **przy login do online service**.

### Co EAC inspekcjonuje (publicznie znane)

- **Skanowanie pamięci procesu** — wzorce typu Cheat Engine, breakpoint detection, modyfikacja addresses znanych itemów/atrybutów. `[OFFICIAL]`
- **DLL injection / module attach detection**. `[OFFICIAL]`
- **Debugger / driver presence**. `[OFFICIAL]`
- **Integrity binarki gry** — sygnatury PE, hash exe/DLL. `[OFFICIAL]`
- **Integrity pliku save** — *prawdopodobny* hand-off checksumów do/z serwera. `[COMMUNITY]`

### Model checksumów save'a (community-derived)

Najszerzej cytowane źródło: thread fearlessrevolution.com t=19320, str. 29. Twierdzenie:

> *"EAC checks the save file for manipulation and a save file internal encrypted checksum during game startup, before any server connection is established. The encrypted checksum gets transferred during the game's login process and compared against the checksum that EAC holds locally in RAM and receives from the server connection procedure. There is no constant datastream used to monitor data."*

Status: `[COMMUNITY]` — **nie potwierdzone** ani przez FromSoft, ani przez Epic. Traktujemy jako roboczy model.

**Co z tego wynika dla edytora**:
1. Per-slot **MD5** (16-byte block na początku każdego USERDATA entry w BND4 container) jest realnie czytany przez gracza i każdy edytor — patrz [`spec/01-header.md`](01-header.md), [`spec/22-player-hash.md`](22-player-hash.md). My recompute'ujemy go w `backend/core/crypto.go` przy `WriteSave()`. **Required by game loader** — bez tego save się nie załaduje.
2. **Open question**: czy EAC ma drugi, *server-anchored* hash (np. liczony z subset pól krytycznych, nie z całego slotu)? Jeśli tak — żaden edytor go nie przejdzie, co tłumaczy dlaczego "save edits sometimes pass for months and then trigger a wave ban".
3. **Brak constant datastream** ⇒ EAC w ER bazowym jest **boot-time + login-time validated**, nie continuous telemetry. Edycje offline z wyłączonym Steam są niewidoczne dla EAC do momentu pierwszego online login.

### Reakcja EAC na anomalię

| Akcja | Skutek | Confidence |
|---|---|---|
| EAC wykryje anomalię w pamięci/binarce | Disconnect z błędem **"Inappropriate activity detected"** | `[OFFICIAL]` (Bandai support) |
| Sam komunikat | **Kick z sesji**, NIE jest banem konta | `[OFFICIAL]` |
| Częste false-positives | Linux/Proton, mod loadery, Cheat Engine running w tle, TDP zmieniony na handheld, cache mismatch po update | `[COMMUNITY-strong]` |
| Wave ban (180 dni) | Wystawiany **przez serwer FromSoft**, nie przez EAC | patrz §2 |

**EAC sam z siebie nie wystawia 180-day banów.** Robi tylko kick + raport. Ban wystawia FromSoft po stronie serwera.

---

## 2. Server-side validation (FromSoftware / Bandai Namco)

### Co serwer dostaje

- `[COMMUNITY]` Przy login do online: gra wysyła **subset save data + checksumy + character summary**. Czy cały slot idzie up — `[no public data]`. Modderska konsensusowa hipoteza: tylko digest + character header + lista item ID + flags krytyczne.
- `[COMMUNITY]` Sesje multiplayer (summon/invasion/coop/PvP) używają **proxy/replicated state** — twój klient kontroluje lokalnego proxy, serwer trzyma kanoniczny snapshot, **delta mismatches są flagowane**. Tłumaczy to dlaczego runtime memory poke w PvP triggeruje raport, a sam save edit offline często przechodzi.

### Znane / domniemane checki serwera

| Check | Confidence | Notatki |
|---|---|---|
| **Item-ID blacklist** (cut content / debug items) | `[COMMUNITY-strong]` | Najmocniejszy sygnał banu. Kanoniczny case study: debug item `"pavel"` rozdawany w 2022 przez Malcolm Reynolds — receiver picking up item dropped by attacker → soft-ban. (Update: Kotaku częściowo zdementowała skalę, ale mechanizm potwierdzony.) |
| **Runy > 999_999_999** | `[COMMUNITY-strong]` | Auto-flag |
| **Atrybut > 99 ustawiony bezpośrednio** (vs. level-up) | `[COMMUNITY-strong]` | Najczystszy sygnał: *"directly altering attributes will result in a ban; leveling normally won't"* (Steam Discussion 4526764179303674425) |
| **Level > 713** | `[COMMUNITY]` | Konsekwencja stat overflow — często towarzyszy stat>99 |
| **Talisman pouch > 3 sloty** | `[COMMUNITY]` | Auto-flag |
| **Quantity > MaxInventory** | `[COMMUNITY]` | Niejednoznacznie raportowane — niektórzy banują, niektórzy nie. Patrz `spec/34-item-caps.md` dla scaling logic |
| **GaItem map handle integrity** | `[REVERSE-ENGINEERED]` | NIE jest server-checked, ale corrupted handle table = crash przy load = "Inappropriate activity" kick. Patrz `spec/03-gaitem-map.md` |
| **MD5 per-slot (PC)** | `[REVERSE-ENGINEERED]` | Game loader refuses bad MD5 — surface error, nie ban |
| **SteamID match w UserData10** | `[REVERSE-ENGINEERED]` | Foreign SteamID nie flaguje as cheating per se (Save Wizard cross-account import jest powszechny). Serwer używa current Steam ticket. Patrz `spec/23-user-data-10.md` |
| **Matchmaking weapon level** (highest-ever upgraded weapon, nawet po discardzie) | `[COMMUNITY-strong]` | NIE ban trigger, ale **matchmaking pool**. Edycja spirit ash +25 / weapon +25 wpłynie na pool zanim postać przekroczy real progression |

### Reakcja serwera

| Stan | Skutek | Confidence |
|---|---|---|
| **180-day suspension** | Account-level block z online play. Po wygaśnięciu — jeśli offending data nadal w save, ban reapply przy pierwszym online login | `[OFFICIAL]` (Bandai support) |
| **"Cheater pool" segregation** (soft-ban) | Account może grać online, ale matched tylko z innymi flagged accounts | `[COMMUNITY-strong]` — multiple Steam/ResetEra threads. Czy soft-ban i 180-day suspension to jedna stan z różnymi surface, czy dwa stany — `[no public data]` |
| **Timing detekcji** | NIE real-time. Cytowane: ~8h, "weekly review pass", "wave bans". Cadence — `[no public data]`. Najbardziej dramatic case: gracz rolluje save po Elden Beast → flag **miesiące później** | `[COMMUNITY]` |

---

## 3. Wektory bana — community case studies

| Wektor | Confidence | Źródło |
|---|---|---|
| **Cut content w inventory** (debug items, items removed before retail) | `[COMMUNITY-strong]` | Kotaku, Dexerto, CBR — Malcolm Reynolds case (2022). Częściowo debunked w Kotaku update — ale mechanizm potwierdzony |
| **Pre-order bonus na koncie bez entitlement** | `[COMMUNITY]` | Steam threads, no Reddit megathread |
| **DLC items na koncie pre-DLC** | `[COMMUNITY]` | Steam threads |
| **Picking up illegal item dropped by another player (invader)** | `[COMMUNITY-strong]` | **Receiver gets flagged** — to jest realne user-protection concern, NIE tylko self-edit |
| **Bulk grace unlock / map reveal / quest skip** | `[COMMUNITY-weak]` | Frequent caution, brak clear ban-report cluster |
| **Stat consistency drift** (HP/FP/SP nie matchuje VIG/MND/END) | `[SPECULATION]` | Plauzybilne (server może recompute), zero confirmed reports |
| **Backup/restore własnego save** (Steam cloud rollback) | `[COMMUNITY-fuzzy]` | Większość OK, ale jeden cytowany case bana po rollbacku z Elden Beast — official-vs-folklore line jest fuzzy |
| **Disabling EAC offline** (Nexus toggle mod 90) | `[COMMUNITY]` | Sam offline state nie flaguje. Login online z EAC off → kick + flagging |

### Co NIE jest (jak wynika z konsensusu) wektorem

- Edycja **gestur unlocked** (jeśli excludujemy ban-risk gestures z `RiskBadge`).
- Edycja **regions unlocked** (jeśli postać ma już dostęp do tych map fragmentów).
- Edycja **NG+ counter** (samodzielnie — łączenie z rune injection już ryzykowne).
- **Map reveal full** — najczęściej cited "OK" edit. Spoiler concern, nie ban concern.
- **PvP exploits** (Eclipse Shotel + Poison Perfume, Stake of Marika one-shot) — *attacker* nie jest banowany w żadnym raportowanym case. To gameplay exploits, nie save edits.

### Note dot. terminu "Mongo"

Jeśli ktoś mówi o "Mongo item" jako trigger banu — **brak public reference**. Albo Discord-internal nickname, albo confusion z "Mohg" (boss, irrelevant), albo nie istnieje. Traktować jako `[unverified]`.

---

## 4. Pre-flight checklist — checki deterministyczne dla edytora

Tier mapuje do `spec/32-ban-risk-system.md`. Wszystkie poniższe są **implementowalne lokalnie** (bez serwera).

### 4A — Numeric / cap checks (Tier 2)

| Check | Reguła | Już zaimplementowane? |
|---|---|---|
| Runes ≤ 999_999_999 | `getRunesRiskKey()` w `frontend/src/data/riskInfo.ts:378` | ✅ |
| Każdy atrybut (VIG, MND, END, STR, DEX, INT, FAI, ARC) ≤ 99 | `getAttributeRiskKey()` | ✅ |
| Level ≤ 713 = Σ(99×8) − 79 | `getLevelRiskKey()` | ✅ |
| Talisman pouch slots ≤ 3 | `getTalismanPouchRiskKey()` | ✅ |
| Per-item quantity ≤ effective cap (`MaxInventory × (ClearCount+1)` dla `scales_with_ng`) | `effectiveCap()` w `DatabaseTab.tsx`; `getQuantityRiskKey()` | ✅ — patrz `spec/34-item-caps.md` |
| Spirit ash upgrade ≤ +10 | `getSpiritAshRiskKey()` | ✅ |
| Weapon upgrade ≤ +25 (smithing) / ≤ +10 (somber) | n/d (nie ma jeszcze Tier 2 outline na to polu) | ❌ TODO |

### 4B — Identity / catalogue checks (Tier 2)

| Check | Reguła | Już zaimplementowane? |
|---|---|---|
| Item-ID whitelist | Każdy GaItem w save musi istnieć w `backend/db/data/`. Unknown ID → flag | ⚠️ **częściowo** — mamy `GetItemDataFuzzy()` ale nie ma audytu pre-write |
| Flags scan: `cut_content`, `pre_order`, `dlc_duplicate`, `ban_risk` | `RiskBadge` inline w UI; modal "Add Anyway" przed dodaniem | ✅ — patrz `spec/32-ban-risk-system.md` |
| DLC ownership consistency | Save z DLC items + `IsDlcOwned == 0` byte → flag | ⚠️ **TODO** — patrz `spec/21-dlc.md` |

### 4C — Internal-consistency checks (Tier 1)

NIE są server-checked, ale corrupted layout = EAC kick przy load.

| Check | Reguła |
|---|---|
| GaItem handle validity | Każdy inventory entry's `gaitem_handle` musi: istnieć w GaItem map; mieć high bit set (`>= 0x80000000`); nie być `0xFFFFFFFF` (sentinel). Patrz `spec/03-gaitem-map.md` |
| Per-slot MD5 recompute | Robione w `backend/core/crypto.go::WriteSave()`. Required by game loader |
| Steam ID match | `UserData10.SteamID` musi == katalog save'a == active Steam ticket przy online login |
| Stat-vs-attribute derivation | HP/FP/SP musi == formuła z `CalcCorrectGraph` evaluowanej na current attrs. `derived_stat_manual` riskKey już to flaguje |
| Event-flag plausibility | "Boss killed" flag bez prerekwizytów (encountered, area unlocked) = anomalia. **Brak public report** że serwer to checkuje, ale tani enforcement |
| GameClearCount monotonicity | NG+ counter nie maleje. Patrz `spec/14-game-state.md` |

### 4D — "Dirty save" detection

**Rekomendacja**: trzymaj flag **POZA** kanonicznym save'm — np. metadata file obok `.sl2` w katalogu edytora. NIE wpisuj custom bytes do save'a (third-party scanners / serwer mogą flagować unknown bytes).

UI behaviour:
- Po każdym `WriteSave()` zapisz `<save>.sl2.editor-meta.json` z `{lastEdit: timestamp, editorVersion, editsApplied: [list]}`.
- Surface w slot picker jako badge *"Last modified: 2026-04-27 by EldenRing-SaveEditor v1.x"*.
- Po imporcie save od innego usera (cross-account) — automatic Tier 1 confirm + scan na ban_risk flagi.

### 4E — Player Data Hash recalculation

Per `spec/22-player-hash.md`: in-save player hash jest recomputable. **Required by game loader** — robimy. **Brak public evidence** że serwer cross-checkuje niezależny server-side hash. Treat as local integrity check only.

---

## 5. Strategie ochrony użytkownika (UX recommendations)

Posortowane po efektywności:

| # | Strategia | Stan w projekcie |
|---|---|---|
| 1 | **Backup przed edycją** — core funkcjonalność. UWAGA: Steam cloud sync runs przy `Steam exit`. Zamknij Steam **przed** edycją | ✅ — `core.BackupSave()` |
| 2 | **Online Safety Mode** (Tier 2 disabled, Tier 1 modal-confirm) | ✅ — patrz `spec/32-ban-risk-system.md` |
| 3 | **"Dirty save" badge** w slot picker (per §4D) | ❌ TODO |
| 4 | **Refuse to write Tier 2 changes z Online Safety Mode ON** | ✅ |
| 5 | **Offline-only-after-edit nudge**: toast *"Recommended: stay offline ≥ 7 days before next online session"*. UWAGA: 7 dni to **heuristic**, nie published rule | ❌ TODO (consider) |
| 6 | **Cross-user save import**: NIGDY auto. Wymagaj SteamID rewrite + Tier 2 confirm + ban_risk scan | ✅ — `character_import` riskKey + `RiskActionButton` |
| 7 | **Steam alt-account testing**: tylko **drugie konto z własnym zakupem ER**. Family Sharing **nie działa** dla VAC-banned games / Bandai-banned accounts | n/d (porada w docs) |

### Anti-recommendations (czego NIE robić)

- ❌ **Nie sugeruj toggle EAC off przed online play.** To najszybsza droga do "Inappropriate activity detected".
- ❌ **Nie pisz custom bytes do save'a** dla edytor metadata. Third-party scanners flagują unknown bytes.
- ❌ **Nie commit'uj presetów / configów które zawierają cut content** lub `ban_risk` flagged items bez Tier 2 warning headera.
- ❌ **Nie clamp'uj wartości przy load/save w `core`** — gracz który ma 999 Larval Tears z innego edytora zachowuje je przy round-tripie. Clamp tylko w modal write path. Patrz `spec/34-item-caps.md`.

---

## 6. Spojrzenie krytyczne — co tu jest niezweryfikowane

| Twierdzenie | Status |
|---|---|
| "EAC checksum encrypted check w startup" (FearLess thread) | `[COMMUNITY]` — single source, nie potwierdzone |
| "Wave bans nightly / weekly" | `[COMMUNITY]` — folklore, brak FromSoft document |
| "Soft-ban i 180-day to dwa różne stany" | `[no public data]` — może być jeden stan z różnymi surface |
| "Cheater pool segregation" | `[COMMUNITY-strong]` — multi-source ale nie official |
| "7 dni offline po edycji" | `[SPECULATION]` — heuristic, nie rule |
| Malcolm Reynolds "softbanned hundreds" | `[partially debunked]` — Kotaku update sama dementuje skalę. Cytuj z update'm |
| "Stat consistency check by serwer" | `[SPECULATION]` — plauzybilne, zero confirmed |

**Implikacja dla doc maintenancera**: gdy ktoś wprowadza nową regułę do `RISK_INFO`, wymagaj cytatu źródła. Domyślnie: `[COMMUNITY]` plus link do thread. Bez source → nie dodawajmy.

---

## 7. Roadmap pre-flight implementacji

Co warte zaimplementowania w nadchodzących iteracjach (priorytet zstępujący):

1. **Pre-write audit pipeline** (`backend/vm/preflight.go`) — pojedynczy `Audit(save) → []Issue` zwracający listę problemów z severity (Tier 1/2). Wywołany przy `SaveBtn.click` jeśli `OnlineSafetyMode == true`.
2. **Item-ID whitelist scan** — iteruj `GaItem map`, dla każdego entry `lookupItem(id)`. Unknown → Tier 2 issue.
3. **Stat consistency derive** — recompute HP/FP/SP z VIG/MND/END (formula z `CalcCorrectGraph`), jeśli stored != derived → Tier 2 issue.
4. **Editor-meta sidecar file** — `<save>.sl2.editor-meta.json` z edit history (per §4D).
5. **DLC ownership cross-check** — `IsDlcOwned` byte vs presence DLC items. Tier 2 jeśli mismatch.
6. **Weapon upgrade level outline** w EquipmentTab (Tier 2 jeśli > 25 / > 10).
7. **GaItem handle audit** w `core.LoadSave()` — log warning (nie clamp) jeśli sentinel `0xFFFFFFFF` znaleziony w nieoczekiwanym miejscu.

---

## Cross-references

- [`spec/01-header.md`](01-header.md) — BND4 / MD5 / AES-128 (PC encryption)
- [`spec/03-gaitem-map.md`](03-gaitem-map.md) — handle integrity
- [`spec/14-game-state.md`](14-game-state.md) — ClearCount, monotonicity
- [`spec/21-dlc.md`](21-dlc.md) — `IsDlcOwned` flag
- [`spec/22-player-hash.md`](22-player-hash.md) — Player Data Hash
- [`spec/23-user-data-10.md`](23-user-data-10.md) — SteamID
- [`spec/32-ban-risk-system.md`](32-ban-risk-system.md) — UI/UX awareness, Tier 0/1/2, `RISK_INFO` dictionary
- [`spec/34-item-caps.md`](34-item-caps.md) — vanilla-realistic caps + NG+ scaling

---

## Źródła

### Oficjalne
- [Easy Anti-Cheat](https://www.easy.ac/) — vendor (Epic Games)
- [Bandai Namco support — EAC Inappropriate Activity Detected](https://support.bandainamcoent.com/s/article/EldenRingEACInappropriateActivityDetected66449c45dc6ce)

### Press / journalism
- [Kotaku — Malcolm Reynolds softban exploit (with update)](https://kotaku.com/elden-ring-hacker-softban-malcolm-reynolds-dark-souls-a-1848643683)
- [Dexerto — Reynolds mass softban claims](https://www.dexerto.com/elden-ring/elden-ring-players-furious-as-hacker-softbans-hundreds-from-online-play-1782179/)
- [PCGamesN — Inappropriate Activity Detected false positives](https://www.pcgamesn.com/elden-ring/inappropriate-activity-detected-fix)
- [GamesRadar — poison/Death-Blight PvP exploit](https://www.gamesradar.com/games/action-rpg/surprise-tarnished-theres-a-new-elden-ring-pvp-exploit-that-lets-you-insta-kill-everyone-in-the-area/)
- [80.lv — Nightreign kernel-level EAC](https://80.lv/articles/elden-ring-nightreign-comes-with-kernel-level-anti-cheat-software)
- [Windows Central — disable EAC offline](https://www.windowscentral.com/how-disable-anti-cheat-elden-ring)

### Community / reverse engineering
- [Souls Modding Wiki — SL2 file format](https://sites.google.com/view/soulsmods/file-formats/sl2-files)
- [GitHub: ClayAmore ER-Save-Editor](https://github.com/ClayAmore/ER-Save-Editor)
- [GitHub: Ariescyn EldenRing-Save-Manager (`hexedit.py`)](https://github.com/Ariescyn/EldenRing-Save-Manager/blob/main/hexedit.py)
- [GitHub: Jeius er-save-manager](https://github.com/Jeius/er-save-manager)
- [Fextralife — Summon Range Calculator](https://eldenring.wiki.fextralife.com/Summon+Range+Calculator)
- [Nexus mod 90 — Anti-cheat toggler / offline launcher](https://www.nexusmods.com/eldenring/mods/90)
- FearLess Cheat Engine — ER thread t=19320 str. 29 (EAC checksum description) — single source, treat as `[COMMUNITY]`

### Steam / GameFAQs / ResetEra (community ban reports)
- [Steam — 180-day ban reports](https://steamcommunity.com/app/1245620/discussions/0/4147320059826168947/)
- [Steam — Banned for save backup?](https://steamcommunity.com/app/1245620/discussions/0/4423184732111534212/)
- [Steam — Can I get banned for cheating levels](https://steamcommunity.com/app/1245620/discussions/0/4526764179303674425/)
- [Steam — Warning: Malcolm Reynolds will softban you](https://steamcommunity.com/app/1245620/discussions/0/3183487594858422139/)
- [ResetEra — Kotaku Malcolm Reynolds discussion](https://www.resetera.com/threads/kotaku-infamous-dark-souls-hacker-is-now-getting-people-softbanned-in-elden-ring.561670/)
