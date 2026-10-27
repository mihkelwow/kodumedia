KoduMedia 0.8.58 - Windows x64

KÄIVITAMINE
1. Paki ZIP täielikult lahti kohalikule kettale.
2. Käivita start-kodumedia.cmd.
3. Ava brauseris http://127.0.0.1:8096.

0.8.58 SUUREM PUHVER JA PÜSIV ENCODER'I VARU
- Tavaline 1080p transcode alustab 24 sekundi varuga ja hoiab eesmärgina 90 sekundit.
- 4K/HDR transcode kasutab 30 sekundi stardivaru ja kuni 120 sekundi eesmärki.
- Seek'i järel oodatakse enne jätkamist taas täielik turvapuhver.
- NVIDIA NVENC-i eelistatakse automaatselt QSV-le ja AMD AMF-ile, kui see töötab.
- Encoder'i diagnostika näitab nüüd viimast tegelikku tootmiskiirust, mitte kogu
  sessiooni algusest kogunenud eksitavat keskmist.

0.8.57 PLAYERI KÄIVITUMISE HOTFIX
- Filmid ja seriaalid ei jää enam brauseris lõputult 14–20 sekundi puhvrit ootama.
- Player alustab töökindla brauseripuhvriga ja jätkab aeglase voo korral ise.
- Windowsi Chrome ja Edge kasutavad taas töökindlat hls.js esitusteed ega satu
  castimise native HLS-i kaudu MediaError 4 kordusveasse.
- Samsungi/DLNA 12–20 sekundi serveripuhver ning AirPlay parandused säilivad.

0.8.56 TÖÖKINDEL AIRPLAY JA PUHVERDATUD SAMSUNG DLNA
- iPhone ja Safari annavad AirPlay-telerile nüüd päris native HLS võrguaadressi.
- Samsungi DLNA kasutab töökindlamat MPEG-TS HLS-i ja 12–20 sekundi käivituspuhvrit.
- Ühilduv 4K/HEVC säilib; vältimatu transkodeerimine kasutab 1080p H.264 fallback'i.

0.8.55 AIRPLAY, GOOGLE CAST JA SAMA VÕRGU TELERID
- Playeri uus castimise nupp koondab AirPlay, Google Casti ja DLNA/UPnP seadmed.
- Samsung RU7172 leitakse samas võrgus DLNA kaudu; Apple'i seadmetest töötab AirPlay 2.
- Välist esitust saab playerist peatada, jätkata, kerida ja lõpetada.
- Valitud subtiitrid põletatakse teleri jaoks vajadusel videopilti.
- Google Cast vajab HTTPS-i; kohaliku HTTP-aadressi korral kasuta AirPlayd või DLNA-d.

0.8.54 ÜHTNE ÜMAR ESITAMISNUPP
- Hero, seriaali-, filmi- ja episoodivaade kasutavad sama pill-kujundusega nuppu.
- Kõigi esitamisnuppude miinimummõõt on 160 × 48 px ning puuteala on mugav.
- Tume, hele, hover-, vajutus- ja klaviatuurifookuse olekud on ühes stiilis.
- Jagatud komponent hoiab nupud ka tulevastes muudatustes kujunduslikult koos.

0.8.53 MUGAVAM EPISOODILIIKUMINE DESKTOPIL
- Hiirega lohistatud episoodirida ei hüppa enam lühikese liigutuse järel tagasi.
- Puuteekraani hästi toimiv snap-käitumine jääb muutumatuks.
- Vasak ja parem nooleklahv liigutavad fookuse eelmisele või järgmisele kaardile.
- Home ja End viivad episoodirea algusesse või lõppu.

0.8.52 HORISONTAALNE CVI-PÕHINE HOOAJAVALIK
- Hooaja tekst-tabid paiknevad alati kõrvuti ühe horisontaalse reana.
- Aktiivne tab kasutab rakenduse CVI pinda, fookust ja tagasihoidlikku sügavust.
- Osade arv on eraldi metainfo ega tekita enam hooajanupu sisse teist tekstirida.
- Ühe hooaja korral kuvatakse rahulik pealkiri, mitte eksitav valikunupp.

