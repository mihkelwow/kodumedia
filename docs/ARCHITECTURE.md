# KoduMedia arhitektuur

### Püsiv vaatamisolek

Vaatamispositsioon on kataloogist teadlikult eraldatud `data\watch-state.json` faili. `WatchService` seob kontrollpunktid API vastustega ainult lugemisel, nii et meediakogu uuesti skannimine ei muuda vaatamisajalugu. Kirjutused on aatomilised ja player piirab kontrollpunktide sagedust; brauser ei hoia eraldi püsivat vaatamisindeksit.


## Eesmärk

KoduMedia on üks Windows x64 Go-protsess, kuid mitte monoliitne läbipõimunud koodiplokk. Iga funktsionaalsus peab olema asendatav või laiendatav ilma HTTP, püsikihi, codec'i, välise provider'i ja kasutajaliidese loogikat kokku segamata.

Põhisõltuvus:

```text
Windows process / embedded web / future TV clients
                         |
                         v
               HTTP transport + OpenAPI
                         |
                         v
                 application services
                         |
                         v
                    domain + ports
                    ^      ^      ^
                    |      |      |
                catalog  media  platform/integration
                         tools      adapters
```

Sõltuvused liiguvad sissepoole. Adapter võib importida domeeni ja porte; domeen ei impordi adapterit, HTTP-d, Windows COM-i, failisüsteemi ega Reacti.

## Composition root

`apps/server/internal/app/app.go` on ainus koht, kus konkreetsed adapterid tervikuks ühendatakse.

0.8.58 koostab seal:

- `jsoncatalog.Store` kataloogi jaoks;
- `jsonsettings.Store` runtime-seadete jaoks;
- `jsonwatch.Store` püsiva ühe profiili vaatamisoleku jaoks;
- `local.Browser` kerge serveripoolse failisirvimise fallback'ina;
- `nativepicker.Picker` Windowsi native faili- ja kaustavalijana;
- dünaamilise FFprobe adapteri;
- dünaamilise FFmpeg transcode manager'i, Quality Engine'i, JIT-cache'i ja VMAF hoidla;
- dünaamilise TMDb provider'i;
- IMDb `DatasetManager`-i serveripoolse cache'i jaoks;
- kompaktse IMDb `RatingsProvider`-i;
- library-, scan-, media-, metadata-, settings-, watch- ja playback-teenused;
- `MetadataMaintenanceService`-i käivituse, perioodilise refresh'i ja backfill'i jaoks;
- `selfupdate.Manager`-i kontrollitud Windowsi uuenduspaketi staging'u ja helperi käivitamise jaoks;
- `tray.Tray`-d eraldatud Win32 message-loop'i, native menüü ja ikooni jaoks;
- `startup.Manager`-i kasutajapõhise Windows Run registrivõtme haldamiseks;
- `singleinstance.Lock`-i duplikaatserveri vältimiseks ning `shell.OpenURL`-i brauseri avamiseks;
- `UpdateService`-i rakenduse kontrollitud restart-signaali jaoks;
- `CastService`-i ja kohaliku võrgu UPnP adapteri;
- protsessisisese `JobManager`-i;
- versioonitud HTTP routeri;
- binaari sisse pakitud veebikliendi.

Ükski Reacti komponent ega HTTP handler ei loo FFmpeg käsku, ei loe otse kataloogifaili, ei paki IMDb faili lahti ega tee TMDb matching'ut.

## Serveri kihid

### `internal/domain`

Sisaldab äriliselt tähenduslikke tüüpe ja stabiilseid vigu:

- `Library`, `LibrarySource`;
- `MediaItem`, `MediaType`, `EditorialMetadata`;
- `SeriesDetails`, `SeasonGroup`;
- `ServerSettings`, `SystemConfiguration`, `ComponentStatus`;
- `DirectoryListing`, `FileSystemEntry`, `FilePickerRequest`;
- `ClientCapabilities`, `VideoDecodeCapability`, `PlaybackPreferences`, `PlaybackPlan`, `PlaybackSession`;
- `CastDevice`, `CastSource`, `CastSession`, `CastCommand` ja stabiilsed castimise olekud;
- `QualitySettings` ja kvaliteediprofiili/cache/VMAF väärtused;
- `SubtitleTrack`, `SubtitleSearchResult` ja `PlaybackOptions`;
- `Job`;
- `UpdateStatus` ja `UpdateState`.

