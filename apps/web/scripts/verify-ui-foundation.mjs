import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const read = (path) => readFileSync(resolve(root, path), 'utf8')
const failures = []

function sourceFiles(directory) {
  const files = []
  for (const entry of readdirSync(directory)) {
    const path = resolve(directory, entry)
    if (statSync(path).isDirectory()) files.push(...sourceFiles(path))
    else if (/\.(ts|tsx)$/.test(entry)) files.push(path)
  }
  return files
}
const assert = (condition, message) => { if (!condition) failures.push(message) }

const index = read('index.html')
const globalCss = read('src/design/global.css')
const tokensCss = read('src/design/tokens.css')
const primitivesCss = read('src/design/primitives.css')
const motionCss = read('src/design/motion.css')
const performanceCss = read('src/design/performance.css')
const interactionCss = read('src/design/interaction.css')
const mainSource = read('src/main.tsx')
const hero = read('src/features/media/FeaturedHero.tsx')
const ambientBackdrop = read('src/shared/components/AmbientBackdrop.tsx')
const staticBackdropRoute = read('src/app/staticBackdropRoute.ts')
const staticBackdrop = read('src/shared/components/StaticAppBackdrop.tsx')
const publicFiles = readdirSync(resolve(root, 'public'))
const removedRuntimeLayers = ['home-featured', 'initial-setup', 'series-themes']
const embeddedDist = resolve(root, '../server/internal/webui/dist')
const embeddedIndex = readFileSync(resolve(embeddedDist, 'index.html'), 'utf8')
const embeddedScriptMatch = embeddedIndex.match(/<script[^>]+src=["']\/([^"']+\.js)["']/)
const embeddedStylesheetMatch = embeddedIndex.match(/<link[^>]+href=["']\/([^"']+\.css)["']/)
const embeddedScriptPath = embeddedScriptMatch?.[1] ?? ''
const embeddedStylesheetPath = embeddedStylesheetMatch?.[1] ?? ''
const embeddedApp = embeddedScriptPath ? readFileSync(resolve(embeddedDist, embeddedScriptPath), 'utf8') : ''
const embeddedCss = embeddedStylesheetPath ? readFileSync(resolve(embeddedDist, embeddedStylesheetPath), 'utf8') : ''

assert(!index.includes('featured-hero-runtime'), 'the React source build must not load a duplicate hero runtime')
assert(!publicFiles.some((file) => file.startsWith('featured-hero-runtime')), 'duplicate hero runtime files still exist in public/')
assert(!embeddedIndex.includes('featured-hero-bootstrap'), 'embedded index must not load a second hero layer')
assert(!existsSync(resolve(embeddedDist, 'featured-hero-bootstrap.js')), 'legacy embedded hero bootstrap still exists')
assert(!existsSync(resolve(embeddedDist, 'featured-hero-bootstrap.css')), 'legacy embedded hero bootstrap CSS still exists')
assert(!existsSync(resolve(embeddedDist, 'featured-hero-runtime.js')), 'deprecated embedded hero runtime still exists')
assert(!existsSync(resolve(embeddedDist, 'featured-hero-runtime.css')), 'deprecated embedded hero runtime CSS still exists')
assert(Boolean(embeddedScriptPath), 'embedded index must reference its production JavaScript bundle')
assert(Boolean(embeddedStylesheetPath), 'embedded index must reference its production stylesheet')
assert(embeddedApp.length > 0, 'embedded production JavaScript bundle must not be empty')
assert(embeddedCss.includes('.ui-dialog-layer'), 'embedded stylesheet must include shared dialog primitives')
assert(embeddedApp.includes('static-app-backdrop'), 'embedded application bundle must mount the static collection backdrop')
assert(embeddedCss.includes('.static-app-backdrop'), 'embedded stylesheet must include the static collection backdrop')
assert(embeddedCss.includes('--focus-shadow-inset'), 'embedded stylesheet must include the clipping-safe focus token')
assert(embeddedCss.includes('touch-action:pan-x pan-y pinch-zoom'), 'embedded stylesheet must include native multi-axis touch scrolling')
assert(!embeddedApp.includes('featured-hero__dot-indicator'), 'embedded application must retain the protected direct hero-dot markup')

