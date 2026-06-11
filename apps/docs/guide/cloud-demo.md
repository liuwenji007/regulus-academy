# 在线体验版（Cloud Demo）

Cloud 版由维护者部署在 Railway，降低试用门槛。

## 特性

- 无需 Docker / API Key 即可开始（平台提供每日免费教练消息额度）
- 额度用尽后可 [填写自己的 LLM Key](https://your-demo.up.railway.app) 继续使用（BYOK）
- 首页展示共学人数与近 7 天活跃统计

## 限制

- 数据保存在共享实例，请勿存放敏感信息
- 用户列表不公开；角色切换依赖浏览器本地记录
- IM Gateway 默认关闭
- PDF 导入大小与建课并发有限制

## 管理员

维护者访问 `#/admin`，使用 `ADMIN_TOKEN` 登录查看用户与 Token 消耗。

部署说明见仓库 [`deploy/README.md`](https://github.com/liuwenji007/regulus-academy/blob/main/deploy/README.md)。
