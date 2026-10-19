# TradeLens Core v0.4.0 — Phase A 验收

本文件保留原验收时点的历史事实；#314合并后的最新文档集成状态见 [集成状态](integration-status.md)。新main/RC必须单独验收。
验收日期：2026-10-15（Asia/Shanghai）。本报告区分候选本地测试、GitHub CI、既有已发布镜像、真实 Render Live 和尚未执行的 NAS 验收。

## 1. 执行结论

**Core 源码与应用层安全验收通过；Phase A 尚未完成，暂不建议发布。** 精确候选的全量 Go / race、Web、生产 Chromium 安全 35 例、Review Inbox / SBI / 数量 E2E、独立安全审查、SQLite 合成升级恢复及 PostgreSQL 条件覆盖均通过。本地 ARM64/AMD64 API/Web 构建、ARM64 容器联调及恢复已在续验通过；Web 依赖临时 Node 单行固定，对应正式修复 #314 尚未合入候选。v0.4.0 GHCR 发布、NAS 实机验收及 Render 候选部署仍未完成。main/发布候选中的操作文档还有与当前自动化状态不一致的文字；本报告分支已提供文档纠正，尚未合入候选。

没有合并 #293、发布 tag/Release/镜像、批准 Environment、触发 Render 部署、更新 NAS、迁移真实用户数据库或修改线上认证/注册设置。#298、#302、#236、#235 均保持打开；#295 保持 Draft。

本轮新增容器完整证据见 [容器续验报告](container-acceptance.md)，下文早期下载失败作为历史记录保留，不再代表本地构建当前结果。 最新定版判断、NOT_VERIFIED矩阵与后续精确操作计划见 [RC定版审查](release-candidate-finalization.md)；历史PASS不自动覆盖刷新后的RC。

状态定义：`PASS` 有本轮证据；`NOT_RUN` 未执行；`BLOCKED` 前置条件缺失；`FAILED` 实际命令/断言失败；`AWAITING_APPROVAL` 必须由 owner 明确批准。测试通过不授予发布权限。

## 2. 精确基线与候选

