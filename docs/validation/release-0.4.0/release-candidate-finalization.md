# v0.4.0 Release Candidate Finalization — 2026-10-15

本文件保留原验收时点的历史事实；#314合并后的最新文档集成状态见 [集成状态](integration-status.md)。新main/RC必须单独验收。
本轮是候选定版审查与下一阶段验收计划，不是 merge/release/tag/push-image/deploy 授权。仓库为 HoesenBruce/TradeLens；所有 GitHub 操作显式选择该仓库，不使用 upstream 的同号 PR。保留全部既有分支和未提交工作。

## 精确基线与当前判断

| 对象 | 已核实 SHA | 当前状态 |
| --- | --- | --- |
| origin/main | `3d383d892698fd746b112eb7c64d9431d29f1f9a` | 本轮 fetch 后未变化 |
| #314 | `6d65405836d7961ddc60064f527b9a213c96eba0` | OPEN / Draft；MERGEABLE / CLEAN；API/Web/Chromium/title 全 PASS |
| #311 基线 | `d1492f662c6c99bf1e337275d22fc71154e50c91` | OPEN / Draft；MERGEABLE / CLEAN；API/Web/Chromium/title 全 PASS；本补充会产生新的文档 head，需检查新 CI |
| #293 | `e23ccbcf702c3e99989cb755b4b3bb88179d32aa` | OPEN；MERGEABLE / CLEAN；API/Web/Chromium/mobile/title 全 PASS；尚不含 #314/#311 |
| 最终受测 RC | `NOT_FINALIZED` | #314/#311 合入后刷新 #293 才能确定 |
| 最终发布源码 | `NOT_FINALIZED` | 最终 squash 提交/tag SHA 可能不同于 PR head；届时重新绑定镜像及 Demo |

实时读回及检查摘要：[finalization-checks.json](evidence/finalization/finalization-checks.json)。该 JSON 的 #311 CI 是上述基线快照，不伪装为包含自身的新 head CI。最终合并前必须再次核对 main/head/严格检查。

三个 PR 的 GitHub `reviews` 数组均为空；没有正式 GitHub APPROVED review。现有独立只读审查记录是 [Core 安全报告](independent-security-review.md) 与 [Node/容器复核](node-runtime-and-container-review.md)，二者结论 PASS，但不能冒充 owner 审批。main ruleset 要求 PR、strict API/Web/title、线性历史、禁止强推/删除、无 bypass actor，required approving review count=0；Chromium 目前运行成功但不是 ruleset required check。

## #314 与 #311 的可合入范围

#314 相对 main **仅** `web/Dockerfile` 一行：`node:24-alpine` → `node:24.18.0-alpine`。保留 `$BUILDPLATFORM`、pnpm11.16.0、`CI=true`、`RUN pnpm install --frozen-lockfile` 和 nginx stage。`package.json`、锁文件、workspace、API Dockerfile 与依赖版本均不变。Dockerfile SHA256 与受测临时文件完全相同。ARM64/AMD64 构建、ARM64 容器7/7与AMD64仿真smoke的历史证据有效；本轮按 image ID 重读四镜像的完整 ID/架构也一致。

Node 只用于 build stage；最终 nginx/config 没改，两架构164个静态文件hash一致，安全 CI PASS，没有观察到应用 runtime 安全回归。这不是一次全基础镜像 CVE 审计。pnpm 仍额外下载 runtime，pin 不证明外部下载超时根因已消除。

**#314 技术上可供 owner 授权合并**；继续保持 Draft，不自动 ready/merge。若 main/head 变化先重查。

#311 基线只有65个文档/证据文件（含 CONTRIBUTING.md），与 #314 的生产文件不相交；版本生成文件不在 #311 中。**可在 #314 后合入作为历史验收记录**：不重写旧 SHA、原日志或旧失败，不把旧 e23+override 改称新 RC PASS。#314 合入后 strict 检查会要求 #311 跟上新 main；使用无历史重写的分支更新，重查 diff 与新 head CI 后再供 owner 合并。

## 证据准确性与复现限制

