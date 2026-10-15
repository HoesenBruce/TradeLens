# Phase A1/A2 独立安全审查

审查候选：`e23ccbcf702c3e99989cb755b4b3bb88179d32aa`；目录：`/private/tmp/tradelens-phase-a-040`。主分支基线：`3d383d892698fd746b112eb7c64d9431d29f1f9a`；对照 Draft PR #295：`44c8761f4d236438d69a6c421ab4931e11aed893`。日期：2026-10-15，macOS 本机。适用规则：仓库 AGENTS、CONTRIBUTING、ponytail/full 与 ponytail-review。未改生产代码、未接触其他工作树、未写 GitHub、未合并/发布/部署。

这项变更把非 Coach 的 provider 错误脱敏保留在 Core，把旧 Coach 的生成能力固定为不可用。旧的自由文本与 SSE 实现仍在代码中，但本候选从 API、服务、底层请求和持久化入口拒绝进入；这是隔离，不是修复旧输出的可信来源问题。

## 审查结论

在下述范围和合成数据回归内，没有确认需要修复的 B 类 Core 安全缺陷。独立审查可作为 A1 的 PASS 证据；不单独宣称 A2 / #302 / RELEASE_READY PASS。A2 的完整 Go/race、Web、生产浏览器 35 例、PostgreSQL、精确 SHA CI 及部署验收由主任务汇总。

```text
PROVIDER_CONFIDENTIALITY = PASS (受审出口及窄回归)
LEGACY_COACH_ISOLATED = PASS (当前候选服务端与前端源码边界)
INDEPENDENT_SECURITY_REVIEW = PASS (未确认残留 Core 缺陷)
CORE_SECURITY_BLOCKERS = NONE_FOUND_IN_REVIEWED_SCOPE
```

## A — 已独立修复并可重复验证

1. OCR provider 失败不返回其正文、URL、credential 或自定义 prompt。`api/internal/ocr/vision.go:231` 在提取服务出口统一 Safe，`:451` 只保留数字 HTTP status；`api/internal/ocr/models.go:31` 同样包裹 models/test 出口。`api/internal/api/ocr_handlers.go:70-90` timeout/busy 保留稳定语义，其余失败返回固定错误。`api/internal/api/settings_handlers.go:25-32` 将 OCR/Coach 配置、test、models 限于 owner。
2. IBKR XML/transport/statement 错误在 `api/internal/flexsync/client.go:75-76` 统一 Safe；`api/internal/api/flex_sync_handlers.go:203-207` 用 errors.Is 保留 pending，并再次包裹 API 错误。旧 last_error 在 `:59-60` 读取时替换为固定文字，不重写历史 credential/config 数据。
3. Webhook 与 Expo 的整体/逐 token 错误在 `api/internal/alerts/notify.go:75-76,126-132` 包裹；`api/internal/alerts/service.go:332-346` 写状态和日志时再次包裹。旧通知 last_error 在 `api/internal/api/alert_handlers.go:136-137` 读取时隐藏。
4. `api/internal/providererrors/errors.go:16-20,29-44` 保留 errors.Is/As 的错误身份，同时固定 Error 和所有 fmt verb 的格式输出。现有单测包含嵌套错误、timeout/cancel、HTTP 状态和调试格式；重跑通过。
5. Review Inbox 复用 `handlePatchTrade`，`api/internal/api/trade_handlers.go:182` 先按 user_id 取交易；`:195` 起先验证 tag 所有权，再做 journal/link mutation。`api/internal/api/review_acceptance_test.go:43-64` 独立重跑验证 B 的 PAT 被拒绝、A 的 PAT 可写 grade/notes/tag，其他 API 字段、SQL 列、执行/setup/其他数据和文件保持；`:123-140` 验证伪造其他账户 backlog cutoff 只进入自己的 preferences，不能抑制对方提醒，恢复和去重有效。
6. 相同浏览器身份切换的源码修复存在：`web/src/components/SessionQueryProvider.tsx:6-24` 按 session 重建组件树和 QueryClient 并清缓存；`web/src/lib/api/client.ts` 拒绝 session revision 已变的迟到响应/refresh。`web/src/sw.ts:19-34,44-58` 接管旧 worker、认证请求 network-only 并清 retired API caches。浏览器运行证据由主代理提供，本子任务只追踪源码。

