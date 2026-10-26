## 0.8.57 - 2026-10-26

### Playeri käivitumise hotfix

- Parandatud 0.8.56 regressioon, mille tõttu film või seriaali osa võis jääda
  brauseris lõputult olekusse „Puhverdan esituse algust 10 / 14 s…” või oodata
  20-sekundilist läve ka siis, kui esitatav HLS-voog oli juba valmis.
- Brauseri kohustuslik käivituspuhver on nüüd piiratud maksimaalselt kaheksa
  sekundiga. Serveri suurem encoder-headroom ei ole enam brauseri stardi
  blokeeriv tingimus.
- Kui brauser on kuus sekundit oodanud ja meediaelement kinnitab esitatavate
  andmete olemasolu, alustab player tööd ka täpsest puhvrieesmärgist veidi allpool.
- Ka taas-puhverdamine kasutab brauseris lõplikku ülempiiri ega saa jääda
  suure serveripoolse eesmärgi taha lõputult ootama.
- Samsungi ja teiste DLNA-telerite 12–20 sekundi serveripoolne valmisolekupuhver
  jääb muutmata; hotfix puudutab ainult kohalikku brauseriplayerit.
- Lisatud on regressioonitestid kõrge kvaliteediga transkodeeringu puhvriläve
  ning ajaliselt piiratud käivitumise kohta.

## 0.8.56 - 2026-10-26

### Töökindel AirPlay ja puhverdatud Samsung DLNA

- Safari kasutab toetatud seadmetes alati brauseri native HLS-i. AirPlay saab
  nüüd päris HTTP(S) meedia-aadressi, mitte ainult saatva vahelehe sees toimiva
  `blob:`-aadressi.
- DLNA-teleritele pakutakse killustunud MP4 asemel laialt ühilduvat MPEG-TS HLS-i.
  Server ootab enne teleris esitamise käivitamist vähemalt 12–20 sekundi jagu
  täielikult loodud segmente, et teler ei jõuaks transkodeerijale järele.
- Ühilduvad 4K/HEVC rajad säilivad Direct Play või remux'i korral muutmata.
  Kui subtiitrite põletamine või codec nõuab reaalaja transkodeerimist, kasutab
  DLNA töökindlat 1080p H.264/SDR fallback'i.
- MPEG-TS segmendid saavad korrektse `video/mp2t` MIME-tüübi ning castimise
  diagnostikalogisse lisati valitud konteiner, codec'id, rajatoimingud ja
  käivituspuhvri suurus.
- Lisatud on regressioonitestid Safari native HLS-i, Samsungi MPEG-TS remux'i,
  1080p transkodeerimise fallback'i ja serveripoolse käivituspuhvri kohta.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.55 - 2026-10-26

### AirPlay, Google Cast ja sama võrgu DLNA/UPnP-telerid

- Videopleieri juhtnuppudele lisati üks ühine castimise nupp ja klaviatuuriga
  kasutatav seadmevalik AirPlay, Remote Playbacki, Google Casti ning
  DLNA/UPnP-telerite jaoks.
- Apple'i seadmetes avatakse native AirPlay väljundivalik. HTTPS-i kaudu avatud
  KoduMedia kasutab Google Cast Default Media Receiverit ilma eraldi receiveri
  registreerimiseta.
- Server otsib SSDP abil sama kohaliku võrgu MediaRenderer-seadmeid ning juhib
  nende AVTransporti ja RenderingControli teenuseid. Kaetud on käivitus,
  paus/jätkamine, lõpetamine, seek, helitugevus, vaigistus ja olekupäring.
- Samsung RU/NU/Q/LS mudelitele, sealhulgas RU7172-le, kasutatakse 4K, HEVC,
  HDR10/HLG ja mitmekanalilise heli võimekusprofiili; muudele seadmetele jääb
  konservatiivne H.264/AAC 1080p fallback.
- Castimine loob eraldi serveripoolse playback-session'i. Valitud teksti- ja
  pildisubtiitrid põletatakse vajadusel videopilti ning transkodeeritud voo
  absoluutne vaatamispositsioon säilib ka seek'i ja watch-progressi ajal.
- UPnP kirjeldus- ja juhtimisaadressid on piiratud privaatse, loopback- või
  link-local võrguga. XML vastused, URL-id ja SOAP väärtused on suurus- ning
  hostikontrolliga.
- Lisatud on eesti- ja ingliskeelsed kasutajatekstid, cast-menüü komponenditestid
  ning serveri regressioonitestid Samsungi profiili, ajajoone, SOAP-i ja
  DIDL-Lite metaandmete jaoks.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.54 - 2026-08-16

### Ühtne pill-kujundusega esitamisnupp

- Avalehe hero, seriaali detail, filmi detail ja episoodi detail kasutavad nüüd
  sama jagatud esitamisnupu komponenti.
- Kõigi peamiste esitamisnuppude kuju on ühtne: 160 × 48 px miinimummõõt,
  täielikult ümar pill-raadius, sama ikoonivahe ja tüpograafia.
- Tume ja hele teema kasutavad semantilisi esitamisnupu värvi- ning
  varjutokeneid; pildi kohal paiknev hero-nupp säilitab selge heleda kontrasti.
- Hover-, vajutus-, klaviatuurifookuse, suure kontrasti ja vähendatud liikumise
  olekud on kaetud ühe keskse stiiliga.
- Regressioonikontroll takistab hero, seriaali, filmi ja episoodi
  esitamisnuppude tulevast kujunduslikku lahknemist.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.53 - 2026-08-16

### Mugavam desktopi episoodiliikumine ja klaviatuuritugi

- Peene osutiga desktopiseadmetel kasutab episoodikarussell nüüd vaba
  lohistamist. Kaardirida jääb vabastamiskohta ega hüppa lühikese hiireliigutuse
  järel tagasi eelmisele snap-punktile.
- Puuteekraani olemasolev snap-käitumine säilib muutumatuna, sest see toimib
  sõrmega pühkimisel hästi ja annab selge kaardipõhise lõppasendi.
- Vasak ja parem nooleklahv liigutavad nüüd nii klaviatuurifookuse kui
  karussellivaate eelmisele või järgmisele episoodikaardile.
- Home viib rea esimesele ning End viimasele episoodile. Rea servas jäetakse
  suunaklahv vabaks rakenduse ülejäänud ruumilise navigeerimise jaoks.
- Fookuse liikumine kasutab `preventScroll` käitumist, et brauseri enda kerimine
  ei võistleks Embla sihitud kaardiliikumisega.
- Kõik 99 veebiliidese testi läbivad; eraldi kontroll katab desktopi Embla
  seadistuse ja klaviatuurifookuse tegeliku liikumise.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.52 - 2026-08-16

### Horisontaalne CVI-põhine hooajavalik

- Mitme hooajaga seriaalides paiknevad hooaja tekst-tabid nüüd alati ühe
  horisontaalse reana. Kitsas vaates saab rida loomulikult külgsuunas kerida.
- Hooajanupus kuvatakse ainult ühe rea pealkiri. Valitud hooaja osade arv on
  eraldi rahulik metainfo ega muuda tabi kahekihiliseks badge'iks.
- Aktiivne tab kasutab CVI pinna-, fookuse-, liikumise ja ümarduse tokeneid koos
  vaoshoitud sisemise valgusserva ning kerge sügavusega.
- Hover-, klaviatuurifookuse ja vajutusolekud on selgelt eristatavad nii tumedas
  kui heledas teemas. Kõik tabid säilitavad vähemalt 44 px puuteala.
- Ühe hooajaga seriaalis ei kuvata eksitavat nuppu. Hooaeg ja osade arv on
  esitatud lihtsa mitteklõpsatava pealkirjareana.
- Lisatud on mitme ning ühe hooaja regressioonikontroll. Kõik 99 veebiliidese
  testi läbivad.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.51 - 2026-08-16

### Horisontaalne ja puutetundlik episoodikarussell

- Seriaalivaate episoodid on nüüd vertikaalse loendi asemel ühe horisontaalse
  Embla karussellina, mida saab sõrme, puutepliiatsi ja hiirega lohistada.
- Karussell kasutab loomulikku snap-liikumist, klaviatuuri nooleklahve,
  eelmise-järgmise nuppe ning tagasihoidlikku asukohariba.
- Uus 16:9 episoodikaart säilitab kaaneteksti asemel selge hierarhia: osa kood,
  nimi, lühikirjeldus, failinimi, kuupäev, kestus ja vaatamise edenemine.
- Mobiilis jääb järgmise episoodi serv nähtavale ning kõik juhtnupud säilitavad
  vähemalt 44 px puuteala. Vertikaalne lehekerimine toimib karusselli kohal.
- Hooajavaliku ja episoodiala üleliigsed horisontaalsed eraldusjooned on
  eemaldatud. Tume, hele, vähendatud liikumise, vähendatud läbipaistvuse ja
  suurema kontrasti režiimid on eraldi kaetud.
- Lisatud on episoodikarusselli komponenditestid ja hooajavaate
  regressioonikontroll. Kõik 98 veebiliidese testi läbivad.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.50 - 2026-08-16

### Läbipaistev seadete külgmenüü ja loetav mobiilivalik

- Eemaldatud on hiline `performance.css` reegel, mis kirjutas külgmenüü
  läbipaistva tausta üle peaaegu `#171719` värvi pinnaga. Töölauavaate menüü
  konteiner on nüüd päriselt läbipaistev.
- Menüüread säilitavad varasema detailse premium-kujunduse. Taust jääb ainult
  aktiivsele, fookuses või hiire all olevale reale, mitte kogu külgmenüüle.
- Mobiili ja tahvelarvuti seadete jaotisevalik kasutab nüüd rakenduse CVI
  valikukomponenti: 48 px puuteala, selge piir, ümardus, fookusrõngas ja
  rakenduse ikoonisüsteemi allanoole tähis.
- Native-rippmenüü valikutele on tumedas ja heledas teemas määratud eraldi
  `color-scheme`, taust ja tekstivärv, et hele tekst ei satuks heledale taustale.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.49 - 2026-08-16

### Taastatud premium-klassi seadete külgmenüü

- Töölauavaate seadete külgmenüü kasutab jälle 0.8.47 detailset menüürida:
  58 px kõrgus, kahe-realised nimetused koos selgitusega, 14 px ümardus,
  parempoolne suunaviide ja aktsendiga ikoon.
- Aktiivset asukohta näitavad taas pehme lokaalne reataust ning 2 px
  aktsentjoon. Hover- ja klaviatuurifookus järgivad sama visuaalset süsteemi.
- Külgmenüü konteiner, selle pseudoelemendid, piir, vari ja backdrop-filter on
  töölauavaates eemaldatud. Menüü taga ei ole eraldi halli pinda.
- Uuenduste rea varasem püsiv hall ja lillakas erandtaust on nullitud. See rida
  saab tausta ainult siis, kui see on aktiivne, fookuses või hiire all.
- Tahvelarvuti ja mobiili kompaktne jaotisevalik ning kõik 0.8.48 seadete
  töövoo-, tagasiside- ja esmase seadistuse parandused säilivad.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.48 — 2026-08-15

### Selgem seadete infoarhitektuur ja töövoog

- Seadete töölauamenüüst eemaldati eraldiseisev taustakast. Jaotised on nüüd
  rühmitatud serveri, meedia, kasutajaliidese ja hoolduse ülesannete järgi ning
  aktiivset jaotist märgib rahulik 2 px küljejoon.
- Tahvelarvuti ja mobiili vaates asendab külgmenüüd kompaktne native-jaotisevalik,
  mis säilitab klaviatuuri-, ekraanilugeja- ja URL-hashi toe.
- Seadete pealkiri kasutab sama 38–72 px põhiskaalat nagu teised rakenduse
  põhilehed. Töövalmiduse ülevaade, olekuread ja ikoonid kasutavad vähem
  pesastatud pindu ning selgemat tüpograafilist hierarhiat.
- Iga seadeterühm näitab rakendumise ulatust ja käitumist: serveriülene,
  serveriarvuti või ainult see brauser ning kohene või salvestamist nõudev.
- Salvestamata muudatuste tegevusriba püsib vaatealas ka pika lehe korral.
  Mobiilis muutuvad teisejärgulised tegevused kompaktseks, kuid säilitavad
  44 px puuteala ja ligipääsetava nime.
- Salvestamise edu- ja veateated kuvatakse alati vaatealas, toetavad ARIA live
  tagasisidet ning neid saab sulgeda. Edukinnitus kaob kuue sekundi järel.
- Pooleli esmase seadistuse juhend avaneb esmakasutajal automaatselt. Valmis
  seadistuse korral kuvatakse kompaktne kinnitatud olek ning sammude lingid
  viivad täpselt vastava vormiväljani.
- Lisatud on vähendatud läbipaistvuse ja suurema kontrasti meediapäringud.
  Eesti ja inglise kataloogid sisaldavad mõlemad 922 võtit.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.47 — 2026-08-15

### Töökindel ja füüsiliselt vahetu hero swipe

- 0.8.46 esialgne pointer-only žestituvastus on asendatud Embla 8.6.0
  täislaiuses slaidirajaga. Aktiivne ja naaberslaidid liiguvad nüüd puudutuse
  ajal 1:1 sõrmega kaasa, mitte ainult aktiivse slaidi dekoratiivkihid.
- Puute vabastamisel säilivad suund ja liikumiskiirus ning karussell snap'ib
  naaberslaidile kriitiliselt summutatud, ilma liigse põrketa liikumisega.
- Mõõduka pühke lävi on 14% hero laiusest, piiratud 44–64 piksliga. Kiirema
  lühikese viipe puhul kasutatakse 100 ms kiirusajalugu ja 90 ms
  momendiprojektsiooni, mistõttu slaidivahetus ei nõua poole ekraani läbimist.
- Pointer-sisendi kõrval on eraldi Touch Events fallback brauseritele ja
  puutevirnadele, mis raporteerivad pointer-andmeid erinevalt. Mõlemad teed
  valivad alati täpselt ühe naaberslaidi ega tekita topeltvahetust.
- `touch-action: pan-y pinch-zoom` jääb füüsilisele slaidirajale: päris Chromiumi
  touch-test kinnitab, et horisontaalne pühkimine vahetab slaidi ning sama hero
  peal tehtud vertikaalne viibe kerib lehte ja jätab slaidi muutmata.
- Mobiilne all-parema noolepaar, peidetud punktid ja töölauavaate kompaktne
  34 / 7 / 22 px juhtnupugeomeetria säilivad.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.46 — 2026-08-15

### Puutetundlik hero-karussell ja mobiilne noolenavigatsioon

- Avalehe hero-karussell toetab nüüd touch- ja pliiatsisisendiga horisontaalset
  pühkimist mõlemas suunas. Slaid vahetub piisava teepikkuse või kiiruse järel
  ning sisu liigub žesti ajal vahetult osutiga kaasa.