本轮比对38个本地原始文件SHA256，无缺失/不匹配；四份归档构建日志与原始文件仅空白标准化不同。版本、源树差异、四镜像ID和架构已复核。原 API/Web 测试源码、Dockerfile 和构建命令均可按 SHA 获取；没有依赖升级或用 warn 绕过 frozen lockfile。

**完整容器夹具的可移植复现仍为 NOT_VERIFIED**：归档 Playwright config 含当时绝对路径；运行时 Compose、随机JWT/session、合成ID及部分夹具仅在临时目录，没有作为一套可直接运行的干净克隆脚本提交。旧 `start-containers.py` 是早期 bind-mount 调试版本，不能直接当作通过时 named-volume 网络拓扑的启动器。哈希和成功日志证明当时结果，不等于已交付便携验收工具。#311可以作为准确的证据文档合入；刷新RC的重跑必须重新建立夹具并记录实际命令/配置，不能只复制旧PASS。

新夹具必须使用全新专用命名卷、VM内SQLite、API/fixture内部网络、仅Web回环端口；停止API后才注入旧review，禁止宿主跨VM写活跃WAL。参照 [容器报告](container-acceptance.md) 的已验证拓扑与原7个版本化E2E；另行归档脱敏Compose模板、fixture版本/校验值和参数说明，保留实际运行凭据在临时私密文件，不提交session/DB。新的运行不要求复用旧合成ID或相同数据库文件hash。

## Core Security：#298 与 #302

当前候选源码重新检查了 providererrors.Safe/Format、OCR vision/models、IBKR direct/background、notification整体/逐票、状态写入/旧DTO、安全日志出口，以及session QueryClient/迟到响应/旧SW API cache清理；相关生产源码在 main/三个PR之间没有新增业务差异。

`LegacyAvailable()` 固定false。JSON/SSE在调用provider、stream headers/flush之前关闭；直接服务、底层request/fallback、持久化入口均拒绝，历史只给legacy_withheld/空notes，旧记录保留。owner/env=true无法开启；News analyze受同样边界约束。已有容器provider调用0、review无新写入及跨用户JWT/PAT证据仍有效。

本轮在精确 e23 源码重新执行14个顶层 provider/legacy/route/Review 安全测试及子例，全部PASS：[摘要](evidence/finalization/security-recheck-summary.log)，原日志hash在摘要/JSON。另重跑3项Docker metadata与10项Demo部署离线测试通过；mock的Live/失败输出不是实际Render操作。本轮没有重复旧全套/race/35浏览器/7容器E2E；旧全套证据保留原SHA限定。

- `ISSUE298_SECURITY=PASS_REVIEWED_SCOPE; OPEN; FINAL_SHA_RECHECK_REQUIRED`。本轮未重做真实vendor或OCR真实API UI全流程；不能据此关闭Issue或宣称审计所有成功payload。
- `ISSUE302_CORE_D1=INCOMPLETE`。本地应用/隔离通过，但正式Node修复尚未进入RC、新RC/最终main安全smoke及目标环境门禁未完。
- 旧Coach自由文本/provenance/SSE漏洞仍是隔离代码中的未解决风险；#295保持Draft，不整体移入。旧离线客户端、历史日志/截图/已复制字节不能被服务端补丁收回。旧0.3.1回滚不能公开恢复这些危险路径。

