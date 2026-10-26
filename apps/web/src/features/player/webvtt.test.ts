import { describe, expect, it } from 'vitest'
import { activeSubtitleCues, normalizeCueText, parseWebVtt, parseWebVttTimestamp } from './webvtt'

describe('WebVTT subtitles', () => {
  it('parses hours, milliseconds and comma timestamps', () => {
    expect(parseWebVttTimestamp('01:02:03.450')).toBe(3_723_450)
    expect(parseWebVttTimestamp('02:03,45')).toBe(123_450)
  })

  it('parses unicode cues and removes WebVTT/ASS presentation tags safely', () => {
    const cues = parseWebVtt('\ufeffWEBVTT\r\n\r\n1\r\n00:00:01.000 --> 00:00:04.000 align:center\r\n<i>Tere, maailm!</i><br>{\\an8}ÕÄÖÜ šž\r\n\r\n')
    expect(cues).toEqual([{ id: '1', startMs: 1000, endMs: 4000, text: 'Tere, maailm!\nÕÄÖÜ šž' }])
  })

  it('returns every overlapping cue at the source-media position', () => {
    const cues = parseWebVtt('WEBVTT\n\n00:01:00.000 --> 00:01:05.000\nEsimene\n\n00:01:02.000 --> 00:01:04.000\nTeine\n')
    expect(activeSubtitleCues(cues, 62_500).map((cue) => cue.text)).toEqual(['Esimene', 'Teine'])
    expect(activeSubtitleCues(cues, 66_000)).toEqual([])
  })

  it('decodes numeric and common HTML entities without interpreting markup', () => {
    expect(normalizeCueText('&lt;test&gt; &amp; &#352; &#x17e;')).toBe('<test> & Š ž')
  })
})
