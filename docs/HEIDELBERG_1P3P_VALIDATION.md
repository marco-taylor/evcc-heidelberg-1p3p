# Lokaler Prüfbericht: Heidelberg-1p3p auf evcc 0.319.0

Stand: 2026-11-15. GitHub-Account über die verbundene GitHub-Anbindung als `marco-taylor` bestätigt.

## Ergebnis und offene GitHub-Schritte

Der Port, die lokale Prüfung und der vollständige Docker-Build sind vorbereitet und erfolgreich geprüft. Es wurde kein produktiver evcc-Container verändert und keine Wallbox oder Schützsteuerung angesprochen.

**Ein GitHub-Fork und Pull Request konnten noch nicht erstellt werden.** `marco-taylor/evcc` lieferte bei der abschließenden Abfrage über die GitHub-Anbindung weiterhin HTTP 404. Das kann ein fehlendes oder für die Verbindung nicht zugängliches Repository bedeuten. Die angebotenen GitHub-Werkzeuge unterstützen Repository-Inhalte, Branches und PRs, aber keine Fork-Erstellung. Ein angemeldetes GitHub-CLI beziehungsweise ein hierfür bereitgestelltes API-Zugangstoken ist auf dem Server nicht vorhanden. Zugangsdaten wurden nicht aus anderen Anwendungen übernommen.

Die Person wurde gebeten, über https://github.com/evcc-io/evcc/fork den Fork unter marco-taylor/evcc anzulegen beziehungsweise einen vorhandenen zugänglich zu machen. Anschließend sind Branch-Übertragung, PR-Erstellung und Beobachtung der gehosteten CI noch auszuführen. Es wird kein existierender Branch überschrieben.

- Lokaler Entwicklungsbranch: `feature/heidelberg-1p3p-0.319.0`.
- Geplante PR-Basis: `heidelberg-main`, zunächst auf dem offiziellen Tag 0.319.0.
- Geplante Veröffentlichung: `ghcr.io/marco-taylor/evcc:latest` und `ghcr.io/marco-taylor/evcc:sha-<vollständiger-Commit>`.
- Hosted-CI-Status: noch nicht verfügbar, kein PR angelegt.
- Merge: nicht ausgeführt, Freigabe ausstehend.
- GHCR-Veröffentlichung: nicht ausgeführt, Freigabe ausstehend.
- Prüfung des veröffentlichten Digests: erst nach der freigegebenen Veröffentlichung möglich.

## Quellen und Änderungsprüfung

Upstream-Basis: Tag `0.319.0`, Commit `811d77b2c699a0fe0fa5d76ad7623da8cfe7cff4` aus evcc-io/evcc.

Untersuchter zivillian-Branch: `heidelberg-1p3p`, Commit `f1b8f5a8bfc3225b854332f90605e62f89848e39`. Gemeinsame Basis: `c72753770289f1f5caddf8a7adc04d35c88c9db3`.

Der neue Heidelberg-Treiberdiff ist nach Entfernen der Git-Blob-Indexzeile bytegleich zum aus dieser gemeinsamen Basis ermittelten Patch: **18 hinzugefügte Zeilen**. Es waren keine weiteren Kompatibilitätsänderungen am Treiber notwendig. Die 25 archivierten Upstream-Workflows wurden einzeln gegen die Originaldateien des Basistags verglichen und unverändert übernommen.

Unverändert: evcc-Kernlogik, API-Definitionen, Modbus-Schicht, Abhängigkeitsdateien und die gesamte externe Bridge-Firmware. Sponsoring-Prüfung und vorhandene Lizenzhinweise bleiben erhalten. Die automatische Registrierung der Umschaltfähigkeit und die Register-Schreibfunktion entsprechen dem Originalpatch; sie sind keine neu entworfene Sicherheitslogik.

