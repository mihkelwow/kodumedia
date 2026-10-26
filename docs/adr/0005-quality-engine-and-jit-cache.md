# ADR 0005: Quality Engine ja vajaduspõhine JIT segment-cache

- Olek: vastu võetud
- Kuupäev: 2026-07-19

## Kontekst

KoduMedia varasem universaalne H.264/AAC fallback piiras videotranskodeeringu sageli 1080p-ni ja kasutas üht üldist tarkvaraprofiili. See oli töökindel, kuid võis muuta 4K lähtepildi pehmemaks, kasutada ebaefektiivselt serveri GPU-d ning kodeerida sama sisu korduvalt.

Kogu meediakogu ennetav või öine optimeerimine ei sobi koduserveri kerge ja vajaduspõhise mudeliga. Samuti ei tohi kvaliteediotsus, cache, FFmpegi argumendid ja Reacti player moodustada üht läbipõimunud komponenti.

## Otsus

### Kvaliteedipõhine esitusahel

Planner kasutab järjekorda:

1. Direct Play;
2. originaalvideo stream-copy/remux;
3. ainult heli transkodeerimine;
4. kvaliteetne video transkodeerimine;
5. encoder-speed põhine sujuv fallback.

`Originaal` ja `Maksimaalne` säilitavad lähteresolutsiooni, `Tasakaalus` piirab 1080p-ni ja `Andmesäästlik` 720p-ni.

### Kliendi võimekus

Veebiklient kasutab `MediaCapabilities.decodingInfo()` päringuid ning edastab serverile codec'i, profiili, level'i, bitisügavuse, resolutsiooni, kaadrisageduse, bitikiiruse, HDR-i, smooth ja power-efficient signaalid. Capability-cache'i võti sisaldab aktiivse ekraani HDR/Rec.2020/DPR signaali.

### Encoder'id

FFmpeg adapter tuvastab ja proovib enne kasutamist Intel QSV, NVIDIA NVENC, AMD AMF ning tarkvaralisi H.264/HEVC/AV1 encodereid. Igal encoder-perekonnal on oma kvaliteediskaala. HEVC 10-bit kasutab Main10 profiili. Automaatne software AV1 ei ole vaikimisi lubatud.

### JIT-cache

Optimeeritud segment-cache tekib ainult reaalse video transkodeerimise käigus. Playerile saadetavad fMP4 HLS segmendid on samaaegselt cache'i payload; eraldi teist encode'i ei käivitata.

Cache:

- sisaldab absoluutse ajavahemiku `BaseStartMS` väärtust;
- toetab täieliku ja osalise vahemiku korduskasutust;
- jagab sama cache-võtmega aktiivse encoder'i klientide vahel;
- invalideerub lähtefaili, profiili või pipeline'i muutumisel;
- kasutab mahu piiri ja LRU-koristust;
- ei kustuta aktiivset kirjet.

Öist ega kogu meediakogu eelkodeerimist ei tehta.

### VMAF

VMAF analüüs käivitatakse valikuliselt ainult juba loodud lühikese SDR-segmendi peal, see ei blokeeri playback'i ning mõjutab sama teose järgmist kvaliteedivalikut. HDR-lähte ja SDR-väljundi otsest eksitavat võrdlust ei tehta.

## Tagajärjed

Positiivsed:

- originaalvideo kvaliteet säilib sagedamini;
- 4K ja 10-bit väljund ei ole varjatud 1080p/8-bit piiranguga;
- hardware encoder'eid saab kasutada ilma pimesi nende olemasolu eeldamata;
- sama sisu korduv playback võib kasutada juba loodud segmente;
- cache tekib ainult siis, kui kasutaja seda päriselt vajab;
- TV-kliendid saavad kasutada sama serveripoolset kvaliteediotsust.

Kompromissid:

- cache kasutab kettaruumi ning sisaldab meediast tuletatud vaadatavat sisu;
- encoder-speed hindamine lisab esimese transkodeerimise alguses väikese readiness-kulu;
- hardware encoder'ite kvaliteediparameetrid vajavad eraldi hooldust;
- üks kvaliteediprofiil session'i kohta ei ole veel adaptive multi-bitrate ladder.