for (const name of removedRuntimeLayers) {
  assert(!index.includes(name), `index.html still loads the deprecated ${name} runtime layer`)
  assert(!publicFiles.some((file) => file.startsWith(name)), `public/${name} runtime files still exist`)
  assert(!embeddedIndex.includes(name), `embedded index still loads the deprecated ${name} runtime layer`)
  assert(!existsSync(resolve(embeddedDist, `${name}.js`)), `embedded ${name}.js runtime file still exists`)
  assert(!existsSync(resolve(embeddedDist, `${name}.css`)), `embedded ${name}.css runtime file still exists`)
}

assert(!/^:root\s*\{/m.test(globalCss), 'global.css must not define the root token set')
assert(tokensCss.includes('--ease-out: cubic-bezier(.23, 1, .32, 1)'), 'the shared strong ease-out token is missing')
assert(tokensCss.includes('--motion-panel: 240ms'), 'the shared panel duration is missing')
assert(tokensCss.includes('--motion-cinematic: 480ms'), 'the shared cinematic duration is missing')
assert(motionCss.includes('[data-pressable="true"]:active:not(:disabled)'), 'shared press feedback is missing')
assert(primitivesCss.includes('.ui-popover--bottom-start'), 'origin-aware popover placement styles are missing')
assert(!/transition\s*:\s*all\b/.test(`${tokensCss}\n${motionCss}\n${primitivesCss}\n${interactionCss}`), 'foundation styles must not use transition: all')
assert(mainSource.indexOf("import './design/interaction.css'") > mainSource.indexOf("import './design/performance.css'"), 'interaction safeguards must be the final stylesheet')
assert(!hero.includes('onFocusCapture={handleFocus}'), 'keyboard focus must not unmute the featured trailer')
assert(!hero.includes('keyboardActive'), 'featured trailer still derives audio from keyboard focus')
assert(hero.includes('loading="eager" fetchPriority="high"'), 'featured artwork must be requested with high priority')
assert(!hero.includes('onTrailerUnavailable'), 'trailer failures must never remove their hero slide')
assert(hero.includes("const legacyTrailerFailureCacheKey = 'kodumedia.trailer-failures.v1'"), 'the false-positive trailer cache migration is missing')
assert(hero.includes("const trailerFailureCacheKey = 'kodumedia.trailer-failures.v2'"), 'the corrected trailer failure cache is missing')
assert(hero.includes('const permanentTrailerErrorCodes = new Set([2, 100, 101, 150])'), 'only permanent video-specific YouTube errors may be cached')
assert(hero.includes('onError: ({ data }) =>'), 'YouTube error codes must be preserved for failure classification')
assert(!globalCss.includes('featured-hero-reveal'), 'featured hero must not replay a reveal animation on every homepage mount')
assert(
  /\.featured-hero__trailer\s*\{[^}]*transition:\s*opacity var\(--motion-cinematic\) var\(--ease-out\)/s.test(globalCss),
  'featured trailer must retain the established 480ms poster-to-video crossfade',
)
assert(hero.includes("import useEmblaCarousel from 'embla-carousel-react'"), 'featured hero must use the production touch-carousel engine')
assert(hero.includes('className="featured-hero__viewport" ref={carouselRef}'), 'featured hero must expose the carousel viewport directly to Embla')
assert(hero.includes('className="featured-hero__slides"'), 'featured hero must render adjacent full-width slides in one physical track')
assert(hero.includes("carouselApi.on('select', syncSelectedSlide)"), 'featured hero state must follow touch snap selection')
assert(!hero.includes('handlePointerDown'), 'featured hero must not regress to the synthetic pointer-only swipe recognizer')
assert(hero.includes('onTouchStart={(event: ReactTouchEvent<HTMLElement>) =>'), 'featured hero must retain a direct touch fallback across browser input stacks')
assert(hero.includes('const projectedX = deltaX + velocityX * heroGestureProjectionMs'), 'featured hero must project touch momentum before choosing the target slide')
assert(hero.includes('const distanceThreshold = Math.min(64, Math.max(44, width * 0.14))'), 'featured hero must accept a deliberate swipe without requiring a half-width drag')
assert(/\.featured-hero__slides\s*\{[^}]*display:\s*flex;[^}]*touch-action:\s*pan-y pinch-zoom;/s.test(globalCss), 'hero track must preserve native vertical scrolling while handling horizontal touch drag')
assert(/\.featured-hero__slide\s*\{[^}]*flex:\s*0 0 100%;[^}]*min-width:\s*0;/s.test(globalCss), 'hero carousel slides must each occupy one stable viewport width')
assert(/@media \(max-width: 640px\)[\s\S]*\.featured-hero__carousel-controls\s*\{[^}]*top:\s*auto;[^}]*right:\s*16px;[^}]*bottom:\s*16px;/s.test(globalCss), 'mobile hero arrows must stay in the bottom-right corner')
assert(/@media \(max-width: 640px\)[\s\S]*\.featured-hero__pagination\s*\{[^}]*display:\s*none;/s.test(globalCss), 'mobile hero must hide pagination dots and retain arrow-only navigation')
const homePageSource = read('src/pages/HomePage.tsx')
const libraryPageSource = read('src/pages/LibraryPage.tsx')
const musicPageSource = read('src/pages/MusicPage.tsx')
const appSource = read('src/app/App.tsx')
assert(homePageSource.includes("const HOME_CACHE_KEY = 'kodumedia.home.v2'"), 'homepage session cache is missing')
assert(homePageSource.includes('async function loadPrimaryHomeData()'), 'homepage hero data must load separately from secondary and category rows')
assert(homePageSource.includes('async function loadSecondaryHomeData()'), 'secondary homepage rows must not delay the hero')
assert(homePageSource.includes("readHomeCacheFrom(\'localStorage\')"), 'homepage cache must survive refreshes and browser restarts')
assert(homePageSource.includes('await preloadImage(firstBackground)'), 'first hero artwork must decode before the initial uncached hero is mounted')
assert(!homePageSource.includes('unavailableHeroIds'), 'homepage must not filter slides after trailer playback failures')
assert(globalCss.includes('.featured-hero__arrow {\n  display: inline-grid;\n  width: 34px;'), 'featured carousel arrows must retain the established compact 34px pointer visual')
assert(!hero.includes('featured-hero__dot-indicator'), 'featured carousel must retain the protected direct dot markup')
assert(globalCss.includes('.featured-hero__dot {\n  width: 7px;\n  height: 7px;'), 'inactive featured carousel dots must retain the established compact 7px geometry')
assert(globalCss.includes('.featured-hero__dot[aria-selected="true"] {\n  width: 22px;'), 'active featured carousel dot must retain the established 22px geometry')
assert(!globalCss.includes('.featured-hero__dot-indicator'), 'featured carousel must not reintroduce a larger dot wrapper')
assert(!globalCss.includes('.featured-hero__dot::before'), 'featured carousel must not use the oversized pseudo-element indicator variant')
assert(ambientBackdrop.includes('ambientLayerIndexes.map((index)'), 'ambient backdrop must keep exactly two persistent buffer layers mounted')
assert(ambientBackdrop.includes("isBase ? 'ambient-backdrop__layer--base'"), 'ambient backdrop must preserve a settled base layer during replacement')
assert(ambientBackdrop.includes("isIncoming ? 'ambient-backdrop__layer--incoming'"), 'ambient backdrop must stage replacements in the hidden incoming buffer')
assert(!performanceCss.includes('.ambient-backdrop--transitioning'), 'ambient layers must not be promoted or demoted during thumbnail hover')
assert(performanceCss.includes('transition: opacity 360ms var(--ease-in-out);'), 'ambient incoming buffer must use the soft 360ms opacity crossfade')
assert(performanceCss.includes('will-change: opacity;'), 'both persistent ambient buffers must retain stable compositor promotion')


