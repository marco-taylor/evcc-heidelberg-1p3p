# Heidelberg-1p3p-Fork für Unraid

## Stand und Freigabe

Dieser Fork basiert auf evcc **0.319.0**, Commit `811d77b2c699a0fe0fa5d76ad7623da8cfe7cff4`.
PR-Ziel und spätere Veröffentlichungslinie ist **master**. Der Fork-master steht derzeit auf `7b590b84e36484f73c94cb813abf2d555dd5f193`, 90 Commits nach 0.319.0. Die Wahl zwischen dieser neueren Basis und dem ursprünglich getesteten Release ist vor einem Merge ausdrücklich zu klären. Ein PR-Mergelauf prüft die Kombination mit dem aktuellen master und ist kein Nachweis eines unveränderten 0.319.0-Releases.

Der erste Merge und die erste Veröffentlichung benötigen Marcos ausdrückliche Freigabe. Der Workflow veröffentlicht erst, wenn die Repository-Variable `HEIDELBERG_PUBLISH_APPROVED` exakt `true` ist. Solange die Variable fehlt oder anders gesetzt ist, erfolgen nur Prüfungen.

Das geplante Image heißt `ghcr.io/marco-taylor/evcc-heidelberg-1p3p:latest`. Dazu wird derselbe geprüfte Image-Inhalt als `ghcr.io/marco-taylor/evcc-heidelberg-1p3p:sha-<vollständiger-Commit>` veröffentlicht. Ein Pull Request veröffentlicht keine Registry-Tags. Ein manueller Workflow-Start veröffentlicht ebenfalls nichts.

Eine Veröffentlichung ist keine Installation auf Unraid. Es gibt keinen Zugriff der Actions auf den Server und keinen automatischen Container-Update-Schritt.

## Herkunft und Änderungen

Quellen:

