# Heidelberg Safety.1: Iststand, Entwicklung und Prüfplan

Stand 22.11.2026. Kein Leistungsbetrieb freigegeben. Produktiver Container, Hardware und Firmware bleiben unverändert.

## Istzustand vor Änderungen

Fork-master 9118fafa720cbcfcd0059a39ec4b61c6a39cd840, Imageversion 0.320.0-dev.heidelberg.7b590b84. PhaseSwitcher ist integriert; PhaseGetter aus PR #2 (642f346c) ist ungemergt. PR #4 (675aa90f) ergänzt Audit und Mocktests, PR #5 (ff03e916) die SHA256-gepinnte Firmware-Simulation. Der Entwicklungsbranch baut auf PR #5 auf, um diese Arbeit zu erhalten. PR #3 ist eine unabhängige Workflowänderung und wird nicht übernommen.

Originalfirmware v1.2.0: 1e6654dd. Eigener ausschließlich lokaler Firmwarestand v1.2.0-safety.1: 63d6c0de80813309464f29273186c6231c9f7e79. Produktionsquellen sind bytegleich mit Firmware-PR #1 / 4682a33. Register 4 liefert 1/3 erst nach Kontakt-/Spannungsbestätigung und aktuellem exklusivem Hilfskontaktmuster, sonst 0. Das ist keine unabhängige elektrische Messung. Nach Kaltstart/Brownout/Watchdog/Panic bleibt AwaitingStart, Web-Freigabe erforderlich. F01–F06 werden in der Firmware behandelt; diese Logik wird hier nicht verändert.

Offizielles aktuelles Release: 0.319.1 (16.11.2026), Commit 29ef1eb4d0fff3186f6d9574496691b09cd6ae43. Upstream-master a9f1206bfc50944988249b1a3b1ab74058c6f8c6; gemeinsamer Vorfahr 7b590b84e36484f73c94cb813abf2d555dd5f193. Fork-master enthält vier eigene Commits, Upstream-master 67 weitere. Im relevanten Heidelberg-Treiber und der Phasensynchronisierung gibt es seit dem gemeinsamen Vorfahren keine neuen Upstreamänderungen. Ein blindes Ersetzen des Treibers durch Upstream entfernt unseren PhaseSwitcher. Die neueren allgemeinen Geräte-/Datenquellenänderungen können separat portiert werden; ein vollständiger Versionswechsel wird mit dieser Sicherheitsintegration nicht vermischt. Die Dev-Versionsnummer 0.320.0 bedeutet nicht, dass das offizielle Release 0.319.1 bereits vollständig enthalten ist.

Referenzen: https://github.com/evcc-io/evcc/discussions/6168 , https://github.com/evcc-io/evcc/releases/tag/0.319.1 , https://github.com/jetronic18s/heidelberg-1p3p . Der Diskussionsvorschlag ist keine elektrische Freigabe.

## Nachgewiesene Integrationslücken

1. Der bisherige Konstruktor erwartet bei Initialisierung 1/3. Register 0 verhindert die Fähigkeitserkennung; zudem können Standby-/Timeoutzugriffe im kalten AwaitingStart mangels Wallbox-Cache scheitern. Ein nicht erreichbarer Controller darf den anderen Ladepunkt nicht beim Start unbrauchbar machen.
2. PR #2 registriert einen Getter auch für alte Firmware, deren Register nur einen Auftrag meldet. Ohne expliziten Vertrag kann dieser Wert nicht als Bestätigung gelten.
3. Der Kern ignoriert ErrNotAvailable und behält die zuletzt angenommene Phase. Das ist kein frischer Bestätigungsnachweis. Beim bekannten Safety-Vertrag muss ein explizit unbekannter Zustand die Annahme verwerfen.
4. Status 10 kann im bisherigen Treiber ein Entsperren auslösen. Im Safety-Modus darf evcc die Sperre nicht selbst aufheben, keine WakeUp-Sperr-/Entsperrsequenz ausführen und bei unbekannten Phasen keine positive Stromvorgabe oder Phasenanforderung schreiben.
5. Gemeinsame Felder current/wakeup können bei Heartbeat und normalen Zugriffen konkurrieren. Zwei Controller benötigen getrennte Verbindung, Zustand und Fehlerbehandlung.

