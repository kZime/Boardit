import { describe, expect, it } from 'vitest'

import {
  extractFirstImageFromMarkdown,
  getCoverUrlFromContent,
  parseFrontmatterAndBody,
} from './markdownCover'

describe('parseFrontmatterAndBody', () => {
  it('parses scalar and array frontmatter while preserving the body', () => {
    const markdown = `---
cover: "https://example.com/cover.jpg"
tags: [ai, "notes"]
---
# Hello`

    expect(parseFrontmatterAndBody(markdown)).toEqual({
      frontmatter: {
        cover: 'https://example.com/cover.jpg',
        tags: ['ai', 'notes'],
      },
      body: '# Hello',
    })
  })

  it('treats an unclosed frontmatter block as normal markdown', () => {
    const markdown = '---\ntitle: Draft\n# Still content'

    expect(parseFrontmatterAndBody(markdown)).toEqual({
      frontmatter: {},
      body: markdown,
    })
  })
})

describe('cover extraction', () => {
  it('prefers a frontmatter cover over an image in the body', () => {
    const markdown = `---
cover: https://example.com/frontmatter.jpg
---
![fallback](https://example.com/body.jpg)`

    expect(getCoverUrlFromContent(markdown)).toBe(
      'https://example.com/frontmatter.jpg',
    )
  })

  it('falls back to Markdown and HTML images', () => {
    expect(extractFirstImageFromMarkdown('![cover](https://example.com/a.png)')).toBe(
      'https://example.com/a.png',
    )
    expect(extractFirstImageFromMarkdown('<img src="https://example.com/b.png">')).toBe(
      'https://example.com/b.png',
    )
  })

  it('returns null when no usable cover exists', () => {
    expect(getCoverUrlFromContent(undefined)).toBeNull()
    expect(getCoverUrlFromContent('# Text only')).toBeNull()
  })
})