0.8.51 HORISONTAALNE JA PUUTETUNDLIK EPISOODIKARUSSELL
- Episoodid paiknevad seriaalivaates nüüd kõrvuti vasakult paremale.
- Embla karusselli saab pühkida sõrmega ning juhtida hiire, klaviatuuri ja nooltega.
- 16:9 episoodikaardid näitavad selge hierarhiaga pealkirja, kirjeldust ja metainfot.
- Üleliigsed horisontaaljooned on eemaldatud ning mobiilis jääb järgmine kaart nähtavale.

0.8.50 LÄBIPAISTEV SEADETE MENÜÜ JA LOETAV MOBIILIVALIK
- Hiline stiilireegel ei lisa külgmenüüle enam #171719-lähedast taustakasti.
- Töölauavaate menüü konteiner on päriselt läbipaistev; premium menüüread säilivad.
- Mobiilivalik järgib rakenduse CVI-d: 48 px puuteala, piir, ümardus ja korrektne nool.
- Avatud valikute tekst ning taust on tumedas ja heledas teemas selgelt loetavad.

0.8.49 TAASTATUD PREMIUM-KLASSI SEADETE KÜLGMENÜÜ
- Taastatud on suuremad ümardatud menüüread, kirjeldused ja suunanooled.
- Aktiivne ja klaviatuuriga fokuseeritud rida kasutab jälle kohalikku aktsentolekut.
- Kogu külgmenüü konteiner on läbipaistev ega moodusta enam halli taustakasti.
- Uuenduste real ei ole püsivat halli erandtausta.
- 0.8.48 mobiilne menüü ja muud seadete töövoo parandused säilivad.

0.8.48 SELGEM JA RAHULIKUM SEADETE LEHT
- Külgmenüüst on eemaldatud tarbetu taustakast ning jaotised on loogiliselt rühmitatud.
- Mobiili ja tahvelarvuti vaates kasutatakse kompaktset jaotisevalikut.
- Seadete pealkiri, töövalmiduse ülevaade ja jaotiste hierarhia vastavad ülejäänud rakendusele.
- Iga rühm näitab, kus ja millal muudatus rakendub.
- Salvestamise tegevused ja tagasiside püsivad ka pika lehe korral nähtaval.
- Pooleli esmase seadistuse juhend avaneb uuel kasutajal automaatselt.

0.8.47 PREMIUM-KLASSI HERO SWIPE
- Kõrvuti asuvad täislaiuses slaidid liiguvad nüüd sõrmega vahetult kaasa.
- Mõõdukas pühkimine või kiire lühem viibe vahetab slaidi kindlalt.
- Liikumise hoog kandub loomulikku snap-animatsiooni ja töötab mõlemas suunas.
- Eraldi touch-fallback katab erineva Pointer Events käitumisega puuteekraanid.
- Vertikaalne pühkimine kerib lehte ega vaheta hero slaidi.
- Mobiili nooled jäävad alumisse paremasse nurka ning punktid on peidetud.

0.8.46 PUUTETUNDLIK HERO-KARUSSELL
- Hero slaide saab puutetundlikul ekraanil vasakule ja paremale pühkida.
- Vertikaalne pühkimine jätkab lehe tavapärast kerimist.
- Mobiilis asuvad eelmise ja järgmise slaidi nooled all paremal.
- Mobiilis on slaidipunktid ja loendur peidetud; töölauavaade ei muutu.

0.8.45 MUUSIKAVAATE PEALKIRJADE HIERARHIA PARANDUS
- Muusika lehe pealkiri ja ruumikasutus vastavad nüüd teistele sisulehtedele.
- Uus brauser alustab ingliskeelse kasutajaliidesega.
- Uue serveri metaandmete keel on en-US ja eelistatud subtiitrikeel en.
- Olemasolevaid salvestatud keelevalikuid uuendus ei muuda.
- Eesti ja inglise kataloogid on 897 võtmega täieliku 1:1 katvusega.
- Inglise vaade tõlgib ka tuntud serveripoolsed oleku- ja veateated.

0.8.43 TÖÖKINDEL LOGIN-STARTUP JA INSTALLER
- Exploreri hilise käivitumise korral proovitakse tray ikooni lisamist uuesti.
- Tray püsiv tõrge ei lõpeta enam KoduMedia veebiserverit.
- Logi eristab käsitsi ja Windowsi sisselogimisel tehtud käivitust.
- Installer kasutab stabiilset kasutajapõhist Programs kausta ilma UAC-ta.
- Olemasolevad seaded, vaatamisajalugu ja teegiandmed migreeritakse automaatselt.
- Uninstaller säilitab kasutaja data-kausta, et vältida andmekadu.