## Geplante, gezielte Änderungen

- Expliziter Modus phasefeedback: true ausschließlich für den dokumentierten Safety-Firmwarevertrag. Kein Raten der Firmwareversion aus Register 4. Normale Heidelberg-Geräte und alte Brücken behalten den bisherigen PhaseSwitcher-Pfad, erhalten aber keinen unbewiesenen Istphasen-Getter.
- Fähigkeiten im expliziten Modus bereits bei Start bekannt; Gerätedaten erst nach frischer 1/3-Rückmeldung initialisieren. Kein evcc-Neustart nach bewusster Web-Freigabe nötig. Firmwareversion bleibt eine vom Betreiber zu prüfende Voraussetzung, da kein verlässliches Versionsregister existiert.
- Neuer expliziter Fehler für unbestätigte Phasen. Nur bei diesem Vertrag verwirft der Kern angenommene/gemessene Phasen. Allgemeines ErrNotAvailable anderer Geräte behält seine bisherige Bedeutung.
- Jeder freigebende Befehl benötigt frische bestätigte Phase. Abschalten/0 A bleibt zulässig. Keine automatischen Remote-Unlock-/WakeUp-Befehle im Safety-Modus. Keine Schütz-/Nullstromlogik in evcc nachbauen.
- Tests mit zwei getrennten Modbus-TCP-Simulatoren: gleichzeitige Zugriffe, unabhängige Wechsel, Kaltstart, Fehler, Verbindungsabbruch, Wiederverbindung und Isolation. Zusätzliche Kern-/Circuit-Tests für PV und gemeinsame Leistungsbudgets mit ausdrücklich synthetischen Testgrenzen.
- Zwei Ladepunkte jeweils maximal 16 A konfigurieren. Gemeinsame Zuleitungsgrenze wird nicht angenommen; sie benötigt spätere Festlegung, Messung und elektrische Prüfung.

## Prüfplan und offene Grenzen

Vorhandene/neue Go-Tests, Race-Tests, 55 Firmware-Hostsimulationen, vollständiger Build einschließlich Weboberfläche und lokales Linux/AMD64-Docker-Image. Laufzeittest ausschließlich in isoliertem Docker-Netz mit simulierten Controllern, ohne Host-Netz, Geräte, Produktivkonfiguration oder Produktivdatenbank. Kein Push als latest, kein Merge.

Hardwareprüfungen bleiben offen: GPIO-/Boot-ROM-Impulse, Versorgung/Brownout, mechanische/elektrische Verriegelung, echte Hilfskontakte, klebende Schütze, Strommessverzug, RS485-Timing/Terminierung und reale Ethernet-/OTA-Neustarts. Die unveränderte Firmware-Fault-Notreaktion und physische Resets können Schütze ohne bestätigten Nullstrom abfallen lassen. Simulationen sind keine Freigabe für Leistungsbetrieb.

Die folgenden Abschnitte werden nach Implementierung um tatsächliche Ergebnisse und verbleibende Grenzen ergänzt.

## Implementierter Vertrag

`phasefeedback: true` registriert Getter und Switcher auch bei unbekanntem Startzustand. Dieser Modus darf nur mit dem dokumentierten Safety-Firmwarestand verwendet werden. Ohne diese Auswahl bleiben normale Heidelberg-Geräte und alte Brücken auf dem bisherigen Switcher-Pfad, ohne unbewiesenen Getter. Die Konfigurationsvorlage docs/heidelberg-safety1-two.evcc.yaml verwendet zwei Dokumentations-IP-Adressen, getrennte Controller, Modbus-ID 1 je unabhängigem RS485-Bus und ausgeschaltete Ladepunkte. Eine reale gemeinsame Anschlussgrenze ist dort bewusst nicht festgelegt.

