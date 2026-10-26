# ADR 0004: serveri väljalasked on ainult Windows x64 jaoks

- Olek: vastu võetud
- Kuupäev: 2026-07-11

## Kontekst

KoduMedia esmane päris kasutuskeskkond on Windowsi koduserver. Mitme töölaua-operatsioonisüsteemi buildide, paigaldusjuhiste ja regressioonimaatriksi ülalpidamine hajutas arendusressurssi enne, kui põhifunktsioonid olid igapäevaseks kasutuseks piisavalt küpsed.

Android TV ja Tizen jäävad tulevasteks **klientideks**. See otsus puudutab serveriprotsessi väljalaset, mitte HTTP API kliendilepingut.

## Otsus

- serveri käivituspunkt kompileeritakse ainult Windowsile;
- ametlik sihtarhitektuur on Windows x64 (`GOOS=windows`, `GOARCH=amd64`);
- buildi-, käivitus- ja testitöövood kasutavad PowerShelli;
- Docker, POSIX shelli skriptid ning Linuxi ja macOS-i binaarid eemaldatakse;
- domeen, teenused ja adapteriliidesed jäävad operatsioonisüsteemist võimalikult sõltumatuks, et neid saaks kiiresti unit-testida ja et Windowsi-spetsiifiline infrastruktuur ei lekiks äriloogikasse.

## Tagajärjed

Positiivne:

- üks release-pakett ja üks toetatud paigaldustee;
- Windowsi failiteede, protsesside ning tulevase riistvaralise kodeerimise optimeerimine saab olla sihipärane;
- vähem dokumentatsiooni- ja testimaatriksi dubleerimist.

Negatiivne:

- serverit ei väljastata Linuxile ega macOS-ile;
- platvormi muutmine tulevikus nõuab uut teadlikku arhitektuuriotsust ja eraldi käivituspunkti.
