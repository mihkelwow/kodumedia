import { readFileSync, readdirSync, statSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const sourceCatalog = JSON.parse(readFileSync(resolve(root, 'src/i18n/locales/et-EE.json'), 'utf8'))
const englishCatalog = JSON.parse(readFileSync(resolve(root, 'src/i18n/locales/en-US.json'), 'utf8'))
const sourceMessages = sourceCatalog.messages ?? {}
const englishMessages = englishCatalog.messages ?? {}
const failures = []

const sourceKeys = Object.keys(sourceMessages).sort()
const englishKeys = Object.keys(englishMessages).sort()
if (JSON.stringify(sourceKeys) !== JSON.stringify(englishKeys)) {
  const missing = sourceKeys.filter((key) => !(key in englishMessages))
  const extra = englishKeys.filter((key) => !(key in sourceMessages))
  if (missing.length) failures.push(`English catalog is missing: ${missing.join(' | ')}`)
  if (extra.length) failures.push(`English catalog has extra keys: ${extra.join(' | ')}`)
}

const placeholders = (value) => [...value.matchAll(/\{\{\s*([\w.-]+)\s*\}\}/g)].map((match) => match[1]).sort()
for (const key of sourceKeys) {
  const translated = englishMessages[key]
  if (typeof translated !== 'string' || translated.trim() === '') {
    failures.push(`English translation is empty: ${key}`)
    continue
  }
  if (JSON.stringify(placeholders(key)) !== JSON.stringify(placeholders(translated))) {
    failures.push(`English translation changes placeholders: ${key}`)
  }
  if (/[õäöüÕÄÖÜ]/.test(translated)) {
    failures.push(`English translation still contains Estonian-specific characters: ${key}`)
  }
}

function sourceFiles(directory) {
  const files = []
  for (const entry of readdirSync(directory)) {
    const path = resolve(directory, entry)
    if (statSync(path).isDirectory()) files.push(...sourceFiles(path))
    else if (/\.(ts|tsx)$/.test(entry) && !/\.test\.(ts|tsx)$/.test(entry)) files.push(path)
  }
  return files
}

const directEstonianPatterns = [
  /throw\s+new\s+Error\(\s*(['"`])[^\n]*[õäöüÕÄÖÜ][^\n]*\1\s*\)/g,
  /set(?:Error|Message)\(\s*(['"`])[^\n]*[õäöüÕÄÖÜ][^\n]*\1\s*\)/g,
  /\b(?:aria-label|title|placeholder|alt)=['"][^'"]*[õäöüÕÄÖÜ][^'"]*['"]/g,
  />[^<{\n]*[õäöüÕÄÖÜ][^<{\n]*</g,
]

for (const path of sourceFiles(resolve(root, 'src'))) {
  if (path.endsWith('/i18n/i18n.ts')) continue
  const source = readFileSync(path, 'utf8')
  for (const pattern of directEstonianPatterns) {
    for (const match of source.matchAll(pattern)) {
      const line = source.slice(0, match.index).split('\n').length
      failures.push(`Untranslated Estonian UI text: ${path.slice(root.length + 1)}:${line}`)
    }
  }
}

if (failures.length) {
  console.error('Localization verification failed:')
  for (const failure of failures) console.error(`- ${failure}`)
  process.exit(1)
}

console.log(`Localization verification passed: ${sourceKeys.length} Estonian and English messages.`)
