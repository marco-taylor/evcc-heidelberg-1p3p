# Prüfung des Heidelberg-Ports auf aktuellem master

Stand: 2026-11-15. Ziel: PR #1 konfliktfrei vorbereiten, ohne Merge, Veröffentlichung, Produktionsänderung oder Hardwarekontakt.

## Exakte Basis und Herkunft

| Referenz | Commit / Bedeutung |
| --- | --- |
| Fork-master | 7b590b84e36484f73c94cb813abf2d555dd5f193 |
| Letzter erreichbarer stabiler Upstream-Tag | 0.319.0, 811d77b2c699a0fe0fa5d76ad7623da8cfe7cff4 |
| Upstream-Abstand | 90 Commits nach 0.319.0 |
| Neuer Fork-Versionsstring | 0.320.0-dev.heidelberg.7b590b84, ausdrücklich kein stabiles Release 0.320.0 |
| Bisheriger PR-Kopf vor diesem Auftrag | 2a03a0efa8c50f648c4dd0ae610bc4be66ddd0bd |
| Ursprünglicher lokaler Port | 068022a72158b85e0e5f606d459d2892edc18572 |
| Inhaltsgleiche erste GitHub-Übertragung | 35784a7e356e4261a09300a819018ba696c464b9 |
| Gemeinsamer Git-Tree der beiden ursprünglichen Ports | fd1b0caf036b782b05272de3e95598cb321500b6 |

Die bisherige PR-Spitze wird vor der Aktualisierung zusätzlich gesichert. Der neue Entwicklungsstand wird direkt von master abgeleitet. Übernommen werden nur Heidelberg-Treiberdiff, Simulatortests, zugehörige Dokumentation und die ausdrücklich beauftragte CI-/Container-Einrichtung. Keine anderen Anpassungen aus zivillians Fork und keine Rücknahme der 90 Upstream-Commits.

