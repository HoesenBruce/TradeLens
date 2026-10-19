# #311 集成状态 — 2026-10-15

本页记录文档PR本次集成，不替代后续新main/Release PR的验收报告。

- 最新已获取main：`8b8f44b40a2382623100169286e3d54c65a543ba`。#314已由owner合并，merge SHA同上；正式Web Dockerfile已固定Node24.18.0-alpine，原frozen-lockfile/依赖策略不变。
- #311此前head：`aa793cc89848f6ade476c17c40354af5bd09f126`；本轮在隔离工作树正常merge当前main，无冲突，无rebase/历史重写。相对main仍仅文档/脱敏证据，不引入生产文件差异。
- #293当前head：`c3ae0d32df35e23b210102e8365dd808ab30ba66`，已纳入#314；尚未包含本次#311文档集成。#311经独立审查、最终head CI通过后才可按本轮用户明确授权squash合入；其后必须再次刷新#293并验证新head。

## 历史证据与现状

[Phase A报告](phase-a-acceptance.md)、[容器报告](container-acceptance.md)、[前次定版审查](release-candidate-finalization.md)和JSON快照保留各自原始日期/SHA，不把其中旧“#314未合并”描述当作本页之后的实时状态。旧e23源码+临时Dockerfile覆盖、6d修复PR head、8b squash main、本次#311集成head与未来Release SHA是不同身份。

旧四镜像是本地image ID/index身份，不是官方0.4.0 GHCR digest。ARM64实际runtime/7例与AMD64仿真smoke已通过；原生AMD64、NAS真实数据副本/实机、v0.4.0 Render候选、正式GHCR匿名完整pull仍NOT_VERIFIED。便携完整临时容器夹具尚未交付；旧start-containers调试脚本不可作为最终named-volume拓扑的启动器。

#311合入不会使旧测试自动成为新main/RC测试。后续选择最新main/刷新后的#293，记录文件级来源等价或重跑所需用例，并单独归档当时实际source、命令、镜像身份、容器fixture和结果。

## Journal 并发范围

原报告的“并发更新/两设备”只对应现有测试断言：重新读取最新详情后保存，或有顺序地交错操作；不证明两个请求在GET/PATCH之间竞争时不会丢失更新。当前Journal handler及Review编辑器没有版本条件写入保证。若另一编辑在GET返回后、PATCH提交前更新journal，后续保存可能覆盖该编辑；两个后端PATCH也可能读取同一旧journal再upsert而丢失未修改字段。因此**严格并发安全不成立，不作PASS承诺**。

新Core独立验收必须检查该已知GET/PATCH覆盖限制及实际release合同。单用户也可能有两个tab/设备，不自动等于不存在竞争。不能擅自把多用户/多设备合同缩为单写入者以取得PASS；若该风险需owner接受或代码修复，单列为发布审批前的阻塞。#311作为准确记录限制的文档可审查，不等于接受数据完整性风险。

## 授权与停止边界

本轮用户明确授权在#311全部条件通过后mark ready并squash merge #311，随后刷新/验证#293；这更新前次“未授权合并#311”的操作边界，不改变历史执行事实。#293合并、Release/tag、正式镜像push、GitHub Environment审批、Render部署和NAS真实数据操作仍未授权。安全Issues保持OPEN，Legacy Coach仍隔离，不称原漏洞修复。