Vor jeder Freigabe/Stromvorgabe/Umschaltanforderung wird Register 4 frisch gelesen. Gültig sind ausschließlich 1/3. Ein erfolgreicher Schreibauftrag ist noch keine Bestätigung: ErrPhaseSwitchPending markiert die Annahme, ErrPhaseStateUnknown verwirft angenommene/gemessene Phasen im Kern. Positive Befehle bleiben bis zur angeforderten Rückmeldung blockiert. 0 A/Disable bleibt möglich. Allgemeines ErrNotAvailable anderer Treiber bleibt unverändert. Die Anzeige kann weiterhin die generische konservative 3P-Kapazitätsschätzung für unbekannte Phasen verwenden; das ist keine bestätigte Schützstellung.

Der Safety-Modus entriegelt bei Status 10 nicht selbst und verweigert Remote-WakeUp. Initialisierung und Standby-Konfiguration erfolgen erst bei bestätigter Phase. Bei Kommunikationsverlust werden Fehler weitergereicht und alte Phasenannahmen verworfen. Bei Rückkehr mit gültiger Rückmeldung kann die Initialisierung ohne evcc-Neustart nachgeholt werden. Gemeinsame Treiberfelder werden pro Instanz serialisiert, der Heartbeat besitzt einen beendbaren Ticker. Der Firmware-Cache kann bereits während Delay eine bestätigte Phase melden; dies beweist kein Running und hebt die Firmware-Sperrzeiten nicht auf.

Schreibzugriffe werden nicht blind wiederholt. Eindeutige Modbus-Ablehnungen (illegal function/address/value, busy) behalten keinen angenommenen Auftrag. Nach einem verlorenen Schreib-ACK bleibt ein möglicherweise angenommener Zielauftrag unbestätigt, bis dieses Ziel tatsächlich rückgemeldet wird. Bei dauerhaft anderem Ziel ist manuelle Prüfung beziehungsweise Neuinitialisierung erforderlich. Das ist absichtlich konservativ, keine automatische Neuauslösung.

Gezielt portierter Upstream-Fix: b2ee30ab54ad8497811278bbdee1acdd5976ebbc, Healthcheck berücksichtigt EVCC_NETWORK_PORT und doppelte mDNS-Portdeklaration entfernt. Zusätzlich wird .git aus dem Docker-Kontext ausgeschlossen, damit lokale Testcaches und Metadaten nicht in Buildschichten gelangen. Die Go-Buildparallelität ist im Builder auf zwei begrenzt; das Laufzeitimage erhält diese Beschränkung nicht. Imagekennung des Entwurfs: 0.320.0-dev.heidelberg.safety1.dual, ausschließlich lokaler Testtag, kein latest-Push.

## Nachweise und Abgrenzung der Tests

Die zusätzlichen TCP-Tests verwenden zwei voneinander unabhängige Server mit eigener Verbindung, eigener Phasenrückmeldung und eigenen Schreibprotokollen. Gleiche Unit-ID auf getrennten TCP-Endpunkten entspricht den zwei getrennten RS485-Bussen; lokale unterschiedliche Ports ersetzen in der Simulation die zwei IP-Adressen. Geprüft werden Kaltstart mit Register 0, manuelle Wiederfreigabe durch gültige Rückmeldung, ungültige Werte 2/65535, Neustart einer Treiberinstanz, zwei gleichzeitige Phasenwechsel, einseitiger Verbindungsabbruch/Wiederverbindung, Modbus-busy, verlorenes Schreib-ACK und konkurrierende Treiberzugriffe. Es gibt keine Wiederholung eines möglicherweise bereits angenommenen Phasenauftrags. Der Kern wird zusätzlich direkt mit zwei tatsächlichen Heidelberg-Treiberinstanzen über TCP geprüft, einschließlich unbekannter Phasen, gemessener Leistung und Circuit-Begrenzung.

