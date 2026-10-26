# KoduMedia teekaart

## 0.6.4 — valmis: väljalaske terviklus ja lame Settings UI

- executable'i sisse embeditud SemVer versioon ja build marker;
- build-info fail serveri ning updateri suuruse ja SHA-256 räsiga;
- pakendamise range VERSION/embedded/build-info/binaarräsi kontroll;
- `dev` versiooniga vigase väljalaske automaatne blokeerimine;
- Meediakogude läbipaistev juursektsioon ja üks top-level pind iga kogu kohta;
- seotud kaustade ning kausta lisamise lamedad read ilma kast-kastis kompositsioonita;
- jagatud CVI select säilib kvaliteediseadetes ja uue kogu vormis.

## 0.6.3 — valmis: Settings UI CVI ühtlustamine

- Pildikvaliteedi neli rippmenüüd kasutavad ühist KoduMedia CVI native-select komponenti;
- sama komponent on kasutusel „Loo oma kogu” sisutüübi valikus;
- Meediakogude vaade kasutab pehmeid kihilisi kaarte, selget visuaalset hierarhiat ja eraldi kaustaplokke;
- eemaldatud on järjestikuste eraldusjoonte ehk „kriipsutatud” tabeli ilme;
- lisatud on tumeda ja heleda teema hover-, focus-, disabled- ja responsive-olekud;
- native select säilitab klaviatuuri, ekraanilugeja, puute ja operatsioonisüsteemi menüü töökindluse.

## 0.6.2 — valmis: HDR hardware pipeline ja kontrollitud taastumine

- HDR/4K automaatne NVENC → QSV → AMF → software järjestus;
- eraldi `-hwaccels` capability-probe ning CUDA/QSV/D3D11VA tuvastus;
- päris ühekaadriline hardware encoder'i käivitustest ja 30-sekundiline negatiivse tulemuse TTL;
- NVIDIA NVDEC/CUDA decode koos NVENC encode'iga;
- hardware decoder'i vea korral software-decode retry ning encoder'i vea korral järgmise pere proov;
- HDR→SDR varajane sihtresolutsiooni skaleerimine enne float tone mapping'ut;
- encoder-speed kontroll ka fallback-session'itele;
- 1080p → 720p → 480p stabiilsusredel;
- decoder'i, encoder'i ja fallback'i põhjuse täpne playeri diagnostika;
- Quality Engine JIT-cache revision v4.

Teekaart on järjestatud nii, et igas etapis säiliks modulaarne arhitektuur, Windowsi serveri igapäevane kasutatavus ja kerge klient.

## 0.6.1 — valmis: puhvri töökindlus ja diagnostika

- algpuhvri, rebufferingu ja voo ettevalmistamise eraldi olekud;
- Direct Play, remux, 1080p ning 4K/HDR jaoks eraldi puhvri- ja madala veemärgi poliitikad;
- FFmpegi suurem kontrollitud initial burst ja 1.30–1.50× tootmisvaru;
- 1.20–1.25× encoder-speed turvavaru, libisev keskmine ja runtime stats;
- korduva stall'i või jätkusuutmatu encoder'i korral samast kohast 1080p/720p kvaliteedifallback;
- codec'i/resolutsiooni/profiili põhine VBV maxrate ja bufsize;
- täpne playeri kvaliteediinfo ning puhvri, encoder'i, bitrate'i, stallide ja cache'i diagnostika;
- versioonitud playback-session stats endpoint tulevastele TV-klientidele.

## 0.6.0 — valmis: Quality Engine

- Direct Play ja video stream-copy eelistamine enne video transkodeerimist;
- ainult heli transkodeerimine, kui video ise sobib;
- `Originaal`, `Maksimaalne`, `Tasakaalus` ja `Andmesäästlik` kvaliteediprofiil;
- täpne `MediaCapabilities.decodingInfo()` codec/profiil/level/bitdepth/resolutsioon/fps/bitrate/HDR raport;
- ekraani HDR/Rec.2020/DPR signaali arvestav capability-cache;
- Intel QSV, NVIDIA NVENC ja AMD AMF H.264/HEVC/AV1 encoder'i tuvastus ning päris käivitustest;
- encoder-perekonna põhised kvaliteediskaalad ja HEVC Main10;
- automaatne software AV1 vältimine ning encoder-speed põhine sujuv fallback;
- vajaduspõhine jagatud JIT HLS segment-cache ilma öise eelkodeerimiseta;
- täieliku ja osalise cache'i korduskasutus ka jätkatud vaatamisel;
- cache'i suuruspiir, LRU-koristus, lähtefaili/profiili invalidatsioon ja grace-periood;
- mitteblokeeriv väikese prooviala VMAF analüüs ning järgmise esituse kvaliteedinõuanne;
- settings schema v4, Pildikvaliteedi UI ja laiendatud OpenAPI leping.