| 对象 | SHA / 状态 | 来源 |
| --- | --- | --- |
| origin/main | `3d383d892698fd746b112eb7c64d9431d29f1f9a` | [main commit](https://github.com/HoesenBruce/TradeLens/commit/3d383d892698fd746b112eb7c64d9431d29f1f9a) |
| upstream/main | `4db1abb2c8dc4118010a9d278f2d8bec6a2fcee0` | fetch upstream 后记录；未整合其代码 |
| Core / release 测试候选 | `e23ccbcf702c3e99989cb755b4b3bb88179d32aa` | [#293](https://github.com/HoesenBruce/TradeLens/pull/293) 当前 head，main 是其祖先 |
| Review Inbox #292 | MERGED，`3d383d892698fd746b112eb7c64d9431d29f1f9a` | [#292](https://github.com/HoesenBruce/TradeLens/pull/292)，2026-10-15 06:20:06 UTC |
| 数量显示 #310 | MERGED，`b63aead3074e08435979a93c6aa84acbe021d189` | [#310](https://github.com/HoesenBruce/TradeLens/pull/310) |
| SBI 方向 #309 | `6016623aa3f1e088f450eafef8d8265c1dbec257` | [#309](https://github.com/HoesenBruce/TradeLens/pull/309) |
| 独立 provider 修复 #305 | `021ffa091ca9a0e4c29a3d5acdffeb12d54a2f01` | [#305](https://github.com/HoesenBruce/TradeLens/pull/305)，已在 main |
| Legacy 隔离 #306 | `66e0616a4673e8602dc5830d66abd3da180431bd` | [#306](https://github.com/HoesenBruce/TradeLens/pull/306)，已在 main |
| 对照 Draft #295 | `44c8761f4d236438d69a6c421ab4931e11aed893` | [#295](https://github.com/HoesenBruce/TradeLens/pull/295)，未合并 |
| 最新已发布稳定版 | v0.3.1，`58f6a3c1a80cc195c3650fd48ef71680684ccb50` | [Release](https://github.com/HoesenBruce/TradeLens/releases/tag/v0.3.1) |

开始时主工作树在 main 且无修改。检查了全部 25 个已有 worktree：final-security-295 有两份未跟踪独立测试，upstream-audit 有七个修改文件；issue-201/login-animation 有未跟踪依赖链接；均未触碰。候选验证在 `/private/tmp/tradelens-phase-a-040`，文档在从 main 分出的 `/private/tmp/tradelens-phase-a-report` / `docs/302-phase-a-report`。报告分支不携带 #293 的生成版本文件。

原始命令包括 `git fetch origin --prune`、`git fetch upstream --prune`、`git worktree list --porcelain`、逐 worktree `git status --short`、`gh pr list`、`gh issue list`、`gh pr view 293`。开放 PR/Issue、workflow、Environment 和规则快照见 [evidence](evidence/)。本报告不覆盖其他 Coach 开发分支。

## 3. A1 — #298 独立安全处置

独立审查员只读检查精确候选及 #295 对照，运行 17 个顶层测试与其子例，两个命令均 exit 0。[完整审查](independent-security-review.md) / [原始测试日志](evidence/independent-security-tests.log)。

| 类别 | 结论 | 证据与范围 |
| --- | --- | --- |
| A 已修复 | PASS | OCR extraction/test/models、IBKR 手动/后台同步、webhook/Expo 逐 ticket、状态持久化与旧 last_error DTO、safe fmt/slog；保留 errors.Is/As、cancel/timeout/pending/dead-device 行为 |
| A 授权边界 | PASS | owner/member JWT/PAT、拒绝请求零 provider 调用/零越权持久化；响应、header、日志 sentinel；无关表与 foreign 状态哈希 |
| B 独立 Core 缺陷 | 本轮未确认 | 不需要另建 #298 修复 PR，不整体移植 #295 |
| C Legacy Coach | ISOLATED | 原自由文本、来源和 SSE 风险未修复；当前构建常量 false，API/服务/直接 transport/持久化 fail closed |

JSON、SSE、fallback、history、导出/分享、旧设置/env true/false、新安装、同库重建、stale JWT/PAT 与直接服务调用均在隔离回归内。历史记录不删除，返回元数据和 withheld 状态；禁用路径不调用 provider、不写旧 review。浏览器认证切换、迟到响应、前缀 API、旧 worker 激活/清理及离线不回放由本轮 35 例补证。

`structured output provenance` 的安全实现仍属于 #300/#301/#303；不能以 Core 隔离验收关闭 Coach 恢复工作。既存日志/备份、旧二进制和完全离线客户端已复制的字节不能由本次服务器隔离收回。

```text
PROVIDER_CONFIDENTIALITY = PASS
LEGACY_COACH_ISOLATED = PASS
INDEPENDENT_SECURITY_REVIEW = PASS (限定于上述 Core/隔离审查)
CORE_SECURITY_BLOCKERS = NONE_FOUND_IN_TESTED_SCOPE
```

## 4. A2 — #302 应用层验收

环境：macOS arm64；Go 1.27.0；Web 使用已安装 Node 24.18.0 / pnpm 11.16.0 / 锁文件依赖；Playwright 已安装 Chromium。Web node_modules 复用主工作树的依赖目录，候选源码、构建输出和测试均来自固定候选工作树；相同 head 的干净 Linux CI 提供独立安装验证。

| 验证 | 本轮结果 |
| --- | --- |
| `go test ./... -count=1 -json` | PASS，1169 个测试事件通过，10 个 PostgreSQL 条件 skip；随后补测全部 10 个 |
| `go vet ./...` | PASS，exit 0 |
| `go test -race ./... -count=1 -json` | PASS，同样 1169 pass / 10 条件 skip；PostgreSQL race 另补齐 |
| `./node_modules/.bin/vp test run --maxWorkers=2` | PASS，198 文件 / 1240 tests |
| `./node_modules/.bin/tsc -b --noEmit` | PASS |
| `./node_modules/.bin/vp check` | PASS，0 errors / 347 既有 warnings |
| `VITE_API=/api/v1 VITE_APP_VERSION=0.4.0 VITE_APP_BUILD=e23ccbc… ./node_modules/.bin/vp build` | PASS，生产 bundle 和实际 worker |
| `node scripts/run-security-e2e.mjs` | PASS，35/35，56.0s；生产构建、临时 SQLite、回环 synthetic API/market、原始超时/断言/repeatEach=3 |
| 真实 API 集成 E2E | PASS，6/6，1.2m；原有 test 文件和各自超时未修改 |
| 独立窄安全审查 | PASS，17 顶层测试及子例 |

6 个集成文件为 `review-html-boundary.spec.ts`、`review-inbox.spec.ts`、`review-remediation.spec.ts`、`review-sbi-integration.spec.ts`、`sbi-review-direction.spec.ts`、`trade-quantities.spec.ts`。一次性 harness 复用仓库 security harness 的服务生命周期，换成这六个 testMatch、相同生产静态服务器；固定合成登录、API `127.0.0.1:8099` / Web `127.0.0.1:5189`。没有连接真实数据库或 broker。

覆盖 Review Inbox 正向/反向筛选、保存失败重试、取消、重进、reset、back/skip、账户范围、评分/笔记/mistake tags、原 journal HTML/歧义边界、独立编辑器读回及并发更新、backlog dismiss/restore 和提醒去重；SBI cash/margin、long/short、closing-first/reverse reimport/dedup、planned short/actual long 保留；数量含手动/IBKR/Generic/SBI、部分平仓、排序、详情、多语言与紧凑 Web。详细行为以原测试断言为准，不把 6 个顶层数量等同于六个单一步骤。

JWT/PAT 的 account/trade、journal/attachments/tags、imports/exports、backup/admin 边界及会计/估值/FX/多账户/analytics 由完整 Go 套件和独立窄测试覆盖。OCR 成功解析与失败行为有 Go 回归；本轮未重复真实 OCR UI 全流程，不将旧 #305 的浏览器记录冒充本轮候选记录。未运行真实 vendor、mobile 或生产历史 regroup。

Codex 内置浏览器另完成合成用户登录、打开实际 Review Inbox 并检查中文渲染；自动化矩阵使用任务指定的 Chromium harness。检查了 E2E 中的中文 Inbox 和部分平仓截图：[Inbox](evidence/review-zh.png)、[数量详情](evidence/quantities-partial.png)、[SBI review](evidence/sbi-review.png)。

### PostgreSQL 条件覆盖

仓库明确提供 PostgreSQL Compose overlay，因此没有将 skip 当作 PASS。创建仅绑定 `127.0.0.1:55440` 的一次性 PostgreSQL 16.15 aarch64 容器，镜像 `postgres:16@sha256:1a6ab3f5345eb6dbe04a1349529caabdb0ab09293a09590fad07b2246bfa4b54`。未使用现有数据库或持久用户卷。测试完成后仅清理本次创建的容器及其一次性卷。

```sh
TM_TEST_DATABASE_URL=<本次一次性库URL> go test ./internal/db ./internal/store ./internal/predictionvalidation -count=1 -p=1 -json
TM_TEST_DATABASE_URL=<另一全新一次性库URL> go test -race ./internal/db ./internal/store ./internal/predictionvalidation -count=1 -p=1 -json
```

两轮各 92 pass / 0 skip / 0 fail，包含全部原先 10 个条件 skip 的迁移、querier、list/time、bulk/import conformance、news/assets、predictions/persistence。第一次 race 错误复用了普通测试库，固定 `news@example.com` 触发 `users_email_key`，没有 data race；保留失败原日志，在全新库上原断言重跑通过。该测试要求每轮新库，未为验收修改源码。

完整 Go JSON 原始文件留在本机 `/private/tmp/phase-a-040-evidence`；[摘要与 SHA256](evidence/go-summary.json) 区分普通、race 和 PostgreSQL 补测，不把所有 suite 的重叠测试相加。

### SQLite 数据安全

原生二进制从已发布 v0.3.1 与精确候选分别构建。合成账户 1、交易 1、execution 2；停止旧进程后 SQLite backup 到新文件，再启动候选副本。40 张表逻辑 SHA256 全部不变，`PRAGMA integrity_check=ok`，原字段与账户/trade IDs 不变，仅新增 #310 的 `quantities`。候选 owner 手动快照、恢复到第三个文件、重启、旧快照+旧二进制回滚均通过，合成附件文件 hash 不变。[日志](evidence/sqlite-upgrade.log)。

`TM_JOBS_ENABLED=false` 时手动备份成功但调度状态仍为 disabled，符合既有契约；不将 disabled 误判为快照失败。v0.3.1 到候选 migration 文件无差异。此演练是本地合成数据，不能证明 NAS 用户数据或完整附件关系已备份。

```text
CORE_SECURITY = PASS (应用层)
CROSS_USER_ISOLATION = PASS
LEGACY_COACH_INACCESSIBLE = PASS (当前构建)
REVIEW_INBOX = PASS
SBI_ACCOUNTING = PASS (合成/回归)
BACKUP_RESTORE = PASS (合成 SQLite；非 NAS)
POSTGRESQL_COVERAGE = PASS (全部10个条件覆盖，普通及race)
GO_TESTS = PASS
GO_RACE = PASS
WEB_TESTS = PASS
PLAYWRIGHT_E2E = PASS (6项集成)
SECURITY_E2E_35 = PASS (本地35/35；精确head CI 35/35)
EXACT_SHA_CI = PASS
CORE_D1_READY = BLOCKED (Node修复未进入正式候选；NAS/Demo目标验收未完；#302不关闭)
```

## 5. A3 — #293 / Release Please

[PR #293](https://github.com/HoesenBruce/TradeLens/pull/293) OPEN，非 Draft。head `e23ccbc…` 包含 main `3d383d8…`；相对 main 仅 6 个生成文件：manifest、CHANGELOG、VERSION、Web/mobile package version 与 Expo version；五处版本均 0.4.0，无依赖升级。Changelog 包含 #285/#292/#310/#309/#306/#305。未手工修改任何生成文件。

精确 head GitHub runs 全通过：

- [API CI 38691833451](https://github.com/HoesenBruce/TradeLens/actions/runs/38691833451)：vet、API 全量、cross-user/backup 窄 race、build。
- [Web CI 38691833494](https://github.com/HoesenBruce/TradeLens/actions/runs/38691833494)：check、198/1240 tests、build。
- [Browser security 38691833493](https://github.com/HoesenBruce/TradeLens/actions/runs/38691833493)：35 passed；日志已读回。
- [PR title 38691832734](https://github.com/HoesenBruce/TradeLens/actions/runs/38691832734)；[Mobile CI 38691833510](https://github.com/HoesenBruce/TradeLens/actions/runs/38691833510) 亦成功，但不代表 iOS/Android E2E。

main ruleset [24700869](https://github.com/HoesenBruce/TradeLens/rules/24700869) active，无 bypass actor；PR、线性历史、禁止删除/force push，strict required checks 为 API/Web/title。Chromium security 目前成功，但不在 required 列表。传统 branch-protection endpoint 的 404 不等于 ruleset 未保护。

Release Please、Docker publisher、Demo deploy workflow 均 active；`DEMO_AUTO_DEPLOY_ENABLED=true`。Release Please 合并后创建 Release，再调用 publisher；不是普通 main push 自动发布。publisher 解析 Release Please 输出 SHA，复验 tag/Release/version、禁止覆盖既有 version/full-SHA tag、复用 API/Web CI，再在 ghcr Environment 审批后 publish；Render 另一个 Environment 审批。ghcr 只允许 main + v*，render-demo 只允许 main；两者均 HoesenBruce required reviewer、无 admin bypass。没有改任何设置或自批。

**候选文档缺口（本报告分支已纠正当前状态）**：受测候选的 `CONTRIBUTING.md` 仍写 automated publishing disabled；`docs/release.md` 开头仍把 v0.2.1 和 Demo 未激活当当前状态，与本轮 live readback 不符；`.env.example`/README 的生产示例仍 pin 0.2.1。上游历史 changelog、Go module、移动端原名和 attribution 是保留的兼容/历史标识，不能简单删除。实际 Compose/publisher/Web repository link 已指向 TradeLens，没有 upstream image fallback。本报告分支已更新 CONTRIBUTING 与 release.md 的当前发布、Demo 和 ruleset 状态，并保留明确标注的历史 fixture。README/`.env.example` 的旧稳定示例不作为本轮批准值；NAS 清单要求明确批准版本/digest。该文档 PR 接受后还需刷新 #293 并复验新 head；未改生成版本文件。

附加 `scripts/test_release_please.cjs` 未通过：缺少外置 release-please 模块，且脚本是 0.2.1 的历史边界 fixture，不能作为当前 0.4.0 simulation PASS。当前真实 bot PR 的 diff、manifest 和精确 CI 已独立核对。部署流程离线测试 10 项、Docker metadata 3 项通过。一次 `unittest discover` 导入 `test_demo_render.py` 实际运行了默认旧镜像 `tradelens-demo:243` 的自检；它不是候选镜像证据，明确排除出本轮候选 Docker PASS。

```text
RELEASE_VERSION = 0.4.0
RELEASE_PR = https://github.com/HoesenBruce/TradeLens/pull/293
RELEASE_HEAD = e23ccbcf702c3e99989cb755b4b3bb88179d32aa
RELEASE_DIFF_VALID = PASS
CHANGELOG_VALID = PASS
VERSION_CONSISTENCY = PASS
RELEASE_READY = NO (#314未合入并刷新候选；发布操作文档/目标环境门禁未完)
```

将来 squash merge 的提交 SHA 通常不同于当前 PR head。发布审批必须重新记录实际 tag SHA、绑定 API/Web image revision 和 Render Live；不能把 `e23ccbc…` 自动当最终发布 SHA。main 或 release head 再变化时，本轮证据不能直接外推。

## 6. A4 — GHCR 与本地候选容器

匿名 token 获取 manifest 成功，两个现有 0.3.1 OCI index 均含 linux/amd64、linux/arm64 和两份 attestation manifest；读取 arm64 config labels，source=TradeLens、revision=`58f6a3c…`、version=0.3.1。只验证了 manifest/config 公共读权限，不冒称新空 Docker config 完整 pull 已通过。

| 已发布回退基线 | immutable index digest |
| --- | --- |
| `ghcr.io/hoesenbruce/tradelens-api:0.3.1` | `sha256:f841b33b8de2b0b03163ce72585f47e4d04b329a94f87c97927cb5de79eb0234` |
| `ghcr.io/hoesenbruce/tradelens-web:0.3.1` | `sha256:109fa4d1b7e09f749b75bd298a74d4b971b08c0a57bf16e77112048de43a701f` |

两个 0.4.0 manifest 均 HTTP 404，符合尚未授权发布；**不存在本轮可用于生产 pin 的 v0.4.0 GHCR digest**。[匿名读回](evidence/ghcr-anonymous.json) 和 [API](evidence/ghcr-api-stable-config.json)/[Web](evidence/ghcr-web-stable-config.json) config。

候选本地 Docker 使用仓库原 Dockerfile 和 build args：

```sh
docker build -f api/Dockerfile --build-arg VERSION=0.4.0 --build-arg COMMIT=e23ccbcf702c3e99989cb755b4b3bb88179d32aa -t tradelens-phase-a-api:e23ccbc .
docker build -f web/Dockerfile --build-arg APP_VERSION=0.4.0 --build-arg APP_BUILD=e23ccbcf702c3e99989cb755b4b3bb88179d32aa -t tradelens-phase-a-web:e23ccbc .
```

早期原 Dockerfile 的两次 Web 下载超时与 API 中止日志保留在 evidence；其后续验已完成原 API Dockerfile 的 ARM64/AMD64 构建，并用 Web 单行 Node 24.18.0 覆盖完成两架构构建。没有采用 runtimeOnFail=warn，也没有改 lockfile。ARM64 真容器的7项浏览器回归、API/权限/Legacy零调用、40表快照、重启、第二新卷恢复及附件读回通过；详见 [续验](container-acceptance.md)。本地镜像source为e23加Web构建覆盖，正式修复 [#314](https://github.com/HoesenBruce/TradeLens/pull/314) 尚未合入。没有改变宿主代理/证书设置或清理共享缓存；下载成功不证明网络根因已修复。

配置静态审查：API `/data` 存 SQLite/attachments/默认 backups；Compose env 优先于镜像 ENV，shell/.env 是变量替换源，不会自动把任意 TM_* 注入服务。`TM_DATABASE_URL` 优先 `TM_DB_PATH`；mount 路径若移到 `/data` 之外需另持久卷。默认 compose 允许 change-me insecure JWT，仅开发用；实际部署必须显式安全 secret、`TM_ALLOW_INSECURE_JWT=false`、注册策略另行确认，未修改 live。API/Web runtime 默认 root，Render demo 固定 UID 10001；没有伪称非 root 全面支持。healthcheck、unless-stopped、same-origin nginx `/api`、启动 migration 都有配置；本地容器运行与默认 root 权限范围见续验；NAS 实际权限和持久卷仍须验证。

```text
API_IMAGE = ghcr.io/hoesenbruce/tradelens-api (0.4.0未发布)
WEB_IMAGE = ghcr.io/hoesenbruce/tradelens-web (0.4.0未发布)
API_DIGEST = NOT_AVAILABLE_FOR_0.4.0
WEB_DIGEST = NOT_AVAILABLE_FOR_0.4.0
MULTIARCH = PASS_LOCAL_ARM64_AMD64_BUILDS; NOT_PUBLISHED_0.4.0_INDEX
IMAGE_SOURCE_SHA = e23ccbcf702c3e99989cb755b4b3bb88179d32aa (Web另加Node单行覆盖；非正式发布commit)
PUBLIC_PULL = PASS_0.3.1_MANIFEST_CONFIG / NOT_RUN_0.4.0_FULL_PULL
DOCKER_SMOKE = PASS_LOCAL_ARM64; AMD64_BUILD_AND_RUNTIME_SMOKE_PASS (仿真环境)
GHCR_READY = NO
```

## 7. A5 — NAS 操作员验收包

没有本任务可用的已授权 NAS connector 或执行环境；未探测地址、SSH、卷或用户数据。现有版本、image IDs/digests、卷、备份和权限未知。[NAS 操作清单](nas-operator-checklist.md) 给出只读 inventory、隔离副本演练、批准后更新与回滚步骤；其步骤是操作说明，**并非已执行证据**。

```text
NAS_ACCESS = NOT_AVAILABLE
NAS_CURRENT_VERSION = UNKNOWN
NAS_BACKUP_VERIFIED = NOT_RUN
NAS_UPGRADE_DRY_RUN = NOT_RUN_ON_NAS_COPY (本地原生及容器合成演练PASS)
NAS_SMOKE = NOT_RUN
NAS_DEPLOY_APPROVED = NO
NAS_DEPLOYED = NO
```

## 8. A6 — Render Demo

目标 [TradeLens Demo](https://tradelens-demo.onrender.com/home)。`render.yaml` runtime=docker，Dockerfile=`deploy/demo/Dockerfile`，native autoDeployTrigger=off；发布脚本验证服务是 GitHub repo + main，提交 exact `commitId` 给 Render。**Render 从 Git 源构建独立合成 Demo 镜像，非直接运行 GHCR API/Web 两镜像。** GHCR digest artifacts 是批准部署的前置证明，不会因 GHCR push 自动改变 Live。

当前 authenticated `/api/v1/system/info` 读回 version=0.3.1、commit=`58f6a3c1a80cc195c3650fd48ef71680684ccb50`，与旧 [发布/部署 run 37943690036](https://github.com/HoesenBruce/TradeLens/actions/runs/37943690036) 的 redacted artifact 相符。[当前实测](evidence/demo-live-system-info.json) / [历史部署记录](evidence/render-previous-deploy-record.json)。没有 Render 管理 API connector，当前 provider 控制面的 latest deploy status 未直接读回；不能仅凭历史 artifact 称当前控制面状态已确认。

本轮直接调用现有脚本的 `smoke(contract)`（不执行 deploy）：PASS，31.48s。覆盖 public ready、/demo、匿名 accounts=401、合成用户非 admin、JPY account、107 trades / 214 executions / P&L 25000、account value 975000 / capital 950000、3 条 FICTIONAL news 与日历、accounts/imports/password/tokens/admin/share/analyze 禁写403，以及前后 trade 不变。首次 `/readyz` 200 耗时27.76s；未刻意停止服务制造 cold start，不把该请求判定为完整冷启动验收。

上述是旧 Live 的公开 smoke，未覆盖 v0.4.0 Review Inbox Demo restrictions、新 worker 升级或候选容器。代码级 Demo read-only/backup/AI restrictions 由候选 Go 测试覆盖；候选实际 Demo build/start/reset、Review Inbox 行为、阻止 uploads/admin/token、内存和冷启动仍应在本地候选 Demo 镜像和批准后的 Live 上验收。部署脚本在成功 smoke 后再次确认 Live deploy ID，记录 source/version/digests，失败不自动把旧版重新上线。

```text
DEMO_AUTO_DEPLOY_ENABLED = true
DEMO_EXPECTED_SHA = AWAITING_FINAL_APPROVED_RELEASE_TAG_SHA (当前受测head e23ccbc…)
RENDER_LIVE_SHA = 58f6a3c1a80cc195c3650fd48ef71680684ccb50
RENDER_DEPLOY_STATUS = SERVING_0.3.1; current provider control-plane NOT_READ
DEMO_READ_ONLY = PASS_CURRENT_0.3.1_PUBLIC_SMOKE
DEMO_PUBLIC_SMOKE = PASS_CURRENT_0.3.1_ONLY
DEMO_ROLLBACK_READY = CONDITIONAL (见回滚安全限制；候选尚未部署)
DEMO_DEPLOY_APPROVED = NO
```

## 9. 未解决事项与回滚

1. 本地 ARM64/AMD64 构建及 ARM64容器验收已通过；#314未合入正式Release候选，0.4.0多架构index/完整匿名pull/正式provenance仍未建立。不可用本地镜像替代正式发布身份。
2. NAS 真实备份、离线副本、附件关联、现有账户/历史完整性与实际升级尚无证据。
3. Render 当前是旧版；候选 Demo 镜像、Review Inbox Demo 限制及新版本 Live 尚未验收。
4. 修正发布操作文档的过时当前状态，必要时在 owner 接受文档修复后刷新 #293，再验证新 head。报告和文档 PR 一旦合入 main，也会使当前 release baseline 过时。
5. PostgreSQL 补齐的是仓库已有条件覆盖，不是全 API PostgreSQL E2E 或生产 dump 恢复证明；内置 backup 仍只支持 SQLite。

**回滚计划**：批准前记录当前 API/Web digests、配置/secret 的安全离线副本、完整 DB+attachments、Compose project 与卷映射；在维护窗口停止 writer，校验快照和文件 hash；先在完全隔离副本验证新旧读回；新版本失败先维护/下线，保留失败数据另存，再用匹配的旧镜像+升级前完整快照恢复，绝不在已升级库上盲目跑旧 migration、不 `down -v`。新版写入后回旧快照会丢失窗口内写入，必须 owner 接受或另做导出恢复。

**安全限制**：v0.3.1 不含本轮 #294/#298/#299 隔离修复。原生合成“能回滚”只证明数据可恢复，不能证明旧版本可安全公开。NAS 回旧版应保留维护隔离、不暴露旧 Coach/provider 路径并撤除相关 vendor 凭据；优先在当前安全候选恢复快照或发布保留隔离的前向补丁。Demo 旧 read-only guard 本轮通过，但回退仍要独立验证 guard/source/public smoke，不能把同一个已发布版本的 deploy 脚本当任意 downgrade 工具。

## 10. Owner 批准与下一步

当前先审阅 #314 的单行固定和 #311 证据、刷新正式候选并完成目标环境验证；**不建议现在批准 #293**。具备完整新证据后，owner 需要分别批准：#293 精确最终 head 的合并（触发 Release）、实际 tag/SHA 对应 GHCR 两个发布 job、Render `render-demo` Environment、明确 NAS project/卷/完整备份对应的更新和持久数据迁移、若需要的 live auth/registration 修改。每个关口重新核对 SHA/digests，不能从一次测试或本报告推导授权。

建议下一开发任务：审阅已提供的发布操作文档与 #314；明确固定修复接受后的新Release候选并重验；按 NAS 清单回传脱敏 inventory/副本结果；候选 Demo 本地验收；Core 全部门禁补齐后更新 #302，再请求 #293 owner 批准。Coach #300/#301/#303 继续单独推进，不重新开放旧 generation。

## 11. 必填最终状态

```text
PHASE_A_STATUS = BLOCKED_NOT_COMPLETE
MAIN_SHA = 3d383d892698fd746b112eb7c64d9431d29f1f9a
CORE_CANDIDATE_SHA = e23ccbcf702c3e99989cb755b4b3bb88179d32aa

ISSUE_298 = INDEPENDENT_REVIEW_PASS; OPEN
LEGACY_COACH_ISOLATION = PASS_CURRENT_BUILD
ISSUE_302 = OPEN; APPLICATION_AND_LOCAL_CONTAINER_GATES_PASS; FIXED_CANDIDATE_AND_TARGET_GATES_PENDING
CORE_SECURITY_GATE = INCOMPLETE; APPLICATION_AND_LOCAL_ISOLATION_PASS; FIXED_RELEASE_CANDIDATE_AND_TARGET_GATES_PENDING

GO_TESTS = PASS; POSTGRES_SKIPS_COMPLETED_SEPARATELY
GO_RACE = PASS; POSTGRES_RACE_COMPLETED_ON_CLEAN_DB
WEB_TESTS = PASS_198_FILES_1240_TESTS
PLAYWRIGHT_SECURITY_35 = PASS_35_OF_35_LOCAL_AND_EXACT_HEAD_CI
FINAL_HEAD_CI = PASS_e23ccbcf702c3e99989cb755b4b3bb88179d32aa

RELEASE_VERSION = 0.4.0
PR_293_STATUS = OPEN; NOT_MERGED
RELEASE_READY = NO

GHCR_API_DIGEST = NOT_AVAILABLE_FOR_0.4.0
GHCR_WEB_DIGEST = NOT_AVAILABLE_FOR_0.4.0
GHCR_READY = NO

API_BUILD = PASS_ARM64_AND_AMD64_ORIGINAL_DOCKERFILE
WEB_BUILD = PASS_ARM64_AND_AMD64_WITH_NODE_24_18_OVERRIDE
ARM64_SMOKE = PASS
AMD64_BUILD = PASS_LOCAL_EMULATION_AND_RUNTIME_SMOKE
CONTAINER_INTEGRATION = PASS_7_E2E_AND_API_BACKUP_RESTART_RESTORE
NAS_READY = NO

NAS_UPGRADE_READY = BLOCKED_NO_ACCESS_OR_REAL_COPY_EVIDENCE
NAS_DEPLOYED = NO

RENDER_EXPECTED_SHA = AWAITING_FINAL_APPROVED_RELEASE_TAG_SHA
RENDER_LIVE_SHA = 58f6a3c1a80cc195c3650fd48ef71680684ccb50
DEMO_DEPLOY_READY = NO_FOR_0.4.0
DEMO_DEPLOYED = NO_FOR_0.4.0

OWNER_APPROVAL_REQUIRED = MERGE_293; RELEASE_TAG_IMAGE_PUBLICATION; ENVIRONMENT_APPROVALS; RENDER_PRODUCTION; NAS_UPDATE_AND_PERSISTENT_MIGRATION; ANY_LIVE_AUTH_CHANGE
REMAINING_BLOCKERS = NODE_FIX_NOT_IN_RELEASE_CANDIDATE; RELEASE_OPERATION_DOCS; NAS_REAL_COPY_AND_TARGET_SMOKE; CANDIDATE_DEMO_VALIDATION; FORMAL_GHCR_ARTIFACTS_AND_UNGRANTED_APPROVALS
NEXT_ACTION = REVIEW_314_AND_311; REFRESH_FIXED_293_AND_REVALIDATE; COMPLETE_TARGET_ACCEPTANCE_BEFORE_RELEASE_APPROVAL
```

未验证 mobile、真实 provider、生产用户数据迁移和 NAS；本地、CI、旧 Live 的 PASS 各自限定，不能外推为 v0.4.0 已发布或已部署。