另有 [#212](https://github.com/HoesenBruce/TradeLens/issues/212) 仍OPEN，负责Git历史隐私清理及仓库改为public前的门禁；本轮未重写历史/改变可见性，不能用Core隔离PASS替代该独立门禁。#300/#301/#303仍属于Coach恢复轨道，不授权旧生成器重开。

## Release Please 与 #293

Release Please、Docker publication、Demo deploy均active；最新main run [38691793415](https://github.com/HoesenBruce/TradeLens/actions/runs/38691793415)成功。`DEMO_AUTO_DEPLOY_ENABLED=true`。本轮GitHub读回无v0.4.0 Release；未创建tag/Release。 匿名registry复核：两个image的0.4.0及sha-e23标签均404，未把404当可用产物。

当前#293相对main仅六个生成文件；VERSION、manifest、Web/mobile package、Expo version全部0.4.0；changelog含backup、Review、数量、SBI、Legacy隔离与provider修复。#314的fix应纳入最终changelog；#311 docs可不出现在release notes，但源码仍必须包含它。不要手工强制版本或改生成文件。

**两PR合入后必须刷新#293**。先让main push上的Release Please处理；docs不改生成body时bot可能不更新branch，不能把成功run当作main已被包含。记录main/head，确认main是新head祖先；若落后，按 [已审阅恢复流程](../../release.md#recover-a-release-pr-that-is-behind-main) 使用expected_head_sha保护的update-branch合并更新（非rebase/force push，不合入main、不发布）。本轮不执行该更新，因为前置PR未合入。

新head必须仍只相对main更改预期六个生成文件、版本一致、changelog包含Node修复；重跑严格required checks及Chromium/full Go race/业务与容器验收，再次安全复核。实际发布tag SHA之后再次核对 source/version/CI，不能继承旧head的绿色状态。普通main push不直接发镜像；合并release PR会创建tag/Release并启动受审批的发布链，因此#293保持不可合并判断。

publisher绑定Release Please输出source SHA，再解析published tag/Release/VERSION/manifest；API/Web CI与构建checkout解析后的同一SHA，OCI revision/version同步；version/full-SHA标签若已存在即拒绝。两镜像multiarch，保留各自index digest artifact。移动tag不是两个镜像间的原子事务；部分成功时不盲重跑/覆盖tag，走已审阅缺失产物修复或新patch。

`ghcr` 允许main和v*，`render-demo`仅main；均required owner reviewer、禁止admin bypass。本轮未改变量、权限、环境policy或secret，也未批准任何等待job。

## 后续精确验收计划（均未执行）

以下 `RC_SHA` 是刷新后的完整40位候选SHA；`RELEASE_SHA` 是另经授权发布后的真实tag commit，不预填e23。所有记录绑定这两个身份，不只记录版本号。

### 1. 发布前：精确候选与原生AMD64

1. 接受#314/#311、刷新#293并固定RC_SHA；保存main/RC diff、五处版本、changelog、CI run URL。干净独立工作树构建，无临时Dockerfile override。
2. 在受控Linux AMD64主机/CI runner记录宿主`uname -m=x86_64`与Docker daemon `Architecture=x86_64`，确认未通过ARM主机的QEMU；仅容器`uname=x86_64`不足以证明native。
3. 使用原Dockerfile构建API/Web，完整记录stdout/stderr/退出码，禁止依赖策略绕过：

```sh
# 未来受控runner：RC_SHA必须先由owner固定，不在本轮执行。
docker build --platform linux/amd64 --progress=plain -f api/Dockerfile \
  --build-arg VERSION=0.4.0 --build-arg COMMIT="$RC_SHA" \
  --label org.opencontainers.image.revision="$RC_SHA" \
  -t "tradelens-rc-api:$RC_SHA-amd64" .
docker build --platform linux/amd64 --progress=plain -f web/Dockerfile \
  --build-arg APP_VERSION=0.4.0 --build-arg APP_BUILD="$RC_SHA" \
  --label org.opencontainers.image.revision="$RC_SHA" \
  -t "tradelens-rc-web:$RC_SHA-amd64" .
```

4. 全新命名卷/内部fixture/安全随机JWT启动API+Web，原nginx代理；验证0.4.0/RC_SHA、setup/login、JWT/PAT外部资源404/member403、两股合成交易P&L20、Review/SBI/数量7项原E2E、附件hash、备份40表逻辑hash/完整性、restart/new-volume restore、Coach JSON/SSE/v2/history/owner+env零provider调用及零写入。新夹具按上文归档，不沿用早期调试启动器。
5. ARM64在同一RC_SHA重建/复验；记录架构、镜像ID、环境、脱敏日志。任一失败即停止定版，不把旧e23结果自动升级。

### 2. 官方GHCR产物与匿名拉取（仅另获release/publication批准后）

1. Owner分别审查#293合并/发布与ghcr环境审批；检查发布的v0.4.0 tag commit=RELEASE_SHA、latest stable与manifest版本；若与RC_SHA不同，先做最终source差异核对与final-main安全smoke。
2. 等同一批准run的API/Web发布job全部success。保存两个`ghcr-digest-{api,web}-{RELEASE_SHA}` artifact、index digest、各平台manifest digest、OCI revision/version/source与run URL。0.4.0和sha-RELEASE_SHA必须均指向该image对应index；列出且实际读取linux/amd64、linux/arm64 config。attestation的unknown平台不算runtime架构。
3. 新无登录Docker config按digest拉取两镜像两架构；没有既有credential/helper，不能把manifest HTTP200当完整pull。

```sh
# 未来批准后执行；digest只取批准run的记录，不用本地image ID替代。
TL_PULL_CONFIG=$(mktemp -d)
printf '{"auths":{}}\n' > "$TL_PULL_CONFIG/config.json"
TL_API_REF="ghcr.io/hoesenbruce/tradelens-api@${TL_APPROVED_API_DIGEST:?approved digest required}"
TL_WEB_REF="ghcr.io/hoesenbruce/tradelens-web@${TL_APPROVED_WEB_DIGEST:?approved digest required}"
for TL_PLATFORM in linux/amd64 linux/arm64; do
  DOCKER_CONFIG="$TL_PULL_CONFIG" docker pull --platform "$TL_PLATFORM" "$TL_API_REF"
  DOCKER_CONFIG="$TL_PULL_CONFIG" docker pull --platform "$TL_PLATFORM" "$TL_WEB_REF"
done
```

4. 在native AMD64及ARM64目标按两个批准digest运行上面的新库smoke，API `/system/info` 与Web内嵌build均为RELEASE_SHA/0.4.0；新来源不能借旧本地镜像通过。任一image缺失/标签已存在/摘要不一致则停，禁止覆盖或盲重跑。

### 3. NAS数据库副本升级与回滚

仅在owner明确批准NAS地址、旧版本、源卷、备份窗口和**副本**后，按 [NAS清单](nas-operator-checklist.md) 先只读inventory；本轮不探测NAS。停止writer取得新完整DB快照+附件+backup identity+私密配置，校验integrity/hash及非空业务计数。复制到全新project/目录/命名卷，不挂生产卷；关闭jobs/外部provider，使用独立secret与回环端口。

先在副本用固定RC_SHA本地镜像验迁移，再在正式产物获批准后用同源两个GHCR digest重复。核查登录/历史/附件/Review/SBI合成重导入/数量/Coach隔离、重启与第二副本恢复；不对生产历史regroup。记录uid/mode/持久路径/配置优先级，不将本机root结果当NAS权限通过。全部PASS后再单独请求真实NAS升级授权；不得`down -v`。

回滚先维护隔离，保留失败库；停止writer，恢复升级前DB+附件+identity，绝不叠旧WAL/SHM。验证integrity、计数、登录、附件及业务读回。若需旧0.3.1二进制，保持隔离、不公开服务；优先同安全版本快照恢复/受审前向修复。明确窗口写入可能丢失，需owner接受；不把一次空库启动当恢复。

### 4. Render Demo候选与回滚

Render使用`deploy/demo/Dockerfile`从Git源码构建，**不使用上述两张GHCR镜像**，且它仍有独立的`node:24-bookworm-slim` build stage；#314的Alpine修复不能证明Demo可构建。先按RC_SHA本地构建该Dockerfile并传RENDER_GIT_COMMIT，运行`python3 scripts/test_demo_render.py <候选镜像>`，禁止省略参数误用默认旧镜像。验证非root、JWT fail-closed、provider死亡整体失败、restart重建合成库/旧JWT失效和read-only policy；再用内置浏览器核对Demo/Review的真实渲染与被禁写入口。

现有正式`demo-deploy.yml`要求**已经发布的latest stable Release和成功GHCR publication run**，不是任意未发布RC入口。因此若owner要求真实Render为发布前硬门禁，需要另行批准隔离候选service及精确RC_SHA构建；不能绕过正式脚本的release检查，也不能拿旧0.3.1 Live替代。候选service/目标授权当前NOT_VERIFIED；本轮不创建/部署。

正式链批准后：重新核对服务repo/main、Dockerfile、native auto-deploy off、旧Live SHA、无并行deploy、publication两个digest、latest stable和RELEASE_SHA；保留render-demo审批。脚本提交exact commitId，验证Live commit和`/readyz`/`/system/info`、合成107trades/214executions/P&L25000/account975000/capital950000/3news、backend所有禁写403、Coach off；内置浏览器做真实UI并检查启动/代理/缓存状态。记录冷启动/重启和脱敏deploy-id/run，不改变注册/secret/资源配置。

失败不自动rollback/deploy。记录失败状态/最后Live SHA，停止继续推广；由owner决定维护隔离、重试前向修复或恢复已验证安全候选。回退旧0.3.1需单独评估其已知缺少安全修复的暴露风险，不为了恢复演示重新开放legacy。

## 未验证矩阵与最终输出

| 范围 | 当前状态 |
| --- | --- |
| 新正式RC/最终main SHA全套与容器复验 | NOT_VERIFIED |
| v0.4.0官方GHCR产物/双架构config/匿名完整pull | NOT_VERIFIED |
| 原生Linux AMD64 smoke | NOT_VERIFIED（已有本地AMD64仿真PASS） |
| NAS真实数据库副本升级/权限/完整恢复及实机升级 | NOT_VERIFIED |
| v0.4.0独立Demo镜像构建/真实Render候选 | NOT_VERIFIED（旧Live证据限定0.3.1） |
| 真实vendor、容器真实FX/混币种、mobile E2E | NOT_VERIFIED |
| 干净克隆直接重跑完整临时容器夹具 | NOT_VERIFIED（源码测试/命令及历史日志可审阅） |

```makefile
MAIN_SHA = 3d383d892698fd746b112eb7c64d9431d29f1f9a
PR314_READY = YES_TECHNICAL_PENDING_OWNER_MERGE
PR311_READY = CONDITIONAL_AFTER_314_BASE_REFRESH_AND_FRESH_CI
ISSUE298_SECURITY = PASS_REVIEWED_SCOPE_OPEN_FINAL_SHA_RECHECK_REQUIRED
ISSUE302_CORE_D1 = INCOMPLETE
RELEASE_CANDIDATE_SHA = NOT_FINALIZED (current PR293=e23ccbcf702c3e99989cb755b4b3bb88179d32aa)
PR293_READY = NO_REFRESH_AND_FINAL_ACCEPTANCE_REQUIRED
GHCR_PUBLICATION_READY = NO
NATIVE_AMD64_VERIFIED = NOT_VERIFIED
NAS_READY = NOT_VERIFIED
RENDER_DEMO_READY = NOT_VERIFIED
REMAINING_BLOCKERS = ACCEPT_314_311; REFRESH_RC; FINAL_SHA_CHECKS; REPRODUCIBLE_FIXTURE; NATIVE_AMD64; DEMO_CANDIDATE; NAS_COPY; OFFICIAL_ARTIFACTS_AND_PULLS
OWNER_APPROVAL_REQUIRED = EACH_MERGE; RELEASE_TAG; GHCR_PUBLICATION; NAS_COPY_ACCESS_AND_LIVE_UPGRADE; RENDER_TARGET_AND_DEPLOY
NEXT_ACTION = OWNER_REVIEW_314_THEN_311; REFRESH_293_AFTER_BOTH; PIN_AND_REVALIDATE_NEW_RC
```

本轮未合并、release、tag、push镜像、NAS/Render部署或关闭Issue。没有改真实数据库、配置/secret/审批设置；不自动解除Draft或恢复旧Coach。仅在#311补充准确证据和计划，并在相关PR/Issue同步脱敏判断。
