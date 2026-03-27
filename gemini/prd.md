# Product Requirements Document: ER-Save-Editor-Go

## 1. Cel Projektu
Stworzenie nowoczesnego, wieloplatformowego edytora plików zapisu do gry Elden Ring w języku Go (Golang). Projekt ma na celu zastąpienie wersji Rust (`tmp/org-src`), oferując 100% zgodność funkcjonalną, błyskawiczne działanie oraz nowoczesny interfejs użytkownika oparty na frameworku Wails.

### Wspierane Platformy
- **Windows**: x64 (Natywny plik .exe)
- **macOS**: Apple Silicon & Intel (Uniwersalna paczka .app)
- **Linux**: x64 (Binarka)

---

## 2. Stack Techniczny
- **Język**: Go 1.21+
- **Framework UI**: `Wails` (Go backend + Web frontend)
    - *Powód*: Pozwala na tworzenie natywnych aplikacji desktopowych przy użyciu HTML/CSS/JS dla UI, zachowując wydajność i bezpieczeństwo natywnego kodu Go.
- **Parsowanie Binarne**: Wbudowany pakiet `encoding/binary`
    - *Powód*: Natywne, bezpieczne typologicznie i niezwykle szybkie mapowanie surowych bajtów na struktury Go.
- **Kryptografia**: Wbudowane pakiety `crypto/aes`, `crypto/cipher`, `crypto/md5`, `crypto/sha256`.
- **Pakowanie**: `wails build` (tworzenie samodzielnych plików wykonywalnych bez zewnętrznych zależności).

---

## 3. Architektura Systemu
Aplikacja musi zachować ścisły podział na warstwy:

### A. Warstwa Danych (Backend - Go)
- **Binary Templates**: Definicje struktur `struct` dla PC (.sl2) i PlayStation (Save Wizard decrypted) oparte na `tmp/org-src`.
- **Checksum Logic**: Implementacja MD5 (PS/PC) oraz SHA256 (PC BND4) do walidacji i naprawy zapisów.
- **Game Database**: Baza danych przedmiotów, łask i flag bossów wyeksportowana z oryginału.

### B. Warstwa Logiki (ViewModel - Go)
- Walidacja danych wejściowych (np. limity statystyk).
- Mapowanie surowych bajtów na obiekty zgodne z logiką oryginału.
- **SteamID Support**: Możliwość zmiany identyfikatora Steam dla zapisów PC.
- Wystawianie API dla frontendu (Wails Bindings).

### C. Warstwa Prezentacji (Frontend - Web)
- **Styl**: UI wzorowany na oryginale (egui), z możliwością późniejszych ulepszeń.
- **Responsywność**: Płynne skalowanie i dopasowywanie do rozmiaru okna.
- **Motywy**: Dynamiczne przełączanie między trybem jasnym i ciemnym.

---

## 4. Wymagania Funkcjonalne
1. **Zarządzanie Postacią**:
    - Zmiana imienia (UTF-16), poziomu, statystyk (Vigor, Mind, itd.) i liczby dusz.
    - Zmiana płci i klasy początkowej.
2. **Edytor Ekwipunku**:
    - Dodawanie/usuwanie przedmiotów, broni, talizmanów i popiołów wojny (zgodnie z `tmp/org-src`).
    - Funkcja "Bulk Add" dla szybkich buildów.
3. **Postęp Świata**:
    - Odblokowywanie Miejsc Łaski (Graces), Pól Przywołań i Koloseów.
    - Flagi Bossów (zabijanie/wskrzeszanie).
4. **Narzędzia Dodatkowe**:
    - **SteamID Changer**: Migracja zapisów PC między kontami.
    - **Character Importer**: Kopiowanie postaci między różnymi plikami zapisu.
5. **Bezpieczeństwo i Weryfikacja**:
    - Automatyczny backup oryginalnego pliku przed zapisem zmian.
    - **Weryfikacja Integralności (Post-Write Validation)**: Po każdym zapisie aplikacja musi automatycznie spróbować odczytać nowo powstały plik, aby potwierdzić poprawność sum kontrolnych.
    - Walidacja rozmiaru pliku po zapisie (identyczny z oryginałem).

---

## 5. Wymagania pozafunkcjonalne
- **100% Parity**: Logika biznesowa musi być identyczna z wersją Rust.
- **Wydajność**: Czas wczytywania/zapisu < 0.1s.
- **UX**: Intuicyjna nawigacja, nowoczesny wygląd desktopowy.
- **Dystrybucja**: Pojedynczy plik wykonywalny.
