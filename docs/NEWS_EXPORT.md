# News Thesis Markdown 导出

`GET /api/v1/news/:id/export` 使用与 News API 相同的登录认证，下载单条
News Thesis 的 UTF-8 Markdown。不存在或不属于当前用户的记录返回 404。

文件名固定为 `news-thesis-<安全化记录 ID>.md`，不包含下载时间。
内容包含 Obsidian YAML frontmatter、新闻来源/原文、资产、AI/User 预测、
交易日周期、已保存的验证/基准结果和复盘笔记。收益数值为比例，例如 `0.1` 表示 10%。
自由文本按字面值保存，避免 Markdown、HTML 或 wikilink 改变笔记结构。

导出在同一数据库事务中读取，不调用 AI 或行情服务，也不重新验证预测。
当前周期没有验证结果时标为 `pending`，缺失数值保持 `(blank)`；历史验证明确标记
`Current: false`，不作为当前预测结论。相同保存状态生成相同字节。

这两个导出 Issue 的交付范围为 API；下载按钮、Web 交互和移动端不在范围内。

## 批量导出

`GET /api/v1/news/export` 下载 `news-theses.md`。不提供筛选条件时导出当前用户的全部
Thesis。可重复提供 `id` 参数导出明确选中的集合，例如
`?id=<first-id>&id=<second-id>`；重复 ID 只导出一次，任一 ID 不存在或不属于当前用户时返回 404。

筛选参数复用 `/api/v1/news/performance` 的解析、校验及匹配规则：

- `from` / `to`：包含边界的 UTC 新闻发布日期，格式 `YYYY-MM-DD`。
- `category`：精确分类。
- `symbol`：忽略首尾空格并转大写后的完整资产代码；`asset_type`：stock/etf/index。
- `source`：预测来源 ai/user；`horizon`：1/3/5/10/20 个交易日。

条件之间为 AND；资产、来源与周期必须匹配同一个预测。没有来源/周期限制时，
没有预测的 Thesis 也可以按新闻或资产条件入选。ID 集合与筛选条件取交集。
匹配后导出完整 Thesis（包括 AI/User 和其他周期），不截断记录内的对照信息。
这遵循报告 API 语义；Web 列表目前的客户端模糊代码匹配和显示时区不是此接口的筛选语义。

顺序与 News API 相同：发布时间降序，再按 ID 降序。文档含索引和记录分隔符，
每条复用单条渲染器，内嵌 YAML 用代码块展示。空集合仍返回 200 Markdown，明确显示
`count: 0` 及无匹配记录说明；非法筛选参数返回 400。
