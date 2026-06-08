#!/usr/bin/env node
/**
 * 截取 README 用界面图。需先启动后端 (8080) 与前端 dev (5173)。
 * 用法：node scripts/capture-screenshots.mjs
 */
import { execFileSync } from 'node:child_process'
import { mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const outDir = join(root, 'docs/screenshots')
const base = process.env.SCREENSHOT_BASE ?? 'http://localhost:5173'
const chrome =
  process.env.CHROME_PATH ??
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'

const profile = JSON.stringify({ id: 'default', displayName: '默认用户' })
const seed = encodeURIComponent(profile)

const shots = [
  { file: 'home.png', path: `/?seedProfile=${seed}#/` },
  { file: 'graph-galaxy.png', path: `/?seedProfile=${seed}#/graph`, budget: 12000 },
  { file: 'graph-outline.png', path: `/?seedProfile=${seed}#/graph?view=outline`, budget: 8000 },
  { file: 'courses.png', path: `/?seedProfile=${seed}#/courses` },
  { file: 'import.png', path: `/?seedProfile=${seed}#/import` },
]

mkdirSync(outDir, { recursive: true })

for (const s of shots) {
  const out = join(outDir, s.file)
  execFileSync(
    chrome,
    [
      '--headless=new',
      '--disable-gpu',
      '--hide-scrollbars',
      '--window-size=1280,800',
      `--screenshot=${out}`,
      `--virtual-time-budget=${s.budget ?? 8000}`,
      `${base}${s.path}`,
    ],
    { stdio: 'inherit' }
  )
  console.log(`wrote ${out}`)
}
