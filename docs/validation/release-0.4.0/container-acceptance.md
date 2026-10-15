# Phase A 容器续验 — 2026-10-15

本记录补充 [Phase A 主报告](phase-a-acceptance.md) 的 A4 本地容器门禁。所有镜像仅在本机 Docker Desktop 使用，没有推送 GHCR、修改 NAS、部署 Render、合并 Release PR 或关闭 Issue。

## 来源与修复边界

- 受测源码：`e23ccbcf702c3e99989cb755b4b3bb88179d32aa`，受跟踪的 API/Web 源码未修改。
- API：仓库原 `api/Dockerfile`，SHA256 `474b9451dcef44137a8f07b0b825a0bfa068c4181db3963c0e958a182ebb78a2`；两架构使用同一 VERSION/COMMIT build args。
- Web：原 Dockerfile 仅替换 `node:24-alpine` 为 `node:24.18.0-alpine`，保留 `$BUILDPLATFORM`、原始 `pnpm install --frozen-lockfile`、pnpm 11.16.0、`CI=true` 和 nginx stage；没有使用 runtimeOnFail=warn，没有改 package/lockfile/依赖策略。
- 正式最小修复：[Draft #314](https://github.com/HoesenBruce/TradeLens/pull/314)，head `6d65405836d7961ddc60064f527b9a213c96eba0`，从 main `3d383d8…` 分出，仅一行变更；独立审查通过。其 Dockerfile 与上述临时文件 SHA256 相同：`4ad85c4a67d2e6fdb12568265d1b70c04f43d2a98fd8c03342b9ec2e6720d680`。
- 当前 #293 仍是未修 Dockerfile 的旧 head。#314 若获准合入，必须刷新 Release Please 候选并在新 SHA 复验；不把本轮 `e23ccbc… + override` 当成一个已存在的正式 Git commit。

Node 24.18.0 与 `.node-version`、devEngines runtime、锁文件一致。官方 Node index `sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd` 包含 amd64/arm64。**pin 没有消除 pnpm 的额外 runtime 下载**：ARM64 日志仍出现 unofficial-builds musl runtime 的重试，最终安装成功。因此本修复是版本对齐，不宣称已修复外部网络超时根因。

## 本地构建与镜像身份

实际命令（仓库根，Web 使用上述临时文件）：

```sh
docker build --platform linux/arm64 --progress=plain -f api/Dockerfile \
  --build-arg VERSION=0.4.0 \
  --build-arg COMMIT=e23ccbcf702c3e99989cb755b4b3bb88179d32aa \
  --label org.opencontainers.image.revision=e23ccbcf702c3e99989cb755b4b3bb88179d32aa \
  --label org.opencontainers.image.version=0.4.0 \
  -t tradelens-phase-a-api:e23ccbc-arm64 .

docker build --platform linux/amd64 --progress=plain -f api/Dockerfile \
  --build-arg VERSION=0.4.0 \
  --build-arg COMMIT=e23ccbcf702c3e99989cb755b4b3bb88179d32aa \
  --label org.opencontainers.image.revision=e23ccbcf702c3e99989cb755b4b3bb88179d32aa \
  --label org.opencontainers.image.version=0.4.0 \
  -t tradelens-phase-a-api:e23ccbc-amd64 .

docker build --platform linux/amd64 --progress=plain \
  -f /private/tmp/phase-a-040-evidence/web-node-24.18.0.Dockerfile \
  --build-arg APP_VERSION=0.4.0 \
  --build-arg APP_BUILD=e23ccbcf702c3e99989cb755b4b3bb88179d32aa \
  --label org.opencontainers.image.revision=e23ccbcf702c3e99989cb755b4b3bb88179d32aa \
  --label org.opencontainers.image.version=0.4.0 \
  -t tradelens-phase-a-web:e23ccbc-node2418-amd64 .
```

ARM64 Web 的成功命令与上一条相同，但不带显式 platform/OCI revision label，tag 为 `tradelens-phase-a-web:e23ccbc-node2418`。实际 image inspect 为 linux/arm64；构建日志、干净源码工作树与 bundle 中的完整 e23 SHA 共同记录本地来源，不能把构建参数视作独立的加密 provenance 证明。

四镜像完整 tags、local index IDs/RepoDigests、架构与 OCI labels 见 [image-identities.json](evidence/container/image-identities.json)。这些是**本地镜像身份，不是 GHCR 的 0.4.0 digest**。ARM64 API 的 `/healthz`、`/system/info` 返回0.4.0/e23 SHA，编译器 go1.27.2。

AMD64 在 macOS arm64 Docker Desktop 的仿真 Linux 环境构建；不是原生 AMD64 主机性能验收。Web 的静态 build stage 按原 `$BUILDPLATFORM` 在 ARM64 原生运行，最终 nginx 为 AMD64；两架构运行 `uname -m` 分别 aarch64/x86_64，164 个静态文件逐一 hash 完全相同。[比较结果](evidence/container/web-cross-architecture-assets.json)。

AMD64 API/Web 另在独立内部网络与全新 tmpfs SQLite 上实际启动，仅 Web发布回环18344。`uname -m` 均 x86_64；nginx健康代理、0.4.0/e23 system info、SPA路由、新安装/setup/login、账户和buy/sell闭环（2股、合成P&L20）、Coach off及手动快照通过。[AMD64 smoke](evidence/container/amd64-smoke.json) / [日志](evidence/container/amd64-smoke.log)。这是仿真运行 smoke，没有冒称在AMD64重跑完整7项浏览器或原生负载测试。

完整构建日志：

- [ARM64 API](evidence/container/api-arm64-build.log)
- [ARM64 Web（Node固定）](evidence/container/web-arm64-build.log)
- [AMD64 API](evidence/container/api-amd64-build.log)
- [AMD64 Web](evidence/container/web-amd64-build.log)

## ARM64 实际容器联调

专用 Compose project `tradelens-phase-a-040-qa`。API/合成 provider 在内部网络；Web 另接前端网络，仅公开 `127.0.0.1:18340`。API 使用全新 `tradelens-phase-a-040-qa_qa_data` named volume；DB=/data/qa.db，attachments=/data/attachments，backup=/data/backups。没有挂载现有数据库或 NAS 卷。

临时 JWT 随机生成，allow_insecure=false；注册仅在这份合成测试实例打开以创建 owner/member。jobs、经济日历、OCR 关闭，market 指向内部合成 bars provider；Coach env 故意 true，地址/密钥是内部 synthetic fixture。fixture 使用既有 Demo image 仅作为 Python 运行时，**不是候选 Demo 的启动验收**。正式 nginx 配置未覆盖。

临时夹具调试时发现宿主 macOS 与 VM 不宜交叉修改活跃 SQLite/WAL，接受结果改用全新 named volume，并在 API 停止后通过 VM 内 Python 注入合成 legacy 行。早期 bind-mount 调试目录没有作为通过证据。夹具还按实际 API 契约处理注册后登录与必填 News 发布时间，没有改产品代码来迎合测试。

### API 和 Legacy 隔离

[preflight](evidence/container/arm64-preflight.json) / [日志](evidence/container/container-preflight.log)：

- 原镜像健康检查、Web `/healthz` 反向代理、`/`/`/review`/`/trades` SPA 路由通过。
- owner/member JWT/PAT，外部用户 review 写入/history 404，member backup GET/POST 403；真实 PNG 上传及授权读回哈希一致，foreign 读取404。
- JSON、SSE、output_version=2/fallback 入口均 source=off、空 notes、no-store，不进入 event-stream；history 仅返回两条 legacy_withheld，正文/模型/next_action 不回放，原行保留。
- News analyze、Coach test/models 被关闭；合成 provider POST 计数0，review 行无新增/改写。
- TM_DATABASE_URL 优先于故意冲突的 TM_DB_PATH，后者文件未创建。独立空 tmpfs 容器使用 insecure secret 且禁用例外时启动失败（exit1），[记录](evidence/container/api-config-negative.json)。
- 额外 owner 配置 enabled=true（DB确认1），与 env=true 同时存在，重启后再测 JSON/SSE/output_version=2 仍 off、两条 legacy 行保持、provider0。[记录](evidence/container/owner-setting-restart.json)。这不是重新启用 Legacy 的授权。

### 原有真实 API 浏览器用例

```sh
E2E_API_URL=http://127.0.0.1:18340/api/v1 \
E2E_WEB_URL=http://127.0.0.1:18340 \
E2E_EMAIL=<本次合成owner> E2E_PASSWORD=<本次合成密码> \
E2E_COACH_FIXTURE=<本次合成trade/news ID JSON> \
./node_modules/.bin/node node_modules/@playwright/test/cli.js test \
  -c /private/tmp/phase-a-040-container-evidence/container-e2e.config.ts
```

7/7，59.4s；原有断言、每文件超时和测试步骤未修改，未把产品 API 替换为前端 mock（原测试本身的保存/加载失败注入保留）。包含 Coach/manual News、Review HTML boundary、Inbox 全转移、独立编辑器、SBI reverse reimport、SBI closing-first/dedup、四 broker 数量/partial close/排序/locale/compact Web。此7项不同于先前的生产安全35例，不混计。

[原输出](evidence/container/arm64-container-e2e.log)、[原测试配置](evidence/container/container-e2e.config.ts)、[中文截图](evidence/container/review-zh.png)、[部分平仓截图](evidence/container/quantities-partial.png)。已检查渲染截图；Codex 内置浏览器另完成真实容器登录和账户选择/Review UI 检查。

**FX 范围限制**：合成 provider 只提供 bars，内部网络不放通 Yahoo。全部账户混合 USD/JPY 且不指定 target currency 时实际返回400 mixed_currencies；指定USD则502 fx_unavailable。[响应](evidence/container/multi-currency-fixture-limit.json)。选择USD合成账户后 Inbox正常。未把这组容器证据当成真实FX/混币种聚合PASS；账户范围内功能通过，既有Go离线FX/多账户测试的证据仍单独限定。

### 完整快照、恢复及重启

[结果](evidence/container/arm64-backup-restart-restore.json) / [日志](evidence/container/arm64-backup-restart-restore.log)：owner手动快照通过、integrity=ok、源/快照全部40表（2 users、12 accounts、30 trades、61 executions、2旧Coach行、1附件）行数和逻辑hash一致。

API/Web restart 后，先在重新登录前比较全部表hash均保持，再验证新登录、账户、基线账户交易、附件和Coach off。backup identity文件保持；快照恢复到第二个新 named volume，复制附件及 identity，启动第二个同源 API，再验证同样的身份/账户/基线交易/附件/history withheld。没有用宿主SQLite操作活跃VM的WAL。所有SQLite操作均在VM内。

jobs关闭使备份调度status=disabled，但manual_allowed=true且实际快照成功；不混淆调度状态与手动快照结果。此演练使用合成数据，不能替代NAS真实完整备份、升级副本与实机验收。

快照实际mode为0600、owner UID0、size585728字节，[权限读回](evidence/container/snapshot-permissions.json)。API/Web保持原镜像默认root运行；没有声称已验证NAS的非root UID映射。

## CI、审批与剩余门禁

#314精确head `6d65405836d7961ddc60064f527b9a213c96eba0`：

- [API CI](https://github.com/HoesenBruce/TradeLens/actions/runs/38701555142)：PASS。
- [Web CI](https://github.com/HoesenBruce/TradeLens/actions/runs/38701555191)：PASS。
- [Chromium security CI](https://github.com/HoesenBruce/TradeLens/actions/runs/38701555090)：PASS。
- 两个标题check：PASS。[状态快照](evidence/container/pr-314-ci.json)。

该CI head基于main，版本文件仍0.3.1；本地容器基于0.4.0 release head+一行覆盖。两份生产代码一致，区别已明确记录，不能将二者合并冒充一个已经选定的新Release SHA。

```text
API_BUILD = PASS (原Dockerfile，ARM64/AMD64)
WEB_BUILD = PASS (Node24.18.0单行覆盖，ARM64/AMD64)
ARM64_SMOKE = PASS
AMD64_BUILD = PASS (Docker Desktop仿真构建及运行smoke)
CONTAINER_INTEGRATION = PASS (7/7、API隔离、快照/重启/恢复)
GHCR_READY = NO (正式固定候选与registry产物未建立)
NAS_READY = NO (未访问NAS或真实副本)
RELEASE_READY = NO
```

机器可读结果：[container-gates.json](evidence/container/container-gates.json)。完整原始构建日志留在 `/private/tmp/phase-a-040-container-evidence`（ARM64 Web原记录在 `/private/tmp/phase-a-040-evidence`）；仓库日志仅标准化空白，原文件SHA256在 [raw-log-hashes.json](evidence/container/raw-log-hashes.json)，未更改断言/超时或输出结果。没有提交临时JWT、session、数据库或完整运行环境。

本地容器通过不代表0.4.0已经发布。#314和#311均未合并，#293仍OPEN；GHCR无0.4.0正式digest，NAS未访问/未更新，Render仍无本次批准部署证据，全部Issues保持打开。下一步需owner审阅最小修复和证据，修复接受后刷新#293、重验新head，另经原审批关口发布/部署；不得从测试PASS推导授权。