Domeen ei tea:

- kas püsikiht on JSON või SQLite;
- kas metadata tuleb TMDb-st või mõnest teisest teenusest;
- kas klient on Chrome, Android TV või Tizen;
- kuidas Windows COM dialog avatakse;
- kuidas IMDb remote cache alla laaditakse;
- milliste argumentidega FFmpeg käivitatakse.

### `internal/ports`

Tarbijakesksed väikesed liidesed:

- `Catalog`;
- `SettingsRepository` ja `SettingsInspector`;
- `DirectoryBrowser`;
- `FilePicker`;
- `MediaProbe`;
- `MetadataProvider` ja `RatingsProvider`;
- `RatingsDataset`;
- `TranscodeSessionManager`;
- `CastController` — seadmeotsingu, meedia laadimise ning DLNA transpordi- ja helikäskude leping;
- encoder-capability, segment-cache ja kvaliteedianalüüsi lepingud on hoitud FFmpeg adapteri sisemiste moodulitena, sest teised teenused ei vaja nende detaile;
- `SubtitleStore` ja `OnlineSubtitleProvider`;
- `JobRunner`;
- `UpdateInstaller`.

Liidesed ei ole koondatud üheks „god interface'iks”. Teenus sõltub ainult meetoditest, mida ta tegelikult vajab.

### `internal/service`

Rakenduse kasutusjuhud:

- `LibraryService` — kogude elutsükkel ja allikakaustad;
- `SettingsService` — runtime-väärtuste merge, validatsioon, maskimine, testimine ja salvestamine;
- `ScanService` — skannimise orkestreerimine;
- classifier ja release-name parser — failinime, kaustastruktuuri ning metadata otsingukandidaatide puhas provider-sõltumatu loogika;
- `MediaService` — kataloogipäringud ja seriaali hierarhia koostamine;
- `MetadataService` — editorial ja rating providerite eraldatud rakendamine;
- `MetadataMaintenanceService` — startup refresh, periodic refresh, manual refresh ja kataloogi backfill;
- `PlaybackPlanner` — puhas Direct Play → stream-copy/remux → audio-only transcode → video transcode otsus, mis arvestab tracke, täpset decode-capability infot, HDR-i ja kvaliteediprofiili;
- `PlaybackService` — playback options, vajadusel vana kirje reprobe, planner, sessiooni käivitamine ja lõpetamine;
- `CastService` — eraldi välise playback-session'i, teleri võimekuste,
  absoluutse ajajoone, subtiitrite põletamise ning castimise elutsükli
  orkestreerimine;
- `SubtitleService` — kohalike/sisemiste radade avastamine, WebVTT teisendus ning online-otsingu/allalaadimise orkestreerimine;
- `JobManager` — protsessisisene taustatöö olek;
- `UpdateService` — valideeritud installi käivitamine ja serveri kontrollitud sulgemissignaal.

`MetadataMaintenanceService` serialiseerib `workMu` lukuga kataloogi läbivad metadata backfill'id ja batch-kirjutused. IMDb võrguallalaadimine ei hoia seda lukku: vana kataloogi TMDb pealkirju, kirjeldusi ja artwork'i saab parandada samal ajal, kui reitingufail serverisse voogedastatakse. IMDb provideri enda kompaktse indeksi laadimine ja vahetus on eraldi mutex'iga kaitstud, mistõttu sama andmestikku ei parsita paralleelselt mitu korda.

### `internal/adapters`

#### `cast/upnp`

Serveripoolne adapter leiab kohaliku võrgu DLNA/UPnP MediaRenderer-seadmed SSDP
M-SEARCH päringuga. Seadme XML-kirjeldusest kasutatakse AVTransporti ja
RenderingControli control URL-e; käsud saadetakse SOAP-iga ning meedia
kirjeldatakse DIDL-Lite metaandmetega.

