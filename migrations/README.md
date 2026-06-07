# 已弃用：请使用 `internal/storage/migrations/`

本目录仅保留早期 `001_init.sql` 历史副本，**运行时不再使用**。

SQLite 迁移真相源为 [`internal/storage/migrations/`](../internal/storage/migrations/)（`001`–`013`，由 Go embed 在启动时按序执行）。

新增或修改表结构请只改 `internal/storage/migrations/` 并递增编号。