## 0.2.2 — valmis

- release-name parser ning filename/parent-folder metadata kandidaadid;
- TMDb tulemuste skoorimine, nõrga vaste tagasilükkamine ja keelefallback;
- olemasoleva TMDb ID põhine metadata repair;
- Windows x64 brauseripõhine ZIP-uuendus localhostist;
- manifesti, SemVer versiooni, failisuuruste ja SHA-256 kontroll;
- eraldi updater-helper, rakendusfailide varundus, health-check ja rollback;
- update API, OpenAPI leping, PowerShell package workflow ja regressioonitestid.

## 0.2.1 — valmis

- native Windowsi faili- ja kaustavalija localhost-brauserile;
- kerge, tsentreeritud ja piiratud serverifailisüsteemi fallback LAN-kliendile;
- automaatne TMDb metadata skannimisel, käivitamisel ja settings repair'i järel;
- üksiku faili metadata nupu eemaldamine;
- vana `data` kataloogi missing-artwork backfill;
- IMDb ametlik URL vaikeallikana;
- serveripoolne ETag/Last-Modified cache refresh;
- käivituse, ööpäevase ja käsitsi IMDb värskenduse job;
- kompaktne reitinguindeks ja batch-kataloogiuuendus;
- settings schema v1 → v2 migratsioon;
- picker, dataset, backfill ja frontend regressioonitestid.

## 0.2.0 — valmis

- brauseris muudetavad FFmpeg, FFprobe ja metadata seaded;
- seadete testimine enne salvestamist;
- serveripoolne kausta- ja failisirvija;
- automaatne film/episood/muusika klassifitseerimine;
- TMDb ametlikud filmi-, seriaali- ja episoodinimed;
- sünteetiline seriaali parent;
- seriaal → hooaeg → episood hierarhia;
- tüübipõhine avaleht;
- algse failinime nähtavus enne playback'i;
- OpenAPI leping ja regressioonitestid.

## 0.5.2 — valmis

- püsiv ühe serveriprofiili vaatamisprogress ja **Jätka vaatamist**;
- progressiriba filmide, episoodide ja seriaalide kaartidel;
- faili `Date modified` põhine viimati lisatud järjestus;
- ametliku release date'i järgi uued episoodid;
- ametlikult ilmunud vs serveris saadaval hooajad ja osad;
- skannimisel püsiv FFprobe tehniline analüüs;
- lamedam seriaali hooaegade ja episoodide UI.


## 0.4.0 — valmis

- vähese kroomiga kinoelamuslik veebiliides ilma püsiva külgmenüü ja kast-kastis paigutuseta;
- hõljuv ülanavigatsioon, dünaamiline hajutatud taust ning eraldi tume ja hele kujundussüsteem;
- laiaformaadilised jätkamise/esiletõstmise kaardid, õhulised posteriread ja posteriruudustikud;
- IMDb skoor thumbnail'ide peal ning selge klaviatuuri- ja TV-fookus;
- täisekraaniline filmi- ja seriaali detailvaade, näitlejate rida ning lamedam tehnilise info paigutus;
- globaalne filmi-, seriaali-, episoodi- ja muusikaotsing ülamenüüst, `/` või `Ctrl+K` / `Cmd+K` abil;
- serveripoolne mitme sõltumatu märksõna otsing pealkirja, osatäitjate, žanri, kirjelduse, albumi ja failinime järgi;
- 180 ms viitega päringud, vananenud request'ide katkestamine ning sisutüübi järgi grupeeritud tulemused;
- varasem Windows tray, startup, self-update, playback, subtiitrid ja metadata töövood säilivad.

## 0.3.4 — valmis

- native Windows system tray ilma püsiva konsooliaknaga;
- vaikimisi bold **Ava Kodu**, startup-lüliti, kõikide kogude skannimine ja seadete avamine;
- kasutajapõhine Windows Run registri startup ilma administraatoriõiguseta;
- single-instance mutex ja olemasoleva kliendi avamine duplikaatkäivituse korral;
- self-update restart sama tray protsessina;
- Kodu logo veebikliendis ning Kodu ikoon trayl ja faviconina;
- installikausta `.env` automaatne laadimine ning failipõhine logimine.