Adapter lubab ainult privaatseid, loopback- ja link-local sihtaadresse, piirab
XML/SOAP vastuste suurust ning ei järgi redirecti avalikku võrku. Castitav
playback URL kirjutatakse teleri suunas marsruuditava kohaliku serveri-IP peale.
Brauseri native AirPlay/Remote Playback ning HTTPS-i nõudev Google Cast Sender
jäävad veebikihi adapteriteks, sest nende seadmevaliku peab avama kasutaja
žestist brauser ise.


#### `system/selfupdate`, `updatepackage` ja `updater`

Brauseripõhine uuendus on jaotatud kolmeks eraldi osaks:

1. `selfupdate.Manager` võtab localhost-endpoint'ilt ZIP-voo vastu, salvestab selle suuruspiiriga, kutsub paketivalideerija ning loob installiplaani;
2. `updatepackage` valideerib manifesti, platvormi, SemVer versiooni, failiteed, suurused ja SHA-256 väärtused ning pakib payload'i eraldatud staging-kausta;
3. eraldi `kodumedia-updater.exe` ootab serveriprotsessi lõppu, teeb failidest varukoopia, vahetab rakendusfailid, käivitab uue serveri ja kontrollib health-endpoint'i.

Helper käivitatakse staging-kaustast, mitte installikaustast. Seetõttu saab see turvaliselt uuendada ka installitud helperit ennast. `data` ja `.env` on nii manifestivalideerijas kui ka installiplaani path-kontrollis keelatud. Rollback kasutab eraldi backup-state faili, et eristada varem olemas olnud faile uutest failidest.

Update olek on püsivalt `data\updates\status.json` failis. Veebiklient pollib protsessi taaskäivituse ajal ainult `/api/v1/health` endpoint'i; see ei käivita FFmpeg ega FFprobe kontrollprotsesse.

#### `catalog/jsoncatalog`

Praegune püsikiht:

- hoiab olekut ühe protsessi sees RW-luku taga;
- kirjutab ajutisse faili ja asendab kataloogifaili kontrollitud Windowsi operatsiooniga;
- hoiab absoluutse meediafaili tee API JSON-ist väljas;
- normaliseerib vana `sources: null` väärtuse tühjaks massiiviks;
- toetab batch-uuendust, et metadata backfill kirjutaks kataloogi ühe korraga.

#### `settings/jsonsettings`

Runtime-seadete adapter:

- loob esimesel käivitamisel `data\settings.json`;
- kasutab keskkonnamuutujaid ainult bootstrap-vaikeväärtustena;
- salvestab schema version 4 kuju;
- migreerib varasemad skeemid, lisab Quality Engine'i profiili/cache/VMAF seaded ning säilitab subtitle/OpenSubtitles saladused;
- salvestab faili piiratud õigustega;
- väljastab teenusele immutable väärtuskoopia;
- ei saada tokenit ise HTTP-sse — maskimine kuulub `SettingsService`-ile.

#### `filesystem/nativepicker`

Windows-only adapter:

- kasutab `IFileOpenDialog` COM liidest;
- avab kausta valimiseks `FOS_PICKFOLDERS` režiimi;
- töötab lukustatud OS thread'il STA COM apartment'is;
- seob dialoogi foreground window'ga;
- tagastab kasutaja cancel-toimingu eraldi olekuna;
- valideerib failirežiimis lubatud laiendi;
- ei sisalda HTTP ega Reacti loogikat.

HTTP transport lubab native pickerit ainult loopback-päringule. See hoiab desktop-dialogi serveriarvuti interaktiivse sessiooni küljes ega luba LAN-kliendil suvaliselt serveri ekraanil dialoogi avada.

#### `filesystem/local`

Kerge fallback-browser:

- loetleb Windowsi loogilised draivid ühe süsteemipäringuga;
- kontrollib absoluutset teed;
- tagastab kaustad enne faile;
- filtreerib faililaiendeid;
- peidab valitud süsteemikaustad;
- hoiab lühikest directory cache'i;
- loeb kataloogi 256-kirjeliste native batch'idena ega lae enne vastamist kogu suurt kataloogi mällu;
- peatub kohe pärast 1001. sobiva kirje leidmist, piirab vastuse 1000 kirjega ja tagastab `truncated` signaali;
- austab request context'i katkestamist;
- ei sisalda modali, nupu ega Reacti loogikat.

#### `probe/ffprobe`

- loeb iga käivituse eel runtime-seadetest FFprobe tee;
- teisendab FFprobe JSON-i neutraalseks tehniliseks metadata mudeliks;
- salvestab stream ordinal/index, kanalid, bitisügavuse, pixel format'i, värviinfo, disposition'id ja HDR-vormingu;
- ei klassifitseeri filmi ega seriaali.

#### `transcode/ffmpeg` ja Quality Engine

Adapter on jagatud väikesteks eesmärgipõhisteks mooduliteks:

- `profile.go` moodustab remux-, audio-only- ja videoencoder'i argumentide ahela;
- `encoder.go` tuvastab ja proovib Intel QSV, NVIDIA NVENC, AMD AMF ning tarkvaralisi H.264/HEVC/AV1 encodereid;
- `progress.go` loeb FFmpegi progressi, hoiab encoder-speed libisevat keskmist, arvutab segmentide/bitrate'i runtime-statistika ja märgib püsivalt alakiire töö fallback-kandidaadiks;
- `cache.go` haldab vajaduspõhist fMP4 HLS segment-cache'i, absoluutseid ajavahemikke, LRU-koristust ja cache'i invalidatsiooni;
- `quality.go` salvestab väikese VMAF-proovianalüüsi ning annab järgmise esituse CRF/kvaliteediväärtuse nõu;
- `manager.go` koordineerib session'e, jagatud encoder-töid, grace-perioodi, fallback'i ja lifecycle'i.

Põhimõtted:

- originaalvideo kopeeritakse alati, kui kliendi decode-capability seda lubab;
- audio võib transkodeeruda ilma video kvaliteeti muutmata;
- `Originaal` ja `Maksimaalne` ei sisalda varjatud 1080p piirangut;
- hardware encoder'i nimekirjas olemine ei ole piisav — enne kasutamist tehakse päris ühekaadriline proov;
- HEVC 10-bit/HDR kasutab selget Main10 profiili;
- automaatne software AV1 on välditud, kui reaalajakiirus pole usaldusväärne;
- encoder'i kiirus hinnatakse enne session'i avaldamist ja kogu esituse ajal; 1080p töö vajab vähemalt 1.20× ning 4K/HDR vähemalt 1.25× turvavaru;
- playeri puhvri madal veemärk ja korduvad stallid võivad käivitada samast positsioonist 1080p või 720p taastumisprofiili;
- codec'i/resolutsiooni põhine VBV `maxrate`/`bufsize` tasandab keeruliste stseenide bitrate'i tippe;
- runtime-statistika on versioonitud HTTP endpoint'i kaudu kättesaadav ka tulevastele TV-klientidele;
- JIT-cache luuakse ainult reaalse video transkodeerimise käigus, mitte skannimisel ega öötööna;
- sama cache-võtmega kliendid jagavad üht aktiivset encoder-session'it;
- cache-võti sisaldab lähtefaili signatuuri, kvaliteediprofiili, codec'it, encoder'it, resolutsiooni, HDR-i, heli- ja subtiitrivalikut ning pipeline'i versiooni;
- VMAF ei blokeeri playback'i ega võrdle HDR-lähtepilti eksitavalt SDR-väljundiga;
- adapter ei tea HTTP request'i ega Reacti playerit.

#### `metadata/tmdb`

- loeb tokeni ja keele runtime-providerist;
- otsib filmi või seriaali classifier'i antud pealkirja/aasta järgi;
- valib kandidaadi normaliseeritud nime, aasta ja popularity signaali põhjal;
- laadib filmi, seriaali või episoodi detailid;
- teisendab TMDb response'i `domain.MetadataResult` kujule;
- ei kirjuta kataloogi ise.

