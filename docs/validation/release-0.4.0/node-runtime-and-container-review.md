# Node 固定与容器续验独立复核

2026-10-15，独立审查员只读检查 #314 的一行变更与本轮容器报告、日志和摘要；未修改代码、重跑重型套件或执行 GitHub/发布操作。

Node 24.18.0 与 `.node-version`、devEngines、锁文件 runtime 一致；保留 BUILDPLATFORM、pnpm11.16、CI=true、frozen-lockfile 和 nginx stage，Compose/发布调用不变。官方基础镜像manifest含amd64/arm64。Dockerfile SHA256与受测临时文件一致，diff whitespace通过。结论：可审阅聚焦修复，不声称消除了额外runtime下载或修复网络超时根因。

容器报告明确区分 e23 release源码+Web临时覆盖、#314 的6d654058提交与尚未刷新的#293；ARM64完整7例和AMD64仿真smoke未混同。独立比对164个静态文件hash一致；7例59.4s、40表快照/重启/新卷恢复、owner/env enabled=true重启仍off的断言与结果记录相符。

相对链接全部存在；证据未检出JWT/Bearer/GitHub token/私钥模式，未提交session或运行环境。FX的400/502未被算成PASS；GHCR正式产物、NAS、Render、正式新候选与整体门禁仍未完成，回滚保持维护隔离。文档准确性复核PASS，无新增阻塞；这不授予合并、发布或部署许可。
