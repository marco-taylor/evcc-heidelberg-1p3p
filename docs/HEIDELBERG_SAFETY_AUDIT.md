# Sicherheitsprüfung Heidelberg-Phasenumschaltung

Stand: 18.11.2026. Ergebnis: **keine Freigabe für den Leistungsbetrieb**. Der Audit ergänzt Tests und Dokumentation, ändert aber weder evcc-Regelung noch Firmware-Schaltlogik. Softwarebefunde und Hardwarefragen werden unten getrennt bewertet.

## Geprüfte Stände

- evcc `master`: `9118fafa720cbcfcd0059a39ec4b61c6a39cd840`, Version `0.320.0-dev.heidelberg.7b590b84`.
- [PR #2](https://github.com/marco-taylor/evcc-heidelberg-1p3p/pull/2), Commit `642f346cbfc4f5dd1175f8fe8ea1ab550f5ce07d`: bedingter PhaseGetter, frisches Lesen von Holding-Register 4, ausschließlich Werte 1/3, sonst `api.ErrNotAvailable`; Kommunikationsfehler bleiben Fehler.
- Ethernet-Firmware [jetronic18s/heidelberg-1p3p](https://github.com/jetronic18s/heidelberg-1p3p/tree/1e6654ddc7d15faa034e8e2980d6c3aef9f9d238), `v1.2.0`, Commit `1e6654ddc7d15faa034e8e2980d6c3aef9f9d238`. SHA256 von `src/switch.cpp`: `6ecba1351b0fde8af0287aa05d3c2066688df4386e3f472b13f7073a06874fdf`.
- Die lokale Ethernet-Kopie liegt unter `/mnt/nvme/appdata/codex-workspace/heidelberg-1p3p-ethernet`. Ihr Zielrepository `marco-taylor/heidelberg-1p3p-ethernet` ist zum Auditbeginn weiterhin leer. Hier wird kein Ersatzimport vorgenommen. Der gemeinsame Prüfbericht und die Tests liegen daher in einem separaten evcc-Auditbranch, als Folge-PR zu PR #2.

## Nachweisbare Softwarefehler

Prioritäten sind Audit-Einschätzungen. Eine bewiesene unsichere Softwareentscheidung beweist noch nicht den zeitlichen Verlauf der realen Schütze.

| ID / Priorität | Fundstelle und Nachweis | Auswirkung / empfohlene Korrektur |
|---|---|---|
| F01 / hoch | Firmware `src/switch.cpp:894`, insbesondere 931–935: `handleRestart()` ruft nach 10 s `restartNow()` auf, obwohl Input 5–8 keinen Status 10 mit drei Nullströmen bestätigt. Simulation `restart_timeout` und `restart_read_error` reproduziert dies. | Ein angeforderter Neustart kann die Stoppbarriere umgehen. Kontrollierten Neustart bei nicht bestätigtem Stopp abbrechen und diagnostizierbaren Sperrzustand beibehalten. Exakte Abschaltreaktion unter Last gesondert mit Schutzkonzept abstimmen. |
| F02 / hoch | Firmware `src/pages.cpp:209` bis 237: `scheduleRestart()` erzwingt nach 15 s `esp_restart()` unabhängig von Stoppbestätigung; bei Timerfehler sogar unmittelbar. Dies ist statisch nachgewiesen, kein Host-Timertest. | Auch eine Korrektur nur von F01 würde den Web-/OTA-/WLAN-Neustartpfad nicht absichern. Alle kontrollierten Neustarts durch dieselbe bestätigte Stoppbarriere führen; Ausweich-Timer darf keine Sicherheitsbedingung aufheben. |
| F03 / hoch | Firmware `src/switch.cpp:48`: `persistsFor()` speichert `nowMs() | 1`. Bei gerader Zeit, z. B. 1000 ms, und unveränderter Uhr beim zweiten Lesen wird `1000 - 1001` als uint32 zu 4294967295. `debounce_even` reproduziert die sofort erfüllte 100-ms-Prüfung. | Ein kurzer Eingangsimpuls kann unmittelbar einen Fault mit Abschalten beider Ausgänge auslösen. Beginn/Zustand separat speichern und nur tatsächlich vergangene Zeit werten; Tests für gerade/ungerade Zeit, Null und Überlauf vorsehen. Ob dieser Zeitfall auf dem ESP eintritt und Schütze unter Last abfallen, muss gemessen werden. |
| F04 / hoch | Firmware `src/switch.cpp:638` bis 646: `Running` verwirft nur gleiche Rückmeldewerte. Soll 3P mit ausschließlich 1P-Rückmeldung wird akzeptiert. `wrong_feedback` reproduziert dies über mehrere Schleifendurchläufe. | Falsche einzelne Schützstellung wird nicht als solche verriegelt. Rückmeldemuster mit dem bestätigten Ziel vergleichen und nach definierter Entprellung sperren; Spannungskontrolle im Betrieb zusätzlich bewerten. Kein automatischer Korrekturwechsel unter Last. |
| F05 / mittel | Firmware `src/switch.cpp:876`: `writeRestartStop()` liefert nach erfolgreichem Stromschreiben auch bei fehlgeschlagenem Lock-Schreiben `true`. `restart_lock_error` reproduziert dies. | `_restartStopWritten` verhindert weitere Sperrschreibversuche. Erfolg erst nach beiden erfolgreichen Transaktionen melden; anschließend weiterhin tatsächlichen Lock und Nullstrom bestätigen. F01/F02 bleiben unabhängig zu korrigieren. |
| F06 / mittel | Firmware `src/switch.cpp:846` bis 869: Entsperr-Timeout schreibt erneut Lock=1 und geht selbst bei Fehler oder fehlender Rückmeldung zu `Running`; `_lastWallboxOkMs` wird dennoch aktualisiert. `unlock_requires_confirmation` reproduziert dies. | Zustand und Kommunikationsdiagnose können einen nicht bestätigten Erfolg behaupten. Rückmeldung oder expliziten Fehlerzustand verlangen, keine Erfolgstimestamps ohne erfolgreiche Transaktion setzen. Dies beweist für sich kein unerwünschtes Einschalten, weil die Wallbox auch gesperrt bleiben kann. |

Direkte Quellen: [Firmware-Zustandsautomat](https://github.com/jetronic18s/heidelberg-1p3p/blob/1e6654ddc7d15faa034e8e2980d6c3aef9f9d238/src/switch.cpp), [Web-/OTA-Neustartpfade](https://github.com/jetronic18s/heidelberg-1p3p/blob/1e6654ddc7d15faa034e8e2980d6c3aef9f9d238/src/pages.cpp).

## Nachweisbares Verhalten mit offener Sicherheitsanforderung

Diese Punkte sind nicht allein aufgrund ihrer Existenz Programmierfehler. Sie begrenzen aber die Aussage, ein Neustart oder ein Phasenwert könne eine sichere Schützstellung garantieren.

### Kaltstart, Brownout und Absturz

Firmware `src/switch.cpp:150` initialisiert `WaitingForOff`, Ziel 1P, keinen angeforderten Wechsel und keinen Safety Fault. `begin()` ab Zeile 234 setzt im Anwendungscode beide Relais aus. Erst nach gestartetem Modbus-TCP-Server arbeitet der Zustandsautomat weiter. Wenn beide Rückmeldungen aus sind, wartet er 2 s und zieht in `ConfirmedOff` das gewählte Relais an, **ohne neuen externen Einschaltauftrag und vor erfolgreicher Wallbox-Validierung** (Zeilen 587–637). `boot_without_request` dokumentiert diese automatische Einschaltpolitik.

`loadBootCache()` ab Zeile 244 akzeptiert RTC-Cache ausschließlich bei `ESP_RST_SW` plus Magic/CRC. Bei Brownout oder Panic wird der Cache nicht übernommen. Ein vorheriges 3P-Ziel wird dann durch den Konstruktorstandard 1P ersetzt. Die Simulation prüft den originalen Resetgrund-Zweig für Software-Reset, Brownout und Panic; sie löst keinen echten Reset aus. Ein Absturz in der Umschaltung kann damit nach Reset einen anderen Zielpfad nehmen als ein kontrollierter Software-Neustart.

Der Fault ist nur bis zum Neustart verriegelt. Sein Code wird nicht im BootCache gespeichert. Eine explizite erneute Bedienfreigabe nach Sicherheitsfehler ist nicht implementiert. Empfehlung zur gesonderten Entscheidung: nach unerwartetem Reset oder Fault zunächst beide Ausgänge aus lassen und einen expliziten, geprüften Wiederanlauf verlangen. Einen alten Sollwert blind wiederherzustellen wäre ebenfalls kein Nachweis einer sicheren Iststellung.

Nicht bewiesen: dass die realen Schütze beim Reset sofort abfallen, wie lange sie abfallen, ob Rückmeldungen vor dem Neuboot bereits stabil sind oder ob Kontakte unter Last öffnen. Dafür sind GPIO-/Relais-/Kontaktmessungen erforderlich.

### Ladefreigabe und Strom nach Neustart

Firmware `src/switch.cpp:769` bis 843: Im normalen Bootpfad ohne `_phaseSwitchRequested` wird zuerst Strom 0 (Register 261) geschrieben und danach Lock=1 (Register 259). Das ist **keine harte dauerhafte Ladesperre**: Lock=1 gibt die Wallbox frei, aber die angebotene Stromvorgabe ist zunächst 0. `boot_current_zero` bestätigt die Schreibreihenfolge. Nach einem regulären Phasenwechsel wird ein alter Strom auf maximal 6 A begrenzt; ein während des Wechsels neu geschriebener Strom wird übernommen. `switch_old_current_capped` prüft den alten 16-A-Wert.

Die Wallbox kann nach dem Wiedereinschalten bereits laufen, bevor dieser `Delay`-Zweig erreicht ist. Das Verhalten ihrer eigenen gespeicherten Strom-/Lock-/Failsafe-Werte während des Bootens ist hier nicht gemessen. Der Anwendungscode allein beweist daher kein lückenloses 0-A-Fenster ab Spannungswiederkehr.

evcc kann in einem laufenden Ladeauftrag eine erneut deaktivierte Wallbox wieder freigeben; die vorhandene Synchronisierung enthält beispielsweise `Enable(true)` während eines noch nicht abgeschlossenen Phasenwechsels (`core/loadpoint.go:1001`). Der Heidelberg-Treiber kann bei Input-Status 10 außerdem den Remote-Lock aufheben (`charger/heidelberg-ec.go:170`). Das sind vorhandene Steuerpfade, keine neuen Änderungen aus PR #2. Die Firmware muss Schreibzugriffe in kritischen Zuständen sicher puffern beziehungsweise abweisen. Eine allgemeine Zusage „nach Neustart niemals Ladefreigabe“ ist damit falsch; das gewünschte Wiederanlaufverhalten muss ausdrücklich festgelegt werden.

### PR #2, Sollwert und Modbusfehler

PR #2 verändert ausschließlich die bedingte Registrierung und das Lesen der Phaseneinstellung. Keine GPIOs, kein Neustart, keine neue Abschaltsequenz. In `charger/heidelberg-ec.go:349` werden nur 1/3 akzeptiert; Fehler ersetzen den Wert nicht durch einen vermeintlich sicheren Standard.

Firmware `reportedPhases()` ab `src/switch.cpp:1003` liefert einen vorgemerkten Auftrag oder `_desiredPhases`. Schon während `WaitingForZero` kann Register 4 daher 3 melden, während das 1P-Schütz noch aktiv ist. Auch bei `Fault` kann ein Sollwert 1/3 bestehen bleiben; die Fehlermeldung erfolgt separat über Input-Status 11. `phase_report_is_confirmation` demonstriert die falsche Annahme, Register 4 sei eine Schützbestätigung. Das ist eine **Vertragsgrenze**, kein Fehler der Werteprüfung in PR #2.

evcc übernimmt einen gültigen Getterwert in `core/loadpoint_phases.go:172`. `api.ErrNotAvailable` lässt den alten beziehungsweise unbekannten Wert bestehen. Kommunikationsfehler werden dort weitergereicht. Aufrufpfade behandeln diese unterschiedlich: im konsistent aktivierten Zustand als Fehler; bei deaktivierter Wallbox beziehungsweise beim Anstecken teilweise nur als Logeintrag (`core/loadpoint.go:635`, 987, 995). Somit erzwingt ein fehlerhafter Phasenread nicht überall eine Ladesperre. Unsere zusätzlichen strikten Mocktests bestätigen: der reine Phasenabgleich schreibt weder Strom noch Freigabe noch Phasenbefehl und erfindet keinen Ersatzwert. Sie behaupten nicht, den kompletten Regelzyklus sperrend geprüft zu haben.

Empfehlung: Soll-/Ist-/Übergangs-/Fehlerzustand im Firmwareprotokoll getrennt definieren. Erst danach den Getter passend anbinden und alle Verbraucher einschließlich Neustart berücksichtigen. PR #2 ist ein Einstellungsabgleich und keine eigenständige Sicherheitsfunktion. Die bestehende `phases1p3p(int)`-Schreibmethode validiert interne Werte nicht zusätzlich; dies ist durch PR #2 unverändert und benötigt bei Härtung einen eigenen Vertragstest.

## Was die reguläre Umschaltung bereits prüft

- Register 261 auf 0 und Lock 259 auf 0 müssen erfolgreich geschrieben sein.
- Input 5–8 muss Status 10 und alle drei Ströme exakt 0 liefern, bevor Relais ausgeschaltet werden.
- Bei fehlender Antwort oder weiter fließendem Strom bleibt `WaitingForZero`; nach 15 s werden Stopp-/Lock-Schreibversuche wiederholt, kein normaler Zwangswechsel.
- Beide Rückmeldungen aus, 2 s Ausschaltpause, nur ausgewähltes Relais ein. Hängenbleibende Ausschalt-Rückmeldung führt nach 8 s zu E105.
- Nach Einschalten werden ausgewählter Hilfskontakt und Spannungsmuster geprüft. Fehlende Bestätigung führt zu Fault. Zwei aktive Rückmeldungen führen zu E101, beide fehlenden im Betrieb zu E103, jeweils mit der beschriebenen fehlerhaften Entprellhilfe.
- `enterSafetyFault()` ab Zeile 1228 schaltet beide Ausgänge unmittelbar aus. Das ist keine unterbrechungsfreie Nullstromsequenz; ein Fehler kann bewusst eine Abschaltung unter Last auslösen. Die dafür nötige Schütz-/Schutzdimensionierung ist eine Hardwarefrage.
- Der evcc-Watchdog ab Zeile 1013 übernimmt Register 262 als Failsafe-Strom. Bei 0 wird 0 geschrieben; bei 16 A wird 16 A geschrieben. Ein konfigurierter Nichtnull-Failsafe ist keine automatische Abschaltung bei evcc-Ausfall. Die Tests prüfen beide Einstellungen.

## Nur durch Hardware-/Systemmessung klärbar

| Offene Frage | Warum Softwaretests nicht ausreichen | Benötigter Nachweis vor Leistungsbetrieb |
|---|---|---|
| GPIO-/Relaispegel beim Einschalten, Brownout, Panic, Watchdog und Reset | Der Host simuliert nur logische GPIO-Werte. Boot-ROM, Strap-Pins, Treiber, Pull-Widerstände und Versorgung fehlen. GPIO2 ist Strap-Pin; GPIO36/39 haben keine internen Pull-Widerstände. | Tatsächliche Boardrevision/Schaltplan und definierte Eingangspegel; zeitaufgelöste Messung der GPIOs, Relaisansteuerung und Hilfskontakte unter kontrollierten Reset-/Versorgungsfällen. |
| Elektrische/mechanische Verriegelung und klebende Leistungskontakte | Ein Hilfskontakt ist nicht ohne Weiteres ein beweiskräftiger Spiegelkontakt. Softwareausgänge können keine verschweißten Kontakte trennen. | Verdrahtung, geeignete Kontakte, Verriegelung, Ausschaltvermögen und Schutzkonzept fachlich prüfen. |
| Stromfreiheit bei Relaiswechsel | Modbus-Stromwerte haben unbekannte Abtast-/Übertragungsverzögerung, Exakt-Null im Register ist keine unabhängige Strommessung. | Strom- und Kontaktverlauf mit ausreichender Zeitauflösung, auch bei Kommunikationsverlust. |
| Wiederanlauf der Heidelberg nach Spannungsausfall | Deren interne Firmware, Lock-/Stromspeicherung und Watchdog sind außerhalb dieses ESP-Programms. | Konkreter Wallbox-Firmwarestand/Registerlayout und Verhalten ab Spannungswiederkehr prüfen. |
| Gleichzeitige HTTP-/Modbus-/OTA-Zugriffe, Flash-Stalls und Scheduler | Hosttests führen ausgewählte Originalfunktionen deterministisch aus; FreeRTOS, Interrupts, Netzwerk und reale Flashzeiten fehlen. | Separater isolierter Prüfstand ohne Leistungskreise, anschließend fachlich freigegebene Systemprüfung. |

ESP-IDF-Primärquelle zu Strap-Pins, fehlenden Pull-Widerständen und veränderlichen GPIO-Startwerten: [GPIO-Dokumentation 5.5.6](https://docs.espressif.com/projects/esp-idf/en/v5.5.6/esp32/api-reference/peripherals/gpio.html). Keine dieser Messungen wurde hier durchgeführt.

## Testlücken und neue automatisierte Simulationen

Die 16 Fälle aus PR #2 prüfen Registererkennung, gültige/ungültige Rückmeldungen, Kommunikationsfehler und Schreibbefehle auf Loopback. Sie prüfen keine physische Schützfolge. Die vorhandenen Core-Tests decken viele Phasenabgleiche ab, kombinieren aber nicht alle Fehler-/Neustartbedingungen mit ausbleibenden Schreibzugriffen.

Die Firmware-Bench enthält bereits Neustart-, OTA-, Failsafe- und Kommunikationsfälle, darunter `bench/bench_test.py:1142` mit simuliertem Fahrzeugnachlauf von 3 s. Dafür braucht sie ein echtes ESP-Board und steuert dessen Relais. Sie wurde **nicht ausgeführt**. Der 3-s-Fall prüft gerade nicht den gefährlichen 10-s-/15-s-Timeout. Brownout-/Panic-Pegel und falscher einzelner Hilfskontakt sind damit nicht nachgewiesen.

Neu:

1. `core/loadpoint_heidelberg_safety_test.go`: sechs strikte Mockfälle zu Neustart mit unbekannten Phasen, ungültigem Register, Kommunikationsfehler bei aktivierter/deaktivierter Wallbox, Übernahme nach Neustart und abweichendem Bridge-Sollwert trotz alter Messphasen. Unerwartete Schalt-/Strom-/Freigabeaufrufe lassen den Test scheitern.
2. `scripts/heidelberg_safety_sim.py`: lädt ausschließlich den oben gepinnten Originalquellstand oder liest eine lokale Kopie, prüft deren SHA256 und extrahiert die unveränderten C++-Funktionen. Host-GPIO, Zeit, RTU und Neustart werden simuliert. Es wird kein ESP-Binary geflasht, kein Gerät geöffnet und kein Relais angesteuert. Ein separates temporäres C++-Programm wird mit `-Wall -Wextra -Werror` gebaut.
3. 28 Hostfälle: 20 Regressionen und acht ausdrücklich als `expected failure` markierte nicht erfüllte Sicherheitsanforderungen. Diese entsprechen nicht acht unabhängigen Bugs. Bekannte Ursachen werden auf exakte Meldungen geprüft; Absturz, unbekannter Rückgabecode und Timeout können sich nicht als erwarteter Fehler tarnen. Eine später erfüllte Anforderung wird als `unexpected success` sichtbar und erzwingt eine bewusste Anpassung der Kennzeichnung.
4. `.github/workflows/heidelberg-safety-audit.yml` führt diese Hostsimulation bei relevanten PR-Änderungen aus. Nur Leserechte, kein Release-/Flash-/Hardwarepfad. Ein erfolgreicher Job mit erwarteten Fehlern ist ausdrücklich **keine Sicherheitsfreigabe**.

Grenzen: kein vollständiges ESP-IDF-Programm, kein echter Brownout/Panic, keine MCU-Emulation, keine Aussage über Schaltzeiten. Konstruktorannahmen werden am gepinnten Quelltext geprüft; RTC-CRC und Betriebssystem-/RTU-Nebenfunktionen sind Stubs. Resettests prüfen den Originalzweig für den Resetgrund, nicht physische Resetfolgen. Der Web-Fallback F02 ist statisch geprüft, nicht durch einen echten ESP-Timer simuliert. OTA-Konkurrenz, Reset in jedem CPU-Zeitpunkt und physische Kontaktfehler bleiben offen.

Ausführung ohne Netzwerk gegen vorhandene Quelle (Python 3 und g++ erforderlich):

```bash
python3 scripts/heidelberg_safety_sim.py \
  --firmware-root /mnt/nvme/appdata/codex-workspace/heidelberg-1p3p-ethernet
go test -race -tags=release -run '^TestHeidelberg|^TestSyncCharger' ./charger ./core
```

Alternativ lädt `python3 scripts/heidelberg_safety_sim.py --fetch-pinned` ausschließlich die gepinnte öffentliche Quelldatei per HTTPS. Der fremde Code behält seine [ursprüngliche GPL-Lizenz](https://github.com/jetronic18s/heidelberg-1p3p/blob/1e6654ddc7d15faa034e8e2980d6c3aef9f9d238/LICENSE); er wird nur temporär für den Test verarbeitet, nicht als eigener Firmwarecode ausgegeben.

Lokale Ergebnisse: Hostsimulation 28 Fälle, davon 20 erfüllt und acht erwartete Anforderungsverletzungen; Go-/Race-Tests für Heidelberg und Phasensynchronisierung erfolgreich. GitHub-Actions-Ergebnisse werden im zugehörigen Audit-PR verlinkt. Der bereits bekannte nachgelagerte Claude-Review kann mangels Anthropic-Anmeldung scheitern; das wird nicht als Code-/Testfehler oder bestandener Review ausgegeben.

## Empfohlene Reihenfolge der Korrekturen

1. Vor Leistungsbetrieb F01/F02 gemeinsam, F03 und F04 durch gesondert freigegebene Firmwareänderungen beheben. F05/F06 ebenfalls korrigieren und negative Fehlerpfade prüfen. Dieser Audit enthält diese Korrekturen ausdrücklich noch nicht.
2. Verbindliche Wiederanlaufpolitik nach Software-Reset, Panic, Brownout und Safety Fault festlegen. Einschalten, Ladefreigabe und bestätigte Phasen voneinander trennen.
3. Soll-/Ist-/Busy-/Fault-Protokoll und Umgang mit veralteten Messwerten definieren. PR #2 allein löst dies nicht.
4. Danach deterministische Softwaretests erweitern und fachlich geplante, zunächst leistungskreisfreie Hardwaretests durchführen. Kein produktiver Container und keine bestehende Installation wird automatisch geändert.
