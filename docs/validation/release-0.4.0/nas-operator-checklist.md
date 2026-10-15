# NAS 操作员清单 — v0.4.0 Phase A

本清单尚未在 NAS 执行。先只读 inventory；任何 pull/up/迁移/恢复动作均应等 owner 对精确版本、卷和窗口明确批准。不要输出 secret、完整 Compose environment、JWT、交易明细或数据库文件到 Issue。

## 1. 只读 inventory

在现有 TradeLens Compose 目录运行，保留 project 名称，不猜测数据路径：

```sh
docker compose ps
docker compose images
docker compose config --images
docker compose ps -q api
```

使用上一条返回的 API container ID 代入 `<API_CONTAINER>`：

```sh
docker inspect --format '{{.Config.Image}} {{.Image}}' <API_CONTAINER>
docker inspect --format '{{range .Mounts}}{{println .Type .Name .Source "->" .Destination}}{{end}}' <API_CONTAINER>
docker inspect --format '{{json .State.Health}}' <API_CONTAINER>
```

image ID 用 `docker image inspect --format '{{json .RepoDigests}}' <IMAGE_ID>` 读 digest。通过已登录 Web 的系统信息记录 version/commit。用管理 UI 记录 SQLite backup 的 status/manual_allowed/latest 时间，不复制 access token。手动确认 `TM_DATABASE_URL`、`TM_ATTACH_DIR`、`TM_BACKUP_DIR` 对应持久卷；custom 路径在 `/data` 外时须另有 mount。

配置优先顺序：shell env > Compose `.env` 的插值；Compose environment > Dockerfile ENV；`TM_DATABASE_URL` > legacy `TM_DB_PATH`。`.env` 并非自动全部传给服务。只记录相关配置的“已设置/安全/路径”，不上传 secret。安全 JWT、注册策略必须明确；不要沿用开发默认 `change-me` / `TM_ALLOW_INSECURE_JWT=true`。

## 2. 完整备份与隔离副本

1. 记录基线账户数、交易/execution 数、最新时间、journal/tag/attachment 数量和若干 operator 已知账户读回；在本地私密记录，不贴真实交易到公开 Issue。
2. owner 执行 SQLite 手动快照并确认存在实际文件；完整恢复还需 attachments、backup identity 文件及安全保存的配置/secrets。内置快照不含附件。若 PostgreSQL，用独立 `pg_dump` 和附件备份，不能用 SQLite backup API。
3. 将完整备份复制到**新目录和新命名卷**，校验 hash，SQLite `PRAGMA integrity_check` 必须为 `ok`，并验证实际业务表非空且与基线计数吻合。单纯 integrity ok 不证明备份有用户数据。
4. 新 Compose project 例如 `tradelens-v040-qa`，独立 loopback 端口、独立 secret、关闭 jobs 与所有外部 provider。不得复用生产 named volumes、bind paths、ports、vendor keys；真实副本仅在受控私有环境保留。
5. 只对该副本启动已核准候选。启动会迁移数据库，故路径/卷必须先人工核对。保持来源是 TradeLens，绝不交替使用 upstream TraderMemos 数据库。
6. 记录实际 API/Web image digest 与系统 commit；验证 health、Web `/api` proxy、登录、原账户/历史、附件图片读回、Review Inbox 正反筛选/取消/保存/重进/dismiss/restore、数量/partial close、合成 SBI fixture 重导入去重。禁止生产历史 regroup。
7. 验证旧 Coach JSON/SSE 关闭、history withheld、env/owner 无法重开；检查无外部 provider 调用与无新旧 review 写入。
8. 重启该测试 project，验证 DB/attachments/backup identity 仍在；恢复备份到另一新副本，再用匹配版本验证登录/账户/交易/附件。

## 3. 发布后按 digest pin

0.4.0 尚未发布，**不要把文档里的候选 SHA 当镜像 digest**。发布后从同一批准 run 获取两个 index digest，核对 OCI revision/version、arm64/amd64（按 NAS 架构）和匿名完整 pull，再写本地 override：

```yaml
services:
  api:
    image: ghcr.io/hoesenbruce/tradelens-api@${TL_APPROVED_API_DIGEST:?approved API digest required}
  web:
    image: ghcr.io/hoesenbruce/tradelens-web@${TL_APPROVED_WEB_DIGEST:?approved Web digest required}
```

两个值都必须是已验证 `sha256:...`，source/version 必须一致。该 override 不修改 DB/secret/卷。只在 owner 明确批准项目、卷和迁移后，维护窗口停止 writer，取得新完整备份并升级；不得执行 `docker compose down -v`。

## 4. 失败与回滚

先进入维护隔离并保留失败数据库。以匹配的旧镜像和**升级前完整快照+附件**恢复；SQLite 必须停止 writer 后替换，不能把旧 WAL/SHM 叠到恢复文件上。用旧版本先验证 integrity、登录、计数和附件，再考虑对外。

v0.3.1 缺少新的 provider/跨用户/Legacy Coach 隔离修复。能恢复旧数据不等于旧版本安全可公开；旧版回退保持维护隔离，不能通过 env=true 或 owner 恢复旧 generation。优先在保留安全隔离的当前版本恢复快照，或用独立审查的前向补丁。回旧快照可能丢失升级窗口写入，必须另由 owner 接受。

## 5. 脱敏回传

回传：日期/架构、版本/source SHA、API/Web digest、mount 目标和卷名（可脱敏）、备份 integrity/hash/计数是否一致、QA copy 与实机 smoke 结果、重启/恢复结果、owner 批准记录。未跑的步骤写 NOT_RUN；本地合成演练不能替代 NAS 结果。