0.8.42 WINDOWS STARTUP TÖÖTAB JA NÄITAB ÕIGET OLEKUT
- Tray menüü kontrollib nii Run-käsku kui ka Windowsi StartupApproved olekut.
- Windowsis keelatud startup ei kuva enam eksitavalt menüüs linnukest.
- Valiku sisselülitamine parandab vana installitee ja lubab KoduMedia uuesti.
- Valiku väljalülitamine eemaldab mõlemad ainult KoduMediale kuuluvad kirjed.

0.8.41 HERO SÄILIB TREILERI TÕRKE KORRAL
- Treileri viga ei eemalda enam filmi või seriaali hero-karussellist.
- Kui kõik treilerid ebaõnnestuvad, jääb nähtavale staatiline backdrop-pilt.
- Vana 0.8.40 vale 30-päevane tõrkevahemälu kustutatakse automaatselt.
- Ajutisi võrgu-, API- ja playeritõrkeid ei salvestata püsiva sisutõrkena.
- Hero 480 ms ristüleminek ning 34 / 7 / 22 px juhtnupud jäid muutmata.

0.8.40 TAASTATUD HERO SLIDERI JUHTNUPUD
- Hero nooled on taas täpselt 34 x 34 px.
- Tavalised slaidipunktid on 7 x 7 px ja aktiivne punkt 22 x 7 px.
- Puute- või hiirerežiimi vahetumine ei muuda enam juhtnuppude suurust.
- Hero juhtnuppude geomeetriat ei muudeta ilma kasutaja eraldi soovita.
- Kõik ülejäänud 0.8.39 puute-, fookuse-, kerimis- ja klaviatuuriparandused säilivad.

0.8.39 PUUTE- JA KLAVIATUURINAVIGATSIOON
- Avalehe meediaread ja muud horisontaalsed valikud kasutavad native touch-inertsi.
- Klaviatuuri või puldi fookus kuvatakse kaardil ühe 2 px rõngana ainult thumbnail'il.
- Sisemine fookusrõngas ja servavaru hoiavad ääres oleva selektsiooni tervenisti nähtaval.
- Muusikapleieri transport, ajajoon, helitugevus ja detailinupp osalevad ruumilises navigatsioonis.

0.8.38 KERGE STAATILINE ÜLDVAADETE TAUST
- Avaleht, filmide ja seriaalide kogud, muusika avaleht ning seaded kasutavad sama tausta.
- Tume teema jääb domineerivalt tumedaks; hele teema kasutab rahulikku pärlhalli põhja.
- Taust on CSS-gradient: pildifaili, blur-filtrit, animatsiooni ega hover-põhist taustavahetust ei kasutata.
- Filmi, seriaali ja episoodi detailvaadete udustatud meediapildi loogika jäi muutmata.
- Hero meediapilt ja 480 ms pilt–trailer ristüleminek jäid muutmata.

0.8.37 SUJUV HERO PILDI JA TRAILERI ÜLEMINEK
- Hero taustapilt hajub mängivasse trailerisse 480 ms kinolise opacity-ristüleminekuga.
- Trailerit ei näidata laadimise ega puhverdamise ajal: see ilmub alles päris PLAYING olekus.
- Vigase või blokeeritud traileri korral jääb taustapilt katkematult nähtavaks.
- Hero ülejäänud käitumine ja 0.8.36 kettakasutuse parandused jäid muutmata.

0.8.36 PIIRATUD KETTAKASUTUS JA PUHAS ESMATÖÖ
- ZIP sisaldab kõike esmakäivituseks vajalikku; varasemat KoduMedia versiooni ega data-kausta ei eeldata.
- Uuendamisel varundatakse ja asendatakse ainult päriselt muutunud failid.
- Uuenduste tööfailid eemaldatakse pärast lõpptulemust ning alles jäävad kaks viimast rollback-komplekti.
- Optimeeritud video cache hoiab vaikimisi vabana vähemalt 10 GB või 10% kettast.
- Katkenud transkodeeringud ja eemaldatud meedia cache-orvud puhastatakse automaatselt.
- Muusika piltidel, metaandmetel, tunnusmuusikal, teisendatud subtiitritel ja logidel on mahupiirid.
- Seadete jaotis Kettakasutus näitab mahtu ning pakub localhost-brauserist ohutut puhastust.
- Meediafaile ega kasutaja allalaaditud subtiitreid automaatne hooldus ei kustuta.