- 10 px suunalukk eristab horisontaalset karusselližesti vertikaalsest
  lehekerimisest. Hero säilitab brauseri loomuliku vertikaalse kerimise ja
  pinch-zoom'i ning interaktiivse nupu pealt alanud puude ei käivita slaidivahetust.
- Mobiilivaates paiknevad hero eelmise ja järgmise slaidi nooled alumises
  paremas nurgas. Punktindikaatorid ja loendur on mobiilis peidetud, kuid
  töölauavaate kinnitatud 34 / 7 / 22 px juhtnupugeomeetria säilib.
- Lisatud regressioonitestid horisontaalse edasi-tagasi pühkimise, vertikaalse
  kerimise säilimise ja mobiilse ainult-noolte paigutuse jaoks.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.45 — 2026-08-15

### Muusikavaate pealkirjade selge hierarhia

- `/music` peamine pealkiri kasutab nüüd sama 38–72 px skaalat nagu filmi- ja
  seriaalikogude pealkirjad.
- Muusika avalehe „Kõik lood” sissepääsu pealkiri kasutab väiksemat 26–40 px
  sektsiooniskaalat ega konkureeri enam lehe pealkirjaga.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.44 — 2026-08-14

### Ühtlane muusikavaade ja ingliskeelne esmakäivitus

- `/music` peamine lehepealkiri kasutab nüüd sama 38–72 px skaalat nagu filmi-
  ja seriaalikogud. Avalehe „Kõik lood” sissepääsu pealkiri kasutab vaiksemat
  26–40 px sektsiooniskaalat. `/music/tracks` lehepealkiri jääb 42–58 px ning
  albumi ja artisti detailvaadete pealkirjad on piiratud 72 piksliga.
- Salvestatud keelevalikuta uus brauser alustab ingliskeelse kasutajaliidesega.
  Uue serveri metaandmete keel on `en-US` ning eelistatud subtiitrikeel `en`.
  Olemasolevaid brauseri- ja serveriseadeid uuendus ei kirjuta üle.
- Eesti ja inglise kataloog sisaldavad mõlemad 897 võtit. Uus build-kontroll
  kinnitab võtmete 1:1 katvuse, placeholder'ite säilimise, ingliskeelsete
  väärtuste olemasolu ning tõlkefunktsioonist mööda minevate eestikeelsete
  UI-stringide puudumise.
- Inglise kasutajaliides tõlgib nüüd ka serverist tulevad taasesitus- ja
  encoder'i põhjused, post-play soovituste põhjused, uuenduse olekud ning
  tuntud dünaamilised uuendus- ja OpenSubtitlesi vead.
- Väljalase sisaldab portable/uuendus-ZIP-i ja lähtepaki; eraldi setup-EXE-d
  selle versiooni jaoks ei koostata.

## 0.8.43 — 2026-08-09

### Töökindel Windowsi sisselogimiskäivitus ja kasutajapõhine installer

- Windowsi sisselogimisel ei käsitleta Exploreri teavitusalale tray ikooni
  lisamise esimest ebaõnnestumist enam fataalse veana. KoduMedia proovib
  `Shell_NotifyIcon` integratsiooni kuni 31 korda ühe-sekundilise intervalliga.
- Kui tray integratsioon jääb püsivalt kättesaamatuks, ei sulgeta enam töötavat
  veebiserverit. KoduMedia jääb brauseris ja koduvõrgus kasutatavaks ning
  kirjutab vea `data\kodumedia.log` faili.
- Käivituslogi salvestab versiooni, EXE ja data-kausta ning eristab
  `windows-login` ja `interactive` käivitust. See teeb võimaliku järgmise
  Windowsi startup-tõrke nähtavaks ka ilma konsooliaknana.
- Lisatud on NSIS-põhine kasutajapõhine installer. Vaikeasukoht on Windowsi
  User Program Files ehk `%LOCALAPPDATA%\Programs\KoduMedia`; server ei vaja
  administraatoriõigusi ja olemasolev iseuuendaja säilitab kirjutusõiguse.
- Installer migreerib eelmisest portable-asukohast `.env`, seaded, kataloogi,
  vaatamisajaloo ja vahemälud. Mahukad `transcodes` ja `updates` töökaustad ning
  vanad logid jäetakse teadlikult kopeerimata; lähteandmeid ei kustutata.
- Installer loob Start-menüü otseteed, Windows Installed apps eemaldamiskirje
  ning värske `Run` käsu ja kustutab sama nimega vana keelatud
  `StartupApproved` oleku. Eemaldaja säilitab kasutaja data-kausta.
- Regressioonitestid kontrollivad ajutisest tray veast taastumist, katsete
  ammendumist ja login-startup argumendi tuvastamist.

## 0.8.42 — 2026-07-30

### Windowsi sisselogimisel käivitamise olek vastab tegelikkusele

- System tray menüü ei loe automaatkäivitust enam lubatuks ainult
  `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` kirje olemasolu põhjal.
  Arvesse võetakse ka Windowsi eraldi `StartupApproved` olekut, mida kasutavad
  Task Manageri ja Windows Settingsi Startup apps vaated.
- Kui Windows on KoduMedia startup-kirje keelanud, kuvatakse tray menüü valik
  nüüd õigesti ilma linnukeseta. Valiku uuesti sisselülitamine kirjutab
  parajasti töötava `kodumedia.exe` täpse absoluutse tee ning eemaldab vana
  keelatud kinnituse.
- Väljalülitamine eemaldab nii Run-käsu kui ka selle juurde kuuluva
  StartupApproved oleku. Registri puuduvat väärtust käsitletakse ohutult ning
  muid startup-rakendusi ei muudeta.
- Regressioonitestid katavad Windowsi lubatud ja keelatud olekud, vana
  installitee parandamise, eksitava path-substring vaste ning mõlema
  registrikirje eemaldamise.

## 0.8.41 — 2026-07-26

### Hero säilib treileri tõrke korral

- YouTube’i treileri või Player API tõrge ei eemalda enam filmi ega seriaali
  hero-karussellist. Kui ühtegi treilerikandidaati ei saa esitada, jääb slaid
  staatilise backdrop-pildiga alles.
- Eemaldatud on avalehe `unavailableHeroIds` filter, mis kahandas karusselli
  refresh’ide ja järjestikuste treileritõrgete järel 4 → 2 → 0 slaidini ning
  peitis viimase kandidaadi järel kogu hero.
- 0.8.40 vigane 30-päevane `kodumedia.trailer-failures.v1` vahemälu
  kustutatakse automaatselt. Uus vahemälu säilitab koos tegeliku veakoodiga
  ainult video enda püsivad YouTube’i vead; ajutisi võrgu-, API-, watchdog- ega
  HTML5 playeri tõrkeid püsivalt ei blokeerita.
- Hero 480 ms pilt–trailer ristüleminek, vaigistatud autoplay ja kaitstud
  juhtnupugeomeetria (34 / 7 / 22 px) säilivad muutmata.
- Regressioonitestid kontrollivad kahe järjestikuse treileritõrke järel
  staatilise hero säilimist, vana vahemälu migratsiooni ja ajutiste vigade
  mittepüsivat käsitlemist.

## 0.8.40 — 2026-07-26

### Hero slideri juhtnuppude visuaali taastamine

- Hero slideri juhtnupud kasutavad taas kinnitatud 0.8.38 geomeetriat:
  nooled on 34 × 34 px, tavalised punktid 7 × 7 px ja aktiivne punkt 22 × 7 px.
- 0.8.39-s lisatud 32 × 34 px punktikonteinerid ning coarse-pointer-režiimi
  44 px hero juhtnupud on eemaldatud. Sisendirežiimi muutus ei saa enam hero
  juhtnuppude suurust ega juhtnupurea kõrgust kasutamise ajal muuta.
- Hero juhtnuppude suurus, DOM-kuju ja coarse-pointer erand on lukustatud
  lähtekoodi, production-CSS-i ning embedded UI regressioonikontrollidega.
- Kõik ülejäänud 0.8.39 puute-, kerimis-, fookuse- ja klaviatuurinavigatsiooni
  parandused säilivad muutmata.

## 0.8.39 — 2026-07-26

### Puute- ja klaviatuurinavigatsiooni viimistlus

- Avalehe meediaread, mobiilne põhinavigatsioon, settingute navigatsioon,
  hooajavalik ning näitlejakarussell kasutavad native horisontaalset puutekerimist
  koos inertsi, vertikaalse kerimise ja pinch-zoom'i säilimisega.
- Näitlejakarusselli hiirega lohistamine ja nähtav kerimisvihje säilivad, kuid
  touch-ekraanil ei asenda JavaScript enam brauseri loomulikku momentum-scroll'i.
- Meediakaardi klaviatuuri- või puldifookus kuvatakse ühe tagasihoidliku 2 px
  sisemise rõngana ainult thumbnail'il. Kogu pildi ja teksti ümber olnud teine
  fookusraam on eemaldatud.
- Ülejäänud ruumiliselt navigeeritavad juhtelemendid kasutavad clipping-kindlat
  sisemist fookusrõngast ja scroll-margin'i, mistõttu servas olev selektsioon ei
  kao scrollport'i taha.
- Hero sees oleva nupu fookus ei joonista enam teist suurt raami kogu hero
  ümber. Kompaktsed 7 px / 22 px indikaatorid säilisid, kuid coarse-pointer
  seadmetes on nende ning noolte puutealad suuremad.
- Esimene nooleklahvivajutus loob prognoositava algfookuse. Komponentide enda
  käsitletud nooleklahvid ei käivita enam lisaks globaalset ruumilist liikumist.
- Muusikapleieri transport, ajajoon, helitugevus ja detailinupp osalevad nüüd
  klaviatuuri ning tulevase puldi ruumilises navigatsioonis.

## 0.8.38 — 2026-07-26

### Kerge staatiline üldvaadete taust

- Avaleht, filmide kogu, seriaalide kogu, muusika avaleht ja seaded kasutavad
  nüüd sama kinnitatud tumedat või heledat CSS-gradienttausta.
- Taust on üks fikseeritud kiht, milles pole pildifaili, `filter`-blur'i,
  `backdrop-filter`-it, animatsiooni ega thumbnail-hover'ist lähtuvat olekut.
- Avalehe thumbnail-hover ei laadi, dekodeeri ega vaheta enam täisekraani pilti
  ning ei põhjusta selle jaoks Reacti kordusrenderdust.
- Tume teema hoiab tumedad toonid domineerivana ning kasutab väikseid roosa ja
  helesinise kuma alasid. Hele teema kasutab pärlhalli põhja ja madala
  küllastusega roosa-siniseid aktsente.
- Konkreetse filmi, seriaali ja episoodi detailvaate olemasolev udustatud
  meediapildi loogika, hero meediapilt ja 480 ms pilt–trailer ristüleminek jäid
  muutmata.
- Marsruudi- ja build-regressioonid piiravad staatilise tausta täpselt viie
  kinnitatud üldvaatega ning nõuavad detailvaadete blur-kihi säilimist.

## 0.8.37 — 2026-07-26

### Taastatud hero kinolik ristüleminek

- Hero staatiline taustapilt hajub taas sujuvalt mängivasse trailerisse sama
  480 ms `opacity`-ristüleminekuga, mida kasutas varasem soovitud versioon.
- Trailerikiht muutub nähtavaks alles pärast YouTube Player API päris
  `PLAYING` olekut; laadimine, puhverdamine ja vead jätavad taustapildi
  katkematult nähtavaks.
- Hero autoplay, vaigistatud heli, trailerifilter, karusselli juhtnupud,
  indikaatorid, iframe'i crop ja ambient-taust jäid muutmata.
- 480 ms kinolik üleminek on dokumenteeritud püsiva hero-disainireeglina ning
  regressioonikontroll takistab selle asendamist tavapärase 200 ms UI-ajastusega.

## 0.8.36 — 2026-07-25

### Piiratud kettakasutus, diferentsiaalne uuendus ja puhas install

- Uuendaja kontrollib paigaldatud failide suurust ja SHA-256 räsi ning varundab/asendab ainult muutunud manifestifailid. Suured identsed FFmpegi ja FFprobe'i binaarid jäetakse puutumata.
- Pärast edukat või ebaõnnestunud terminalset uuendusolekut eemaldatakse `packages`, `staged` ja `plans` tööfailid. Rollback-ajaloos säilitatakse kaks uusimat eelmist versiooni; uuenduslogi roteeritakse 2 MiB juures.
- Optimeeritud videovahemälu jõustab korraga kasutaja mahupiiri ja vaba ruumi reservi. Vaikimisi hoitakse vabana vähemalt 10 GB või 10% kettast ning seadete vähendamine rakendub kohe.
- FFmpegi build fingerprint põhineb binaari SHA-256 sisuräsil, nii et identne kaasapandud tööriist ei muuda olemasolevat transkodeerimisvahemälu pärast rakenduse uuendust kasutuks.
- Serveri käivitamisel eemaldatakse katkenud eelmise protsessi `data/transcodes` seansid. Optimeeritud cache eemaldab puuduva või muutunud lähtefailiga orvud.
- Muusika artwork normaliseeritakse kuni 800 px JPEG-iks, identsed pildid deduplitseeritakse, eemaldatud meedia cache puhastatakse ning kogumaht piiratakse 2 GiB-ga.
- Muusika metaandmete cache on piiratud 1 GiB-ga, lookup-vastused aeguvad 180 päevaga ning eemaldatud failide fingerprint/audioanalüüs kustutatakse. Tunnusmuusika piir on 2 GiB ja teisendatud subtiitrite piir 1 GiB; kasutaja allalaaditud subtiitreid automaatselt ei kustutata.
- HTTP-logimine väldib tavaliste HLS-segmentide ja staatiliste failide sünkroonset kirjutamist; serverilogil ja uuenduslogil on jooksvalt toimiv suurusepõhine rotatsioon.
- JSON-kataloog kasutab kuni 256 filtreeritud/sorditud päringu indeksit ning korduv lehekülgede sirvimine ei skanni ega sordi enam iga lehe jaoks kogu meediakogu.
- Seadete lehele lisati **Kettakasutus** koos kategooriate, piiride, vaba ruumi ja reservi kuvaga ning localhost-only ohutu hoolduse tegevustega.
- Uus seadete skeem 10 migreerib olemasolevad installid, kuid rakenduse esimene käivitus ei eelda vana `data` kausta ega ühtegi varasemat versiooni.

## 0.8.35 — 2026-07-25

### Püsiv ambient-tausta topeltpuhver

