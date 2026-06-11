#!/usr/bin/env node
/**
 * 将仓库 docs/screenshots 与 banner 同步到 VitePress public/，供文档站引用。
 */
import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const docsRoot = join(here, '..')
const repoRoot = join(docsRoot, '..', '..')
const publicDir = join(docsRoot, 'public')
const screenshotsSrc = join(repoRoot, 'docs', 'screenshots')
const screenshotsDest = join(publicDir, 'screenshots')
const bannerSrc = join(repoRoot, 'docs', 'banner.png')
const bannerDest = join(publicDir, 'banner.png')

mkdirSync(publicDir, { recursive: true })

if (existsSync(screenshotsSrc)) {
  rmSync(screenshotsDest, { recursive: true, force: true })
  cpSync(screenshotsSrc, screenshotsDest, { recursive: true })
  console.log(`synced ${screenshotsSrc} → ${screenshotsDest}`)
} else {
  console.warn(`screenshots source missing: ${screenshotsSrc}`)
}

if (existsSync(bannerSrc)) {
  cpSync(bannerSrc, bannerDest)
  console.log(`synced ${bannerSrc} → ${bannerDest}`)
}