#### `metadata/imdb/DatasetManager`

- aktsepteerib absoluutset HTTPS URL-i või kohalikku `.tsv`/`.tsv.gz` faili;
- kasutab vaikimisi ametlikku `title.ratings.tsv.gz` URL-i;
- saadab remote refresh'il ETag ja Last-Modified tingimused;
- voogedastab vastuse otse ajutisse faili;
- rakendab 512 MiB turvapiiri;
- valideerib TSV päise enne cache'i vahetust;
- säilitab ebaõnnestunud refresh'i korral viimase töötava cache'i;
- hoiab cache'i source/ETag/timestamp/size olekut eraldi JSON failis;
- annab avaliku `ComponentStatus` mudeli, mitte HTTP response'i.

#### `metadata/imdb/RatingsProvider`

- loeb dataset manager'i aktiivset lokaalset faili;
- pakib IMDb `tt...` ID numbriks;
- hoiab kirjeid fikseeritud laiusega sorteeritud slice'is;
- kasutab lookup'iks binaarotsingut;
- laeb indeksi laisalt ja ainult faili signatuuri muutumisel;
- saab refresh'i järel in-memory indeksi odavalt invalidiseerida.

#### `system/inspection`

- käivitab `ffmpeg -version` ja `ffprobe -version` ajapiiranguga;
- kontrollib TMDb Bearer tokenit;
- delegeerib IMDb source'i kontrolli `DatasetManager`-ile;
- tagastab domeeni `SystemStatus`, mitte HTML-i.

### `internal/transport/httpapi`

Õhuke adapter:

1. dekodeerib ja piirab sisendi;
2. loob domeenipäringu või update-käsu;
3. kutsub üht teenust või porti;
4. teisendab domeenivea stabiilseks HTTP veaks;
5. kodeerib vastuse.

0.2.2 asjakohased lepingud:

```text
GET  /api/v1/system/settings
PUT  /api/v1/system/settings
POST /api/v1/system/settings/test
POST /api/v1/system/metadata/imdb/refresh
GET  /api/v1/system/update
POST /api/v1/system/update
GET  /api/v1/filesystem/browse
POST /api/v1/filesystem/pick
GET  /api/v1/jobs/{id}
GET  /api/v1/series/{id}
```

Täielik leping on `api/openapi.yaml`.

## Runtime-seadete voog

```text
first start
.env / process environment
            |
            v
      config.Load defaults
            |
            v
 data\settings.json repository
            |
            +-----------------------------+
            |                             |
            v                             v
 SettingsService                   dynamic adapters
 validate/mask/test          FFmpeg/FFprobe/TMDb/IMDb source
            ^                             |
            |                             |
            +--------- HTTP API ----------+
                          |
                          v
                    browser settings UI
```

Oluline invariant: salvestatud TMDb token ei lähe `PublicServerSettings` vastusesse. API väljastab ainult `tmdbTokenConfigured` ja maskitud `tmdbTokenHint` väärtuse.

Runtime-adapterid ei hoia tee või tokeni koopiat konstruktoris. Nad saavad provider-funktsiooni, mis loeb repository hetkeseisu. Seetõttu rakendub brauseris salvestatud muudatus järgmise operatsiooni ajal ilma serveri restartita.

## Skannimise pipeline

```text
library sources
      |
      v
recursive candidate discovery
      |
      v
path classifier
movie / episode / music
      |
      v
FFprobe technical metadata + tags
      |
      v
series grouping
parent + season/episode links
      |
      v
configured metadata enrichment
series first, then movie/episode
      |
      v
sorted atomic catalog replacement
```

Classifier ei tee võrgupäringut. TMDb matching kasutab classifier'i antud pealkirja, aastat, hooaja- ja episoodinumbrit järgmises etapis.

Skannimine ei sõltu IMDb remote download'i lõppemisest. Kui rating cache pole veel valmis, säilib TMDb editorial metadata ning reitingud parandatakse dataset refresh'i järgse backfill'iga.