- Video kaaderhaaval analüüsiga tuvastatud jada `vana → uus → vana → uus` eemaldati: kaks täisekraani ambient-kihti püsivad nüüd kogu komponendi eluea DOM-is.
- Uus taust dekodeeritakse enne läbipaistvasse puhvrisse asetamist. Puhver püsib kaks animatsioonikaadrit `opacity: 0` olekus ning ainult see kiht hajub 360 ms jooksul vana pildi peale.
- Mõlemad kihid on stabiilselt komposiitorisse tõstetud; hover'i ajal ei muudeta enam `will-change` olekut ega looda või eemaldata suuri blur-kihte.
- Ülemineku lõpus muutub uus kiht ühe commit'iga aluskihtiks. Järgmine järjekorras olev pilt alustab alles pärast peidetud puhvri oleku eraldi ekraanikaadrit.
- Kiire hover jätab järjekorda ainult viimase dekodeeritud soovitud tausta ega katkesta pooleliolevat opacity-üleminekut.
- Lisatud regressioonid püsiva kahe kihi, nähtamatu puhvri, täieliku katvuse ja viimase kiire hover'i eelistamise jaoks.
- Hero juhtnuppude kujundust, vaigistatud autoplay'd, trailerifiltrit ega thumbnaili hover-ajastust ei muudetud.

## 0.8.34 — 2026-07-25

### Taastatud hero juhtnupud ja välkumiseta ambient-taust

- Hero slaidivaliku indikaatorid kasutavad jälle varasemat kompaktset kujundust: mitteaktiivne indikaator on 7 px ja aktiivne 22 px. 0.8.33-s lisatud nähtamatud 22 px kastid ning nende tekitatud hõre paigutus on eemaldatud.
- Juhtnuppude markup, nooled, loendur, helinupp ja muud hero stiilid jäid muutmata.
- Ambient-tausta tugevus rakendub nüüd ühe korra kogu kihikonteinerile. Eelmine pilt püsib täielikult nähtav, kuni uus dekodeeritud pilt selle pehme opacity-üleminekuga katab.
- Eelmise taustakihi eemaldamine toimub tegeliku `opacity` ülemineku lõpus; ajastipõhine puhastus on ainult konservatiivne varuvariant.
- Kiire thumbnailide kohal liikumine järjekorrastab valmis dekodeeritud taustad ega katkesta pooleliolevat ristüleminekut.
- Lisatud on regressioonitestid püsiva vana kihi, üleminekujärgse puhastuse ja kiire järjestikuse taustavahetuse jaoks.

## 0.8.33 — 2026-07-25

### Vaigistatud hero ja snappy kasutajaliides

- Hero trailer käivitub taas vaikimisi vaigistatult ning heli saab kasutaja samast helinupust sisse lülitada. Trailerita ja jooksvalt vigaseks osutunud kirjed jäävad endiselt hero-ringist välja.
- Kõrge sagedusega kerimine on kohene: globaalne, klaviatuuriga käivitatud ning detailvaate ankru-kerimine ei kasuta enam aeglast smooth-scrolli.
- Ambient-taust kasutab blur-kihi jaoks väiksemat TMDb pilti, väldib sama pildi korduvat dekodeerimist ning eemaldab eelmise filtreeritud kihi kohe pärast 200 ms opacity-crossfade'i.
- Näitlejate karusselli scroll ei uuenda enam igal kaadril Reacti olekut; scrollbar liigub otse GPU-sõbraliku transformiga.
- Hero indikaator, teavitused ja subtiitrite asend ei animeeri enam `width`, `padding` ega `bottom` paigutusomadusi. Sagedane thumbnaili tagasiside kestab 120 ms ning piirdub transformiga.
- Fikseeritud ja scrollivad suuremad klaaspinnad kasutavad blur-resamplingu asemel visuaalselt lähedasi läbipaistmatumaid pindu. Avalehe mahukas vahemälu serialiseeritakse idle-ajal.
- Lisatud on `review-animations` ja `improve-animations` auditist lähtuvad täpsed animatsioonijõudluse plaanid ning ambient-tausta regressioonitestid.

## 0.8.32 — 2026-07-25

### Hero heli, trailerifilter ja sujuvam sirvimine

- Hero proovib traileri käivitada kohe häälega. Kui brauser häälega autoplay blokeerib, oodatakse esimest kasutajasisendit ja jätkatakse häälega, mitte vaigistatud varuvariandiga.
- Trailerivõtmeta kirjed filtreeritakse hero kandidaatide seast välja. Jooksvalt vigaseks osutunud trailer eemaldab vastava kirje hero-ringilt.
- Thumbnaili sagedane hover ei renderda enam avalehe ridu kohe uuesti: eelvaade käivitub lühikese 110 ms kavatsuspuhvri järel ning kaardid ja read on memoiseeritud.
- Ambient-taustad dekodeeritakse enne vahetust ning ristuvad komposiitkihis ainult `opacity` kaudu; suuri fikseeritud pindu ei udustata enam kerimise ajal.
- Ekraanivälised read ja kaardid kasutavad `content-visibility` optimeerimist ning `prefers-reduced-motion` vähendab liikumist süsteemiseadistuse järgi.

## 0.8.31 — 2026-07-25

### Detailvaate ja ambient-tausta stabiliseerimine

- Taastatud on näitlejate karusselli nähtav õhuke kerimisriba ning lisatud hiirega lohistamine.
- Filmi-, seriaali- ja hooajavaate fikseeritud taustakiht katab nüüd kogu vaate ilma korduva pildi või horisontaalse õmbluseta.
- Meediainfo kuvab algse failinime ja serveri täieliku failitee ilma ellipsi või ühe rea kärpimiseta.
- Avalehe ambient-taust hoiab eelmise pildi nähtaval kuni uus pilt on dekodeeritud ning vahetab kihid opacity-crossfade'iga, vältides esimesel eelvaatel välgatust.

## 0.8.30 — 2026-07-25

### Kasutajaliidese taastav stabiliseerimine

- Eemaldatud on väljalaskesse jäänud legacy hero bootstrap-kiht. Avalehel renderdatakse nüüd ainult üks Reacti `FeaturedHero`, mistõttu kaovad dubleeriv hero, topelt slaiditaimerid ja kattuvad navigatsioonikihid.
- Taastatud on esmase seadistuse akordioni päris komponentide paigutus: pealkiri ja abitekst on eraldatud, neli sammu kasutavad ühtset hierarhiat ning tegevusnupud järgivad primaarse ja sekundaarse tegevuse olemasolevat CVI-d ilma eraldi glow-efektita.
- Seadete lehe pealkirjade, ülevaate, külgmenüü, vormiveeru ja alajaotiste mõõdud on viidud tagasi loetavasse proportsiooni. Parem sisuveeru laius, väljade maksimaalne realaius ja vertikaalsed vahed on nüüd ühe autoriteetse, seadete vaatele piiratud stiilikihi all.
- Meedia tööriistad, töövalmiduse kaardid ja sisevõrgu hoiatus on rühmitatud järjepidevateks plokkideks; juhuslikud täislaiuses read ja üksteisest lahti hõljuvad kirjeldus-/kontrolliveerud on eemaldatud.
- Filmi ja seriaali detailvaate taustakiht on isoleeritud oma komponendi piiridesse. Eemaldatud on näitlejate ja meediainfo vahele tekkinud hele blur-riba ning meediainfo veerud kohanduvad nüüd sisu järgi.
- Näitlejate rida kasutab kontrollitud horisontaalset overflow'd, peidetud brauseri scrollbari ja stabiilseid kaardimõõte. Pikad failinimed ja asukohad ei venita enam kogu tehnilise info paigutust, vaid kärbitakse loetavalt.
- Lisatud on regressioonikontrollid, mis keelavad legacy hero failide/tagide tagasituleku ning kontrollivad üht Reacti hero arhitektuuri, esmase seadistuse struktuuri ja detailvaate stabiliseeriva stiilikihi olemasolu.
- Väljalaske veebibundlit kontrolliti renderdatud vaadetes: avalehel üks hero, seadetes kuus põhijaotist, esmase seadistuse neli sammu ning detailvaates katkestusteta taust ja tehnilise info kaart.

## 0.8.29 — 2026-07-25

### Seadete sisu ja paigutuse terviklik korrastus

- Seadete parempoolne sisu järgib nüüd täpselt vasakmenüü kuut jaotist ja sama järjekorda: server ja töövalmidus, meediakogud ja metaandmed, taasesitus ja heli, subtiitrid, välimus ja keel ning uuendused ja diagnostika.
- Lisatud on jagatud `SettingsGroup` kompositsioon, mis ühtlustab jaotiste pealkirjad, kirjeldused, ikoonid, vahed ja ankrud. Vasakmenüü märgib kerimisel aktiivse jaotise ilma navigeerimiselemente nihutamata.
- Parempoolse sisu maksimaalne laius on ühtlustatud 980 pikslile ning vormikontrollide sisulaius 720 pikslile. Seadistuste alajaotised kasutavad läbivalt 180–220 pikslit laia kirjeldusveergu ja paindlikku kontrolliveergu.
- Liiga lähestikku olnud väljad, tegevused ja alajaotised said ühtse 20–32 pikslise vahemustri. Kahe veeruga väljad langevad kitsal ekraanil üheks veeruks ning külgmenüü muutub horisontaalseks kompaktseks navigeerimiseks.
- Meediakogud, metaandmed, taasesitus, tunnusmuusika, subtiitrid, keel, teema ja uuendused on rühmitatud oma sisuliste alajaotiste alla; harva kasutatavad tehnilised valikud jäävad jagatud `Disclosure` komponentide sisse.
- Eemaldatud on vana 0.5.9 seadete CSS-kiht. Seadete paigutusel on nüüd üks autoriteetne stiiliplokk, olemasolevad CVI komponendid ja ühine reduced-motion/hover-poliitika.
- Lisatud on regressioonikontrollid vasakmenüü ja sisu järjekorra, jagatud grupikomponendi, ühe autoriteetse navigatsioonistiili ning kleepuva salvestusriba jaoks.

## 0.8.28 — 2026-07-25

### Veebiliidese käivituseparandus

- Parandatud 0.8.27 veebibundli süntaksiviga, mis takistas Reacti käivitumist ja jättis kõik lehed tühjaks.
- Reacti vendor-kihi eksport lõpetab nüüd vana `var` deklaratsiooni enne globaalse ekspordi loomist.
- Sisseehitatud moodulilaadur toetab nüüd ka `react-dom` moodulit, mida jagatud Dialog kasutab portaalide loomiseks.
- Lisatud regressioonikontrollid vendor-kihi lausepiiri ja `react-dom` ekspordi jaoks; väljalaskepaketi Reacti mountimist kontrolliti päris brauseris.

## 0.8.27 — 2026-07-25

### Seadete ja playeri süsteemne korrastus

- Seadete külgmenüü on koondatud kuueks ülesandepõhiseks alaks: server ja töövalmidus, meediakogud ja metaandmed, taasesitus ja heli, subtiitrid, välimus ja keel ning uuendused ja diagnostika.
- Muudetud seadete korral ilmub lehe allossa olemasolevat CVI-d kasutav kleepuv salvestusriba. Kasutaja saab muudatused salvestada, kontrollida või viimati salvestatud väärtustele taastada; lehelt lahkumisel hoiatatakse salvestamata muudatuste eest.
- Harva kasutatavad tööriistateed ja taasesituse tehnilised valikud on viidud jagatud Disclosure-komponendiga täpsemate seadete alla. Uusi eraldiseisvaid visuaalseid stiile ei lisatud.
- Märkeruudud ja progressiribad kasutavad nüüd jagatud Switch- ja ProgressBar-komponente. Progress muutub transformiga, mitte layouti ümber arvutava laiuse animatsiooniga.
- Playeri nähtavuse, timeline'i, rajamenüüde, diagnostika, taasesitusoleku, subtiitriotsingu ja post-play loogika on eraldatud väiksemateks mooduliteks. Peamine PlayerModal vastutab endiselt sessiooni ja meedia orkestreerimise eest.
- Playeri juhtnupud ilmuvad kasutaja sisendi järel kohe ning peituvad tegevusetuse järel ainult esituse ajal. Peidetud juhtnupud on klaviatuurile inertsed, menüüd kasutavad jagatud Popover/Menu primitiive ja toetavad Arrow-, Home-, End- ning Escape-klahve.
- Playeri üleminekud kasutavad ainult opacity/transform omadusi, hover-liikumine on piiratud täpse osutusseadmega ning reduced-motion režiimis eemaldatakse ruumiline liikumine. Post-play kaartide järjestikune dekoratiivne sisenemine eemaldati.
- Lisatud regressioonitestid seadete dirty-state'i, taastamise, playeri menüü klaviatuurikäitumise, inertsete juhtnuppude ning transformipõhiste progressiribade kohta.

## 0.8.26 — 2026-07-25

### Kohene ja stabiilne avalehe hero

- Avalehe põhiandmed laaditakse nüüd prioriteetselt eraldi aeglasematest episoodi- ja žanriridadest, mistõttu hero ei oota enam kogu avalehe viimast päringut.
- Viimati edukalt kuvatud avaleht ja esiletõstetud valik salvestatakse brauseri püsiva vahemäluna. Avalehele naastes või lehte värskendades taastatakse hero enne järgmist ekraanijoonistust ning andmeid värskendatakse vaikselt taustal.
- Esiletõstetud taustapildid saavad kõrge laadimisprioriteedi ja esimese pildi eelsoojendus ei blokeeri kasutajaliidest üle 180 ms.
- Külmkäivituse ajal ei kuvata osaliselt valmis avalehte, kuhu hero hiljem vahele hüppaks; read muutuvad nähtavaks alles siis, kui esiletõstetud plokk on paigas.
- Hero korduv reveal-animatsioon on eemaldatud. Menüü kaudu avalehele naasmine ei mängi enam dekoratiivset sisenemist uuesti.
- Slaidinavigatsiooni olemasolev kompaktne CVI ja käitumine jäid muutmata.

## 0.8.25 — 2026-07-25

### Hero slaidinavigatsiooni väljalaskeparandus

- Parandatud 0.8.24 väljalaskes tekkinud regressioon, kus hero slaidinavigatsiooni Reacti lähtekood oli olemas, kuid vastav renderdus ei jõudnud sisseehitatud veebibundlisse.
- Taastatud on varem töötanud hero kontroller, mis loob eelmise/järgmise nupud, slaidipunktid ja loenduri ning kasutab täpselt olemasolevaid `featured-hero__*` CVI klasse. Uut visuaalset stiilikihti ei lisatud.
- Hero slaidivalija kasutab endiselt kompaktset 34 px noolte, 7 px punktide ja 22 px aktiivse indikaatori kujundust.
- Lisatud regressioonikontrollid, mis nõuavad nii lähtekoodi kui sisseehitatud distributsiooni puhul hero kontrolleri laadimist ja selle varade olemasolu.

## 0.8.24 — 2026-07-25

### Taastatud hero ning rahulikum sirvimine

