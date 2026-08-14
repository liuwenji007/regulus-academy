# Regulus Academy — an AI coach for fragmented learning

[中文](./README.md) | **English**

![Banner](./docs/banner.png)

![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

> In the gaps of a busy day: explain → practice → feedback → light up → keep knowledge you can take with you.
>
> **A bounded knowledge map · A coach that follows through until you finish · The more you learn, the better it knows you**

The app UI and docs site are currently **Chinese**. This page is a storefront for international visitors; the product itself is Chinese-first.

**Shipped:** online Demo · knowledge graph · coach loop · profile-aware teaching · action assistant · select-to-ask tutor · Obsidian export  
**Planned:** review flashcards · proactive daily push from the agent (full list in the [docs](https://regulus-academy-docs.vercel.app))

### Try it

| | Link |
|---|------|
| **Online Demo** | https://demo.awoshuile.cn |
| **Docs** (Chinese) | https://regulus-academy-docs.vercel.app |
| **GitHub** | https://github.com/liuwenji007/regulus-academy |

- **Demo**: no API key required up front (shared daily quota; when it runs out, paste your own key and continue)
- **Self-host / long-term use**: one OpenAI-compatible key is enough; data stays on your machine. See [self-hosting](https://regulus-academy-docs.vercel.app/guide/self-host)

---

## How it differs

Compared with “learn by chatting with ChatGPT / Claude”:

1. **A bounded knowledge map, not an endless chat** — each node pins down what to teach, what people usually get wrong, and what is out of scope
2. **A coach that follows through until you light the node up, not a teacher who explains and leaves** — explain → practice → grade → light up
3. **The more you learn, the better it knows you** — explanations are cut to your background: skip what you already know, drill what is weak

---

## What you walk away with

| Outcome | What it means |
|------|------|
| **Nodes you can light up** | Explain–practice–grade loop; familiar / mastery layers include applied questions and a mastery check |
| **A map with edges** | Intro / familiar / mastery layers; a multi-domain graph that shows progress |
| **Something you can take with you** | Export a Domain pack, a Coach Skill, and Obsidian notes |
| **Ask anytime** | Select a term while listening; gaps are saved and feed later recommendations and teaching |

Not sure where to start? Use the [built-in catalog](https://regulus-academy-docs.vercel.app/guide/features#建课与导入). Rhythm a mess? Try the [action assistant](https://regulus-academy-docs.vercel.app/guide/action-assistant). Course quality? [Audit and improve](https://regulus-academy-docs.vercel.app/guide/course-audit).

---

## Get started

### A. Online Demo (recommended, no key)

Open [demo.awoshuile.cn](https://demo.awoshuile.cn) → create a profile → type a topic or start from the catalog → pick a node and practice until it lights up.

Quota and limits: [cloud demo](https://regulus-academy-docs.vercel.app/guide/cloud-demo).

### B. Docker on your machine (data stays local; needs a key)

Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) first.

**One-liner** (clone, pull image, start):

```bash
curl -fsSL https://raw.githubusercontent.com/liuwenji007/regulus-academy/main/scripts/install.sh | bash
```

**Or three manual steps** (you control the directory):

```bash
# 1. Clone
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy

# 2. Configure the key (edit .env, set LLM_API_KEY; DeepSeek / OpenAI / any compatible API)
cp .env.example .env

# 3. Start (pulls the prebuilt image; about 30 seconds to 2 minutes)
docker compose -f docker-compose.image.yml up -d
```

Open **http://localhost:8080** and type something like “Go concurrency” to start (if 8080 is taken, set `HOST_PORT` in `.env`). To update, re-run the one-liner.

<details>

<summary><strong>C. Run from source (no Docker)</strong></summary>

> Backend is Go, frontend is Vite/Node. You need both runtimes and two processes.

```bash
git clone https://github.com/liuwenji007/regulus-academy.git
cd regulus-academy
cp .env.example .env            # set LLM_API_KEY

# Terminal 1: Go backend
go run ./cmd/server

# Terminal 2: frontend (Vite dev)
cd web && pnpm install && pnpm dev
```

Open **http://localhost:5173** (backend defaults to 8080; the Vite dev server already proxies to it).

</details>

Also available: download a **Coach Skill** from the home page (install it in Cursor or similar); self-hosting can connect **IM** (Telegram / DingTalk / Feishu / WeCom; some need public HTTPS). Details: [Coach Skill](https://regulus-academy-docs.vercel.app/guide/agent-offline) · [IM channels](https://regulus-academy-docs.vercel.app/guide/im) · [self-hosting](https://regulus-academy-docs.vercel.app/guide/self-host) · [environment variables](https://regulus-academy-docs.vercel.app/reference/env).

Stack: Go + SQLite + an OpenAI-compatible API. No Embedding / RAG setup required.

---

## What it looks like

| AI coach · practice feedback | Knowledge graph · Xuan paper |
|:---:|:---:|
| <img src="./docs/screenshots/coach-exercise.png" width="320" alt="Coach practice and grading" /> | <img src="./docs/screenshots/graph-paper.png" width="320" alt="Knowledge graph · Xuan paper" /> |

<details>

<summary>More screenshots: entry and path / import a course / select-to-ask tutor / Cloud demo</summary>

### Entry and learning path

| Start learning | Course detail | My courses |
|:---:|:---:|:---:|
| <img src="./docs/screenshots/home.png" width="260" alt="Home" /> | <img src="./docs/screenshots/tree.png" width="260" alt="Course detail" /> | <img src="./docs/screenshots/courses.png" width="260" alt="My courses" /> |

### Build a course and the graph

| Import from PDF / URL | Graph · night sky | Graph · outline |
|:---:|:---:|:---:|
| <img src="./docs/screenshots/import.png" width="260" alt="Import a course" /> | <img src="./docs/screenshots/graph-sky.png" width="260" alt="Knowledge graph · night sky" /> | <img src="./docs/screenshots/graph-outline.png" width="260" alt="Knowledge graph · outline" /> |

### Select-to-ask tutor

While listening, select a term and ask “what is this / how is it pronounced / go deeper” without breaking the main thread. Terms and gaps are saved and feed the sidebar’s “today’s recommendation”.

- Docs: [select-to-ask tutor](https://regulus-academy-docs.vercel.app/guide/aside-assistant)
- Demo assets: after recording, put them in `docs/screenshots/aside-selection.gif` (selection) and `aside-panel.png` (term panel); steps in [`docs/screenshots/README.md`](./docs/screenshots/README.md#划词助教截图与动图)

### Cloud demo

| Cloud home | Create a profile | Settings |
|:---:|:---:|:---:|
| <img src="./docs/screenshots/cloud-home.png" width="260" alt="Cloud home" /> | <img src="./docs/screenshots/cloud-profile.png" width="260" alt="Profile picker" /> | <img src="./docs/screenshots/cloud-settings.png" width="260" alt="Cloud settings" /> |

</details>

Full gallery: [screenshots](https://regulus-academy-docs.vercel.app/guide/screenshots).

---

## Why this exists

I am a mid-career engineer with a day job.

Tech moves fast. Work takes most of my energy; evenings leave only scraps of time. Anxiety is the default; learning is one of the few things that eases it. I have bought video courses — 48 lessons to “complete the track,” I watched 3. I have opened textbooks, skimmed the table of contents, and put them down. I have learned by chatting with AI; the knowledge never quite became a system, and after the chat I was left with some notes and not much else.

Worse: pick up a new stack and you cannot tell what “I know this” even means. Read the docs? Ship a demo? Run it in production? That blur turns into fear, and fear into not starting.

At the end of 2025 I decided to build a tool for this. I do not need a thorough lecturer. I need a coach — one that knows where I am, remembers what I already know, and corrects only the next movement that matters, so that in about 15 minutes I can finish one measurable step. I need a little discomfort to leave the comfort zone and start a field I used to avoid. Learning should not have to hurt.

I spent real time with [OpenMAIC](https://github.com/THU-MAIC/OpenMAIC) (structured presentation and multi-agent ideas) and [DeepTutor](https://github.com/HKUDS/DeepTutor) (practice generation and RAG). They are excellent; their classroom pace or setup cost did not fit fragmented time. So I borrowed the good ideas and made a lighter version: predefined boundaries instead of RAG, and one closed loop for one measurable gain:

- **The knowledge tree has clear edges.** Intro / familiar / mastery each mean something; when you finish, you know, and you can say it in an interview
- **The more you learn, the better it knows you.** The profile updates as you talk: courses are cut to you, explanations skip the redundant, depth grows from what you already hold, practice hits weak spots only
- **Setup stays small.** After local install you only need one model key (DeepSeek recommended), then you can stay in the conversation and the tree
- **Each node stands alone.** A spare 15 minutes can finish a level; you are not stuck forever in the middle of a course
- **Missed questions come back quietly.** Not “do this again,” but a different angle next time
- **Progress can follow you across clients** (self-hosted): start on the web, continue in IM

One API key is enough. Self-hosted, your data lives on your machine. If it runs, you can learn.

**This project is a gift to myself, and to anyone else who wants to keep growing in the scraps of a busy day.**

Design principles: [DESIGN.md](./DESIGN.md) (Chinese). A shorter version on the docs site: [Why Regulus](https://regulus-academy-docs.vercel.app/guide/why-regulus).

---

## Contribute

- Improve lighting-up or node teaching quality → [teaching quality](https://regulus-academy-docs.vercel.app/guide/contributing-teaching)
- Contribute a knowledge domain (YAML that defines node boundaries) → [CONTRIBUTING.md](./CONTRIBUTING.md)
- Try it and tell us → [`[体验]` issue](https://github.com/liuwenji007/regulus-academy/issues/new?template=experience_feedback.yml)

English issues and PRs are welcome; a one-line Chinese summary helps the maintainer file them. See [CONTRIBUTING.md](./CONTRIBUTING.md).

**A star is the simplest support. Issues are the best conversation.**

---

## License

Apache 2.0.

Self-host: learning data stays in local SQLite. Cloud Demo: a shared trial environment — do not put sensitive content in it; when quota runs out you can paste your own key and continue.

Report vulnerabilities in [SECURITY.md](./SECURITY.md).
