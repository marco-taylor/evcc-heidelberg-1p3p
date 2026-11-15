# Heidelberg-1p3p-Fork für Unraid

## Versionsbasis und Freigabe

Die freigegebene Arbeitsbasis dieses PR ist der aktuelle Fork-master vom 14.11.2026:

- Basiscommit: `7b590b84e36484f73c94cb813abf2d555dd5f193`.
- Letztes erreichbares stabiles Release: evcc **0.319.0**, Commit `811d77b2c699a0fe0fa5d76ad7623da8cfe7cff4`.
- Abstand: **90 Upstream-Commits**. Das ist ein Entwicklungsstand auf dem Weg zu 0.320.0 und kein veröffentlichtes stabiles 0.320.0.
- Image-Version: `0.320.0-dev.heidelberg.7b590b84`. Der vollständige Build-Commit steht zusätzlich im Binary, OCI-Label und Commit-Tag.
- [Pull Request #1](https://github.com/marco-taylor/evcc-heidelberg-1p3p/pull/1), Zielbranch master. Der bisherige Entwicklungsbranchname mit 0.319.0 bleibt für denselben PR erhalten; er beschreibt nur dessen ursprüngliche Basis.

Master, Produktivcontainer und Hardware bleiben unverändert. Vor dem PR-Merge und der ersten Image-Veröffentlichung ist Marcos ausdrückliche Freigabe erforderlich. Der Publish-Job benötigt zusätzlich `HEIDELBERG_PUBLISH_APPROVED=true`; die Variable wird in diesem Auftrag nicht gesetzt. Es gibt keine automatische Unraid-Aktualisierung.

Geplante Images: `ghcr.io/marco-taylor/evcc-heidelberg-1p3p:latest` sowie `sha-<vollständiger-Buildcommit>`. Beide Tags zeigen bei einer späteren Veröffentlichung auf dasselbe getestete Image. Für Rollback den Digest zusätzlich aufbewahren.

## Treiber und Sicherheitsgrenzen

Die [Erweiterung von zivillian](https://github.com/zivillian/evcc/tree/heidelberg-1p3p), Stand `f1b8f5a8bfc3225b854332f90605e62f89848e39`, wird nur im Heidelberg-Treiber ergänzt:

1. Einmalige Erkennung über Modbus TCP, FC03, Holding-Register 4.
2. Nur der Antwortwert 1 oder 3 registriert dynamisch `api.PhaseSwitcher`.
3. Fehlendes Register, Modbus-Fehler oder andere Werte aktivieren keine Umschaltfähigkeit.
4. Ein Phasenbefehl schreibt mit FC06 den Wert 1 oder 3 auf Register 4 der Bridge. Modbus-Fehler werden weitergegeben.

Die Capability-API, der Heidelberg-Basistreiber, die Modbus-Verbindungsschicht samt Bibliotheksversion und die zentralen Phasen-/Synchronisationsfunktionen sind zwischen 0.319.0 und dem Basiscommit unverändert. Es ist keine API-Anpassung der Erweiterung nötig. Die einzige Änderung am bisherigen Treiberdiff ist gofmt-Ausrichtung.

Sicherheitsrelevante Wirkung der Erweiterung: Ein erkannter Bridge-Treiber erhält erstmals die Fähigkeit, automatische Phasenbefehle aus evcc entgegenzunehmen. Es werden keine eigenen Abschaltfolgen, Wartezeiten oder Verriegelungen hinzugefügt. Der Treiber prüft nicht nochmals den Laststrom und implementiert keine zusätzliche Gültigkeitsprüfung seines internen Integer-Parameters. Es wird keine neue GetPhases-Fähigkeit ergänzt. Die vorhandenen evcc-Abläufe und die Bridge-Firmware bleiben dafür maßgeblich.

Gegenüber dem ursprünglich getesteten Heidelberg-Port wird keine sicherheitsrelevante Schaltlogik geändert. Gegenüber master kommt ausschließlich die beschriebene optionale Fähigkeit hinzu. Der gesamte evcc-Kern, alle übrigen Integrationen, Abhängigkeiten und generierten Dateien bleiben gegenüber master unverändert. Die 90 vorhandenen Upstream-Commits werden erhalten und nicht zurückgenommen.

Das ist eine Quell- und Simulatorprüfung, kein Nachweis sicherer realer Schützsteuerung. Sponsoring-Prüfung und vorhandene Lizenzhinweise bleiben bestehen; insbesondere gilt der Heidelberg-Modulhinweis weiterhin.

## Vorhandene Workflows und zusätzliche CI

Alle 25 vorhandenen Workflowdateien bleiben im aktiven Verzeichnis erhalten. Das frühere Archiv wird nicht übernommen. Keine bestehende Sicherheitsprüfung wird entfernt.

- Der Default-Workflow behält Clean/Generierung/Porcelain, Go-Build, Go-Tests, Go-Lint und Lizenzprüfung, UI-Lint/Tests/Lizenz/Build sowie alle drei Playwright-Integrationsshards und deren Abschlussprüfung.
- Im persönlichen Fork laufen diese Prüfungen auf GitHub-gehosteten Ubuntu-24.04-Runnern statt nicht verfügbarer Depot-Runner. Upstreams bisherige Runnerwahl bleibt für evcc-io/evcc erhalten. Zwei Shell-Umleitungen werden korrekt gequotet.
- CodeQL und alle übrigen Workflows bleiben gegenüber master unverändert. Deren bisherige Ereignisfilter bleiben erhalten; CodeQL läuft zum Beispiel bei master-Push, Zeitplan oder manuellem Start, nicht automatisch in diesem PR.
- Der Claude-PR-Labeler klassifiziert nur Labels. Er ist weder Sicherheitsprüfung noch Build-/Testvoraussetzung und scheiterte hier mangels Anthropic-Zugang. Nur dieser Job ist im Fork standardmäßig ausgeschaltet; mit `HEIDELBERG_CLAUDE_LABELER_ENABLED=true` und gültiger Authentifizierung bleibt er reaktivierbar. Andere Claude-Jobs bleiben erhalten. Die Abschaltung wirkt erst nach einem späteren freigegebenen Merge, weil pull_request_target den Basisbranch verwendet.
- Upstream-Veröffentlichungsworkflows und deren Ziele bleiben erhalten und werden nicht auf eigene Pakete umgebogen. Sie benötigen weiterhin die ursprüngliche Infrastruktur/Zugangsdaten. In diesem Auftrag wird kein solches Ereignis ausgelöst und nichts veröffentlicht.

Der zusätzliche Heidelberg-Workflow führt bei PRs gegen master alle Go-Tests, die Heidelberg-Race-Tests, den vollständigen AMD64-Docker-Build einschließlich UI sowie einen Versionsstart mit network=none aus. Tests und Docker-Job haben contents:read, keine Registry-Anmeldung und kein packages:write. PRs und manuelle Starts veröffentlichen nichts.

Nach einem später freigegebenen master-Push wartet der Publish-Job zusätzlich auf einen wiederverwendeten vollständigen Default-Prüflauf. Erst dann darf er mit GITHUB_TOKEN und ausschließlich im Publish-Job packages:write das getestete Image-Artefakt laden und den Commit-Tag vor latest veröffentlichen. Ein inzwischen überholter master-Kopf wird abgewiesen. Das kann nach einem master-Push einen zusätzlichen Default-Lauf bedeuten; es gibt dadurch keinen ungeprüften Veröffentlichungsweg.

## Branchschutz und Berechtigungen

Der verbundene Account marco-taylor besitzt laut Repository-API admin und push. Master ist derzeit ungeschützt; es bestehen keine Rulesets. Die bereitgestellte GitHub-Anbindung kann Branches, Inhalte und PRs bearbeiten, aber keine Branchschutz-/Ruleset-Administration durchführen. Eine lokale authentifizierte GitHub-CLI ist nicht vorhanden. Der Schutz ist daher nicht als aktiv eingerichtet zu verstehen.

Die Vorlage `./.github/heidelberg-master-ruleset.json` ist für die Repository-Rulesets-API vorbereitet: PR-Pflicht, aktuelle Basis für Pflichtprüfungen, aufgelöste Diskussionen, keine Löschung/Force-Pushes und kein Bypass. Erforderliche Checks: Go tests, Docker linux/amd64, Clean, Build, Test, Lint, UI und Integration. Eine zusätzliche Fremd-Reviewpflicht ist bei einem allein verwalteten Fork nicht automatisch sinnvoll: Der PR-Autor kann seinen eigenen PR nicht genehmigen. Die ausdrückliche menschliche Mergefreigabe bleibt davon unabhängig erforderlich. Die Vorlage verlangt deshalb null zusätzliche formale Reviews.

[GitHub beschreibt die erforderlichen Ruleset-Administrationsrechte und API-Felder hier](https://docs.github.com/en/rest/repos/rules#create-a-repository-ruleset). Vor Einrichtung die tatsächlich angezeigten Check-Namen mit dem aktuellen PR abgleichen. Actions-Richtlinien und tatsächliche GHCR-Paketrechte sind nicht allein durch eine Workflowdatei bewiesen. Die Schreibberechtigung und anonyme Lesbarkeit des künftigen Pakets werden erst nach freigegebener Erstveröffentlichung praktisch prüfbar.

## Firmware-Kompatibilität

Die offizielle [Bridge-Firmware 0.0.2](https://github.com/zivillian/heidelberg-1p3p/releases/tag/0.0.2) verwendet das erwartete Holding-Register 4 und die Phasenwerte 1/3. Die Protokollübereinstimmung wurde anhand der Quellen geprüft. Ein Startwert 0 aktiviert die Fähigkeit im laufenden evcc-Prozess nicht; eine spätere Wiedererkennung wird nicht ergänzt.

Die offizielle Binärdatei hat 964352 Bytes und SHA256 `d87158b73c9c8c8f5f4bb3762edb71a917285def7cfa3cd158bc7e963b55699c`. Sie ist ein ESP32-Anwendungsimage, kein vollständiges Erstflash-Paket mit Bootloader und Partitionstabelle. Flashbelegung, konkrete DT-R002 DEV mit RS485 und sichere Schützsteuerung bleiben eigenständige Hardwareprüfungen. In diesem Auftrag wird keine Firmware gebaut, geflasht oder betrieben.

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

In dieser evcc-Entwicklungsbasis wird der frühere YAML-Parameter `phases` ignoriert. Die Phaseneinstellung wird später in der Oberfläche geprüft beziehungsweise vorgenommen. Automatik erst nach Freigabe und Hardwareprüfung wählen. Nur bei erfolgreicher Erkennung stellt der Treiber die Umschaltfähigkeit bereit. Wenn die Bridge beim evcc-Start nicht 1 oder 3 meldet, bleibt sie in diesem Prozess deaktiviert; es gibt in diesem Port keine nachträgliche Wiedererkennung.

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
4. Bei Upstream-Aktualisierungen vorhandene Workflows und Sicherheitsprüfungen erhalten, neue Anforderungen und Runner nachvollziehbar anpassen. Nicht pauschal archivieren oder Prüfungen deaktivieren.
5. Go-/UI-Versionen und Docker-Basisimages aus der neuen Upstream-Version gemeinsam prüfen. Versionsargument im Docker-Build und Workflow aktualisieren.
6. Simulator-, Go- und Docker-Prüfungen ausführen. PR gegen master erstellen, Änderungen dokumentieren und vor dem Merge freigeben lassen. Keine automatischen Upstream-Merges auf die freigegebene Hauptlinie.

## Prüfumfang

Der Zielhost wurde lesend als Intel N100 mit x86_64 und vier CPUs erkannt. Gebaut wird `linux/amd64`, die Go-CPU-Basis ist v1. Der lokale Image-Smoke-Test benutzt kein Netzwerk und keine Gerätemounts.

Die automatischen Treibertests prüfen sechs Erkennungsfälle sowie erfolgreiche 1P-/3P-Schreibbefehle und eine Fehlerantwort. Sie kontrollieren die erwarteten Modbus-Telegramme, einschließlich der unveränderten Initialisierung, und dass die neue Erkennung keinen Phasenwechsel schreibt. Es werden weder echte Wallboxen noch Platinen kontaktiert.

Konkrete lokale und GitHub-CI-Ergebnisse sowie offene Freigabeschritte stehen im zugehörigen PR beziehungsweise im lokalen Prüfbericht. Eine Prüfung des später veröffentlichten Registry-Digests ist vor der ersten Veröffentlichungsfreigabe nicht möglich und wird nicht vorgetäuscht.
