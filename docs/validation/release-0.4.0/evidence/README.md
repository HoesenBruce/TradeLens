# 证据说明

本目录记录 2026-10-15 的公开 GitHub/registry、合成测试与当前 Demo 读回。报告状态以 phase-a-acceptance.md 的范围说明为准。

`.log` 仅把 tab 展开为空格并去掉行末空白，方便仓库 whitespace 检查；未更改断言、失败信息或结果。完整原始输出留在 `/private/tmp/phase-a-040-evidence`，其 SHA256 在 `local-evidence-hashes.json`；独立测试原始输出另在 `/private/tmp/phase-a-independent-tests.log`。Go JSON 大日志未提交，计数/跳过/失败和原始 hash 在 `go-summary.json`。PR 完整 diff 留在本机并可从 #293 读取。

`demo-runtime-tests.log` 是误经 unittest discover 导入的旧默认镜像自检，`release-tests.log` 是缺少外置历史测试依赖的失败。两者保留为执行记录，均不是当前候选 PASS。`postgres-reused-db-failure.log` 保留第一次复用测试库的唯一键失败；新库重跑结果在 go-summary.json，不能隐藏失败后直接称首次成功。

三张 PNG 仅含合成数据。未保存 JWT、真实用户数据或 vendor credentials。
