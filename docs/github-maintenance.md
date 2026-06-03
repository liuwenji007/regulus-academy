# GitHub 仓库维护配置指南

面向 **Regulus Academy 维护者**：在 GitHub 网页上如何配置分支保护、CI、容器镜像与 Release。路径以 `liuwenji007/regulus-academy` 为例，其他 fork 请替换仓库名。

---

## 1. 分支保护（Branch protection）

**路径：** 仓库 → **Settings** → **Branches** → **Add branch protection rule**

| 配置项 | 建议值 | 说明 |
|--------|--------|------|
| Branch name pattern | `main` | 只保护主分支 |
| **Require a pull request before merging** | ✅ 开启 | 禁止直接 push；维护者也走 PR |
| Require approvals | ❌ 关闭（solo）或 **1**（有协作者时） | 单人项目不必卡 approval |
| **Dismiss stale pull request approvals** | 可选 | 新 push 后需重新 approve |
| **Require status checks to pass before merging** | ✅ 开启 | 必须 CI 绿 |
| Status checks that are required | 勾选 **`test`**（CI job 名） | 与 [.github/workflows/ci.yml](../.github/workflows/ci.yml) 中 `jobs.test` 一致 |
| **Require branches to be up to date before merging** | 建议 ✅ | 避免基于过旧 main 的 PR 被合入 |
| **Do not allow bypassing the above settings** | solo 可 ❌（给自己留紧急通道）；团队 ✅ | 管理员是否可跳过 |
| **Restrict who can push to matching branches** | 可选 | 进一步限制 push 名单 |
| **Allow force pushes** | ❌ 关闭 | 禁止 force push `main` |
| **Allow deletions** | ❌ 关闭 | 禁止删 `main` |

配置完成后：所有人（含 Owner）向 `main` 的改动应通过 **PR + 绿色 CI** 合并。

**本地习惯：**

```bash
git fetch origin
git checkout -b feat/my-change origin/main
# … 改代码 …
git push -u origin feat/my-change
gh pr create --base main --fill
```

---

## 2. Actions 与 CI

**路径：** 仓库 → **Actions**

本仓库有两个 workflow：

| Workflow | 文件 | 触发 | 作用 |
|----------|------|------|------|
| **CI** | `.github/workflows/ci.yml` | push/PR 到 `main` | `go test ./...` + 前端 tsc/build |
| **Docker Publish** | `.github/workflows/docker-publish.yml` | push 到 `main`、tag `v*`、手动 | 构建并 push 到 GHCR |

### 2.1 查看 PR 是否可合

PR 页面底部应显示 **All checks have passed**。若失败，点进 **CI** 日志排查。

### 2.2 手动触发镜像构建

**Actions** → **Docker Publish** → **Run workflow** → 选 `main` → Run。

用于：首次配置 GHCR、main 上 workflow 失败后补发镜像。

### 2.3 Fork PR 的 Actions

默认 fork 来的 PR **不会**跑 write 权限的 workflow（如 Docker Publish）。CI（test）在 Settings → Actions → General 里可设为对 fork PR 开放 **Read and write** 或保持默认只读；**Docker Publish 仅应在 merge 到 upstream `main` 后执行**，无需改。

---

## 3. GitHub Container Registry（GHCR）

镜像地址：`ghcr.io/liuwenji007/regulus-academy:latest`

**路径：** 仓库 → 右侧 **Packages**（或 `https://github.com/users/liuwenji007/packages`）

### 3.1 让一键安装脚本能匿名 pull

`scripts/install.sh` 默认 **不** `docker login`。若 pull 报 401/403：

1. 打开 Package **regulus-academy**
2. **Package settings** → **Change visibility** → **Public**

或文档中说明用户需：

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

（Personal Access Token 需 `read:packages`）

### 3.2 镜像 tag 规则

| 事件 | 产生的 tag |
|------|------------|
| push `main` | `latest`、`<git-sha>` |
| push tag `v1.2.3` | `v1.2.3`、`latest`（见 metadata-action 配置） |
| workflow_dispatch | `latest` |

用户可通过环境变量覆盖：

```bash
REGULUS_IMAGE=ghcr.io/liuwenji007/regulus-academy:v1.2.3 bash scripts/install.sh
```

---

## 4. Pull Request 与合并策略

**路径：** 仓库 → **Settings** → **General** → **Pull Requests**

| 配置项 | 建议 |
|--------|------|
| **Allow squash merging** | ✅ 开启（默认合并方式，历史干净） |
| Allow merge commits | 可选 ❌ |
| Allow rebase merging | 可选 ❌ |
| **Automatically delete head branches** | ✅ 建议开启 |