- Hero slaidinavigatsioon taastati versiooni 0.8.22 kompaktsesse CVI-sse: 34 px nooled, 7 px punktid ja 22 px aktiivne indikaator. 0.8.23 suurendatud pseudoelemendi lahendus eemaldati.
- Windowsi sisseehitatud veebiliidesest eemaldati lõplikult vanad `home-featured`, `initial-setup` ja `series-themes` runtime-failid. Reacti kasutajaliidest ei muudeta enam pärast renderdust teise DOM-kihiga ning hero stiilid ei saa omavahel konflikti minna.
- Meediakaartide sagedast hover-liikumist vähendati: vertikaalne hüpe eemaldati, suurendus on tagasihoidlik ja rakendub ainult täpse osutusseadme korral. Klaviatuurifookus kasutab liikumise asemel selget fookusrõngast.
- Kogu filtrite väljad järgivad ühiseid kontrolli kõrguse, raadiuse, pinna ja fookuse tokeneid.
- Filmide ja episoodide vaatamisprogress renderdatakse `scaleX()` transformiga. Episoodiridade hover ei animeeri enam paddingut ega põhjusta paigutuse ümberarvutamist.
- UI regressioonikontroll kontrollib nüüd ka sisseehitatud distributsiooni, kompaktset hero navigatsiooni, runtime-failide puudumist ja transformipõhiseid progressiribasid.

## 0.8.23 — 2026-07-25

### Ühtne UI vundament

- Kujundustokenid koondati ühte autoriteetsesse `tokens.css` faili. Tume ja hele teema kasutavad samu semantilisi pinna-, teksti-, äärise-, fookuse-, raadiuse- ja kihistustokeneid.
- Lisati ühine motion-poliitika: 120–240 ms tavaliidese kestused, tugev ease-out, 480 ms kinolik hero üleminek, transform/opacity-põhine liikumine ja tsentraalne `prefers-reduced-motion` tugi.
- Lisati jagatud `Dialog`, `Popover`, `Menu`, `IconButton` ja `Field` komponendid. Dialog haldab Escape'i, fookuse püünist ja taastamist ning body scroll lock'i; Popover haldab väljaspool klõpsamist, fookuse taastamist ja avanemissuunale vastavat transform-origin'i.
- Otsing, failibrauser, filmi-, seriaali- ja muusikaloo detailvaated, päise kogude menüü, teavitused ning valitud seadete väljad viidi ühistele primitiividele.
- Reacti lähtekoodi HTML-ist ja `public` kaustast eemaldati `home-featured`, `initial-setup` ning `series-themes` DOM-i ümberkirjutavad runtime-kihid; vastav funktsionaalsus elab Reacti komponentides.
- Hero trailer ei lülita heli enam klaviatuurifookuse tõttu sisse. 44 px sihtalad, press-feedback ja pointer-põhised hover-olekud ühtlustati; carousel'i aktiivne indikaator kasutab laiuse asemel `scaleX()` transformi.
- Lisati Dialogi ja Popoveri regressioonitestid ning `verify-ui-foundation.mjs`, mis kontrollib runtime-kihi puudumist, motion-tokenite olemasolu, määratlemata CSS-muutujaid ja keelatud `transition: all` kasutust.

## 0.8.22 — 2026-07-25

### Avalik npm-register ja korratav lähtekoodibuild

- `apps/web/package-lock.json` kõik 136 sisemise paketipeegli viidet asendati ametliku `https://registry.npmjs.org/` aadressiga.
- Lisati projektipõhine `apps/web/.npmrc`, mis kasutab avalikku npm-registrit ning väldib tarbetut audit- ja rahastusteavet buildi ajal.
- Windowsi `build.ps1` ja `test.ps1` annavad `npm ci` käsule avaliku registri otsese argumendina, mistõttu välise masina vigane `NPM_CONFIG_REGISTRY` ei saa lähtekoodibuildi sisemisele registrile suunata.
- Lisati `verify-public-registry.mjs` regressioonikontroll. See kontrollib lockfile'i enne buildi, tüübikontrolli ja teste ning keelab sisemised OpenAI aadressid ja ootamatud registrihostid.
- `package.json` kirjeldab nüüd toetatud Node.js/npm versioone ja soovituslikku npm paketihalduri versiooni. Rakenduse runtime-funktsionaalsus ei muutunud.

## 0.8.21 — 2026-07-25

### Tunnusmuusika valik kasutab olemasolevat seadete CVI-d

- Seriaalide tunnusmuusika rippmenüü kasutab nüüd täpselt sama `SelectControl` struktuuri, klasse, nooleikooni ja olekustiile nagu teised seadete lehe valikud.
- Versioonis 0.8.20 lisatud eraldi tumeda/heleda teema select-reeglid eemaldati; tunnusmuusika seade ei defineeri enam oma värve, raadiust, kõrgust ega fookuskujundust.
- Sama ühiskomponendi struktuur lisati ka brauseripoolsele ühilduvuskihile, et kujundus oleks ühesugune nii tavapärase Reacti renderduse kui olemasoleva veebibundli korral.
- Tunnusmuusika laadimise, vahemälu, esituse ja seadete väärtuste funktsionaalsus ei muutunud.

## 0.8.20 — 2026-07-25

### Töökindlam tunnusmuusika ja loetav seadete valik

- ThemerrDB tunnusmuusika esmakordne laadimine käib nüüd brauseripäringust sõltumatu serveripoolse taustatööna, mis jätkub ka siis, kui kasutaja seriaali vaatest lahkub.
- Sama seriaali kohta jagavad korduvad päringud ühte ettevalmistustööd; edukas M4A-fail salvestatakse endiselt püsivasse `data/themes/series` vahemällu.
- Veebiliides kontrollib valmisolekut vaikselt taustal, ei kuva ettevalmistamise olekut ning alustab heli ainult siis, kui sama seriaali vaade on veel aktiivne. Episoodi käivitamisel tühistatakse brauseripoolne ootus, mistõttu heli ei saa episoodi vaatamise ajal hiljem iseseisvalt käivituda.
- Ajutine ThemerrDB, YouTube’i või võrguviga ei märgi seriaali enam kogu brauseriseansiks kasutamatuks; server lubab kontrollitud korduskatseid.
- Tunnusmuusika vaikehelitugevus muudeti 20 protsendilt 15 protsendile; kahe sekundi fade-in/fade-out ja automaatse kordamise keeld säilisid.
- Seadete rippmenüüle lisati korrektne tume/heleda teema `color-scheme` ning selged valiku tausta- ja tekstivärvid.
- Lisati regressioonitestid taustatöö jagamise, päringu tühistamisest sõltumatuse ja uue 15% esituskonfiguratsiooni kohta.

## 0.8.19 — 2026-07-25

### ThemerrDB seriaalide tunnusmuusika

- Seriaali, hooaja ja episoodi detailvaates saab nüüd esitada ThemerrDB kaudu leitud seriaali tunnusmeloodiat.
- Heli kasutab vaikimisi 20% helitugevust ning kahe sekundi pikkust sisse- ja väljahajumist; tunnuslugu ei kordu automaatselt.
- Sama seriaali üldvaate, hooaegade ja episoodide vahel liikudes jätkub sama helivoog algusest alustamata. Episoodi käivitamisel või seriaali vaadetest lahkumisel hajub heli välja.
- Seadetesse lisati **Esita seriaalide tunnusmuusikat** valikutega **Alati**, **Ainult esimesel avamisel** ja **Väljas**, koos eesti- ja ingliskeelsete tekstidega.
- ThemerrDB vaste leitakse seriaali TMDb ID järgi. Edukalt hangitud M4A-fail salvestatakse püsivalt serveri `data/themes/series` vahemällu.
- Windowsis hangib server puuduva yt-dlp ametlikust fikseeritud väljalaskest ainult esimesel vajadusel, kontrollib binaari SHA-256 summat ja hoiab seda serveri andmekaustas. Kohandatud tee saab määrata `MEDIA_SERVER_YTDLP` kaudu.
- Lisatud on eraldiseisvad ThemerrDB, yt-dlp, vahemälu, esituse konfiguratsiooni ja seadete migratsiooni moodulid ning regressioonitestid.

## 0.8.18 — 2026-07-24

### Ulmefilmid ja avalehe hero järjekord

- Avalehe žanriridade lõppu lisati **Ulmefilmid / Science Fiction**, toetades žanrinimesid `Science Fiction`, `Sci-Fi` ja `Ulme`.
- Ulmefilmid kasutavad sama kaalutud ja kategooriate vahel kordusteta jaotusloogikat nagu teised avalehe žanriread.
- Esiletõstetud hero kuvatakse alati avalehe esimese sisusektsioonina enne **Jätka vaatamist** rida.
- Parandus rakendati nii Reacti avalehe paigutusele kui brauseripoolsele hero-täiustusele ning lisati eesti- ja ingliskeelsed tõlked.

## 0.8.17 — 2026-07-24

### Kaalutud ja kordusteta avalehe filmikategooriad

- Avalehe žanriridadesse jagamisel kasutatakse filmi žanride järjekorda kaaluna: esimene avalehel toetatud žanr määrab filmi põhikategooria.
- Animatsioon on erandina alati eksklusiivne: animatsioonifilmi ei näidata enam märuli, seikluse, komöödia, romantika ega õuduse reas isegi siis, kui need žanrid on samuti määratud.
- Sama film kuvatakse avalehe žanriridades ainult üks kord; mitme sobiva žanriga film läheb oma suurima kaaluga kategooriasse.
- Kategooriate sees säilib IMDb hinnangu järgi kahanev järjestus ning tühje kategooriaridu ei renderdata.
- Lisatud regressioonitestid animatsiooni välistamise, žanrijärjekorra kaalu ja kategooriatevaheliste duplikaatide vältimise kohta.

## 0.8.16 — 2026-07-24

### Esmase seadistuse tegevused ja täielik kakskeelsus

- Esmase seadistuse akordioni kõik tekstid on olemas nii eesti kui inglise keeles ning keel vahetub kohe koos ülejäänud kasutajaliidesega.
- Viimase sammu **Salvesta seaded** on nüüd päris tegevusnupp, mis salvestab hetkel vormil olevad väärtused.
- **Skanni kõik meediakogud** laadib serverist kõik kaustadega kogud ja käivitab nende skannimise paralleelselt ühe vajutusega.
- Mõlemad tegevused on teistest sektsiooni otseteedest prominentsemad, kuid kasutavad sama KoduMedia CVI-d ja ligipääsetavaid busy-olekuid.
- Juhend mainib, et kõiki kogusid saab skannida ka serveriarvuti KoduMedia system tray menüüst ning samas menüüs saab lubada automaatse käivitumise Windowsi sisselogimisel.
- Lisatud on eestikeelse tegevusvoo, ingliskeelse renderduse ja paralleelse skannimise regressioonikontrollid.

## 0.8.14 — 2026-07-24

### Esmase seadistuse akordion

- KoduMedia töövalmiduse plokki lisati ligipääsetav **Esmane seadistus** nupp, mis avab ja sulgeb kompaktse kontrollnimekirja.
- Juhend selgitab, et seadistus tuleb teha ainult üks kord, kestab tavaliselt alla kümne minuti ning kõik vajalikud API-võtmed on tasuta.
- TMDb, OpenSubtitlesi, meediakaustade, salvestamise, skannimise ja muusika metaandmete sammud viivad ühe vajutusega vastavasse seadete jaotisesse.
- Akordion toetab klaviatuuri, korrektset `aria-expanded` olekut, mobiilipaigutust ja vähendatud animatsiooni eelistust.
- Seadete API-d, väärtused ja salvestusloogika ei muutunud.

## 0.8.13 — 2026-07-24

### Ühtne muusika metaandmete juhend

- Muusika metaandmete lisainfo kasutab nüüd sama `instruction-card` komponenti nagu TMDb seadistamise juhend.
- Infoikoon, pealkiri, kirjeldus ja AcoustID link paiknevad ühes kahesambalises plokis; ikoon ei murdu enam eraldi reale.
- Mõlemad juhendid kasutavad sama pinna-, piiri-, tüpograafia-, vahe- ja lingistiili.
- Seadete funktsionaalsust, API-t ega muusika metaandmete töötlusloogikat ei muudetud.

## 0.8.12 — 2026-07-24

### Vertikaalne ja avar seadete paigutus

- Kõigi seadete jaotiste pealkiri ja kirjeldus paiknevad nüüd sisu ning juhtnuppude kohal, mitte eraldi vasakus veerus. Lehte saab lugeda loomulikus ülalt-alla järjekorras.
- Serveri tööriistad, pildikvaliteet, metaandmed, muusika, IMDb, subtiitrid, uuendamine, keel, välimus ja meediakogud kasutavad sama üheveerulist sektsioonistruktuuri.
- Sektsioonide, päiste ja juhtnuppude vahele lisati järjepidev negatiivne ruum; seotud väljad võivad endiselt suurel ekraanil kõrvuti olla, kuid jaotise põhihierarhia jääb vertikaalseks.
- Uuendamise osa säilitab prominentse pinna, kuid selle päis, olek ja tegevused järgivad nüüd sama ülalt-alla lugemisloogikat.
- Seadete funktsionaalsust, API-t ja salvestusloogikat ei muudetud.

## 0.8.11 — 2026-07-24

### Stabiilne seadete paigutus ja ühtne tüpograafia

- Serveriseadete jaotised ei kasuta enam brauseriti erinevalt paigutuvat `fieldset`/`legend` kombinatsiooni. Jaotised on tavalised semantilised sektsioonid ja päised, mistõttu Keel, Välimus ning Meediakogud ei saa enam üksteise otsa nihkuda.
- Kõik samaväärsed jaotisepealkirjad, kirjeldused, väljasildid ja abitekstid kasutavad nüüd ühist tüpograafilist skaalat. Pildikvaliteedi, TMDb, muusika, subtiitrite, keele, välimuse ja meediakogude kirjeldused on sama tähtsuse korral sama suuruse ja reavahega.
- Kaheveeruline seadete paigutus kasutab ühtset vasaku päiseveeru laiust, joondust ja sektsioonidevahelist ruumi. Välimuse valikud ning meediakogud ei puutu enam eelmise jaotise vastu.
- KoduMedia töövalmiduse esiletõstetud plokk säilib ning uuendamise plokk jääb prominentseks, kuid selle päis paikneb nüüd loomulikus dokumendivoo paigutuses ilma absoluutpositsioneerimiseta.
- Kontrollimise ja salvestamise tegevusriba kasutab sama abiteksti suurust ja sisemist ruumi nagu ülejäänud seadete leht.
- Muudatus puudutab ainult seadete lehe struktuuri ja kujundust; seadete väärtused, API ja salvestusloogika ei muutunud.

## 0.8.10 — 2026-07-24

### Viimistletud ja usaldusväärne seadete leht