## Metadata hooldusvoog

```text
server starts
    |
    +--> HTTP starts immediately
    |
    +--> IMDb refresh job --------------------+
    |       conditional GET / local validate  |
    |                                         v
    |                                  compact rating index
    |                                         |
    +--> missing editorial backfill           |
            series before episodes            |
                    |                          |
                    +----------+---------------+
                               v
                      batch catalog update
```

Käivituse hooldus on non-blocking. `MetadataMaintenanceService` kasutab ühist tööjärjekorra lukku, seega remote refresh ja full catalog backfill ei koorma ketast ning providerit paralleelselt.

Seadete muutmisel:

- IMDb source'i muutus invalidiseerib vana indeksi ja käivitab refresh'i;
- aktiivse refresh'i ajal tehtud source'i muudatus märgitakse pending-tööks;
- TMDb tokeni salvestamine käivitab puuduva editorial metadata repair'i;
- keele muutmine käivitab forced editorial refresh'i.

Üksiku faili detailvaates metadata nuppu ei ole. Kasutaja käivitab süsteemse repair'i settings save'i või skanni kaudu; IMDb-l on eraldi dataset-level refresh.

## Seriaali rühmitamine

Iga episood saab stabiilse `SeriesID`, mis tuletatakse:

```text
library ID + normaliseeritud seriaali nimi + tuvastatud aasta
```

Skanner loob ühe sünteetilise `MediaTypeSeries` parent-kirje, millel on:

- ametlik või failinimest tuletatud nimi;
- `seasonCount`;
- `episodeCount`;
- seriaali metadata ja external ID-d;
- puuduv playable file, sest parent ise pole meediafail.

Seriaali parent rikastatakse enne episoode. Episoodi provider saab parent'i TMDb ID ja küsib konkreetse `/tv/{id}/season/{s}/episode/{e}` kirje. Ametliku parent-nime muutus levitatakse episoodidele batch-uuenduses ilma iga episoodi kohta uut seriaaliotsingut tegemata.

## Esituse otsustusvoog

```text
client capability report
          |
          v
+---------------------+
| PlaybackPlanner     |
+----+-----------+----+
     |           |
 direct       source incompatible
     |           |
     v           v
ServeContent   can copy tracks?
Range support    |         |
              remux     transcode
                 \       /
                  v     v
               fMP4 HLS session
```

- **Direct play** kasutab `http.ServeContent` byte-range tuge.
- **Remux** kopeerib sobivad rajad fMP4 HLS-i.
- **Transcode** toodab praegu tarkvaraliselt H.264/AAC fMP4 HLS-i.

Uus encoder lisatakse `Profile` adapterina; HTTP handler ja React player ei vaja muutmist, kui väljundleping jääb HLS-iks.

## Quality Engine'i esitus- ja cache-voog

```text
Client MediaCapabilities + display/HDR signature
                         |
                         v
                 PlaybackPlanner
       Direct Play / copy / audio-only / video transcode
                         |
                         v
                Encoder capability probe
       QSV / NVENC / AMF / software + exact profile args
                         |
                         v
             startup speed/readiness assessment
                         |
             +-----------+-----------+
             |                       |
             v                       v
       primary quality path      smooth fallback
             |                       |
             +-----------+-----------+
                         |
                         v
         one shared FFmpeg producer per cache key
                         |
             +-----------+-----------+
             |                       |
             v                       v
        active clients          JIT segment cache
                                      |
                                      v
                        later playback / partial resume
```

Cache manifest salvestab `BaseStartMS`, cached duration'i ja täielikkuse. Seetõttu võib 40. minutil alustatud reaalajas transkodeering tekitada korrektse absoluutse ajavahemikuga cache'i ning hilisem 42. minuti esitus seda kasutada. Nullist algavat ja mitte-nullist algavat playlisti ei liideta pimesi üheks vigase timestamp'iga vooluks.