0.8.35 PÜSIV AMBIENT-TAUSTA TOPELTPUHVER
- Kaks udustatud taustakihti püsivad alati DOM-is ja komposiitoris.
- Uus pilt dekodeeritakse ning asetatakse täielikult läbipaistvasse puhvrisse.
- Ainult valmis puhver hajub 360 ms jooksul vana pildi peale; vana kiht ei vilgu vahepeal tagasi.
- Ülemineku lõpus vahetuvad kihtide rollid ühe commit'iga ja DOM-elemente ei looda ega eemaldata.
- Kiire thumbnailide kohal liikumine jätab ootele ainult viimase soovitud tausta.
- Hero juhtnupud, vaigistatud autoplay, trailerifilter ja 0.8.34 kompaktne indikaatoristiil jäid muutmata.

0.8.34 TAASTATUD HERO JUHTNUPUD JA VÄLKUMISETA TAUST
- Hero slaidivaliku indikaatorid kasutavad taas varasemat kompaktset 7 px / 22 px kujundust.
- Nooled, loendur, helinupp ja muud hero juhtnupud jäid muutmata.
- Eelmine ambient-taust püsib täielikult nähtav, kuni uus dekodeeritud pilt selle pehmelt katab.
- Vana blur-kiht eemaldatakse tegeliku opacity-ülemineku lõpus.
- Kiire thumbnailide kohal liikumine ei katkesta pooleliolevat taustavahetust.
- 0.8.33 vaigistatud hero, trailerifilter ning scrolli- ja hover-jõudlusparandused jäävad alles.

0.8.27 SEADED JA PLAYER
- Seaded on koondatud kuueks ülesandepõhiseks jaotiseks.
- Salvestamata muudatuste korral ilmub kleepuv salvestusriba koos salvesta, kontrolli ja taasta tegevustega.
- Harva kasutatavad tööriistateed ja tehnilised taasesitusvalikud asuvad Täpsemad seaded all.
- Playeri juhtnupud, timeline, rajamenüüd, diagnostika, subtiitriotsing ja post-play on eraldi moodulid.
- Peidetud playeri juhtnupud ei ole klaviatuuriga fookustatavad; menüüd toetavad nool-, Home-, End- ja Escape-klahve.
- Progressid kasutavad transformi ning sagedaste toimingute animatsioonid on lühikesed või eemaldatud.

0.8.26 KOHENE AVALEHE HERO
- Avalehele naastes ja refreshimisel taastatakse hero püsivast vahemälust enne järgmist ekraanijoonistust.
- Hero põhiandmed ei oota enam episoodi- ega žanriridade aeglasemaid päringuid.
- Külmkäivituses ei kuvata osaliselt valmis avalehte, kuhu hero hiljem vahele hüppaks.
- Korduv reveal-animatsioon eemaldati; olemasolev slaidinavigatsiooni CVI jäi muutmata.

0.8.25 HERO SLAIDINAVIGATSIOONI PARANDUS
- Taastatud on hero eelmise/järgmise nupud, slaidipunktid ja loendur.
- Kasutatakse sama kompaktset CVI-d nagu enne regressiooni; uusi visuaalseid eristiile ei lisatud.

0.8.25 HERO JA SIRVIMISE VIIMISTLUS
- Hero slaidivalija on tagasi varasemas kompaktses CVI-s.
- Vana Reacti-järgne runtime UI kiht on väljalaskest eemaldatud.
- Meediakaartide hover on rahulikum ning klaviatuurifookus ei liiguta sisu.
- Filmide ja episoodide progressiribad kasutavad transformi, mitte layouti muutvat laiust.
- Kogu filtrid kasutavad ühiseid kontrolli- ja fookustokeneid.

0.8.25 UI VUNDAMENT
- Üks autoriteetne tokeni- ja motion-kiht määrab värvid, pinnad, raadiused, kestused ning reduced-motion käitumise.
- Otsing, detailvaated, failibrauser, päise menüüd ja seadete väljad kasutavad jagatud Dialog/Popover/Menu/IconButton/Field komponente.
- Hero heli käivitub ainult kasutaja helinupust; klaviatuurifookus seda enam sisse ei lülita.
- Interaktsioonialad on vähemalt 44 × 44 px ning hover-olekud piirduvad päris osutusseadmetega.
- Lähtekood ei laadi enam DOM-i ümberkirjutavaid home-featured, initial-setup ega series-themes runtime-kihte.