合并 PR 时选 **Squash and merge**，标题用清晰的一句（如 `feat: 预构建镜像安装与节点 requires`）。

---

## 5. 发布 Release（可选）

merge 大功能后：

```bash
git checkout main && git pull
git tag -a v0.2.0 -m "v0.2.0: 预构建镜像 + 学习前置 requires"
git push origin v0.2.0
```

**路径：** 仓库 → **Releases** → **Draft a new release**

- Choose tag：`v0.2.0`
- 标题：`v0.2.0`
- 说明：用户可见变更（安装变快、图谱前置依赖等）
- 可勾选 “Set as the latest release”

push tag 会触发 **Docker Publish**，镜像带 `v0.2.0` tag。

---

## 6. Security 与 Secrets

### 6.1 在哪里开 Dependabot / Secret scanning？

GitHub 已把 **「Code security and analysis」** 并入 **Advanced Security**，按下面顺序找：

1. 打开仓库：`https://github.com/liuwenji007/regulus-academy`
2. 顶部 **Settings**（需仓库 Admin 权限）
3. 左侧边栏找到 **Security** 分组（可能在 **General** 下面）
4. 点击 **Advanced Security**（中文界面可能显示「高级安全」）

**直达链接（替换为你的仓库）：**

```
https://github.com/liuwenji007/regulus-academy/settings/security_analysis
```

在 **Advanced Security** 页面可开关：

| 功能 | 公开仓库 | 说明 |
|------|----------|------|
| **Dependabot alerts** | 建议 ✅ Enable | 依赖漏洞告警；Go/npm 有 lock 文件时才有意义 |
| **Dependabot security updates** | 可选 ✅ | 自动开 PR 升级有漏洞的依赖 |
| **Secret Protection** / **Secret scanning** | 建议 ✅ Enable | 扫描误提交的 API Key |
| **Push protection** | 可选 ✅ | push 时拦截明文密钥（公开库免费） |

**若 Settings 里完全没有 Security / Advanced Security：**

- 确认你是 **仓库 Owner** 或 Admin（Collaborator 的 write 权限不够改这项）
- 确认打开的是 **GitHub.com** 上的仓库，不是仅本地 clone
- 到 **个人账号** 统一开：`https://github.com/settings/security_analysis` → 对 **所有新仓库默认启用** Dependabot alerts

**查看已产生的告警（不是 Settings）：**

仓库顶栏 **Security** 标签 → **Dependabot alerts** / **Secret scanning alerts**

### 6.2 Actions Secrets

**路径：** 仓库 → **Settings** → **Secrets and variables** → **Actions**

| 类型 | 本仓库 |
|------|--------|
| Repository secrets | 通常 **不需要** 额外 secret；`GITHUB_TOKEN` 由 Actions 自动注入用于 push GHCR |

**漏洞报告（不是 Settings 里的开关）：** 见 [SECURITY.md](../SECURITY.md) — **不要**公开 Issue，用仓库顶栏 **Security** → **Advisories** → **New draft security advisory**。

---

## 7. Issue 与 PR 模板

已配置：

- [.github/pull_request_template.md](../.github/pull_request_template.md)
- [.github/ISSUE_TEMPLATE/bug_report.yml](../.github/ISSUE_TEMPLATE/bug_report.yml)
- [.github/ISSUE_TEMPLATE/feature_request.yml](../.github/ISSUE_TEMPLATE/feature_request.yml)

**路径：** **Settings** → **General** → **Features** 确保 Issues 开启。

---

## 8. Merge 后维护者 Checklist

每次 PR 合入 `main` 后：

- [ ] **Actions**：CI ✅、Docker Publish ✅
- [ ] `docker pull ghcr.io/liuwenji007/regulus-academy:latest` 成功（公开包）
- [ ] 本地 `git pull origin main`；若自用 Docker：`docker compose -f docker-compose.image.yml pull && up -d`
- [ ] 大版本：打 tag + Release 说明
- [ ] 用户可见行为变更：必要时更新 README / CONTRIBUTING

---

## 9. 常见问题

**Q: PR 合了但 install.sh 仍很慢？**  
A: Docker Publish 失败或 GHCR 包为私有。查 Actions 日志与 Package visibility。

**Q: 能否 hotfix 直接 push main？**  
A: 文档约定不走 direct push。开 `fix/hotfix-*` 分支，PR 自 merge，通常 2 分钟。

**Q: CI 在 PR 上红但本地过？**  
A: 看是否缺 `pnpm install`、Go 版本与 `go.mod` 不一致；PR 需基于最新 `main`。

---

> 协作约定摘要见 [CONTRIBUTING.md](../CONTRIBUTING.md) 的「分支与工作流」一节。