Display/HDR capability cache'i võti sisaldab aktiivse ekraani mõõte, device pixel ratio, color depth'i ning `dynamic-range`, `video-dynamic-range` ja `rec2020` signaale. Windowsi HDR-režiimi või ekraani muutus käivitab uue decode-probe'i.

## Veebikliendi kihid

### `src/api`

Ainus võrgupiir:

- `fetch` wrapper;
- stabiilne `ApiError`;
- TypeScripti lepingud;
- väliste payload'ide normaliseerimine.

### `src/features`

Iseseisvad funktsioonid:

- `navigation` — hõljuv `AppHeader`, süsteemikogude route'id ja klaviatuurifookus;
- `search` — kerge `SearchOverlay`, debounce, request cancellation ja tulemuste grupeerimine;
- `media` — posterid, laiformaadis kaardid ning filmi-/seriaali detailvaated;
- `player`;
- `settings`;
- `theme`.

Detailvaated ei käivita enam metadata request'i. Need renderdavad kataloogi hetkeseisu ja avavad playback'i. Globaalne otsing kutsub ainult `src/api` kataloogipäringut; React ei indekseeri ega skanni meediat kliendis.

### Native picker + fallback

`openNativeFilePicker` kutsub kõigepealt `/filesystem/pick` endpoint'i.

- localhost + interaktiivne Windows session → native dialog;
- 503/404 → `FileBrowserDialog` fallback;
- cancel → dialoog sulgub ilma fallback'ita;
- muud vead kuvatakse kasutajale.

Fallback `FileBrowserDialog`:

- kasutab jagatud `Dialog` primitiivi ja renderdatakse portaali kaudu `document.body` alla;
- katkestab eelmise browse request'i;
- ignoreerib hilinenud vastuseid request-sequence'i kaudu;
- pärib sama Escape'i, fookuse, scroll lock'i ja motion-poliitika nagu ülejäänud dialoogid;
- kuvab `truncated` teate, mitte piiramatut DOM loendit.

### `src/shared/components` ja `src/design`

- `Dialog` omab modaalide Escape'i, fookuse püünist, fookuse taastamist ja body scroll lock'i;
- `Popover` omab väljaspool klõpsamist, Escape'i, fookuse taastamist ja placement-põhist transform-origin'i;
- `Menu` lisab ühise vertikaalse Arrow/Home/End klaviatuurinavigatsiooni;
- `IconButton`, `Button`, `Field` ja `SelectControl` määravad kontrollide ühise struktuuri ja vähemalt 44 × 44 px sihtala;
- `tokens.css` on ainus autoriteetne teema-, mõõdu-, raadiuse-, kihistuse- ja motion-tokenite allikas;
- `motion.css` määrab press-feedback'i ning ühe globaalse `prefers-reduced-motion` poliitika;
- feature-stiilid võivad primitiive paigutada, kuid ei defineeri uuesti nende kesta, fookust ega motion-käitumist;
- Reacti lähtekoodi UI-d ei tohi ehitada ega ümber tõsta `MutationObserver`-i või `innerHTML` abil.

### `src/app`, `src/pages`, `src/design`

- `src/app` sisaldab composition root'i, History API routerit, globaalse otsingu olekut ja error boundary't;
- `src/pages` koostab feature'id ega tee otse võrgupäringuid;
- `src/design` hoiab tumeda/heleda teema värvid, atmosfäärsed pinnad, mõõdud ja fookuserõngad tokenites;
- player ja `hls.js` jäävad lazy chunk'i, mistõttu uue avalehe, kogude ning otsingu kasutamine ei laadi playback-mootorit.

## Windows-only piir

Ametlik serveri entrypoint on build-tagiga:

```go
//go:build windows
```

Windowsi-spetsiifiline loogika:

- native COM file picker;
- loogiliste draivide loend;
- kontrollitud failiasendus;
- PowerShelli build/release töövoog;
- NSIS-i kasutajapõhine installer ja portable-data migratsioon;
- Windows x64 PE väljalase.

Domeen ja enamik teenuseid jäävad platvormist sõltumatuks, et neid saaks kiiresti unit- ja race-testida. See ei tähenda, et Linuxi või macOS-i serveriväljalase oleks toetatud.