Der isolierte Docker-Laufzeittest verwendet absichtlich generische HTTP-Ladegerätesimulatoren. Damit werden das vollständige Programm, zwei Lastpunkte, PV-Überschuss, gemeinsame dynamische Leistungsgrenzen, einseitiger Ausfall und Wiederkehr geprüft. Er ersetzt nicht den separaten Heidelberg-Modbus-Test. Ein vollständiger Container-Lauf mit dem Heidelberg-Treiber verlangt die reguläre evcc-Sponsorisierung; Zugangsdaten werden weder angefordert noch aus dem Produktivcontainer übernommen. Die Go-Tests verwenden den bestehenden offiziellen Testmechanismus zur Sponsorisierung, das Laufzeitprogramm hat keine Lizenzumgehung.

Alle Leistungsbudgets in den Tests (unter anderem 12/15/18 kW) sind ausschließlich synthetische Testdaten. Für die Installation sind ein geeigneter gemeinsamer Netz-/Circuit-Zähler und die tatsächlich elektrisch zulässigen Grenzen separat festzulegen. Ein unerreichbarer Controller kann weiterhin laden; ein Kommunikationsfehler bedeutet weder stromlos noch abgeschaltet. Bestehende letzte Leistungsmessungen bleiben grundsätzlich Schätzwerte; eine frische gemeinsame Messung und die lokale Firmware-Watchdogfunktion sind erforderlich. Keine automatische Aktualisierung oder Änderung der Produktivkonfiguration.

Die 55 bestehenden Firmware-Hostsimulationen werden unverändert gegen den lokalen Firmwarecommit 63d6c0de80813309464f29273186c6231c9f7e79 ausgeführt. Der Wrapper verifiziert die geprüften Produktionsquellen per SHA256; diese stimmen mit dem bereits gepinnten öffentlichen Firmware-Kandidaten 4682a33 überein. F01–F06 werden damit erneut regressionsgeprüft, aber nicht elektrisch nachgewiesen. Die Firmware und ihre sicherheitsrelevante Schützlogik werden durch diesen evcc-Branch nicht geändert.

## Offene Prüfungen vor Einsatz

- Die explizite Auswahl phasefeedback setzt die tatsächlich installierte Safety-Firmware voraus. Das Protokoll besitzt kein geeignetes verlässliches Firmware-Identifikationsregister; automatische Unterscheidung alter Auftragsrückmeldung und bestätigter Istphase ist nicht möglich.
- Eine gültige Register-4-Rückmeldung kann während der Firmware-Verzögerung vor Running erscheinen. Nur die Firmware verwaltet Ladeunterbrechung, Nullstrombestätigung, Verriegelung und Sperrzeiten; evcc darf einen ACK oder Istphasenwert nicht als Ende dieser Schutzsequenz verwenden.
- Erst zwei isolierte Development-Boards mit realen, ungefährlichen Ersatzlasten können Ethernet/RS485-Timing, gleichzeitige Verbindungen, Startfreigabe, OTA-/Web-Neustart und Watchdogreaktionen praktisch bestätigen. Solche Prüfungen wurden hier nicht durchgeführt.
- Brownout, harter Reset und Fehlernotabschaltung können physisch einen Schützabfall unter Last verursachen. Einschaltimpulse, klebende oder widersprüchliche Rückmeldekontakte und die elektrische/mechanische Verriegelung benötigen Messungen und Prüfung der konkreten Schaltung.
- Der Root-Circuit-Wert der echten Installation, der Netz-/Leistungsmesser und die elektrische Auslegung sind offen. 16 A je Wallbox ist keine Aussage über die gemeinsame Zuleitung.
- Die übrigen 67 Upstream-Commits und der Release-Backport 0.319.1 sind nicht vollständig übernommen. Ein späterer Versionswechsel benötigt eigene Regressionen; keine Behauptung, der Fork sei vollständig auf dem offiziellen aktuellen Release.