const sourceText = sourceFiles(resolve(root, 'src')).map((path) => readFileSync(path, 'utf8')).join('\n')
assert(!sourceText.includes('new MutationObserver('), 'React source must not create UI through MutationObserver')
assert(!/\.innerHTML\s*=/.test(sourceText), 'React source must not create UI through innerHTML assignment')
assert(!/transition\s*:\s*all\b/.test(globalCss), 'global.css must not use transition: all')
const mediaCardSource = read('src/features/media/MediaCard.tsx')
assert(mediaCardSource.includes('transform: `scaleX('), 'media card progress must use transform-based rendering')
assert(!mediaCardSource.includes('style={{ width:'), 'media card progress must not animate or update layout width')
assert(mediaCardSource.includes('draggable={false}'), 'media card artwork must not hijack horizontal pointer gestures')
assert(tokensCss.includes('--focus-shadow: 0 0 0 2px var(--focus-ring)'), 'keyboard focus must use one restrained two-pixel ring')
assert(tokensCss.includes('--focus-shadow-inset: inset 0 0 0 2px var(--focus-ring)'), 'clipping-safe inset focus token is missing')
assert(/\.media-card:focus-visible\s*\{[^}]*box-shadow:\s*none;/s.test(interactionCss), 'media cards must suppress the duplicate full-card focus ring')
assert(/\.media-card:focus-visible \.media-card__visual\s*\{[^}]*box-shadow:\s*0 13px 34px/s.test(interactionCss), 'media card artwork must suppress the legacy outer focus shadow')
assert(/\.media-card:focus-visible \.media-card__visual::after\s*\{[^}]*var\(--focus-shadow-inset\)/s.test(interactionCss), 'poster focus must be drawn once inside the thumbnail')
assert(/\.featured-hero:focus-within\s*\{[^}]*box-shadow:\s*0 34px 90px/s.test(interactionCss), 'hero child focus must not frame the entire hero')
assert(interactionCss.includes('touch-action: pan-x pan-y pinch-zoom;'), 'horizontal collections must retain native touch panning and pinch zoom')
assert(interactionCss.includes('-webkit-overflow-scrolling: touch;'), 'native momentum scrolling safeguard is missing')
assert(interactionCss.includes('scroll-margin: var(--focus-gutter);'), 'directional focus must reserve a clipping-safe scroll gutter')
assert(!/@media \(pointer: coarse\)[\s\S]*\.featured-hero__(?:arrow|dot)[\s\S]*min-(?:width|height): var\(--tap-target\)/.test(interactionCss), 'coarse-pointer rules must never resize protected hero controls')
const directionalNavigationSource = read('src/shared/hooks/useDirectionalNavigation.ts')
assert(directionalNavigationSource.includes('event.defaultPrevented'), 'spatial navigation must respect component-owned arrow-key handling')
assert(directionalNavigationSource.includes('first.focus({ preventScroll: true })'), 'the first directional key press must establish focus predictably')
const castCarouselSource = read('src/features/media/CastCarousel.tsx')
assert(castCarouselSource.includes("event.pointerType !== 'mouse'"), 'touch cast scrolling must use native momentum instead of mouse drag emulation')
const mediaDetailsSource = read('src/features/media/MediaDetails.tsx')
const seriesDetailsSource = read('src/features/media/SeriesDetails.tsx')
const episodeCarouselSource = read('src/features/media/EpisodeCarousel.tsx')
const mediaPlayButtonSource = read('src/features/media/MediaPlayButton.tsx')
assert(mediaPlayButtonSource.includes('className={`media-play-button'), 'the shared media play button must own the primary playback style hook')
assert(hero.includes('<MediaPlayButton'), 'featured hero must use the shared media play action')
assert(mediaDetailsSource.includes('<MediaPlayButton'), 'movie and episode details must use the shared media play action')
assert(seriesDetailsSource.includes('<MediaPlayButton'), 'series details must use the shared media play action')
assert(/\.media-play-button\.button\s*\{[^}]*min-height:\s*48px;[^}]*border-radius:\s*var\(--radius-pill\);/s.test(globalCss), 'media play actions must share the accessible 48px pill geometry')
assert(episodeCarouselSource.includes('transform: `scaleX('), 'episode progress must use transform-based rendering')
assert(!episodeCarouselSource.includes('style={{ width:'), 'episode progress must not update layout width')
assert(!globalCss.includes('transition: background 150ms ease, padding 180ms ease'), 'episode rows must not animate layout padding')
const settingsPageSource = read('src/pages/SettingsPage.tsx')
const systemStatusSource = read('src/features/settings/SystemStatus.tsx')
const playerModalSource = read('src/features/player/PlayerModal.tsx')
const playerTrackMenuSource = read('src/features/player/PlayerTrackMenu.tsx')
const playerTimelineSource = read('src/features/player/PlayerTimeline.tsx')
const playerControlsSource = read('src/features/player/PlayerControls.tsx')
const playerPostPlaySource = read('src/features/player/PlayerPostPlay.tsx')
const musicPlayerSource = read('src/features/music/MusicPlayer.tsx')
const progressBarSource = read('src/shared/components/ProgressBar.tsx')
assert(settingsPageSource.includes("label: 'Server ja töövalmidus'"), 'settings navigation must group server readiness instead of exposing implementation details')
assert(settingsPageSource.includes("label: 'Meediakogud ja metaandmed'"), 'settings navigation must group libraries and metadata')
assert(settingsPageSource.includes("label: 'Taasesitus ja heli'"), 'settings navigation must group playback and audio')
const settingsNavigationOrder = ['#settings-overview', '#settings-metadata', '#settings-quality', '#settings-subtitles', '#settings-language', '#settings-update']
const settingsGroupOrder = ['id="settings-overview"', 'id="settings-metadata"', 'id="settings-quality"', 'id="settings-subtitles"', 'id="settings-language"', 'id="settings-update"']
assert(settingsNavigationOrder.every((item, index) => index === 0 || settingsPageSource.indexOf(item) > settingsPageSource.indexOf(settingsNavigationOrder[index - 1])), 'settings navigation order must match the task-oriented information architecture')
assert(settingsGroupOrder.every((item, index) => index === 0 || systemStatusSource.indexOf(item) > systemStatusSource.indexOf(settingsGroupOrder[index - 1])), 'settings content order must match the left navigation')
assert(globalCss.includes('KoduMedia 0.8.30 — restorative UI stabilization'), 'restorative UI stabilization styles are missing')
assert(globalCss.includes('KoduMedia 0.8.31 — detail continuity, usable cast scrolling and flash-free ambient art.'), '0.8.31 detail continuity styles are missing')
assert(globalCss.includes('KoduMedia 0.8.32 — responsive hover and scroll rendering'), '0.8.32 responsive hover and scroll styles are missing')
assert(existsSync(resolve(root, 'src/features/media/CastCarousel.tsx')), 'shared cast carousel is missing')
assert(appSource.includes('{staticBackdrop ? <StaticAppBackdrop /> : null}'), 'the application shell must mount the shared static backdrop on approved routes')
assert(staticBackdrop.includes('className="static-app-backdrop"'), 'the shared static backdrop component is missing')
assert(staticBackdropRoute.includes("route.name === 'home' || route.name === 'settings'"), 'home and settings must share the static backdrop')
assert(staticBackdropRoute.includes("new Set(['library-movies', 'library-series'])"), 'only the movie and series collection routes must opt into the static library backdrop')
assert(staticBackdropRoute.includes("route.name === 'music'") && staticBackdropRoute.includes('!route.artist && !route.albumId && !route.tracks'), 'only the music collection root must use the static backdrop')
assert(!homePageSource.includes('<AmbientBackdrop'), 'homepage must not decode or filter hover-driven ambient artwork')
assert(!homePageSource.includes('onPreview='), 'homepage rows must not schedule hidden hover backdrop state')
assert(!libraryPageSource.includes('<AmbientBackdrop'), 'movie and series collection pages must not decode or filter ambient artwork')
assert(musicPageSource.includes('{usesArtworkBackdrop ? <AmbientBackdrop'), 'music detail routes must retain their artwork backdrop')
assert(/\.library-page__header h1,[^}]*font-size:\s*clamp\(38px, 4\.4vw, 72px\)/s.test(globalCss), 'library title scale must remain available as the page-title reference')
assert(/\.music-page__hero h1\s*\{[^}]*font-size:\s*clamp\(38px, 4\.4vw, 72px\)/s.test(globalCss), 'music landing title must match the library page-title scale')
assert(/\.music-library-header h1\s*\{[^}]*font-size:\s*clamp\(42px, 4\.6vw, 58px\)/s.test(globalCss), 'all-tracks title must use the stabilized content-page scale')
assert(/\.music-all-tracks-entry__copy h2\s*\{[^}]*font-size:\s*clamp\(26px, 3vw, 40px\)/s.test(globalCss), 'music all-tracks entry must use the quieter section-title scale')
assert(!globalCss.includes('font-size: clamp(52px, 8vw, 112px);'), 'music pages must not reintroduce the oversized legacy title scale')
assert(performanceCss.includes('.static-app-backdrop {'), 'the lightweight static backdrop styles are missing')
assert(performanceCss.includes('.app-shell {\n  isolation: isolate;\n}'), 'the static backdrop must stay visible inside an isolated application stacking context')
assert(/\.static-app-backdrop\s*\{[^}]*background:/s.test(performanceCss), 'the static backdrop must be drawn with CSS gradients')
assert(!/\.static-app-backdrop\s*\{[^}]*(?:filter|transition|animation|will-change)\s*:/s.test(performanceCss), 'the static backdrop must not use filtered or animated rendering')
assert(mediaDetailsSource.includes('<div className="details__background"'), 'movie and episode details must retain their artwork background')
assert(seriesDetailsSource.includes('<div className="details__background"'), 'series details must retain their artwork background')
assert(performanceCss.includes('.details__background {\n  filter: blur(36px) saturate(1.08);'), 'detail views must retain the established blurred artwork treatment')
assert(globalCss.includes('.settings-subsection {\n  display: grid;'), 'settings subsections must use the stable stacked composition')
assert(globalCss.includes('.details__technical-band {\n  grid-template-columns: minmax(260px, 1.35fr)'), 'technical details must use content-aware columns')
assert(existsSync(resolve(root, 'src/features/settings/SettingsGroup.tsx')), 'settings must use the shared SettingsGroup composition')
assert(systemStatusSource.includes('<SettingsSaveBar'), 'settings must use the shared dirty-state save bar')
assert(systemStatusSource.includes('<Switch'), 'settings toggles must use the shared native Switch component')
assert(systemStatusSource.includes('<ProgressBar'), 'settings job progress must use the shared transform-based progress component')
assert(systemStatusSource.includes("title={tr('Täpsemad taasesituse seaded')}"), 'advanced playback settings must be disclosed progressively')
assert(systemStatusSource.includes("title={tr('Täpsemad tööriistateed')}"), 'advanced tool paths must be disclosed progressively')
assert(/\.system-settings-actions\s*\{[^}]*position:\s*sticky;[^}]*bottom:\s*14px;[^}]*grid-template-columns:\s*minmax\(220px, 1fr\) auto auto auto;/s.test(globalCss), 'dirty settings save bar must stay available on long forms')
assert(playerTrackMenuSource.includes('<Popover'), 'player track menus must use the shared Popover')
assert(playerTrackMenuSource.includes('<Menu'), 'player track menus must use the shared keyboard Menu')
assert(playerTimelineSource.includes('transform: `scaleX(${progress})`'), 'player progress must use transform-based rendering')
assert(!playerTimelineSource.includes('style={{ width:'), 'player timeline must not update layout width')
assert(playerModalSource.includes('usePlayerControlsVisibility'), 'player controls visibility must be isolated in a focused hook')
assert(playerModalSource.includes('<PlayerControls'), 'player transport and tools must be isolated from session orchestration')
assert(playerModalSource.includes('<PlayerPostPlay'), 'post-play UI must be isolated from the player controller')
assert(playerModalSource.includes('<PlayerSubtitleSearchPanel'), 'subtitle search UI must be isolated from the player controller')
assert(playerModalSource.includes('<PlayerDiagnostics'), 'player diagnostics must be isolated from transport controls')
assert(playerControlsSource.includes('inert={hidden'), 'hidden playback controls must be removed from the keyboard focus order')
assert(!playerPostPlaySource.includes('animationDelay'), 'post-play cards must not use staggered entrance delays')
assert((musicPlayerSource.match(/data-focusable="true"/g) || []).length >= 6, 'music transport, timeline, volume and details controls must participate in spatial navigation')
assert(progressBarSource.includes('transform: `scaleX('), 'shared settings progress must render through transform')
assert(!playerModalSource.includes("document.addEventListener('pointerdown'"), 'player must not duplicate shared popover outside-click handling')
assert(globalCss.includes('.player-control-button:focus-visible {') && globalCss.includes(`outline-offset: 2px;
  transform: none;`), 'keyboard focus must not move player controls')