## Püsikihi järgmine etapp

SQLite'i migratsioon ei tohi muuta domeeni ega HTTP lepingut:

1. lisa `internal/adapters/catalog/sqlite`;
2. rakenda sama `ports.Catalog`;
3. lisa versioned migratsioonid ja JSON import;
4. vali adapter composition root'is;
5. hoia `SeriesID`, external ID-d ja runtime-seaded migratsioonis stabiilsena.

## Versioonimine

- avalik API elab `/api/v1` all;
- olemasoleva välja tähendust ei muudeta vaikides;
- uued väljad on kliendile valikulised;
- settings repository kasutab eraldi schema versionit ja migratsiooni;
- murdev HTTP muudatus loob `/api/v2` või dokumenteeritud migratsiooni;
- `api/openapi.yaml` on veebikliendi ja tulevaste platvormiklientide leping.

## Testistrateegia

1. classifier ja playback planner — puhtad table-driven unit-testid;
2. JSON repositoryd — migratsiooni, normaliseerimise, batch-write'i ja salvestamise testid;
3. adapters — executable/status, fallback browser, native picker ABI build, dataset download ja mapping-testid;
4. metadata maintenance — copied catalog backfill, provider failure isolation ja refresh serialization;
5. scan integration — päris ajutine kaust, film + mitu hooaega;
6. HTTP — settings, picker loopback policy, filesystem, series ja SPA route lepingud;
7. frontend — router, API normaliseerimine, hõljuv navigatsioon, globaalne otsing, native/fallback picker flow, runtime-seaded, detailvaade ja hooajagrupp;
8. Windows release — `GOOS=windows`, `GOARCH=amd64`, `CGO_ENABLED=0` PE build;
9. `smoke-test.ps1` — päris `.exe` käivitus Windowsis, API, migratsioon, settings, scan ja grouping.

## Bundled tööriistade piir

Windowsi väljalase kannab `tools\ffmpeg\bin` all konkreetset, litsentsifailidega seotud FFmpeg/FFprobe build'i. `config.PreferBundledTools` valib selle ainult siis, kui seadistus on tühi, generic PATH-nimi või viitab puuduvale failile. Kehtiv custom build säilitatakse. See hoiab paigaldamise iseseisvana, kuid ei seo domeeni ega playback-plannerit ühe konkreetse FFmpegi versiooniga.

## Kerge kliendi põhimõte

Veebiklient ei dekodeeri lähtecodec'e JavaScriptis. Ta saadab capability-mudeli, saab serverilt kas Range-voo või fMP4 HLS-i ning renderdab ainult player UI. `hls.js` on eraldi lazy chunk, puhver on piiratud ja suur FFmpeg/subtitle töö tehakse serveris. Tulevane Tizeni või Android TV adapter võib sama API taga kasutada native playerit.

## Web interface localization

The React client owns interface localization. Translation catalogs are UTF-8 JSON files under `apps/web/src/i18n/locales/`. `et-EE.json` remains the typed source-key catalog, while `en-US.json` is the first-run and runtime fallback catalog. Every locale must contain the same keys and preserve interpolation placeholders.

The selected interface locale is stored per browser under `kodumedia:ui-language`, so clients connected to the same server may use different languages. It is deliberately separate from server-wide metadata and subtitle language preferences. The locale store updates the HTML `lang` and `dir` attributes and drives `Intl` number, plural, date and duration formatting.

`TranslationKey` is derived from the Estonian JSON catalog, making untranslated literal keys a TypeScript build error. Catalog tests enforce key parity, non-empty values and placeholder integrity. Known legacy server status strings pass through `trExternal`; unknown technical messages remain unchanged so diagnostic detail is not lost.


## Notifications

`NotificationService` compares the catalog's official latest episode metadata with locally available episode records. Persistent history and read state live in `data/notifications.json` through the `NotificationRepository` port and the `jsonnotifications` adapter. The browser consumes the versioned notification API; opening the header menu marks all current items read.
