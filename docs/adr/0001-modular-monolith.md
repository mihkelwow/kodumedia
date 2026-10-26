# ADR 0001: modulaarne monoliit enne mikroteenuseid

- Staatus: vastu võetud
- Kuupäev: 2026-07-11

## Kontekst

Koduserver peab olema lihtne paigaldada ja madala ressursikuluga. Samal ajal vajavad codec'id, metadata, kataloog ja kliendid sõltumatuid laienduspunkte.

## Otsus

Server ehitatakse ühe Go-protsessina, kuid sisemiste moodulite vahel kehtib ports-and-adapters sõltuvussuund. Välised protsessid on ainult FFmpeg/FFprobe.

## Tagajärjed

Positiivne:

- üks binaar ja üks port;
- lihtne lokaalne debugging;
- domeeniloogika jääb adapteritest eraldi;
- vajadusel saab töökoormuse hiljem protsessist välja tõsta.

Negatiivne:

- kõik moodulid jagavad protsessi mälu;
- protsessisisene job manager ei ela restarti üle;
- skaleerimine üle mitme masina pole praeguse koduserveri etapi eesmärk.
