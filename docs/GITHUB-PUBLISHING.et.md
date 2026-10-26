# KoduMedia avaldamine GitHubis ilma PowerShellita

See juhend kasutab GitHubi veebilehte ja tasuta GitHub Desktopi. Käsurida ega
PowerShelli ei ole vaja.

Juhend eeldab GitHubi kasutajanime `mihkelwow` ja repo nime `kodumedia`.

## 1. Loo GitHubis tühi repo

1. Logi GitHubi kasutajaga `mihkelwow`.
2. Ava <https://github.com/new>.
3. Täida väljad:
   - **Owner:** `mihkelwow`
   - **Repository name:** `kodumedia`
   - **Description:**

     ```text
     Self-hosted Windows media server for movies, TV series and music, with a React web interface, HLS transcoding, subtitles, metadata, AirPlay, Google Cast and DLNA.
     ```

   - **Visibility:** `Public`
   - **Add README:** `Off`
   - **Add .gitignore:** `No .gitignore`
   - **Add license:** `No license`

README, `.gitignore` ja GPL-3.0 litsents on GitHub-ready paketis juba olemas.
Kui GitHub need ise lisaks, tekiksid topeltfailid ja esimese ühendamise konflikt.

4. Vajuta rohelist **Create repository** nuppu.

## 2. Paigalda GitHub Desktop

1. Ava <https://desktop.github.com/download/>.
2. Laadi GitHub Desktop Windowsile alla.
3. Paigalda ja käivita rakendus.
4. Vali **Sign in to GitHub.com**.
5. Kinnita sisselogimine avanevas brauseris.

GitHub Desktop teeb taustal samad Giti toimingud, mille jaoks muidu kasutataks
PowerShelli.

## 3. Klooni tühi repo arvutisse

Kõige lihtsam viis GitHubi veebilehelt:

1. Ava äsja loodud <https://github.com/mihkelwow/kodumedia>.
2. Vajuta rohelist **Code** nuppu.
3. Vali **Open with GitHub Desktop**.
4. Luba brauseril GitHub Desktop avada.
5. Vali **Local path**, kuhu repo arvutis salvestada, näiteks
   `C:\Users\SinuNimi\Documents\GitHub`.
6. Vajuta **Clone**.

GitHub Desktop loob sinna tühja kausta:

```text
C:\Users\SinuNimi\Documents\GitHub\kodumedia
```

## 4. Paki GitHub-ready lähtekood lahti

1. Tee failil `kodumedia-0.8.56-github-ready.zip` paremklõps.
2. Vali **Extract All... / Ekstrakti kõik...**.
3. Ava lahtipakitud `kodumedia-0.8.56-github` kaust.
4. Seal peavad olema muu hulgas:
   - `.github`
   - `apps`
   - `docs`
   - `.gitignore`
   - `LICENSE`
   - `README.md`

Ära kasuta selleks valmis Windowsi programmi ZIP-i.

## 5. Kopeeri lähtekood kloonitud repo kausta

1. Märgi lahtipakitud `kodumedia-0.8.56-github` kaustas kogu sisu.
2. Kopeeri failid.
3. Ava GitHub Desktopi loodud tühi `kodumedia` kaust.
4. Kleebi failid sinna.

Kopeeri kausta **sisu**, mitte välimist `kodumedia-0.8.56-github` kausta ennast.
Õige tulemuse korral asub fail `README.md` otse kloonitud `kodumedia` kaustas.

## 6. Vaata muudatused GitHub Desktopis üle

Mine tagasi GitHub Desktopi. Vasakul peaks ilmuma failide loend.

Kontrolli, et loendis ei oleks:

- `.env`;
- `data` kausta;
- API-võtmeid või credentials/secrets faile;
- `node_modules` kausta;
- `tools/ffmpeg/bin` kausta;
- EXE- või ZIP-faile;
- isiklikke filme, seriaale, muusikat ega subtiitreid.