0.8.22 AVALIK NPM-REGISTER JA KORRATAV LÄHTEKOODIBUILD
- package-lock.json kasutab ainult ametlikku registry.npmjs.org registrit.
- apps\web\.npmrc ja buildi/testi skriptid välistavad vigase sisemise registri pärandumise.
- Automaatne kontroll peatab buildi, kui lockfile'i satub sisemine või ootamatu registriaadress.
- Rakenduse kasutusfunktsioonid ei muutunud.

0.8.21 ÜHTNE TUNNUSMUUSIKA RIPPMENÜÜ
- Tunnusmuusika esitusrežiimi valik kasutab nüüd sama komponenti ja CVI-d nagu teised seadete lehe rippmenüüd.
- Eraldi tunnusmuusika select-värvid, mõõdud ja fookusstiilid eemaldati.
- Esituse, vahemälu ja seadete funktsionaalsus ei muutunud.

0.8.20 TÖÖKINDLAM SERIAALIDE TUNNUSMUUSIKA
- Esmakordne laadimine jätkub serveri taustatööna ka siis, kui kasutaja lehelt lahkub.
- Sama seriaali kohta käivitatakse korraga ainult üks allalaadimine ja edukas tulemus salvestatakse data\themes\series vahemällu.
- Episoodi käivitamisel katkeb brauseris ootamine ning tunnusmuusika ei hakka episoodi vaatamise ajal hiljem iseseisvalt mängima.
- Helitugevus on 15%, fade-in ja fade-out kestavad 2 sekundit ning heli ei lähe kordusele.
- Seadete Alati / Ainult esimesel avamisel / Väljas rippmenüü on tumedas ja heledas teemas loetav.

0.8.19 SERIAALIDE TUNNUSMUUSIKA
- ThemerrDB kaudu leitud tunnusmeloodia mängib seriaali, hooaja ja episoodi detailvaates.
- Sama seriaali vaadete vahel heli jätkub; episoodi käivitamisel või seriaali vaadetest lahkumisel hajub see välja.
- Edukalt hangitud heli salvestatakse data\themes\series vahemällu.

0.8.17 KAALUTUD JA KORDUSTETA AVALEHE FILMIKATEGOORIAD
- Filmi žanride järjekord määrab tema tugevaima avalehe kategooria.
- Sama filmi näidatakse žanriridades ainult üks kord.
- Animatsioonifilm kuvatakse alati ainult Animatsiooni reas, sõltumata teistest žanritest.
- Kategooriate sees säilib IMDb hinnangu järgi järjestus ja tühje ridu ei näidata.

0.8.16 ESMANE SEADISTUS: TEGEVUSED JA INGLISE KEEL
- Esmase seadistuse juhend on täielikult olemas nii eesti kui inglise keeles.
- Salvesta seaded salvestab kohe vormi väärtused; Skanni kõik meediakogud käivitab kõik kaustadega kogud korraga.
- Juhend mainib ka system tray menüü skannimis- ja Windowsiga automaatse käivitumise valikuid.

0.8.14 ESMASE SEADISTUSE AKORDION
- KoduMedia töövalmiduse plokis on nüüd Esmane seadistus nupp.
- Avatav juhend koondab tasuta TMDb, OpenSubtitlesi, meediakaustade ning salvestamise ja skannimise sammud.
- Iga samm viib otse õigesse seadete jaotisesse.
- Muusika põhituvastus töötab ilma võtmeta; valikuline AcoustID võti on samuti tasuta.

0.8.13 ÜHTNE MUUSIKA METAANDMETE LISAINFO
- Muusika metaandmete juhend näeb nüüd välja samasugune nagu TMDb seadistamise juhend.
- Infoikoon, pealkiri, kirjeldus ja AcoustID link paiknevad korrektselt ühes plokis.
- Ikoon ei murdu enam eraldi reale ning mõlemad juhendid kasutavad sama tüpograafiat ja vahemikke.
- Funktsionaalsus ei muutunud.