## 0.3.3 — valmis

- brauseri dünaamilisest TextTrack olekumasinast sõltumatu WebVTT overlay;
- embedded-, adjacent- ja downloaded tekstisubtiitrite ühine esitusahel;
- absoluutse lähtepositsiooni järgi subtitle sync seek'i ja HLS sessioni restartide järel;
- Unicode/RTL, kattuvate cue'de, WebVTT märgendite ja entiteetide parser;
- no-store subtitle vastused ning playeris nähtav, mitteblokeeriv diagnostika.


## 0.3.1 — valmis

- fMP4 HLS `init.mp4` sessioni töökataloogi parandus;
- playlist + init + esimese fragmendi readiness barrier;
- lühike generated-file wait atomaarse rename'i jaoks;
- FFmpegi reaalaja sisend, threadide piir ja 2-sekundilised segmendid;
- fataalse playerivea korral serveripoolse sessioni kohene lõpetamine;
- kaheminutiline orphan-session cleanup;
- päris Windowsi FFmpeg playback smoke-test.

## 0.3.0 — valmis

- bundled Windows x64 FFmpeg/FFprobe ning automaatne vana tool-path migratsioon;
- konservatiivne Direct Play → Direct Stream → universaalne transcode pipeline;
- HEVC/AV1/legacy codec fallback H.264/AAC fMP4 HLS-i;
- HDR10/HDR10+/HLG/Dolby Vision tuvastus ja HDR→SDR software tone mapping;
- audio track'i valik, audio-only/video-only ja serveripoolne seek;
- Apple/tvOS põhimõtetest inspireeritud lazy player controls;
- embedded/adjacent/downloaded subtiitrite avastus;
- UTF-8 WebVTT teisendus ning pildisubtiitrite burn-in;
- OpenSubtitles.com otsing ja allalaadimine;
- playback/subtitle API ja OpenAPI leping;
- playback-planner'i, FFmpeg profiili ja release regressioonitestid.

## 0.3.x — järgmised töökindluse parandused

- käsitsi codec/profile capability raport seadetes;
- hardware acceleration adapterid Intel QSV, NVIDIA NVENC ja AMD AMF jaoks;
- adaptive multi-bitrate HLS ning transcode ressursipiirangud;
- subtitle sync/offset, font attachment ja ASS styling'u valikuline burn-in;
- inkrementaalne file watcher ning scan summary;
- local artwork cache ja käsitsi metadata match/re-match.

## 0.4 — playback'i kvaliteet

- adaptive multi-bitrate HLS;
- transcode job prioriteedid ja ressursipiirangud;
- Intel Quick Sync Video;
- NVIDIA NVENC;
- AMD AMF;
- parem HDR/SDR capability model ja tone mapping profiilid;
- codec/profile registry koos brauseris nähtava availability raportiga.

## 0.5 — kasutajad ja turvaline LAN

- lokaalsed kasutajakontod;
- HttpOnly/SameSite sessioonid;
- CSRF kaitse;
- admin / viewer rollid;
- library-level õigused;
- seadete ja failisirvija piiramine ainult adminile;
- source-root allowlist;
- auditlogi;
- Windows Service paigaldus ja piiratud teenusekonto juhend;
- automaatsed varukoopiad ja restore kontroll.

## 0.7 — muusika

- artist → album → track hierarhia;
- MusicBrainz provider;
- album artwork cache;
- playlistid ja queue;
- gapless playback, kui klient toetab;
- replay gain metadata.

## 0.8 — TV kliendid

### Android TV

- generated OpenAPI client;
- Compose for TV UI;
- Media3 playback adapter;
- D-pad focus testid;
- client capability report;
- progress sync serveriga.

### Samsung Tizen

- sama API semantika;
- Tizen Web App shell;
- AVPlay/HTML5 player adapter;
- remote focus layer;
- capability report mudeliaasta järgi.

TV kliendid ei jaga React DOM komponente, kuid jagavad OpenAPI lepingut, disainiprintsiipe ja serveripoolset playback negotiation'it.

## Enne internetipõhist kaugjuurdepääsu

Avalik remote access ei ole versiooninumbri järgi automaatselt lubatav. Enne seda peavad olema valmis vähemalt:

- autentimine ja rollid;
- TLS;
- CSRF ja rate limiting;
- session revocation;
- auditlogi;
- lubatud source-root'ide poliitika;
- threat model Windows Service'i, update'i ja pöördproksi jaoks;
- sõltumatu turvaülevaatus.
