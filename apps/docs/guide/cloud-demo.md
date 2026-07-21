# 在线体验版（Cloud Demo）

Cloud 版由维护者提供公网实例，降低试用门槛。第一次使用建议配合 [快速上手](./quick-start.md) 阅读。

**入口**：[demo.awoshuile.cn](http://demo.awoshuile.cn)

## 快速试用

1. 打开在线 Demo，创建学习角色（输入昵称）
2. 在首页输入学习主题（如「Go 并发」），或打开 `#/catalog` 从内置课目录开练
3. 选节点开始 AI 教练对话；完成练习后节点点亮
4. 可选：侧栏打开行动助手，梳理今日学习后再开练
5. 额度用尽时按提示填写自己的 LLM Key（BYOK）继续使用

![角色创建](/screenshots/cloud-profile.png)

## 特性

- 无需 Docker / API Key 即可开始（平台提供每日免费额度：**30 条教练消息** + **3 次自定义建课**）
- 命中内置 Skill 的主题（如「Go 并发」）建课**不计入** 3 次建课额度
- 额度用尽后可 [填写自己的 LLM Key](http://demo.awoshuile.cn) 继续使用（BYOK）
- 首页展示共学人数与近 7 天活跃统计
- 内置课目录、行动助手 / 学习捷径、纵深扩展、课程体检、Coach Skill / Domain 包与学习笔记导出等核心学习与沉淀功能可用

![Cloud 首页](/screenshots/cloud-home.png)

## Cloud vs 自托管

| | 在线体验版 | 自托管 |
|---|-----------|--------|
| 数据位置 | 共享公网 Demo 实例 | 本机 SQLite |
| 用户隔离 | 浏览器本地 UUID，非强多租户 | 单用户 / 多角色 |
| 日配额 | 平台 Key：消息 30/天 + 建课 3/天（Skill 快路径豁免） | 无限制（用自己的 Key） |
| IM 机器人 | ❌ 未开放 | ✅ Telegram / 钉钉 / 飞书 |
| 管理员 | `#/admin` + `ADMIN_TOKEN` | — |

![Cloud 设置](/screenshots/cloud-settings.png)

## 限制

- 数据保存在共享实例，请勿存放敏感笔记或生产密钥
- 学习角色 ID 存于浏览器本地，换设备或清缓存即新账号（弱身份模型）
- 用户列表不公开；角色切换依赖浏览器本地记录
- IM Gateway 默认关闭（长连接与回调无法在演示环境开放）
- PDF 导入大小、建课并发与每日建课次数有限制

## 管理员

维护者访问 `#/admin`，使用 `ADMIN_TOKEN` 登录查看用户与 Token 消耗。

部署说明见仓库 [`deploy/README.md`](https://github.com/liuwenji007/regulus-academy/blob/main/deploy/README.md)。

需要完整能力（含 IM）请 [自托管部署](./self-host.md)。
