#!/usr/bin/env node
/**
 * 截取 README / 文档站用界面图。需先启动后端 (8080) 与前端 dev (5173)。
 *
 * 用法：
 *   node scripts/capture-screenshots.mjs
 *   SCREENSHOT_MODE=cloud node scripts/capture-screenshots.mjs
 *   DOMAIN_ID=xxx SESSION_ID=yyy node scripts/capture-screenshots.mjs
 */
import { execFileSync } from 'node:child_process'
import { mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const outDir = join(root, 'docs/screenshots')
const base = process.env.SCREENSHOT_BASE ?? 'http://localhost:5173'
const mode = process.env.SCREENSHOT_MODE ?? 'default'
/** all | default | cloud — 仅截取某一类页面 */
const only = process.env.SCREENSHOT_ONLY ?? 'all'
const domainId = process.env.DOMAIN_ID
const sessionId = process.env.SESSION_ID
const chrome =
  process.env.CHROME_PATH ??
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'

const profile = JSON.stringify({
  id: 'default',
  displayName: '默认用户',
  onboardedAt: '2026-01-01T00:00:00.000Z',
})
const seed = encodeURIComponent(profile)
const seedQuery = `/?seedProfile=${seed}`

const defaultShots = [
  { file: 'home.png', path: `${seedQuery}#/` },
  { file: 'graph-galaxy.png', path: `${seedQuery}#/graph`, budget: 12000 },
  { file: 'graph-outline.png', path: `${seedQuery}#/graph?view=outline`, budget: 8000 },
  { file: 'courses.png', path: `${seedQuery}#/courses` },
  { file: 'import.png', path: `${seedQuery}#/import` },
]

const cloudShots = [
  { file: 'cloud-home.png', path: `${seedQuery}#/`, budget: 25000 },
  { file: 'cloud-profile.png', path: '/#/', budget: 25000 },
  { file: 'cloud-settings.png', path: `${seedQuery}#/settings`, budget: 25000 },
]

function optionalShots() {
  const shots = []
  if (domainId) {
    shots.push({
      file: 'tree.png',
      path: `${seedQuery}#/tree/${domainId}`,
      budget: 10000,
    })
    shots.push({
      file: 'tree-extend.png',
      path: `${seedQuery}#/tree/${domainId}`,
      budget: 10000,
    })
  } else {
    console.warn('跳过 tree.png / tree-extend.png：请设置 DOMAIN_ID（完成度 ≥80% 的课程 ID）')
  }
  if (sessionId) {
    shots.push({
      file: 'coach-exercise.png',
      path: `${seedQuery}#/coach/${sessionId}`,
      budget: 10000,
    })
  } else {
    console.warn('跳过 coach-exercise.png：请设置 SESSION_ID')
  }
  return shots
}

const shots = [
  ...(only === 'all' || only === 'default' ? defaultShots : []),
  ...(mode === 'cloud' && (only === 'all' || only === 'cloud') ? cloudShots : []),
  ...(only === 'all' || only === 'default' ? optionalShots() : []),
]

mkdirSync(outDir, { recursive: true })

function capture(s) {
  const out = join(outDir, s.file)
  execFileSync(
    chrome,
    [
      '--headless=new',
      '--disable-gpu',
      '--hide-scrollbars',
      '--window-size=1280,800',
      `--screenshot=${out}`,
      `--virtual-time-budget=${s.budget ?? 15000}`,
      `${base}${s.path}`,
    ],
    { stdio: 'inherit' }
  )
  console.log(`wrote ${out}`)
}

for (const s of shots) {
  capture(s)
}
