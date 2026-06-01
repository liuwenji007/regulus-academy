# 安全策略

## 支持的版本

| 版本 | 支持状态 |
|------|----------|
| `main` 分支最新代码 | ✅ 积极维护 |
| 已发布 Tag | ✅ 仅安全修复 |

## 报告漏洞

Regulus Academy 是本地优先的应用：学习进度与 SQLite 数据默认留在你的机器上，LLM 请求发往你自己配置的 API。

如果你发现**可被利用的安全问题**（例如未授权访问本地 API、路径穿越、SQL 注入、IM Webhook 伪造等），请**不要**公开 Issue，优先私下联系维护者：

- GitHub：[Security Advisories](https://github.com/liuwenji007/regulus-academy/security/advisories/new)（推荐）
- 或在私有渠道联系仓库 Owner

请在报告中尽量包含：

1. 问题描述与影响范围
2. 复现步骤或 PoC
3. 受影响版本 / 提交
4. 你的环境（OS、部署方式：源码 / Docker）

我们会在确认后尽快回复，并在修复发布后致谢（如你希望署名）。

## 不在范围内的报告

以下通常**不视为**安全漏洞，可直接提 Issue：

- 未配置 `LLM_API_KEY` 导致功能不可用
- 本地单用户场景下未启用认证（当前设计为个人/小团队本地部署）
- 用户自行泄露 `.env` 或 IM Bot Token
- 对第三方 LLM 提供商本身的攻击面

## 安全使用建议

- 不要将实例无防护地暴露到公网；若必须暴露，请在前置反向代理上限制访问
- `.env` 与 SQLite 数据库文件不要提交到 Git
- IM Gateway 凭证仅保存在本机，定期轮换 Bot Token
- 生产环境使用 `FEISHU_ALLOWED_USERS` 等配置限制可绑定用户