- Seadete külgmenüüst eemaldati üleliigne hele püstjoon ja uuendamise lingi eraldi vertikaalne marker; navigeerimine kasutab nüüd ühtset puhast tausta- ja ikooniaktsenti.
- Tavalised süsteemiseaded kasutavad avarat kaheveerulist editorial-paigutust. Korduvad horisontaaljooned ja servast servani taustaribad asendati järjepideva vahe, joondamise ja väheste sihipäraste pindadega.
- KoduMedia töövalmiduse ülevaade säilitab esiletõstetud kinoliku pinna ning olekukaartide hover-liikumine ja hover-taust on endiselt eemaldatud.
- Uuendamise sektsiooni pealkiri ja ikoon paiknevad nüüd täielikult sama esiletõstetud pinna sees; olek, ZIP-i valik ja turvateave on joondatud üheks selgeks töövooks.
- Kontrollimise ja salvestamise tegevusriba sai korrektse sisemise paddingu, ümarduse, tausta ja responsiivse paigutuse. Abitekst ei puutu enam konteineri servadesse.
- Keel, välimus, meediakogud, väljad, valikud, lülitid, juhised ja teavitused kasutavad sama tüpograafilist hierarhiat, raadiuseid, fookusolekuid ja ruumirütmi.
- Muudatus puudutab ainult seadete lehe välimust; API, seadete väärtused ja salvestusloogika jäid samaks.

## 0.8.9 — 2026-07-24

### Kergem ja avaram seadete leht

- Seadete lehe sektsioonid ei ole enam kõik eraldi kaartides: tavasektsioonid kasutavad avarat paigutust, selgeid eraldusjooni ja rohkem negatiivset ruumi.
- Eemaldatud on suurem osa seadete lehe pesastatud taustadest, varjudest ja `backdrop-filter` efektidest, mis vähendab brauseri renderduskoormust.
- KoduMedia töövalmiduse ülemine ülevaade säilitab 0.8.8 kinoliku esiletõstmise, kuid olekukaartidel pole enam hover-liikumist ega hover-tausta muutust.
- Vormilülitid, juhised, inline-tegevused ja meediakogud on muudetud lamedamaks: neid eristavad peamiselt vahed, tüpograafia ja õrnad jooned, mitte kastid kastide sees.
- KoduMedia uuendamise osa jääb tahtlikult kõige prominentsemaks tegevussektsiooniks, kuid selle sisemine uuendusolek ei moodusta enam teist eraldi kaarti.
- Salvestusriba ei ole enam ujuv klaaskaart ega sticky-element; see paikneb vormi lõpus lihtsa tegevusreana. Funktsionaalsus ei muutunud.

## 0.8.8 — 2026-07-23

### Ühtne kinolik seadete lehe CVI

- Seadete lehe päis, jaotiste navigeerimine, olekukaardid, vormisektsioonid, sisendväljad, valikud, infokaardid, tegevused, teema- ja keelevalik ning meediakogude haldus kasutavad nüüd ühtset CVI-põhist visuaalset süsteemi.
- Kujundus järgib avalehe ning filmide ja seriaalide vaadete avarat hierarhiat, pastelset brändigradienti, klaaspindu, suuri ümardusi ja selgeid fookusolekuid.
- KoduMedia uuendamise sektsioon ja selle navigeerimislink on tõstetud olulisema tegevusena esile, muutmata uuendamise funktsionaalsust.
- Väiksematel ekraanidel muutub jaotiste navigeerimine horisontaalselt keritavaks ning kõik vormid, olekukaardid ja tegevusnupud lülituvad üheveeruliseks.
- Muudatus on ainult kasutajaliidese välimuses; serveri API, seadete väärtused ja salvestusloogika jäid samaks.

## 0.8.7 — 2026-07-23

### Järjestikune esiletõstetud autoplay

- Treileri lõppedes liigub esiletõstetud karussell automaatselt järgmisele sisule ning käivitab selle treileri.
- Viimase treileri järel autoplay peatub ja viimasele slaidile jääb staatiline taustapilt; automaatset ringi tagasi esimesele ei tehta.
- Noolte, punktide või klaviatuuri/TV-puldi nooltega käsitsi liikudes aktiveerub autoplay uuesti valitud slaidist.

## 0.8.6 — 2026-07-23

### Kuni kümne sisuga esiletõstetud karussell

- Avalehe esiletõstetud karussell kuvab nüüd kuni kümme serveris olevat vaatamata või alla 5% vaadatud filmi ja seriaali senise viie asemel.
- Kvaliteedipõhine kandidaatide reserv suurenes kahekümne sisuni, et päevapõhine rotatsioon saaks kümnest nähtavast kaardist hoolimata valikut värskena hoida.
- Noolte, punktindikaatorite, loenduri ning klaviatuuri ja TV-puldi navigeerimise käitumine jäi samaks.
- Lisatud regressioonitest kontrollib kümmet karussellipunkti ja loendurit `2 / 10`.

## 0.8.5 — 2026-07-23

### Kuni viie sisuga esiletõstetud karussell

- Avalehe hero kasutab nüüd kogu sisuala laiust ning valib kuni viis serveris olevat vaatamata või alla 5% vaadatud filmi ja seriaali.
- Esiletõstetud sisude vahel saab liikuda eelmise/järgmise nupu, punktindikaatorite ning klaviatuuri või TV-puldi vasak-/paremnoolega.
- Aktiivse sisu number ja koguarv on nähtavad; vahetamisel uuenevad logo, kirjeldus, taust ja treiler ühe tervikliku kaardina.
- Fookus säilib ka klaviatuuriga järgmisele või eelmisele sisule liikudes ning aktiivne hero jätkab olemasolevat automaatse heli loogikat.

### Sisseehitatava treileri kontroll ja automaatne fallback

- TMDB-st salvestatakse nüüd ühe video asemel järjestatud nimekiri kuni kaheksast ametlikust treilerist või teaserist.
- Red-band-, uncensored-, 18+ ja muude tõenäoliselt piiratud lõigete prioriteet langetatakse juba metadata valikul.
- YouTube IFrame Player API kontrollib päriselt, kas kandidaat hakkab rakenduse sees mängima. Embed-keelu, vanusepiirangu, eemaldatud video või käivitumata playback'i korral proovitakse automaatselt järgmist kandidaati.
- Blokeeritud YouTube'i teadet ei näidata hero peal: videokiht muutub nähtavaks alles pärast tegelikku `PLAYING` olekut. Kui ükski kandidaat ei sobi, jääb alles kvaliteetne staatiline taustapilt.
- Ebaõnnestunud videovõti jäetakse 30 päevaks kohalikku brauseri cache'i, et sama mittesobivat treilerit igal avalehe avamisel uuesti mitte proovida.
- Olemasolevad filmid ja seriaalid rikastatakse `heroAssetsVersion: 2` migratsiooniga uue kandidaatide nimekirjaga ühe korra taustal.

## 0.8.4 — 2026-07-23

### Prominentne esiletõstetud sisu avalehel

- Avalehe senine tavaline **Esiletõstetud** horisontaalrida on asendatud ligikaudu kaks korda laiema kinoliku hero-kaardiga, mis paikneb **Jätka vaatamist** järel.
- Esiletõstetuks valitakse ainult serveris olev film või seriaal, mida pole vaadatud või mille vaatamisprogress on alla 5%. Kõrgema prioriteedi saavad treileri, TMDB logo, taustapildi ja sisukirjeldusega teosed.
- Võrdse kvaliteediga kandidaatide seast vahetub valik kord päevas, et avaleht püsiks värske, kuid ei hüppaks ühe sessiooni jooksul juhuslikult.
- Pealkirja tekst asendatakse võimalusel TMDB title-treatment logoga, mis kuvatakse Kodu visuaalis valgena. Logo puudumisel jääb ligipääsetav tekstipealkiri.

### Netflixi-laadne treileri eelvaade

- TMDB metadata päring laadib nüüd koos detailidega title-logo pildid ning YouTube'i treilerid/teaserid. Valik eelistab ametlikku treilerit, kasutaja keelt ja seejärel ingliskeelset videot.
- Treiler käivitub hero nähtavaks muutumisel vaikimisi hääletult, kordub ning jätab taustapildi alles turvaliseks fallback'iks.
- Paremas ülanurgas on heli sisse-/väljalülitamise nupp. Klaviatuuri või puldiga hero tegevusele liikudes lülitub heli sisse; fookuse lahkumisel vaigistatakse treiler uuesti.
- `prefers-reduced-motion` või andmesäästurežiimi korral treilerit automaatselt ei laadita. Taustale minnes video peatatakse ja vaigistatakse.
- Rotten Tomatoesi kanalit ei seota jäigalt: kasutatakse TMDB teosega seotud kureeritud YouTube'i videot, vältides eraldi YouTube Data API võtit ja ebatäpseid pealkirjaotsinguid.

### Metadata migratsioon ja töökindlus

- Filmide ja seriaalide metadata saab `heroAssetsVersion` markeri. Olemasolevad teegid rikastatakse ühe korra logo ja treileri väljadega, kuid puuduvate varade tõttu ei tehta lõputuid korduspäringuid.
- YouTube'i video võti ja nimi salvestatakse kohalikku meediakataloogi; igal avalehe avamisel TMDB-d uuesti ei küsita.
- Lisatud on TMDB logo- ja treilerivaliku regressioonitestid ning API skeemi uued väljad.

## 0.8.3 — 2026-07-23

### Muusika kasutajaliidese parandused

- Muusika avalehe **Kõik lood** sissepääs on kujundatud Kodu visuaalse identiteedi järgi prominentseks pastelsete aktsentide, suurema tüpograafia, muusikaikooni ja selge esmase tegevusnupuga kaardiks.
- `/music/tracks` otsing saadab päringu iga klahvivajutusega, kuid ei asenda enam kogu vaadet laadimisekraaniga. Sisestusväli säilitab fookuse ja teksti ning tulemused uuenevad taustal.
- Otsingu ajal kuvatakse väljas tagasihoidlik **Otsin** olek; aegunud päringute tulemusi ei rakendata uuema otsingu peale.
- Muusika **Kõik lood** vaatest eemaldati sorteerimise ja filtreerimise plokk. Lood kuvatakse stabiilselt pealkirja järgi kasvavas järjekorras ning otsing jääb ainsaks kitsendamisvahendiks.
- Audiofaili detailvaate artwork kasutab nüüd `contain` paigutust ruudukujulises neutraalses raamis, mistõttu horisontaalsete või vertikaalsete kaante servad ja tekst ei lõiku enam ära.
- Lisatud on regressioonitestid fookust säilitavale klahvivajutuspõhisele otsingule, eemaldatud filtritele, prominentsele **Kõik lood** kaardile ja artworki raamile.

## 0.8.2 — 2026-07-23

### Artworki vale jagamise parandus

- Embedded artwork, genereeritud kaardid, kataloogipildid ja negatiivne cache on nüüd seotud konkreetse füüsilise audiofailiga, mitte ainult artisti ja albumi tagidega.
- Ühesugused või üldised tagid nagu `Muusika` / `2026` ei saa enam panna ühe loo pilti teistele mitteseotud lugudele.
- Mitme looga lahtises kaustas ning üldiste tagide korral proovitakse Shazami/kataloogi pilti enne embedded artworki; see parandab ka faile, millele on juba allikas sama playlisti pilt sisse kirjutatud.
- Hästi tagitud albumikaustades ja ühe loo kaustades jääb embedded artwork endiselt prioriteetseks.
- Välise kataloogi artwork jääb track-spetsiifiliseks ka siis, kui mitmel failil on ekslikult sama MusicBrainzi ID.
- Faili suuruse, muutmisaja või välise vaste ID muutumine invalideerib vana artworki cache'i automaatselt.
- Kogu teegi sunnitud kordustöötlus ei jää enam olemasoleva kataloogipildi cache'i taha kinni.
- Brauseri artworki URL-i cache-versioon tõsteti, et vana vale pilt ei jääks pärast uuendust 24 tunniks nähtavaks.

### Kohalike kaustapiltide turvalisem kasutus

- Audiofailiga sama nimega `.jpg`, `.jpeg` või `.png` käsitletakse alati selle loo track-spetsiifilise sidecar-pildina.
- Üldist `cover.jpg`, `folder.jpg`, `front.jpg` või `album.jpg` pilti kasutatakse mitme looga kaustas ainult siis, kui kausta nimi vastab sisulisele albuminimele.
- Aasta-, `Music`-, `Muusika`-, `Downloads`-, `Singles`-, `Remixes`- ja muud üldised albumisildid ei luba enam terve kausta pilti automaatselt jagada.
- Lisatud on regressioonitestid üldiste tagide, lahtiste lugude kausta, track-sidecar'i, albumikausta ja faili muutumise jaoks.

## 0.8.1 — 2026-07-21

### Shazam esimese automaatse tuvastusena

- Lisatud on native Go audio-fingerprint, mis kasutab esimese automaatse otsinguna tasuta ja API-võtmeta mitteametlikku Shazami tuvastusendpointi. Tuvastuseks saadetakse ainult kompaktne genereeritud signatuur, mitte algne audiofail.
- Shazami vaste annab pealkirja, esitaja, albumi, laia žanri, artworki ja avaliku Shazami lingi. Tulemus salvestatakse püsivasse vahemällu ning sama muutmata faili ei tuvastata igal käivitamisel uuesti.
- Kui Shazam vastet ei leia, on ajutiselt kättesaamatu või piirab päringuid, jätkub olemasolev AcoustID/MusicBrainz → Deezer → lokaalne audioanalüüs ahel. Shazami mitteametlik endpoint ei ole rakenduse ainus sõltuvus.
- Seadetesse lisati eraldi vaikimisi aktiivne Shazami lüliti ning keskkonnamuutuja `MUSIC_SHAZAM_ENABLED`. Shazam ei vaja AcoustID API võtit.
- Metadata cache'i skeem tõsteti versioonile 4 ja seadete skeem versioonile 8, et olemasolevad 0.8.0 automaatsed hinnangud uue prioriteediga uuesti läbi vaadata.

### Vale Hip-Hop liigituse parandused

- Eemaldatud on BPM-vahemikul põhinev automaatne Hip-Hop määramine; tempo üksi ei ole žanri tõend.
- Lokaalse audioanalüüsi tulemust ei rakendata, kui see läheb vastuollu tugevama fingerprint- või kataloogivaste žanriga.
- `Hip-Hop/Rap` normaliseeritakse üheks Hip-Hop põhikategooriaks ega tekita enam ekslikku eraldi Rap kirjet.
- Audioanalüüsi cache'i skeem tõsteti versioonile 2, et varasemad BPM-põhised Hip-Hop hinnangud ei jääks kasutusse.

## 0.8.0 — 2026-07-21

### Laiema katvusega muusika rikastamine

- MusicBrainzi ja Cover Art Archive'i järele lisati tasuta Deezeri avaliku kataloogi fallback. Vaste hinnatakse pealkirja, esitaja, versioonitunnuste ja kestuse järgi; bootlegi või remixi ei asendata nõrga originaalversiooni vastega.
- Kataloogist saadud laiem žanr normaliseeritakse KoduMedia klassifikatsiooni. Täpsem stiil tuletatakse eraldi ja märgitakse oma tegeliku allikaga, mitte eksitavalt kataloogiandmena.
- Puuduva klassifikatsiooni jaoks lisati lokaalne FFmpeg-põhine audioanalüüs. Analüüs kasutab lühikest PCM-lõiku, tempo-, rütmi-, dünaamika- ja spektritunnuseid ning salvestab tulemuse püsivasse vahemällu.
- Pealkirjas, albumis, failinimes või kaustatees olevad selged stiilitunnused (näiteks Deep House, Liquid DnB või Psytrance) täidavad ainult puuduvaid välju ja märgitakse hinnanguliseks.
- Kui usaldusväärset või hinnangulist stiili ei leita, salvestatakse tühi vaade vältiv madala kindlusega `Other`, mitte ei jäeta lugu lõputult töötlemata.