Der zusätzliche Umfang besteht aus Simulatortests, Fork-Workflow, Docker-Versionsargumenten/AMD64-Basis/Quelllabels sowie Betriebsdokumentation. Alle Änderungen wurden auf offensichtliche Geheimnisse geprüft; die Dokumentation enthält nur öffentliche Quellen, Beispieladressen und vorgesehene Testpfade.

## Build- und Testergebnisse

| Prüfung | Ergebnis |
| --- | --- |
| 207 Go-Pakete außerhalb des exakt abgegrenzten Hauptpakets tariff | Exit 0 im isolierten internen Docker-Netz, einschließlich aller dort vorhandenen Tests |
| Hauptpaket tariff | Exit 0 mit Zugriff auf die öffentliche aWATTar-Preis-API, erneute tatsächliche Ausführung mit count=1 |
| Gesamtumfang | Alle 208 Go-Pakete abgedeckt: 97 Pakete mit Tests, 111 ohne Testdateien |
| Neue Heidelberg-Simulatortests | Sechs Erkennungsfälle und drei Schreib-/Fehlerfälle, im Charger-Testpaket enthalten |
| Fokussierter Race-Test | Exit 0, `go test -race -tags=release -run '^TestHeidelberg' ./charger` |
| Workflow-Syntax | actionlint 1.7.12, Exit 0 |
| Formatierung | Neue Go-Testdatei mit gofmt formatiert |
| Git-Diff-Prüfung | git diff --cached --check ohne Befund |
| Vollständiger Docker-Build einschließlich Weboberfläche und Go-Binary | Exit 0, linux/amd64 |
| Native Ausführung auf Intel N100 | Exit 0, Ausgabe `evcc version 0.319.0-heidelberg.1` |

Die lokalen Go-Prüfungen benutzten Go 1.27.3, zwei CPUs, 8 GiB Speicherlimit und den Build-Tag release. Standardtests liefen wie das Upstream-Makefile mit CGO_ENABLED=0; der Race-Test mit CGO und den notwendigen Compiler-/Linux-Header-Paketen. Der erste Testwerkzeug-Aufbau ohne Linux-I2C-Header wurde korrigiert, ohne Anwendungscode zu ändern.

Die gesamte Testsuite ist nicht vollständig netzwerkfrei ausführbar: EEBus-Tests benötigen eine lokale mDNS-fähige IP-Adresse, das Tarif-Template awattar benötigt öffentliche Preisdaten. Die anfänglichen vollständigen Läufe mit network=none beziehungsweise rein internem Netz schlugen an diesen Voraussetzungen fehl. Diese fehlgeschlagenen Läufe wurden nicht als Erfolg umgedeutet und keine Tests wurden deaktiviert. Stattdessen wurden alle Pakete systematisch in passenden Testprofilen geprüft:

1. Ein eigenes internes Docker-Netz `evcc-hec-review-20261115` ohne LAN-Zugang, ohne veröffentlichte Ports und ohne andere Anwendungskonfigurationen für die 207 lokalen Pakete. Nur das exakt benannte Hauptpaket tariff wird in diesem Profil herausgenommen; dessen Unterpakete bleiben enthalten.
2. Das Hauptpaket tariff separat mit regulärem Internetzugang für seine öffentliche Preisabfrage. Keine produktive Konfiguration, Datenbank oder Geräte waren eingebunden. Damit bleibt kein Go-Paket ungetestet.
3. Die Heidelberg-Race-Tests mit network=none. Der Modbus-Simulator bindet ausschließlich an 127.0.0.1 und verwendet eine zufällige freie Portnummer.

GitHub Actions führt das unveränderte vollständige `make test` auf einem GitHub-gehosteten Ubuntu-Runner mit dessen normaler Testnetzwerkumgebung aus. Dieses gehostete Ergebnis muss nach dem noch ausstehenden PR separat kontrolliert werden. Lokale Ergebnisse werden nicht als gehosteter CI-Erfolg ausgegeben.

## Lokales Docker-Image und N100