与 #295 的 Core 对比：OCR vision/models、Flex transport/sync、notification transport 的对应生产文件无差异；main 的 providererrors 增加安全 Format，Flex handler 与 alert service 再次脱敏。没有发现 #295 中独立 Core provider 修复遗漏。

## B — 仍可利用且独立适用于 Core

本轮未确认。没有为了凑齐清单而把不可达 legacy 分支算作 Core 缺陷，也没有扩大到非 provider 相关的全仓审计。

## C — 仅旧 Coach；隔离边界必须保留

1. JSON 和 SSE 原输出、provider error、fallback、free-form/provenance 风险仍属于旧实现，不能据此声称已修复。PR #295 的 outputGuard、encoded/provenance 检查和 SSE 校验未被整体移入 main。
2. `api/internal/coach/availability.go:5` 固定 false，无法用 env 或 owner 例外开启。`api/internal/api/coach_handlers.go:48-58` 与 `coach_stream_handler.go:44-53` 先验证所有权，再返回 source=off；发生在构建详情/provider调用/SSE headers 和 flush 之前。
3. 直接服务也关闭：`api/internal/coach/generate.go:184,216`、`stream.go:51,81`、`news.go:58` 在调用 provider 前拒绝。JSON schema/object fallback、stream fallback 均位于关闭边界之后。`api/internal/api/coach_handlers.go:129-130` 禁止直接持久化旧 review。
4. 历史读取 `api/internal/api/coach_handlers.go:163-185` 保留所属用户的 UUID/时间元数据，仅返回 legacy_withheld 与空 notes，不解析/反射旧 model、next_action、headline、detail；损坏 JSON 也不泄漏。导出/公开分享不包含旧 review 正文。
5. 隔离回归针对已启用旧设置、env true/false、新安装、同库重建、stale JWT/PAT、output_version=2、并发请求、跨用户、匿名、持久化直接调用；合成 provider 调用计数为 0，数据库状态不变。底层直接请求与取消上下文也拒绝，不调用 callback。

## 执行的测试

原始输出：`/private/tmp/phase-a-independent-tests.log`。所有命令在候选 API 目录运行，均 exit 0；17 个顶层测试及其子用例 PASS。第一条命令中的 OCR 包无匹配测试，随后第二条明确执行 3 个 OCR 测试，未把 no-tests 当成覆盖。

```sh
go test ./internal/providererrors ./internal/ocr ./internal/flexsync ./internal/alerts ./internal/coach ./internal/api -run 'Test(SafePreservesIdentityWithoutDiagnostics|ClientSafeErrorIdentity|ClientDiagnosticFailureMatrix|ProviderNotificationFailureMatrix|LegacyEntryPointsFailClosed|LegacyPersistenceCannotBypassAvailability|CoachIsolationAcrossConfigurationAndRestart|CoachNewInstallationAndEnvironmentCannotEnableLegacy|ReviewAcceptance)' -count=1 -v

go test ./internal/ocr ./internal/api -run 'Test(ExtractTradeFromImage_(timeout|busyAfterRetries|noRetryOnClientError)|CrossUserIsolation(ProviderConfigurationDiagnostics|AsyncProviderErrors|StoredProviderErrors|ProviderFailureMatrix))$' -count=1 -v
```

Provider API 矩阵实际覆盖 owner/member JWT/PAT、OCR配置 test/models 的授权/无provider副作用、6种 provider 失败（status、embedded、malformed、transport、timeout、cancel）、响应/headers/日志 sentinel、异步 IBKR/notification 失败、数据库 last_error 和旧错误读取、其他用户状态不变。

## 范围与限制

按单实例多用户部署及可升级浏览器客户端评估。未重复主任务负责的重型全套、race、Web浏览器或35例；未运行 PostgreSQL、mobile、NAS、Render/真实provider、真实用户数据库迁移或线上日志检查。服务端隔离能约束新响应和写入，不能收回已下载到旧离线客户端/旧二进制/截图或历史日志的字节。上线仍需相应客户端和 credential 处理计划。

仓库受跟踪源码 diff 为空；主任务创建的未跟踪 `web/node_modules` 未碰。该审查不授予 merge/release/deploy 许可。
