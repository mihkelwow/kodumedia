# ADR 0003: metadata ja reitingud eraldi provideritena

- Staatus: vastu võetud
- Kuupäev: 2026-07-11

## Kontekst

Üks teenus ei kata alati hästi pealkirja matching'ut, lokaliseeritud kirjeldust, artwork'i ja reitinguid. Teenuste litsentsid ning API-d muutuvad.

## Otsus

`MetadataProvider` tagastab neutraalse `MetadataResult` mudeli. `RatingsProvider` rikastab kirjet välise ID järgi. Teenused proovitakse composition root'is määratud järjekorras.

## Tagajärjed

- TMDb, IMDb ja tulevased allikad on asendatavad;
- domeen ei sisalda ühe teenuse response-mudelit;
- erinevate allikate conflict policy peab olema selgelt teenuses määratud;
- providerite tokenid jäävad serverisse.