- [Offizieller evcc-Tag 0.319.0](https://github.com/evcc-io/evcc/tree/0.319.0)
- [Heidelberg-Fork von zivillian](https://github.com/zivillian/evcc/tree/heidelberg-1p3p), geprüfter Stand `f1b8f5a8bfc3225b854332f90605e62f89848e39`
- Gemeinsame Basis des untersuchten Patches: `c72753770289f1f5caddf8a7adc04d35c88c9db3`
- [Bridge-Projekt und Dokumentation](https://github.com/zivillian/heidelberg-1p3p)

Der eigentliche Treiber-Port entspricht dem 18-Zeilen-Diff aus zivillians Fork unverändert:

1. Der Heidelberg-Treiber liest einmal bei der Initialisierung Holding-Register **4** mit Function Code 03.
2. Nur bei Antwortwert **1** oder **3** wird die optionale `api.PhaseSwitcher`-Fähigkeit registriert.
3. Bei fehlendem Register, Modbus-Fehler oder anderen Werten bleibt die normale Heidelberg-Integration ohne diese Fähigkeit erhalten.
4. Ein Phasenwechsel sendet Function Code 06, Register **4**, Wert **1** oder **3**, an die externe Bridge. Die Fehlerantwort der Bridge wird weitergegeben.

Die bereits vorhandene Capability-API von 0.319.0 ist kompatibel. Es werden keine alten Änderungen an Decorator-Generatoren und keine anderen Commits aus dem fremden Fork übernommen. Der normale Heidelberg-Treiber behält Sponsoring-Prüfung, Heartbeat, Stromregelung und Statusbehandlung.

Die Bridge-Firmware und deren Registerprotokoll, Schütz-Verriegelungen, Abschaltbedingungen und Wartezeiten werden nicht geändert. Auch evccs zentrale Lastpunkt- und Umschaltlogik bleibt unverändert. Das ist keine eigenständige Sicherheitsprüfung der ursprünglichen Umschaltlösung.

Weitere Änderungen:

- Lokale Modbus-TCP-Simulatortests für Erkennung, unveränderte Wallbox, nicht unterstützte Werte, genaue Schreibtelegramme und Fehlerweitergabe. Es werden ausschließlich Loopback-Verbindungen geöffnet.
- Der vorhandene Docker-Multistage-Build einschließlich kompletter Weboberfläche bleibt erhalten. Versions- und Commit-Argumente sowie Fork-Quelllabels werden ergänzt. `GOAMD64=v1` vermeidet die Voraussetzung neuerer x86-Instruktionsstufen.
- Die geerbten Workflows liegen unverändert unter `./.github/upstream-workflows/`. Sie benötigen teils evcc-interne Depot-Runner, Dienste und Geheimnisse und dürfen im persönlichen Fork keine unbeabsichtigten Release-Pfade eröffnen. Aktiv ist stattdessen `./.github/workflows/heidelberg.yml`.
- Keine Änderung an Produktionskonfiguration, Datenbank, produktivem Container oder Platinen.

Die vorhandenen Lizenzhinweise bleiben erhalten. Einzelne evcc-Module, darunter der Heidelberg-Treiber, tragen einen gesonderten Rechtehinweis. Dieser Fork erklärt sie nicht nachträglich für MIT-lizenziert und entfernt keine Sponsoring-Anforderung. Maßgeblich bleiben die jeweiligen Upstream-Bedingungen.

## GitHub Actions und erste Veröffentlichung

Der Workflow verwendet GitHub-gehostete Ubuntu-24.04-Runner. Es werden keine eigenen Runner auf Unraid eingesetzt.

| Ereignis | Prüfung | Veröffentlichung |
| --- | --- | --- |
| PR gegen master | Go-Tests aller Pakete, fokussierte Race-Tests, vollständiger AMD64-Docker-Build und Versions-Smoke-Test | Nie |
| Push/Merge auf master | Dieselben Prüfungen | Nur im richtigen Repository und mit Freigabevariable |
| Manueller Workflow-Start | Dieselben Prüfungen | Nie |
| Push auf feature/heidelberg-1p3p-0.319.0 | Dieselben Prüfungen auf der Entwicklungsbasis, auch bei blockiertem PR-Merge | Nie |
| Andere Branches | Kein neuer Fork-Workflow | Nie |

Die Test- und Docker-Jobs haben nur `contents: read`. Nur der Publish-Job besitzt zusätzlich `packages: write`. Die Anmeldung bei GHCR erfolgt mit `GITHUB_TOKEN`, ohne persönliche Zugangstoken oder Docker-Hub-Geheimnisse. PR-Jobs erhalten keine Registry-Anmeldung. Es wird kein privilegierter `pull_request_target`-Workflow verwendet.

Der Publish-Job lädt das vom Docker-Job dieses Push-Laufs getestete Image. Es wird nicht nochmals anders gebaut. Er prüft außerdem, dass der Commit noch der Kopf von master ist, damit ein alter wiederholter Lauf nicht versehentlich latest zurücksetzt. Erst wird der Commit-Tag, dann latest gepusht.

Vor dem ersten Merge durch den Repository-Eigentümer erledigen beziehungsweise überprüfen:

1. Der öffentliche Fork `marco-taylor/evcc-heidelberg-1p3p` ist erreichbar; der verbundene Account besitzt admin und push. Den Versionsunterschied zwischen Fork-master und 0.319.0 vor dem Merge klären. Kein bestehender Hauptbranch wird zurückgesetzt.
2. Den PR und die Sicherheitsgrenzen prüfen. Der Standardbranch ist bereits master. Bis zum Merge liegen dort noch die geerbten Upstream-Workflows; deren Ausführung muss in den Actions-Einstellungen geprüft werden.
3. Actions im Fork aktivieren und auf die verwendeten Actions beschränken, soweit gewünscht. Für master Branchschutz gegen direkte Pushes und die Pflichtprüfungen **Go tests** und **Docker linux/amd64** konfigurieren.
4. Die Repository-Variable `HEIDELBERG_PUBLISH_APPROVED` bleibt bis zur ausdrücklichen Freigabe ungesetzt. Dadurch kann auch ein versehentlicher früher Push unseres Workflows keine Veröffentlichung auslösen.
5. Erst nach Marcos ausdrücklicher Merge- und Veröffentlichungsfreigabe `HEIDELBERG_PUBLISH_APPROVED=true` setzen und den geprüften PR zusammenführen. Der Merge-Push startet die Veröffentlichung automatisch. Auch spätere Merges benötigen die vereinbarte Freigabe, lösen aber keinen zusätzlichen Deployment-Review aus.
6. Nach Veröffentlichung GHCR-Sichtbarkeit und Paketberechtigungen prüfen. Öffentliche Forks bedeuten nicht automatisch öffentlich lesbare Containerpakete. Für anonymes Unraid-Pullen muss das Paket öffentlich sein; ansonsten separat authentifizieren.
7. Commit-Tag und Registry-Digest notieren. Ein abschließender Test genau des veröffentlichten Digests auf dem N100 steht erst dann an. Vor Freigabe ist nur ein lokaler Test des vorgeschlagenen Images zulässig.

Die administrative Einrichtung von Default-Branch, Branchschutz und Freigabevariable ist nicht allein durch die Workflowdatei bewiesen. Noch nicht durchgeführte Schritte müssen im PR offen bleiben.

Dokumentation: [GitHub GHCR-Veröffentlichung](https://docs.github.com/en/actions/tutorials/publish-packages/publish-docker-images).

## Firmware-Kompatibilität

Die offizielle [Bridge-Firmware 0.0.2](https://github.com/zivillian/heidelberg-1p3p/releases/tag/0.0.2) verwendet das vom Port erwartete Modbus-TCP-Protokoll mit Holding-Register 4 und Phasenwerten 1 beziehungsweise 3. Diese Protokollübereinstimmung ist anhand der Quellen geprüft. Ein Startwert 0 aktiviert die Umschaltfähigkeit in diesem evcc-Prozess nicht.

Die separat untersuchte offizielle Binärdatei hat 964352 Bytes und SHA256 `d87158b73c9c8c8f5f4bb3762edb71a917285def7cfa3cd158bc7e963b55699c`. Sie ist ein gültiges ESP32-Anwendungsimage, kein vollständiges Erstflash-Paket mit Bootloader und Partitionstabelle. Eine bestätigte Flashbelegung und die Inbetriebnahme der konkreten DT-R002 DEV mit RS485 bleiben eigenständige offene Prüfungen. Es gibt hier keine Flashfreigabe und keinen Nachweis sicherer Schützsteuerung.

## Konfiguration der Bridge

Die externe Bridge spricht **Modbus TCP**, nicht Modbus RTU über TCP. Als Ziel wird die Bridge-IP verwendet, nicht ein weiterer Adapter, der die Wallbox direkt erreicht. Beispiel nur für die spätere, ausdrücklich freigegebene Einrichtung, mit einer nicht routbaren Dokumentationsadresse:

```yaml
chargers:
  - name: heidelberg_bridge
    type: heidelberg
    uri: 192.0.2.10:502
    rtu: false
    id: 1

loadpoints:
  - title: Heidelberg
    charger: heidelberg_bridge
    mode: "off"
```

Das Beispiel ist ein Ausschnitt, keine vollständige Produktionskonfiguration. Die tatsächlich konfigurierte Wallbox-ID muss mit Bridge und Wallbox übereinstimmen. Es ist kein neuer Charger-Typ und kein Sponsoring-Bypass erforderlich. Das normale RS485-Heidelberg-Template kann für diesen Zweck versehentlich `rtu: true` erzeugen; deshalb ist hier die direkte Treiberkonfiguration gezeigt.

In evcc 0.319.0 wird der frühere YAML-Parameter `phases` ignoriert. Die Phaseneinstellung wird später in der Oberfläche geprüft beziehungsweise vorgenommen. Automatik erst nach Freigabe und Hardwareprüfung wählen. Nur bei erfolgreicher Erkennung stellt der Treiber die Umschaltfähigkeit bereit. Wenn die Bridge beim evcc-Start nicht 1 oder 3 meldet, bleibt sie in diesem Prozess deaktiviert; es gibt in diesem Port keine nachträgliche Wiedererkennung.

Auch `mode: off` und reine Diagnoseaufrufe garantieren keinen schreibfreien Gerätestart: Der unveränderte Heidelberg-Konstruktor schreibt bereits die Standby-Konfiguration. Deshalb die neue Konfiguration nicht zur Probe gegen eine produktiv angeschlossene Bridge starten. Ein hardwarefreier Smoke-Test verwendet ausschließlich `--version` und `--network none`.

## Installation auf Unraid, erst nach gesonderter Freigabe

Für einen anfänglichen hardwarefreien Test nach der späteren Veröffentlichung:

```bash
docker pull ghcr.io/marco-taylor/evcc-heidelberg-1p3p:sha-VOLLSTAENDIGER_COMMIT
docker run --rm --network none --entrypoint evcc \
  ghcr.io/marco-taylor/evcc-heidelberg-1p3p:sha-VOLLSTAENDIGER_COMMIT --version
```

Keine produktive Konfiguration, Datenbank, Geräte, privilegierter Modus oder Host-Netzwerk für diesen Test. Er prüft nur Startfähigkeit und Architektur, nicht Ladebetrieb oder Schützsicherheit.

Für eine spätere Installation über Unraids Docker-Verwaltung:

- Zunächst ein eigenes Template und einen separaten Container verwenden. Repository `ghcr.io/marco-taylor/evcc-heidelberg-1p3p:sha-VOLLSTAENDIGER_COMMIT` oder einen notierten Digest wählen. latest ist der bequeme bewegliche Tag, ein Commit-Tag beziehungsweise Digest ist für kontrollierte Updates vorzuziehen.
- Konfiguration und Daten unter einem separaten Appdata-Verzeichnis ablegen. Beispielpfade für den späteren Test: `/mnt/user/appdata/evcc-heidelberg-test/evcc.yaml` und `/mnt/user/appdata/evcc-heidelberg-test/data/`. Produktionsdateien nicht gleichzeitig von zwei Containern öffnen lassen.
- Die Konfiguration beispielsweise nach `/etc/evcc.yaml:ro` und das separate Datenverzeichnis nach `/data` mounten. Anwendungsargumente `--config /etc/evcc.yaml --database /data/evcc.db` verwenden.
- Bei einem später freigegebenen UI-Test mit simulierten Geräten Bridge-Netzwerk und eine freie Host-Portzuordnung, etwa 17070 nach 7070, verwenden. Kein Host-Netzwerk und keine echten Geräte für diesen Test.
- Für die reale Migration zuerst eine konsistente Sicherung von Konfiguration, Datenbank und Unraid-Template erstellen, dann in einem geplanten Wartungsfenster umstellen. Hier wird das ausdrücklich nicht ausgeführt.
- Keine Watchtower-Regel und kein automatisches Update-Plugin für diesen Container aktivieren. Die Veröffentlichung eines neuen latest darf den laufenden Container nicht selbst ändern.

Die genauen bisherigen Produktionspfade und Containerparameter wurden nicht ausgelesen und sind keine Annahme dieses Dokuments.

## Kontrollierte Updates und Rollback

Vor einem Update notieren: bisheriger Image-Digest beziehungsweise Commit-Tag, Unraid-Template, komplette Konfiguration und konsistente Datenbanksicherung. Bei einer späteren produktiven Umstellung die Datenbank erst nach regulärem Stop sichern oder ein geeignetes SQLite-Backup-Verfahren verwenden. Eine lose Kopie einer geöffneten SQLite-Datei kann WAL-Daten übersehen.

Ein Update erfolgt manuell: neue PR-Version prüfen, nach Freigabe mergen und veröffentlichen, neuen Digest prüfen und erst dann einen separaten Test beziehungsweise die geplante Unraid-Umstellung durchführen. Kein automatisches Aktualisieren des produktiven Containers.

Für einen Rollback den vorher notierten Image-Digest/Commit-Tag zusammen mit der passenden vorherigen Konfiguration und Datenbanksicherung wiederherstellen. Ein älteres Image allein ist nach Datenbankmigrationen nicht zwingend ausreichend. latest nicht als historische Rollback-Adresse verwenden. Alte Images und Sicherungen erst nach erfolgreicher Abnahme bereinigen.

## Wartung des Forks

1. Upstream-Remote auf evcc-io/evcc belassen, Fork als origin, zivillians Repository nur als Referenzquelle. Upstream-Tags separat abrufen.
2. Für ein neues evcc-Release einen neuen Entwicklungsbranch erstellen. Den kleinen Heidelberg-Treiberdiff bewusst auf die neue Basis übernehmen und mit dem letzten freigegebenen Stand vergleichen.
3. Änderungen an Registerprotokoll, Auswahl der Phasen, Timing oder Steuerungslogik benötigen eine gesonderte Prüfung. Ein Upstream-Merge ist dafür keine automatische Freigabe.
4. Bei Upstream-Aktualisierungen neu hinzugekommene Workflows ebenfalls aus dem aktiven Verzeichnis nehmen oder ausdrücklich prüfen. Archivierte Workflows werden nicht automatisch ausgeführt.
5. Go-/UI-Versionen und Docker-Basisimages aus der neuen Upstream-Version gemeinsam prüfen. Versionsargument im Docker-Build und Workflow aktualisieren.
6. Simulator-, Go- und Docker-Prüfungen ausführen. PR gegen master erstellen, Änderungen dokumentieren und vor dem Merge freigeben lassen. Keine automatischen Upstream-Merges auf die freigegebene Hauptlinie.

## Prüfumfang

Der Zielhost wurde lesend als Intel N100 mit x86_64 und vier CPUs erkannt. Gebaut wird `linux/amd64`, die Go-CPU-Basis ist v1. Der lokale Image-Smoke-Test benutzt kein Netzwerk und keine Gerätemounts.

Die automatischen Treibertests prüfen sechs Erkennungsfälle sowie erfolgreiche 1P-/3P-Schreibbefehle und eine Fehlerantwort. Sie kontrollieren die erwarteten Modbus-Telegramme, einschließlich der unveränderten Initialisierung, und dass die neue Erkennung keinen Phasenwechsel schreibt. Es werden weder echte Wallboxen noch Platinen kontaktiert.

Konkrete lokale und GitHub-CI-Ergebnisse sowie offene Freigabeschritte stehen im zugehörigen PR beziehungsweise im lokalen Prüfbericht. Eine Prüfung des später veröffentlichten Registry-Digests ist vor der ersten Veröffentlichungsfreigabe nicht möglich und wird nicht vorgetäuscht.