Minu koostatud GitHub-ready paketis on need juba eemaldatud ja ignoreeritud.

## 7. Tee esimene commit

GitHub Desktopi alumises vasakus nurgas:

1. Sisesta **Summary** väljale:

   ```text
   Initial public release v0.8.56
   ```

2. Description võib jääda tühjaks.
3. Vajuta sinist **Commit to main** nuppu.
4. Vajuta ülal **Push origin**.

Pärast Push origin toimingut on lähtekood avalikult aadressil
<https://github.com/mihkelwow/kodumedia>.

## 8. Kontrolli GitHubis

1. Ava repo veebileht uuesti.
2. Veendu, et KoduMedia logo ja README kuvatakse avalehel.
3. Ava **Actions**. Esimene automaattest käivitub ise.
4. Roheline linnuke tähendab, et kontroll läbis.
5. Ava **Settings → Code security and analysis** ning kontrolli secret scanning'u
   ja push protection'i olekut.
6. Ava **Settings → Security → Private vulnerability reporting** ning luba
   privaatne turvaraport, kui valik on saadaval.

## 9. Lisa About ja teemad

Repo avalehe paremas servas on **About**. Vajuta selle kõrval hammasratast.

Description:

```text
Self-hosted Windows media server for movies, TV series and music, with a React web interface, HLS transcoding, subtitles, metadata, AirPlay, Google Cast and DLNA.
```

Website jäta esialgu tühjaks.

Topics:

```text
media-server
self-hosted
go
react
typescript
hls
dlna
airplay
windows
```

Märgi vajaduse korral **Releases** ja **Packages** nähtavaks ning salvesta.

## 10. Loo KoduMedia 0.8.56 väljalase

1. Ava repo avalehel paremal **Releases**.
2. Vajuta **Draft a new release**.
3. Vajuta **Choose a tag**.
4. Kirjuta `v0.8.56` ja vali uue tag'i loomine harust `main`.
5. Release title:

   ```text
   KoduMedia 0.8.56
   ```

6. Kopeeri kirjeldus failist `docs/releases/v0.8.56.md`.
7. Lohista **Assets** alale:
   - `kodumedia-0.8.56-windows-x64.zip`;
   - `kodumedia-0.8.56-windows-x64.zip.sha256`.
8. Vajuta **Publish release**.

GitHub lisab automaatselt ka kaks arendajatele mõeldud Source code pakki.
Tavakasutaja peab valima Assets alt `windows-x64.zip` faili.

## 11. Mida kasutaja GitHubi lehel näeb

README alguses on:

- lühike selgitus, mis KoduMedia on;
- tähtsamad võimalused;
- turvahoiatus, et porti 8096 ei avata internetti;
- Windowsi allalaadimise ja käivitamise sammud;
- link täielikule esmase kasutuse juhendile;
- castimise, ehitamise ja litsentsi info.

Eestikeelne täielik kasutusjuhend asub failis
`docs/ESMANE-KASUTUS.et.md` ning ingliskeelne failis
`docs/GETTING-STARTED.md`.

## Edasised muudatused GitHub Desktopiga

Kui muudad lähtekoodi või dokumentatsiooni:

1. ava GitHub Desktop;
2. vaata vasakul muudatuste loend üle;
3. kirjuta Summary väljale lühike muudatuse kirjeldus;
4. vajuta **Commit to main**;
5. vajuta **Push origin**.

Uue versiooni Windowsi ZIP lisa alati uue GitHub Release'i Assets alla, mitte
lähtekoodifailide hulka.

## Kas kogu lähtekoodi saab laadida ainult brauseris?

Tehniliselt saab GitHubis valida **Add file → Upload files** ning lohistada
brauserisse faile või kaustu. GitHub lubab aga korraga kuni 100 faili ja
veebiupload ei rakenda `.gitattributes` reegleid. KoduMedia paketis on üle 400
faili, mistõttu GitHub Desktop on kindlam ja lihtsam viis.