### Artworki täielik fallback-ahel

- Artworki järjekord on nüüd kohalik pilt, embedded artwork, täpne Cover Art Archive, kõrge kindlusega kataloogipilt ja viimase võimalusena KoduMedia genereeritud neutraalne loo kaart.
- Genereeritud kaart sisaldab loo ja esitaja nime ning remix/edit/bootleg tähist, kuid seda ei esitata ametliku albumikaanena.
- Kogu teegi kordustöötlus proovib genereeritud fallback-kaarti hiljem uuesti ametliku pildiga asendada.
- Parandatud on puuduva artisti või albumiga lugude cache-võtme kokkupõrge, mis võis näidata eri lugudel sama vale pilti.

### Taustatöö ja seaded

- Seadetesse lisati eraldi lülitid laiemale kataloogivastele, lokaalsele audioanalüüsile ja genereeritud artworkile.
- Lisatud on nupp **Töötle kogu muusikateek uuesti**, mis käivitab püsiva progressiga taustatöö ning säilitab käsitsi määratud sidecar- ja failitagid.
- Lisatud on `POST /api/v1/music/enrich` taustatöö API.
- Muusika metadata cache'i skeem tõsteti versioonile 3 ja seadete skeem versioonile 7.

## 0.7.4 — 2026-07-21

### Tasuta MusicBrainzi ja AcoustID metaandmed

- Eemaldatud on tasulist SoundCloud Artist Pro ligipääsu eeldav integratsioon ja selle Client ID/Client Secret seaded.
- Lisatud on MusicBrainzi täpse recording-vastena salvestatavad recording, release ja release-group MBID-d.
- Valikuline tasuta AcoustID API võti lubab Chromaprint `fpcalc` audio fingerprint'i; ilma võtmeta töötab konservatiivne pealkirja, esitaja, albumi ja kestuse vaste.
- Windowsis laaditakse ametlik fpcalc 1.6.0 vajadusel alla, kontrollitakse kinnitatud SHA-256 summaga ning salvestatakse kohalikku tööriistavahemällu.
- Positiivsed vasted, negatiivsed tulemused ja fingerprint'id salvestatakse `data/music-metadata` alla, et skannimine ei kordaks samu päringuid.
- MusicBrainzi žanrid ja tugevaimad mittetavalised tagid täidavad ainult puuduvaid klassifikatsioonivälju; sidecar ja embedded tagid jäävad prioriteetseks.
- Cover Art Archive'i artwork laaditakse ainult täpselt valitud release'i või release-group'i ID järgi ja salvestatakse kohalikku vahemällu.

### Tagasihoidlik allikainfo

- Looridadelt eemaldati `Faili tagidest`, `Kinnitatud` ja välise vaste badge'id.
- Žanri ja stiili päritolu salvestatakse nüüd eraldi väljadena.
- Allikainfo kuvatakse ainult audiofaili detailvaates **Žanr ja stiil** all väikese sekundaarse tekstina.
- Eraldi prominentne **Liigitus** rida eemaldati.

## 0.7.3 — 2026-07-21

### SoundCloudi metaandmed ja artwork

- Lisatud on valikuline SoundCloudi avaliku API integratsioon puuduva muusikainfo ja artworki jaoks.
- Seadetes saab turvaliselt salvestada SoundCloud Client ID ja Client Secreti; saladusi ei tagastata brauserisse.
- Avaliku track search API vasted hinnatakse pealkirja, esitaja ja kestuse järgi ning mitmetähenduslikud vasted jäetakse automaatselt kõrvale.
- Sidecar ja embedded failitagid säilitavad prioriteedi; SoundCloud täidab ainult puuduva klassifikatsiooni ning märgitakse allikana eraldi.
- Positiivsed vasted, negatiivsed tulemused ja OAuth tokenid hoitakse kohalikus vahemälus.
- Puuduva kohaliku/embedded artworki korral proovitakse enne MusicBrainzi SoundCloudi kõrge kindlusega vaste artworki.
- KoduMedia ei striimi SoundCloudi audiofaili ning jätkab kohaliku originaalfaili või kadudeta PCM-varutee kasutamist.

### Kompaktne audiofaili detailvaade

- Codec, konteiner, diskreetimissagedus, bitisügavus ja kanalite arv koondati reale **Helikvaliteet**.
- Žanr, stiil ja meeleolu koondati reale **Žanr ja stiil**.
- Kestus, failisuurus ja bitikiirus koondati reale **Fail ja kestus**.
- SoundCloudi vaste korral kuvatakse allika nimi ja link avalikule track-lehele.
- Töölaua detailvaade kasutab kompaktsemat artworki, tüpograafiat ja kuut infirida ning ei vaja enam sisemist scrollriba.
- Kõik uued tekstid lisati `et-EE.json` ja `en-US.json` tõlkefaili.

## 0.7.2 — 2026-07-21

### Täiustatud „Kõik lood” vaade

- `/music/tracks` kasutab nüüd eraldi muusika-API-t, mis tagastab pagineeritud lood koos artisti-, albumi-, žanri-, stiili-, aasta-, codec'i ja kvaliteedifiltrite valikutega.
- Lisatud on sorteerimine pealkirja, artisti, albumi, aasta, lisamisaja, viimase kuulamise, kestuse ja helikvaliteedi järgi ning kasvava/kahaneva järjekorra valik.
- Lisatud on filtrid artisti, albumi, žanri, stiili, aasta, formaadi, kadudeta heli, Hi-Res heli ja liigitamata lugude järgi.
- Loo real kuvatakse nüüd žanr, stiil, metadata päritolu, codec, diskreetimissagedus ja bitisügavus.

### Usaldusväärne žanri- ja stiiliinfo

- KoduMedia ei arva žanri faili- või artistinime põhjal ega kasuta ebakindlat avalikku vastet faktina.
- Embedded `GENRE`, `STYLE`, `SUBGENRE`, `GROUPING`, `CONTENTGROUP` ja `MOOD` tagid loetakse failipõhiseks metadataallikaks ning märgitakse kasutajaliideses „Faili tagidest”.
- Mitteametlike remixide, bootlegide ja edit'ite jaoks lisati failipõhine `<audio-file>.kodumedia.json` sidecar. Sidecar on kasutaja kinnitatud allikas ning ületab embedded tagid.
- Kui usaldusväärne klassifikatsioon puudub, kuvatakse ausalt „Liigitamata” ja lugu on eraldi filtriga leitav.
- FFprobe metadata versioon tõsteti 3 peale, et olemasolevad muusikafailid skannitaks ühe korra uute stiili- ja meeleolutagide jaoks uuesti.

## 0.7.1 — 2026-07-21

### Rakenduseülene muusikapleier

- Muusikapleier renderdatakse nüüd rakenduse juurtasemel, mitte ainult `/music` marsruudil. Aktiivne lugu jääb nähtavaks ja jätkab õigelt positsioonilt ka teistesse vaadetesse liikudes.
- Pausile pandud lugu ei käivitu marsruudi vahetamisel uuesti ega alusta algusest.
- Filmi või seriaali videoesituse alustamine peatab muusika, eemaldab aktiivse muusikajärjekorra ja vabastab Media Sessioni muusikakontrollid.
- Muusikapleieri olemasolul lisatakse teistele lehtedele alumine turvaruum, et player ei kataks sisu.

### Kõik lood

- Muusika avalehele lisati **Kõik lood** sektsioon koos serveris olevate lugude koguarvuga.
- Lisatud on `/music/tracks` vaade kogu muusikateegi sirvimiseks.
- Vaade toetab otsingut loo, artisti, albumi, žanri, failinime ja failitee järgi.
- Suured kogud laaditakse 200 loo kaupa ning **Laadi veel** nupp lisab järgmise lehe olemasoleva loendi lõppu.
- `MusicLanding` API sisaldab nüüd `totalTracks` välja.
- Kõik uued kasutajaliidese tekstid on lisatud `et-EE.json` ja `en-US.json` tõlkefaili.

## 0.7.0 — 2026-07-20

### Audiofiilne muusikaesitus

- Muusikafailid esitatakse brauseri toe korral alati otse algsest failist; WAV, FLAC ja muud toetatud formaadid säilitavad lähtefaili codec'i, diskreetimissageduse, bitisügavuse ja kanalipaigutuse.
- Brauseri poolt mittetoetatud audioformaadi korral kasutatakse ainult kadudeta PCM WAV varuteed. Muusika puhul ei kasutata ühtegi kadudega encoder'it.
- FFprobe salvestab nüüd audio diskreetimissageduse ja eraldi audio bitisügavuse, mida näidatakse playeri tehnilises infovaates.
- Muusika süsteemikogu skannib ainult audiofailide laiendeid ning videofailid ei satu enam muusikakataloogi.

### Artist → album → lugu

- Lisatud on serveripoolne artistide, albumite ja lugude projektsioon ilma sünteetilisi albumi- või artistikirjeid kataloogi salvestamata.
- Failitagid `title`, `artist`, `album_artist`, `album`, `track`, `date` ja `genre` rakendatakse skannimisel; puuduv track-number tuvastatakse ka failinime algusest.
- Lisatud on API-d `GET /api/v1/music`, `GET /api/v1/music/artists/{artist}` ja `GET /api/v1/music/albums/{albumId}`.
- Muusika avaleht sisaldab **Viimati kuulatud**, **Viimati lisatud**, **Albumid** ja **Artistid** sektsioone.
- Artisti vaade kuvab albumid ning albumivaade järjestab lood track-number'i järgi.

### Püsiv alumine muusikapleier

- Muusika lehe allservas on kogu muusikaosas püsiv player koos play/pause, eelmise/järgmise loo, ajajoone, helitugevuse, järjekorra ja artwork'iga.
- Loole vajutamine alustab kohe esitamist; lugudel ei ole eraldi eelvaate- või detaillehte.
- Playerist avatav audiofaili infovaade näitab algset failinime, codec'it, konteinerit, diskreetimissagedust, bitisügavust, kanaleid, bitikiirust, kestust ja failisuurust.
- Media Session metadata ja riistvaralised play/pause/previous/next juhtelemendid töötavad muusikapleieriga.

### Artwork ja kuulamisajalugu

- Albumi artwork otsitakse esmalt albumikausta `cover`/`folder`/`front` pildist, seejärel embedded cover'ist ja viimase võimalusena MusicBrainz + Cover Art Archive kaudu.
- Artwork salvestatakse `data/music-artwork` vahemällu; albumipõhine lukustus väldib dubleerimist ning MusicBrainzi päringud on kiiruspiiratud.
- Muusikafailid on **Jätka vaatamist** loogikast täielikult välja jäetud. Sama püsivat watch-state hoidlat kasutatakse ainult **Viimati kuulatud** järjestuse jaoks.
- Kõik uued frontend-tekstid lisati `et-EE.json` ja `en-US.json` tõlkekataloogi.

## 0.6.9 — 2026-07-20

### Teavitused

- Päise paremas servas on uus teavituste kell. Lugemata teavitused märgitakse playeri heliraja aktiivmärgiga samas stiilis roosa punktiga.
- Teavituste rippmenüü kasutab playeri subtiitrimenüü visuaalset struktuuri ning kuvab viis viimast teavitust.
- Menüü avamisel märgitakse kõik teavitused loetuks ja roosa märgis kaob kohe.
- Lisatud on eraldi `/notifications/` leht kogu teavituste ajaloo vaatamiseks.
- Esimene teavitusetüüp näitab seriaali uut ilmunud episoodi, mida serveris veel pole, koos episoodikoodi, pealkirja ja ilmumiskuupäevaga.
- Kui episood lisatakse hiljem serverisse, säilib teavitus ajaloos lahendatud olekus.
- Teavitused salvestatakse püsivalt faili `data/notifications.json`; sünkroniseerimine on lukustatud, ajutiselt cache'itud ning ajalugu piiratud 1000 kirjega.
- Lisatud API-d `GET /api/v1/notifications` ja `POST /api/v1/notifications/read`.
- Kõik uued frontend-tekstid on lisatud nii eesti kui inglise tõlkekataloogi.

## 0.6.8 — 2026-07-20

### Rakenduse keelevalik

- Lisatud on täielik eesti ja inglise kasutajaliides ning uus **Seaded → Keel** jaotis.
- Keele vahetamine rakendub kohe ilma serveri taaskäivitamiseta ja salvestatakse brauseripõhiselt `localStorage`-isse; eri kliendid saavad kasutada eri UI keelt.
- Olemasolevate installide vaikekeel jääb eesti keel, kuni kasutaja valib inglise keele.
- Valitud locale määrab ka kuupäevade, arvude, häälte arvu ja kestuste vormingu ning uuendab dokumendi `lang` ja `dir` atribuute.

### Tõlkefailid ja kogukonnatõlked

- Tõlked asuvad eraldi UTF-8 JSON kataloogides `apps/web/src/i18n/locales/et-EE.json` ja `en-US.json`.
- Locale'id avastatakse buildis automaatselt failimustri järgi; uue keele lisamine ei vaja Reacti komponentides registreerimist.
- Eesti kataloog toimib lähte- ja fallback-kataloogina. Inglise kataloog on täielik ning sisaldab kõiki kasutajaliidese võtmeid.
- Lisatud on `docs/translations/README.md` juhend BCP 47 locale'i, autoriinfo, placeholder'ite ja GitHubi pull request'ide jaoks.
- TypeScript seob `tr()` võtmed eesti kataloogiga, mistõttu puuduv tõlkevõti põhjustab buildivea.
- Testid kontrollivad kataloogide võtmepaarsust, 100% täielikkust, tühjade väärtuste puudumist ja kõigi `{{placeholder}}` väärtuste säilimist.
- Serverist tulevad tuntud job'i-, komponentide oleku- ja API veateated läbivad samuti tõlkekihi; tundmatu detailne tehniline tekst jääb diagnostikaks muutmata.

## 0.6.7 — 2026-07-19

### Seriaali detailvaate kujunduse tagasiparandus

- Taastati seriaali detailvaate parempoolne infoala 0.6.4 lihtsa tekstilise kujunduse järgi.
- Eemaldati 0.6.6 lisatud metaandmete paneel, žanrikapslid ja täiendav kastidesse jaotamine.
- **Väljas**, **Vaata kohe**, žanrid ja IMDb hinnangute kogus kuvatakse taas vabalt ning ilma ümbritseva kaardita.
- **Uuem osa on ilmunud** jääb soovitud asukohta paremas veerus, kohe aasta ja IMDb skoori alla.
- Lisati regressioonitest, mis välistab `series-meta-panel` ja `series-meta-stack` struktuuri tagasituleku.

