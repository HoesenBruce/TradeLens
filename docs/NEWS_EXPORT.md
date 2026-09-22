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
