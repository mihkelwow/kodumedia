# ADR 0002: võimekuspõhine playback negotiation

- Staatus: vastu võetud
- Kuupäev: 2026-07-11

## Kontekst

Chrome, Android TV ja Tizeni eri aastakäigud toetavad erinevaid konteineri-, video-, audio- ja subtitle kombinatsioone. User-agent'i järgi kõvakodeeritud tabel vananeb kiiresti.

## Otsus

Klient saadab `ClientCapabilities` payload'i. Serveri puhas `PlaybackPlanner` valib iga kirje jaoks direct play, remux või transcode plaani. FFmpeg profiil saab plaani, mitte HTTP request'i.

## Tagajärjed

- uus klient ei vaja serveris user-agent'i erandit;
- planner on deterministlikult testitav;
- riistvaraline encoder on eraldi profiil;
- capability väärtuste valetamine võib anda esituse vea, mistõttu kliendid vajavad oma teste.