Der Host wurde lesend als Intel N100, x86_64, vier CPUs erkannt. Der vorhandene vollständige Upstream-Multistage-Build wurde verwendet, mit Fork-Version und explizitem GOAMD64=v1. Es wurde kein Hardwarebetrieb gestartet.

- Lokaler Image-Name: `evcc-heidelberg:review-0.319.0`.
- Lokale Image-ID: `sha256:857f33566544224080f7c13d8581c3af4bbe67df00cf31657e570777c56779a4`.
- Docker-Architektur: `linux/amd64`.
- OCI-Quelllabel: `https://github.com/marco-taylor/evcc`.
- Build-Argumente für diesen lokalen Review-Build: TAG_NAME=0.319.0-heidelberg.1, SHA=review-811d77b. Die Review-Kennung ist kein veröffentlichter Commit-Tag. Der spätere Workflow verwendet den tatsächlichen GitHub-Commit.

Der Smoke-Test führte lediglich `evcc --version` mit network=none, ohne Konfigurations-/Datenmounts und ohne Geräte aus. Ein unveränderter Upstream-Initialisierer protokolliert dabei eine mDNS-Meldung wegen der fehlenden Schnittstellen. Der Prozess beendet sich dennoch mit Exit 0 und der erwarteten Version.

Die Image-ID ist eine lokale Docker-Konfigurations-ID, kein bereits veröffentlichter GHCR-Manifest-Digest. Ein Start der vollständigen Installation mit produktiver Konfiguration, ein Test von RS485 und ein Nachweis sicherer Schützsteuerung sind ausdrücklich nicht Bestandteil dieses Ergebnisses.

## Veröffentlichungsprüfung

Der aktive Fork-Workflow wurde manuell und mit actionlint geprüft:

- PR-Ereignisse besitzen keinen Publish-Pfad und keine Registry-Anmeldung.
- Manuelle Workflow-Starts besitzen ebenfalls keinen Publish-Pfad.
- Nur ein Push auf heidelberg-main im Repository marco-taylor/evcc ist grundsätzlich berechtigt.
- Zusätzlich muss HEIDELBERG_PUBLISH_APPROVED exakt true sein. Ohne diese noch ausstehende Freigabe bleibt Veröffentlichung gesperrt.
- Tests und Docker-Prüfung müssen beide erfolgreich sein.
- Das getestete Image aus genau demselben Workflow-Lauf wird aus dem Artefakt geladen und mit vollem Commit-SHA sowie latest getaggt.
- Ein alter Commit wird abgewiesen, wenn er nicht mehr der aktuelle Hauptbranch-Kopf ist.
- Nur der Publish-Job hat packages:write. Verwendet wird ausschließlich GITHUB_TOKEN; kein persönliches Veröffentlichungstoken.
- Es gibt keinen Deployment-Zugriff auf Unraid und keine automatische Aktualisierung des dortigen Containers.

Die Repository-Einstellungen, tatsächliche GitHub-Checks, Paketberechtigungen und ein Registry-Pull können vor zugänglichem Fork beziehungsweise erster freigegebener Veröffentlichung noch nicht bestätigt werden. Die konkrete Vorgehensweise steht in `./docs/HEIDELBERG_1P3P.md`.

## Lokale Nachweise

Die vollständigen lokalen Logs liegen absichtlich außerhalb des veröffentlichten Quellumfangs unter `./.cache/logs/`:

- `./.cache/logs/docker-build.log`
- `./.cache/logs/go-tests-local-packages.log`
- `./.cache/logs/tariff-tests-online.log`
- `./.cache/logs/eebus-tests.log`
- `./.cache/logs/heidelberg-race.log`
- `./.cache/logs/actionlint-final.log`
- `./.cache/logs/n100-smoke.log`

Die anfänglichen fehlgeschlagenen Umgebungsprüfungen sind zusätzlich in `./.cache/logs/go-tests.log` und `./.cache/logs/go-tests-isolated.log` erhalten. Der vorbereitete PR-Text liegt lokal unter `./.cache/PR_BODY.md`.