assert(globalCss.includes('.player-next-up__timeline span') && globalCss.includes('transition: transform 500ms linear;'), 'next-up progress must animate through transform')

for (const file of [
  'src/shared/components/Dialog.tsx',
  'src/shared/components/Popover.tsx',
  'src/shared/components/Menu.tsx',
  'src/shared/components/IconButton.tsx',
  'src/shared/components/Field.tsx',
  'src/shared/components/Disclosure.tsx',
  'src/shared/components/Switch.tsx',
  'src/shared/components/ProgressBar.tsx',
  'src/features/settings/SettingsSaveBar.tsx',
  'src/features/player/PlayerControls.tsx',
  'src/features/player/PlayerDiagnostics.tsx',
  'src/features/player/PlayerPlaybackStatus.tsx',
  'src/features/player/PlayerPostPlay.tsx',
  'src/features/player/PlayerSubtitleSearchPanel.tsx',
  'src/features/player/PlayerTrackMenu.tsx',
  'src/features/player/PlayerTimeline.tsx',
  'src/features/player/playerLabels.ts',
  'src/features/player/usePlayerControlsVisibility.ts',
]) {
  assert(existsSync(resolve(root, file)), `${file} is missing`)
}

const css = `${tokensCss}\n${motionCss}\n${globalCss}\n${primitivesCss}\n${performanceCss}\n${interactionCss}`
const references = new Set([...css.matchAll(/var\(--([\w-]+)/g)].map((match) => match[1]))
const definitions = new Set([...css.matchAll(/--([\w-]+)\s*:/g)].map((match) => match[1]))
const missingTokens = [...references].filter((name) => !definitions.has(name)).sort()
assert(missingTokens.length === 0, `undefined CSS tokens: ${missingTokens.join(', ')}`)

if (failures.length > 0) {
  console.error('UI foundation verification failed:')
  for (const failure of failures) console.error(`- ${failure}`)
  process.exit(1)
}

console.log('UI foundation verification passed.')