Keine Freigabe für elektrischen Leistungsbetrieb. Kein Merge und kein Image-Push durch diesen Auftrag.

## Tatsächliche lokale Ergebnisse am 22.11.2026

- Vollständiger Go-Testlauf: `CGO_ENABLED=0 go test -tags=release ./...`, Exit 0. Zusätzlich Race-Tests für alle Pakete charger, core, core/circuit und api, Exit 0. Die zusätzlichen Heidelberg-TCP-/Kernprüfungen sind dabei enthalten.
- Firmware-Wrapper gegen den lokalen Safety.1-Stand: 55/55 harte Regressionen, keine erwarteten Fehlschläge.
- Vollständiger Docker-Build einschließlich neu gebauter Weboberfläche: Exit 0, linux/amd64. Lokaler Testtag `evcc-heidelberg:safety1-dual-test`, Version `0.320.0-dev.heidelberg.safety1.dual`, Image-ID `sha256:f2c2a8b649e1e64c3553fcc04c8c5cac20317795cccbd963d1163a0e2415f0a6`. Kein Registry-Upload, daher kein veröffentlichter Registry-Digest.
- Das lokale Image trägt Produktionsquellcommit `097548d48a2a352e7a3e416917a399a3114a87f8`. Nachfolgende Änderungen betreffen ausschließlich Tests, CI und Dokumentation; die produktiven Go-/Template-/Dockerquellen sind identisch. Der GitHub-Build baut den jeweiligen PR-Commit separat.
- Image-Versionsaufruf ohne Netzwerk: erfolgreich. Die erwartete Zeroconf-Diagnose bei vollständig fehlendem Netzwerk ist kein Hardwarezugriff. HTTP/Weboberfläche und konfigurierbarer Port 7071 im eigenen internen Testnetz erfolgreich geprüft.
- Docker-App-Simulation: beide Ladepunkte mit PV aktiv; gemeinsames synthetisches Budget eingehalten; dynamische Verringerung von 18 auf 15 kW verarbeitet; danach 18 kW für den unabhängigen Wiederanlauf des zweiten Ladepunkts; erster Controller offline, zweiter separat stoppbar und wieder ladend; erster Controller nach Wiederkehr wieder steuerbar. Alle Prüfbedingungen bestanden.
- Der erste Lauf des neu erstellten Docker-Testskripts scheiterte an der Annahme einer veralteten API-Hülle `result`. Korrigiert auf die tatsächliche direkte Antwort von `/api/state`; derselbe vollständige Test danach erfolgreich. Keine Änderung am Laufzeitprogramm dafür erforderlich.

Entwurfs-PR: https://github.com/marco-taylor/evcc-heidelberg-1p3p/pull/6 . Basis master 9118fafa720cbcfcd0059a39ec4b61c6a39cd840; frühere PRs bleiben unverändert. Die anfängliche GitHub-Revision 81d384443351be667b5b8d6d4a47c06ff3d291dd wird für die Testskriptkorrektur fortgeschrieben. CI-Ergebnisse sind im PR unter Checks verbindlich nachvollziehbar; zum Zeitpunkt dieser lokalen Dokumentation läuft die erneute GitHub-Prüfung noch. Ausstehende oder fehlgeschlagene Checks dürfen nicht als bestanden gewertet werden.

Lokale vollständige Logs liegen im Git-Metadatenverzeichnis: `.git/safety1-dual-all-tests.log`, `.git/safety1-dual-firmware.log`, `.git/safety1-dual-docker-build.log` und `.git/safety1-dual-runtime-retry.log`. Sie werden nicht in das öffentliche Repository oder Docker-Image aufgenommen. Der Fehlversuch bleibt in `.git/safety1-dual-runtime.log` nachvollziehbar. Die CI wiederholt die relevanten Prüfungen öffentlich und veröffentlicht auf Pull Requests kein Image.