0.8.12 VERTIKAALNE JA AVAR SEADETE PAIGUTUS
- Iga seadete jaotise pealkiri ja kirjeldus paiknevad nüüd juhtnuppude kohal, mitte vasakus külgveerus.
- Kogu seadete leht järgib sama loomulikku ülalt-alla lugemisjärjekorda ja järjepidevat sektsioonivahet.
- Uuendamise osa jääb esiletõstetuks, kuid kasutab sama vertikaalset hierarhiat.
- Funktsionaalsus ei muutunud.

0.8.11 STABIILNE SEADETE PAIGUTUS JA TÜPOGRAAFIA
- Keel, Välimus ja Meediakogud ei saa enam üksteise otsa nihkuda: brauserist sõltuv fieldset/legend paigutus asendati tavaliste sektsioonipäistega.
- Kõik samaväärsed jaotisepealkirjad, kirjeldused, väljasildid ja abitekstid kasutavad sama fondisuurust, kaalu ja reavahet.
- Kaheveeruline paigutus, vasaku päiseveeru laius ning sektsioonide vahed on üle terve seadete lehe ühtlustatud.
- Töövalmiduse plokk säilib esiletõstetuna ja uuendamise osa jääb prominentseks, kuid kumbki ei kasuta haprat absoluutset päisepaigutust.
- Funktsionaalsus ei muutunud.

0.8.10 VIIMISTLETUD SEADETE LEHT
- Seadete külgmenüüst eemaldati topelt püstjooned; navigeerimine on nüüd puhas, tasakaalus ja ühetaoline.
- Tavalised seadistusjaotised kasutavad avarat kaheveerulist editorial-paigutust, mitte korduvaid kaste ega horisontaaljooni.
- KoduMedia töövalmiduse ülevaade jäi esiletõstetud visuaalseks ankruks ning selle kaartidel pole hover-liikumist.
- Uuendamise plokk on terviklikult joondatud: pealkiri, olek, tegevus ja turvateave paiknevad ühes selges pinnas.
- Kontrollimise ja salvestamise tegevusriba sai korrektse sisemise paddingu, ümarduse ja ühtse tausta.
- Keel, välimus ning meediakogud kasutavad sama hierarhiat, vahemikke, sisendstiile ja fookusolekuid. Funktsionaalsus ei muutunud.

0.8.9 KERGEM JA AVARAM SEADETE LEHT
- Tavalised seadete sektsioonid ei ole enam pesastatud kaartides, vaid kasutavad õhku, tüpograafiat ja eraldusjooni.
- Vähendatud on varje, klaasiefekte ja backdrop-filter kasutust, et vaade oleks kliendile kergem.
- KoduMedia töövalmiduse ülevaade jäi esiletõstetuks, kuid olekukaartidel pole enam hover-animatsiooni.
- Uuendamise osa jääb prominentseks, kuid selle sees pole enam eraldi kaarti.
- Salvestusriba on nüüd lihtne vormi lõpus paiknev tegevusrida. Funktsionaalsus ei muutunud.

0.8.8 ÜHTNE SEADETE LEHE CVI
- Kogu seadete leht kasutab nüüd avalehe ning filmide/seriaalide vaadetega sama kinolikku visuaalset süsteemi.
- Kõik sektsioonid, vormiväljad, valikud, olekukaardid, tegevused ja meediakogud järgivad ühtset kujundust.
- KoduMedia uuendamise osa on pastelset gradiendi ja selge tegevusnupuga tavalisest prominentsem.
- Funktsionaalsus ja seadete salvestusloogika ei muutunud; uuendus puudutab ainult kasutajaliidest.

0.8.7 JÄRJESTIKUNE ESILETÕSTETUD AUTOPLAY
- Treileri lõppedes liigub karussell automaatselt järgmisele sisule ja käivitab selle treileri.
- Viimase treileri järel autoplay peatub ning viimasele slaidile jääb staatiline taustapilt.
- Viimaselt slaidilt ei tehta automaatset ringi tagasi esimesele ja üksik treiler ei loobi enam.
- Noolte, punktide või klaviatuuri/TV-puldi nooltega käsitsi liikudes aktiveerub autoplay uuesti.

0.8.6 KÜMNE SISUGA ESILETÕSTETUD KARUSSELL
- Avalehe hero sisaldab nüüd kuni kümmet vaatamata või alla 5% vaadatud filmi ja seriaali.
- Valik tehakse kuni kahekümne kõrgeima kvaliteediga kandidaadi seast, et päevapõhine rotatsioon püsiks mitmekesine.
- Noolte, punktide, loenduri ning klaviatuuri/TV-puldi navigeerimine töötab endiselt samamoodi.
- Treileri automaatne fallback ja vanusepiiranguga videote peitmine jäid muutmata.