Die Remote-Übertragung erfolgt über die GitHub-Anbindung. Wegen anderer Commit-Metadaten entsteht dort eine andere Commit-ID; vollständige lokale und entfernte Git-Trees werden vor Abschluss auf Gleichheit geprüft. Der abschließende Commit und aktuelle Status sind im [PR #1](https://github.com/marco-taylor/evcc-heidelberg-1p3p/pull/1) sichtbar.

## Modbus- und Phasenprüfung

Zwischen dem stabilen Tag und dem Basiscommit sind folgende Komponenten unverändert:

- Heidelberg-Basistreiber.
- api.PhaseSwitcher mit Phases1p3p(int) error und dynamische Capability-Implementierung.
- Modbus-Verbindungsschicht und verwendete Modbus-Bibliothek: github.com/grid-x/modbus v1.5.1, ersetzt durch github.com/evcc-io/modbus v0.0.0-20250501165638-8b6f1fbdb7ea.
- Zentrale Phasenverwaltung und Charger-Synchronisation.

Kein Kompatibilitätsumbau der Erweiterung erforderlich. Der Port fügt weiterhin 18 Treiberzeilen hinzu; die Initialisierung ist jetzt gofmt-formatiert. Keine sicherheitsrelevante Logikänderung gegenüber dem bisherigen Port.

Die simulierten Modbus-TCP-Tests prüfen sechs Erkennungsfälle: 1, 3, 0, 2, 65535 und Illegal-Data-Address für unveränderte Wallboxen. Weiterhin werden FC06-Schreibtelegramme für 1P/3P sowie Fehlerweitergabe bei einer Bridge-Ausnahme geprüft. Erkennung darf keine Phasenbefehle schreiben. Initialisierung mit unveränderter Standby-Konfiguration und Timeoutabfrage ist Teil der erwarteten Telegrammfolge. Ausschließlich 127.0.0.1 und kurzlebige Ports, keine echte Bridge.

Sicherheitsrelevante Wirkung gegenüber master: bedingte Registrierung der Umschaltfähigkeit und Weitergabe von Phasenbefehlen an die Bridge. Keine neue Verriegelung, keine neue Gültigkeitsprüfung, keine neue Phasenrückmeldung und keine Änderung an Lastpunkt-Timing, Abschaltfolge oder Firmware. Die unveränderten evcc-Kerntests gehören zur vollständigen Go-Prüfung. Simulation und Kompilierung ersetzen keine Prüfung der konkreten Schützsteuerung.

## Workflows und Prüfungen

Alle 25 master-Workflowdateien bleiben aktiv vorhanden. Die zuvor vorgenommenen Archivverschiebungen werden vollständig aufgehoben. Am Default-Workflow ändern sich nur die Fork-Runnerwahl und zwei gequotete Shell-Umleitungen; alle Jobs und Prüfbefehle bleiben erhalten. Der Claude-Labeler ist eine optionale Metadatenklassifikation und wird im Fork standardmäßig deaktiviert, mit explizitem Opt-in reaktivierbar. Kein Sicherheitscheck wird dafür entfernt.

| Prüfung | Ausführung / Nachweis |
| --- | --- |
| Vollständige Go-Tests auf neuer Basis | make test, Release-Build-Tag, GitHub-gehosteter Runner |
| Heidelberg-Race-Tests | go test -race -tags=release -run '^TestHeidelberg' ./charger |
| Unveränderte Default-Prüfungen | Clean, Build, Test, Lint/Lizenz, UI-Lint/Tests/Lizenz/Build, drei Playwright-Shards und Integration |
| Vollständiger Docker-Build | Originaler Multistage-Build einschließlich UI, linux/amd64, GOAMD64=v1 |
| Smoke-Test | Architekturprüfung und evcc --version mit network=none, ohne Konfiguration/Daten/Geräte |
| Workflow-Syntax | actionlint; proprietäre Depot-Runnerlabels als bekannte Upstream-Labels ausgenommen |
| Veröffentlichungsgrenzen | PR/manueller Start ohne Registry-Login, Publish nur bei freigegebenem master-Push und nach allen benötigten Jobs |
| Unveränderte Anwendung | Diff gegen master beschränkt sich auf Heidelberg-Treiber und dessen neue Testdatei; Kern, Abhängigkeiten und andere Integrationen unverändert |

Verbindliche aktuelle Laufresultate und vollständige Logs: [PR-Prüfungen](https://github.com/marco-taylor/evcc-heidelberg-1p3p/pull/1/checks). Die abschließenden Lauf-IDs und Ergebnisse werden zusätzlich im zugehörigen Notebook-Eintrag festgehalten. Frühere erfolgreiche Läufe auf 0.319.0 sind historische Nachweise und werden nicht als Ergebnis dieser neuen Basis ausgegeben.

CodeQL und andere bestehende Workflow-Ereignisfilter bleiben erhalten; ein nicht vom Ereignis ausgelöster Job wird nicht als bestanden bezeichnet. Die geerbten Veröffentlichungsjobs benötigen weiterhin Upstream-Infrastruktur und werden in diesem Auftrag nicht ausgelöst.

## Berechtigungen und offene Betriebsprüfung

Admin und push sind für marco-taylor bestätigt. Master war bei Prüfung ungeschützt; Rulesets-Liste leer. Die angebotenen Werkzeuge enthalten keinen administrativen Schreibzugriff auf Rulesets/Branchschutz. Die Vorlage unter ./.github/heidelberg-master-ruleset.json wird vorbereitet, aber ist dadurch nicht aktiv. Keine privaten Zugangsdaten wurden aus anderen Anwendungen übernommen.

GHCR wird nicht beschrieben. Der Publish-Job allein besitzt packages:write, verwendet GITHUB_TOKEN und ist zusätzlich über HEIDELBERG_PUBLISH_APPROVED abgesichert. Tatsächliche Paketberechtigung, Sichtbarkeit und Pull des veröffentlichten Digests bleiben bis zu einer separat freigegebenen Erstveröffentlichung offen. Produktiver Unraid-Container, Wallbox und Schütze bleiben unberührt.