## 0.6.6 — 2026-07-19

### Seriaalide detailvaate parem veerg

- Viimistleti seriaali detailvaate parempoolne infohierarhia, et aasta, IMDb skoor, uue episoodi teavitus, saadavus ja žanrid moodustaksid ühtsema ning rahulikuma terviku.
- Saadavuse info (**Väljas / Vaata kohe**) tõsteti eraldi, kuid CVI-ga kooskõlas olevasse pehmesse metaandmete paneeli.
- Žanrid muudeti visuaalselt paremini skaneeritavateks kapselsiltideks, mis sobituvad parema veeru uue klaasja esteetikaga.
- IMDb hinnangute kogus seoti sama metaandmete paneeli alumisse ossa, et tehniline lisainfo oleks olemas, kuid mitte liiga domineeriv.
- Lahendus hoiab detailvaate parema veeru elegantse, õhulise ja loogiliselt järjestatud ka siis, kui uue episoodi teavitus on aktiivne.

## 0.6.5 — 2026-07-19

### Seriaali detailvaate UI

- Seriaali kogumiku lehel tõsteti teavitus **„Uuem osa on ilmunud”** vasakust sisuveerust paremasse metaandmete veergu, otse aasta ja IMDb skoori alla.
- Teavitus kasutab nüüd KoduMedia CVI-ga sobivat pehmet klaasja taustaga kapselkaarti: kumerad nurgad, õrn blur, hele kiht ja tagasihoidlik piirjoon.
- Kujundus on tahtlikult integreeritud metaandmete plokki, mitte eraldi „tükina”, et leht püsiks puhas ja rahulik.
- Lisati regressioonitest, mis kontrollib uue paremveergse teavituse renderdumist.

## 0.6.4 — 2026-07-19

### Väljalaske terviklus

- Serveri versioon loetakse nüüd executable'i sisse kompileeritud `go:embed` ressursist; haavatav linkeri `-X main.version` sõltuvus eemaldati.
- Embedded versioon ja unikaalne `KODUMEDIA_VERSION=<version>` marker valideeritakse käivitamisel ning eraldi Go regressioonitestides.
- Windowsi build genereerib `build-info.json` faili serveri ja updateri SHA-256 räside ning suurustega.
- Pakendamine katkeb, kui root `VERSION`, embedded versioon, build marker, build-info või binaaride tegelikud räsid ei ühti.
- See väldib 0.6.3 viga, kus paketi manifest ootas 0.6.3, kuid server raporteeris `dev` ning updater tegi automaatse rollback'i.

### Settings UI CVI

- `Meediakogud ja kaustad` juursektsioon on nüüd läbipaistev ning ei moodusta meediakogude ümber välist paneeli.
- Iga kogu on üks top-level pind; seotud kaustad, tühiolek ja kausta lisamise tegevus on selle sees lamedad read ilma täiendavate kaartide, ääriste või varjudeta.
- Eemaldati 0.6.3 pesastatud `library-settings-card` ja `library-source-section` struktuur.
- `Loo oma kogu` on eraldi top-level tegevuspind ning kasutab endiselt jagatud CVI `SelectControl` komponenti.
- Lisati regressioonitest, mis kontrollib lamedat DOM-struktuuri ja välistab vanad pesastatud paneeliklassid.

## 0.6.3 — 2026-07-19

### Settings UI ja CVI

- Lisatud on jagatud `SelectControl` komponent, mis säilitab native `select` semantika, kuid kasutab KoduMedia CVI raami, gradienti, fookuserõngast ja eraldi chevron-juhtelementi.
- Kõik **Pildikvaliteedi** rippmenüüd kasutavad nüüd sama komponenti ning ühesuguseid hover-, focus-, disabled-, tumeda ja heleda teema olekuid.
- **Loo oma kogu** sisutüübi valik kasutab sama CVI rippmenüüd; vorm kujundati ümber üheks selgeks, pehmeks loomiskaardiks.
- **Meediakogud ja kaustad** eemaldus triibulisest tabeliilmest. Iga kogu on nüüd iseseisev kihiline kaart, mille sees on selge päis, kogu tüübimärgis, kaustade kokkuvõte ja pehmed kaustaplokid.
- Kaustade puudumise, skannimise, kausta lisamise ja eemaldamise olekud säilitavad olemasoleva funktsionaalsuse, kuid järgivad uut visuaalset hierarhiat.
- Lisatud on responsive-käitumine kitsale ekraanile ning regressioonitestid jagatud select-stiili ja meediakogu kaardistruktuuri jaoks.

## 0.6.2 — 2026-07-19

### HDR hardware pipeline

- 4K-, 10-bit ja HDR-video automaatne encoder'i järjestus eelistab NVIDIA NVENC-i, seejärel Intel QSV-d ja AMD AMF-i. SDR-sisu puhul võib Auto endiselt valida energiatõhusa QSV tee.
- FFmpegi capability-probe ühendab nüüd `-h full`, `-encoders`, `-filters` ja eraldi `-hwaccels` väljundid. CUDA, QSV ja D3D11VA tuvastatakse eraldi ning riistvaralist encoder'it valideeritakse päris ühekaadrilise prooviencode'iga.
- NVIDIA süsteemides saab sobiva H.264, HEVC või AV1 lähtevideo kasutada NVDEC/CUDA hardware decode'i koos NVENC encode'iga. Pildisubtiitrite põletamisel jääb kasutusele konservatiivne software-filter pipeline.
- Hardware encoder'i negatiivne proov ei lukustu terveks serveri tööajaks: ebaõnnestunud tulemus aegub 30 sekundiga ning draiveri või GPU ajutise vea järel saab KoduMedia uuesti proovida.

### Kontrollitud taastumine

- Hardware decoder'i vea korral proovitakse sama hardware encoder'it software decode'iga; encoder'i vea korral välistatakse see konkreetse playback-katse jaoks ja proovitakse järgmist sobivat hardware encoder'it.
- Encoder-speed kontroll rakendub nüüd ka fallback-session'itele. Reaalajast aeglasemat `0.81×` software-voogu ei anta enam playerile lihtsalt seetõttu, et see on juba fallback.
- Stabiilsusredel sisaldab kolme kontrollitud astet: 1080p Tasakaalus, 720p Andmesäästlik ja 480p hädaolukorra fallback. Iga aste peab täitma oma minimaalse encoder-speed turvavaru.
- Player saab liikuda kolmanda taastumisastmeni ka rebuffering-olekus ning jätkab uue session'iga samast absoluutsest vaatamispositsioonist.

### HDR töötlus ja diagnostika

- HDR→SDR pipeline skaleerib pildi sihtresolutsioonini juba lineaarse värviruumi teisenduse käigus ning teeb CPU-mahuka float tone mapping'u väiksema pildiga.
- Playeri kvaliteedi- ja diagnostikavaade näitab eraldi decoder'it, encoder'it, hardware fallback'i põhjust, väljundresolutsiooni, kvaliteediprofiili ja recovery level'it.
- Quality Engine pipeline'i revision on tõstetud `quality-engine-v4` peale, et varasemad sobimatud JIT-cache'i variandid ei läheks uue hardware pipeline'iga taaskasutusse.

## 0.6.1 — 2026-07-19

### Sujuvam käivitamine ja taastuv puhver

- Player eristab nüüd selgelt voo ettevalmistamist, algpuhvri kogumist ja mängimise ajal tekkinud rebufferingut. Algpuhvri tekst ei saa enam kuvada ebaloogilist väärtust nagu `10 / 8 s`.
- Direct Play, remux/audio-transcode, 1080p transcode ning 4K/HDR transcode kasutavad eraldi startup-, rebuffer-, soovitud ja madala puhvri piire.
- 1080p videotranskodeering alustab vaikimisi 14-sekundilise ning 4K/HDR transkodeering 20-sekundilise puhvriga. Rebuffering ootab suurema varu taastumist enne video jätkamist.
- FFmpeg kasutab kvaliteediprofiili järgi suuremat esialgset burst'i ja 1.30–1.50× sisendi tootmisvaru, kuid ei kodeeri kogu filmi kontrollimatult ette.

### Encoder'i turvavaru ja automaatne fallback

- Kvaliteetne 1080p encoder peab hoidma vähemalt 1.20× ja 4K/HDR encoder vähemalt 1.25× reaalajakiirust. Käivituse kontroll kasutab sama turvavaru.
- FFmpegi progressi jälgitakse kogu aktiivse transkodeerimise ajal. Libisev keskmine, genereeritud kestus, segmentide arv ja hinnanguline väljundbitikiirus on saadaval uue runtime-stats endpoint'i kaudu.
- Kui encoder'i keskmine kiirus ja playeri puhver jäävad püsivalt alla turvapiiri, lülitub KoduMedia samast vaatamispositsioonist esmalt stabiilsele 1080p H.264 ning korduva probleemi korral 720p andmesäästlikule profiilile.
- Kaks lühikese aja jooksul tekkinud buffer-stalli käivitavad samuti kontrollitud kvaliteedifallback'i. Ühe juhusliku võrguhetke tõttu kvaliteeti kohe ei muudeta.

### Bitrate'i tipud ja playeri diagnostika

- Transkodeeritud video kasutab nüüd codec'i, resolutsiooni ja kvaliteediprofiili järgi arvutatud VBV `maxrate`/`bufsize` piire. CRF-i sisupõhine kvaliteet säilib, kuid väga suured hetkelised bitrate'i tipud ei tohiks enam puhvrit asjatult tühjendada.
- Playeri olekurida näitab täpselt profiili, videoteed, resolutsiooni, codec'it, encoder'it, heli teisendust, HDR/SDR olekut ja JIT-cache'i kasutust.
- Olekuribale vajutades avaneb kerge diagnostikavaade: puhver ees, sihtpuhver, encoder speed, väljundbitikiirus, VBV, stallide arv/kestus ja cache'i olek.
- Lisatud on `GET /api/v1/playback/sessions/{id}/stats`, mida saavad kasutada ka tulevased Android TV ja Tizeni kliendid.

## 0.6.0 — 2026-07-19

### Quality Engine

- Esitusahel eelistab nüüd maksimaalselt originaalpilti: Direct Play, video stream-copy/remux, ainult heli transkodeerimine ning alles seejärel video transkodeerimine.
- Fikseeritud 1080p transkodeerimispiir eemaldati profiilidelt **Originaal** ja **Maksimaalne**. **Tasakaalus** piirab väljundi 1080p-ni ja **Andmesäästlik** 720p-ni.
- Veebiklient kasutab `MediaCapabilities.decodingInfo()` päringuid codec'i, profiili, level'i, bitisügavuse, resolutsiooni, kaadrisageduse, bitikiiruse ja HDR-i töökindluse hindamiseks.
- Capability-cache arvestab aktiivse ekraani mõõte, device pixel ratio väärtust ning Windowsi/browseri HDR- ja Rec.2020 signaale, mistõttu HDR-režiimi muutus ei kasuta vana otsust.
- FFmpeg adapter tuvastab ja proovib enne kasutamist Intel QSV, NVIDIA NVENC ning AMD AMF H.264/HEVC/AV1 encodereid. Käivitumisvea korral valitakse tarkvaraline fallback enne session'i väljastamist.
- HEVC 10-bit/HDR väljund kasutab selget Main10 profiili nii libx265, QSV, NVENC kui AMF puhul.
- Tarkvaralised kvaliteediprofiilid kasutavad codec'ile sobivat preset'i ja CRF-i; QSV, NVENC ja AMF kasutavad eraldi, oma kvaliteediskaalale vastavaid argumente.
- Automaatne AV1 valitakse ainult valideeritud riistvaralise AV1 encoder'i korral. Tarkvaraline 4K AV1 ei ole automaatne fallback.
- Encoder'i FFmpeg progressi jälgitakse enne playback'i avaldamist. Kui esmane lähteresolutsiooniga kvaliteediprofiil ei püsi reaalajas, valitakse kiirema preseti või 1080p H.264 fallback.

### Vajaduspõhine JIT-cache ja VMAF

- Optimeeritud koopiat ei looda ennetavalt ega öösel. Cache tekib ainult siis, kui video tuleb päriselt reaalajas transkodeerida.
- Playerile saadetavad fMP4 HLS segmendid on samaaegselt korduskasutatav optimeeritud cache; eraldi teist kodeerimist ei käivitata.
- Sama lähtefaili, kvaliteediprofiili, encoder'i, HDR-režiimi, heli- ja subtiitrivalikuga kliendid jagavad üht aktiivset encoder-session'it.
- Täielikku ja osalist cache'i saab jätkata ning taaskasutada ka mitte-nullist jätkatud vaatamise korral. Cache'i manifest salvestab absoluutse ajavahemiku baasi.
- Lähtefaili suurus või `Date modified`, kvaliteediprofiili versioon, codec, encoder või filtrivalik muudab cache'i võtme ning väldib vananenud väljundi kasutamist.
- Cache'i maht on seadistatav ja vanimad mitteaktiivsed kirjed eemaldatakse LRU-põhiselt; aktiivset cache'i koristaja ei kustuta.
- Valikuline VMAF analüüs töötab ainult juba loodud väikese SDR-prooviala peal, ei blokeeri esitamist ning korrigeerib sama teose järgmise transkodeeringu kvaliteediväärtust.

### Seaded, metadata ja API

- `/settings` sisaldab uut **Pildikvaliteedi** sektsiooni: profiil, väljundcodec, hardware encoder, JIT-cache'i poliitika ja maht, grace-periood ning VMAF seaded.
- Runtime settings schema on versioon 4 ning vanemad seaded migreeritakse kvaliteedimootori turvalistele vaikeväärtustele.
- FFprobe tehnilise metadata `ProbeVersion` võimaldab vanad kirjed ühe korra täiendada ning jätab muutmata täieliku info järgmistel skannimistel puutumata.
- OpenAPI 3.1 leping kirjeldab laiendatud decode-capability, kvaliteediseadete, tehnilise metadata ning playback-plaani encoder/cache/VMAF välju.

## 0.5.15 — 2026-07-18

### Live Caption ja playback-stabiilsus

- Täissuuruses playeri Live Caption on nüüd päriselt sisse- ja väljalülitatav. KoduMedia annab aktiivse video heliraja Chrome'i `SpeechRecognition.start(audioTrack)` API-le ning kuvab tuvastatud teksti playeri enda subtiitrikihis.
- Live Caption kasutab aktiivse heliraja keelt, proovib võimaluse korral seadmesisest kõnetuvastust ning taaskäivitub kontrollitult pärast allika või playback-session'i vahetust.
- Eemaldatud on eksitav juhend, mis väitis, et suur player ei saa Live Captionit käivitada.
- Media Sessioni positsiooni ei saadeta enam Chrome'ile iga `timeupdate` sündmusega. Seda sünkroniseeritakse ainult play/pause, session'i, kestuse, seek'i ja PiP-i avamise ajal.
- Vähenes brauseri ja Windowsi media-control IPC koormus, mis põhjustas 0.5.14-s osadel arvutitel korduvat puhverdamist ja katkestusi.
- PiP säilitab endiselt lähtefaili tegeliku kogukestuse ning absoluutse seek-loogika.