0.8.5 ESILETÕSTETUD KARUSSELL JA TREILERI FALLBACK
- Esiletõstetud hero kasutab nüüd kogu sisulaiust ja sisaldab kuni viit vaatamata sisu.
- Liigu eelmise/järgmise nupu, punktide või klaviatuuri/TV-puldi nooltega.
- Rakendus proovib automaatselt järgmist TMDB treilerit, kui YouTube ei luba videot embedis mängida.
- Vanusepiirangu või embed-keelu teadet hero peal ei näidata: video ilmub alles päris playback'i käivitumisel.
- Kui sobivat treilerit pole, jääb ekraanile staatiline TMDB taustapilt.

0.8.4 AVALEHE ESILETÕSTETUD HERO
- Esiletõstetud sisu on nüüd suur kinolik kaart Jätka vaatamist rea järel.
- Valikusse lähevad ainult vaatamata või alla 5% vaadatud serveri filmid ja seriaalid.
- TMDB title-logo kuvatakse võimalusel valgena; muidu kasutatakse tekstipealkirja.
- TMDB-ga seotud YouTube'i treiler käivitub hääletult, kui hero on nähtav.
- Heli saab muuta parempoolsest ülanupust; klaviatuuri/TV-puldi fookus lülitab heli sisse.
- Vähendatud liikumise või andmesäästu korral jääb staatiline taustapilt.
- Olemasolev teek rikastatakse logo ja treileriga ühe korra taustal.

0.8.3 MUUSIKA KASUTAJALIIDESE PARANDUSED
- Muusika avalehe Kõik lood kaart on Kodu CVI järgi kujundatud, kuid selle pealkiri jääb põhilehe pealkirjale selgelt alla.
- Kõik lood otsing töötab iga klahvivajutusega ilma kogu vaate refreshi ja fookuse kadumiseta.
- Muusika sorteerimise ja filtreerimise paneel on eemaldatud; vaikimisi järjestus on pealkiri kasvavalt.
- Audiofaili detailvaates kuvatakse artwork tervikuna, teksti ja servi ära lõikamata.

0.8.2 ARTWORKI VALE JAGAMISE PARANDUS
- Igal audiofailil on nüüd oma artworki cache; üldised artisti/albumi tagid ei jaga enam pilti.
- cover.jpg/folder.jpg kasutatakse mitme looga kaustas ainult päris albumikausta korral.
- Üldiste tagidega lahtises kaustas proovitakse Shazami pilti enne võimalikku korduvat embedded playlisti pilti.
- Audiofailiga sama nimega JPG/PNG jääb turvaliseks loo-spetsiifiliseks sidecar-pildiks.
- Uus brauseri cache-versioon jätab vana vale pildi kohe kõrvale.
- Pärast uuendamist käivita üks kord Seaded -> Muusika metaandmed ->
  Töötle kogu muusikateek uuesti.

0.8.1 SHAZAM ESIMESEKS TUVASTUSEKS
- Tasuta ja API-võtmeta Shazami audio-fingerprint on esimene automaatne vaste.
- Server saadab tuvastuseks ainult kompaktse audio-signatuuri, mitte helifaili.
- Shazami vaste žanr ja artwork on kohaliku audioheuristika ees prioriteetsed.
- Tempo põhjal ei märgita enam lugu automaatselt Hip-Hopiks.
- Kui Shazam vastet ei leia või endpoint ei tööta, jätkub AcoustID/MusicBrainzi,
  Deezeri kataloogivaste, lokaalse audioanalüüsi ja genereeritud artworkiga.

Shazami endpoint on mitteametlik ja pöördprojekteeritud: see ei vaja praegu
makset ega API võtit, kuid võib tulevikus muutuda või päringuid piirata.
Sidecar ja audiofaili tagid jäävad alati prioriteetseks.

UUENDAMINE
Paki uus versioon eraldi kausta. Ära kopeeri vana data-kausta ZIP-i sisse; KoduMedia kasutab käivitamisel kõrval asuvat data-kausta. Enne asendamist tee data-kaustast varukoopia.

Täpsem info: README.md
Portable-paki nimi: kodumedia-0.8.54-windows-x64.zip