## 0.5.14 — 2026-07-18

### Playeri helirajad, Live Caption ja picture-in-picture

- Playeri suur heliraja `select` asendati kompaktse heliikooniga. Ikoon avab sama CVI ja interaktsiooniloogikaga menüü nagu subtiitrite valik ning näitab selgelt aktiivset rada, keelt, codec'it, kanalite arvu ja vaikimisi olekut.
- Subtiitrite ikoon jääb täissuuruses playeris alati nähtavaks; suure helivaliku tõttu ei suruta seda enam juhtribalt välja.
- Täissuuruses playerisse lisati eraldi Chrome Live Caption juhtelement. Kuna brauser ei paku veebilehele Live Captioni sisse-/väljalülitamise API-t, kuvab juhtnupp täpsed Chrome'i seadistusjuhised ja võimaldab seadete aadressi kopeerida.
- Picture-in-picture player sünkroniseeritakse nüüd Media Session API kaudu lähtefaili tegeliku kogukestuse ja absoluutse vaatamispositsiooniga.
- PiP play/pause, 10 sekundi edasi/tagasi liikumine ja `seekto` tegevus kasutavad sama serveripoolset seek-loogikat nagu põhijoon, mistõttu saab ka PiP ajajoonelt liikuda puhverdamata kaugemasse kohta.
- PiP-le edastatakse filmi või seriaali pealkiri, episoodi tähis ja artwork ning Media Sessioni olek puhastatakse playeri sulgemisel.

## 0.5.13 — 2026-07-17

### HDR fallback ja diagnostika

- HDR-lüliti näitab nüüd kasutaja soovitud režiimi eraldi aktiivsest väljundist. Kui HDR-katse ebaõnnestub, kuvatakse selgelt `SDR fallback`, mitte eksitav üldtekst kliendi universaalse transkodeerimise kohta.
- HDR HLS-session ei kuku enam esimese mööduva võrgu- või MediaSource vea järel kohe SDR-i: player teeb kuni kaks kontrollitud võrgutaastust ning ühe meediapipeline'i taastuse.
- Codec'i või MediaSource'i selge ühildumatus läheb endiselt kohe töökindlale SDR fallback'ile.
- Fallback-teade sisaldab nüüd tegelikku HLS/codec'i vea põhjust ning kogu tekst on olekureal tooltipina loetav.
- Native HLS toega brauserites eelistatakse säilitatud HDR-i puhul operatsioonisüsteemi enda HLS pipeline'i, mis on HEVC HDR-i jaoks sageli töökindlam kui JavaScripti MSE.
- Serveri playback-plan kirjeldab HDR→SDR fallback'i nüüd täpselt ega väida ekslikult, et kasutaja palus üldist SDR-transkodeerimist.

## 0.5.12 — 2026-07-17

### HDR playback

- HDR-ekraanil proovib KoduMedia nüüd esmalt HDR passthrough'd, kui brauser teatab sobivast HEVC/HDR toest.
- Parandatud on Windowsi brauserite HDR tuvastamist: arvestatakse `dynamic-range`, `video-dynamic-range` ja `rec2020` signaale ning HEVC MSE tuge.
- HDR MKV saab vajadusel remuxida fMP4 HLS-iks ilma videot ümber kodeerimata; HEVC voog märgitakse brauseri ühilduvuse jaoks `hvc1` sample-entryga.
- Kui HDR otse- või remux-esitus ebaõnnestub, käivitab player automaatselt HDR→SDR tone mapping'u samast vaatamispositsioonist.
- Playeri alumisele olekureale lisati kompaktne `Auto / HDR / SDR` valik. Kasutaja valik säilib selles brauseris.
- `HDR` režiim võimaldab käsitsi proovida passthrough'd ka konservatiivse brauseri capability-reportingu korral; ebaõnnestumisel rakendub endiselt automaatne SDR fallback.

## 0.5.11 — 2026-07-17

### Windowsi uuendaja käivitamise parandus

- `kodumedia-updater.exe` sisaldab nüüd sisemist Windowsi `asInvoker` rakendusmanifesti, mistõttu UAC installer-detection ei käsitle tavakasutaja installikaustas töötavat updaterit administraatoriõigusi nõudva paigaldajana.
- Väljalaskepakis on sama manifest ka sidecar-failina `kodumedia-updater.exe.manifest`, et tagada ühilduvus rangema Windowsi konfiguratsiooni ja vanema serveriversiooni käivitatud esimese veebiuuendusega.
- Uuendaja jääb taustal peidetud GUI-protsessiks ega küsi administraatoriõigusi, kui KoduMedia installikaust on praegusele kasutajale kirjutatav.
- Ebaõnnestunud paketi kontrolli või updateri käivitamise järel eemaldatakse sama katse `staged`, `packages` ja `plans` ajutised failid.
- Uue uuenduskatse alguses koristatakse varasemate lõpetatud katsete ajutised staging-failid, vältides iga katsega uute mahajäänud kaustade kogunemist.

## 0.5.10 — 2026-07-17

### Parandatud

- Universaalne HLS-transkodeerimine ei eelda enam, et FFmpeg toetab uuemat `-readrate_initial_burst` valikut.
- KoduMedia tuvastab FFmpegi käivitusvalikute toe üks kord iga seadistatud executable'i kohta ja koostab käsu ainult toetatud argumentidega.
- FFmpeg 6.0 ning vanemate või kohandatud Windowsi buildide korral kasutatakse automaatselt ühilduvat reaalaja sisendirežiimi.
- Kaasa pandud FFmpeg 6.0 töötab nüüd samade playback-profiilidega ilma „Unrecognized option” veata.

# Changelog

## 0.5.9 — 2026-07-17

- Seadete leht viidi üle KoduMedia kinematograafilisele CVI-le: suurem pealkirjahierarhia, atmosfäärne taust, vähem pesastatud kaste ja selgemad eraldusjooned.
- Lisatud on desktopis kleepuv jaotiste navigatsioon ning väiksematel ekraanidel horisontaalne kiirnavigeerimine.
- Serveri olek, meediatööriistad, TMDb/IMDb, subtiitrid, uuendamine, välimus ja meediakogud on ümber korraldatud ülesandepõhisteks sektsioonideks.
- FFmpeg/FFprobe, metaandmete ja subtiitrite vormid kasutavad uut kaheveerulist seadistusmustrit, mis muutub mobiilis üheveeruliseks.
- Salvestamise ja väärtuste kontrollimise tegevusriba püsib pika vormi kasutamisel nähtaval.
- Ühtlustatud on vormiväljad, olekuread, juhendplokid, teema valikud ja meediakogude haldus nii tumedas kui heledas vaates.

## 0.5.8 — 2026-07-16

- Seriaaliepisoodi lõpu eel kuvatakse animeeritud järgmise osa kaart koos osa nime, SxxExx tähise, automaatesituse loenduri ja väikese ajajoonega.
- Praeguse episoodi lõppemisel käivitub järgmine serveris olev osa automaatselt, kui kasutaja pole automaatesitust tühistanud.
- Kui järgmist osa ei ole saadaval, kuvatakse kolm vaatamata filmi või seriaali soovitust ilma automaatesituseta.
- Soovitused eelistavad sama või sarnast žanri, välistavad juba vaadatud filmid ning seriaali puhul avavad kõige vanema vaatamata osa.
- Lisatud eraldatud serveripoolne post-play teenus ja `/api/v1/media/{id}/post-play` API, et veebiklient ning tulevased Android TV ja Tizeni kliendid kasutaksid sama loogikat.
- Järgmise osa playback-options eellaaditakse kergelt enne vahetust, kuid FFmpeg ei käivitu enne tegelikku esitust.


## 0.5.6 — 2026-07-14

- Parandatud seriaaligruppide lõhenemise tegelik algpõhjus: punktidega eraldatud pealkirja viimast sõna ei käsitleta enam ekslikult faililaiendina. Näiteks `For.All.Mankind` säilib tervikuna ega muutu nimeks `For All`.
- `.to` sisaldavat punktidega pealkirja ei käsitleta enam ekslikult veebiaadressina. Näiteks `IT.Welcome.to.Derry` säilib ühe seriaali nimena.
- Eri release-kaustadest pärinevad aastad on nüüd nõrgad metadata vihjed ega lõhesta sama failinimepõhise seriaali episoode eri gruppidesse.
- Faili- ja kaustatee analüüs kasutab ühtset Windowsi ning testikeskkonna teeparserit, mistõttu erinevad `\` ja `/` eraldajad annavad sama tulemuse.
- Käivitumisel toimuv kerge kataloogimigratsioon ühendab varasemates versioonides lõhenenud grupid, säilitab tugevaima metadata ja järjestab episoodid hooaja ning osanumbri järgi.

## 0.5.5 — 2026-07-14

- Parandatud viga, mille tõttu sama seriaali episoodid võisid erinevate aastate või vanade parent-ID-de tõttu ilmuda eraldi seriaalikaartidena.
- Seriaali grupivõti põhineb nüüd kanoniseeritud failinime pealkirjal ega sõltu episoodi aastast; kirjavahemärkide erinevused ei tekita uut gruppi.
- KoduMedia käivitamisel tehakse kerge kataloogimigratsioon, mis ühendab vanad fragmenteerunud grupid ilma faile uuesti FFprobe'iga analüüsimata.
- Ühendamisel säilitatakse tugevaim olemasolev TMDb/IMDb metadata, artwork ja kirjeldus ning episoodide seosed uuendatakse uuele parent-kirjele.
- Vaatamisajaloo teenus kasutab vana salvestatud SeriesID asemel kataloogis olevat praegust parent-seost, et jätkamine ja progress säiliksid.

## 0.5.4 — 2026-07-14

- Parandatud vale TMDb seriaali vaste, kus `Silo` võis sobituda pealkirjaga `Love of Silom`.
- Osalised pealkirjavasted kontrollitakse tervete sõnade järgi ning varem salvestatud vale vaste parandatakse automaatselt.

## 0.5.3 — 2026-07-14

- Lihtsustasin playeri allosa kujundust: eemaldatud on senine "hõljuv klaas" tunnetus ning juhtimisriba kasutab nüüd kompaktsemat, rahulikumat ja KODU CVI-ga paremini haakuvat stiili.
- Subtiitrite valik on nüüd eraldi selge menüüna: näha on aktiivne valik, "Väljas" olek, olemasolevad subtiitrirajad ning eraldi tegevus OpenSubtitlesest uute subtiitrite otsimiseks.
- Ühtlustatud on ümmarguste juhtnuppude stiil playeris, otsingus, detailvaadetes ja hover-play nuppudel, et kogu kliendi vaade jätaks visuaalselt järjepidevama mulje.

# Muudatuste logi

## 0.5.2 — 2026-07-14

- Uuendas playeri visuaalse keele Kodu CVI-ga: hõljuv klaasist juhtimisriba, brändigradiendiga esitusnupp, õhem ajajoon, selgemad olekud ja minimalistlik subtiitripaneel.
- Lisatud ajajoone eraldi vaadatud ja puhverdatud olek ning kompaktsemad ajamärgised.
- Playeri ülariba, veateated, laadimisolekud ja subtiitrite otsing kasutavad nüüd sama tüpograafiat, värve, raadiuseid ja liikumist nagu ülejäänud Kodu kasutajaliides.
- Lisatud Kodu täielik visuaalse identiteedi juhend PDF-ina eraldi väljalaskedokumendina.

## 0.5.1 — 2026-07-14

- Windowsi FFmpeg, FFprobe ja subtiitrite teisendamise protsessid käivituvad ilma nähtava konsooliaknata.
- Jätka vaatamist värskendub avalehel kohe pärast playeri sulgemist.
- Timeline-seek kasutab puhverdatud vahemikus kohest seek'i ja mujal täpset transkodeeritud restarti.
- FFmpeg loob alguses kontrollitud 10-sekundilise puhvri ning jätkab seejärel reaalaja tempos.
- Player ei käivita videot enne sihtpuhvri täitumist; eemaldatud on brauseri enneaegne autoplay.
- Detailvaade eellaadib kergelt playeri mooduli ja esitusvalikud ilma FFmpegi käivitamata.
- Seriaali põhinupp on lühidalt „Esita” või „Jätka”; episoodikood kuvatakse nupu all.

## 0.5.0 — 2026-07-14

### Püsiv vaatamisajalugu ja jätkamine

- Filmide ja episoodide vaatamispositsioon salvestatakse eraldi aatomilisse `data\watch-state.json` faili, mistõttu meediakogu skannimine ja KoduMedia uuendamine ei kustuta vaatamisajalugu.
- Player jätkab pooleli olevat sisu eelmisest positsioonist ning saadab kerge kontrollpunkti maksimaalselt kord 15 sekundi jooksul, pausil, sulgemisel ja esituse lõpus.
- Avalehele lisati enne esiletõstetud sisu rida **Jätka vaatamist**, kus on kuni viis viimati vaadatud 1–95% pooleli olevat filmi või seriaali episoodi.
- Pooleli või osaliselt vaadatud sisu thumbnailidel kuvatakse alati edenemisriba. Seriaali kaardil eelistatakse aktiivse pooleli oleva osa progressi; muul juhul kuvatakse serveris lõpetatud osade osakaal.
- Seriaali põhivaates on nüüd kontekstipõhine **Jätka vaatamist** või **Esita N. hooaja M. osa** nupp.

### Õige järjestus ja seriaali saadavus

- Avalehe **Viimati lisatud filmid** ja **Viimati lisatud seriaalid** järjestatakse Windowsi faili `Date modified` väärtuse järgi; seriaali kuupäev on selle uusima serveris oleva episoodi kuupäev.
- Vanad kataloogid parandavad seriaali tuletatud modified-aja käivitamisel automaatselt, seega ei pea ainult järjestuse parandamiseks kogu uuesti skannima.
- **Uued episoodid** järjestatakse ametliku väljalaskekuupäeva järgi, mitte failinime ega kataloogi lisamise aja järgi.
- Seriaali detailvaade eristab ametlikult ilmunud hooaegade ja osade arvu serveris kohe vaadatavast sisust.
- Kui TMDb järgi on ilmunud serverist uuem osa, kuvatakse selle hooaja- ja osanumber, nimi ning ilmumiskuupäev esiletõstetud teavitusena.

### UI ja skannimise parandused

- Avalehe plakatiridadel on ühtlasem kaardistiil ning detailvaadetes kasutatakse suuremat taustapilti ja kokkuvõtet.
- Seriaalide ja filmide metaandmete sünkroon on stabiilsem ning skannimine töötleb faile etteaimatavamalt.
