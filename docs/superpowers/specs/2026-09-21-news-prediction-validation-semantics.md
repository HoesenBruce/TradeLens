# 新闻预测结果验证口径

**Issue:** #59 · **Parent:** #47 · **实现:** #60（原始收益）、#61（可选基准）
**口径版本:** `news-validation-v1`
**状态:** 本文冻结后续实现契约；当前尚无结果计算或结果持久化功能。

验证记录预测形成之后的价格表现，不证明新闻导致价格变动，也不是可成交收益或投资回测。
本 PR 只增加规格，不修改 API、数据库或 Web；移动端不在验收范围内。

## 1. 已有能力与实现边界

- 复用 `api/internal/marketdata.Service.GetBars`、`Request`、`Response`、Yahoo/Finnhub
  provider 及现有内存/数据库缓存；统一请求 `Interval = "D"`，不得新建新闻专用下载器或缓存。
- 复用 `MarketTimezone`、`Bar.MarketDate` 与日本代码映射；例如 `285A` 以 stock 请求，
  Yahoo 内部映射为 `285A.T`。不直接把新闻中的任意字符串当作已确认市场。
- 复用 `FindCorporateActionCandidates` / `CorporateActionWarnings`，候选不是已确认公司行动。
- 当前 `predictions` 保存 source、direction、confidence、created_at、updated_at，
  `prediction_horizons` 保存 1/3/5/10/20。结果、预测修订快照和交易日历尚未实现。
- 当前共享服务返回 `adjustment_status = unadjusted`，Yahoo 的 `applyYahooSplits`
  尝试按请求内拆股事件还原价格；旧公司行动规格中的 `split_adjusted` 描述不是当前契约。
  这个标签本身不证明事件覆盖完整，也不证明历史 OHLC 一定可直接跨公司行动比较。
- #60 必须明确 JP/US 交易所日历来源和版本，并在共享市场数据边界使用它；当前仅有时区
  和日期归一化，不能把“有 bar 的日期”或“周一至周五”当作交易日历。来源不可用时显式失败。
  本规格不指定新库，不要求 #59 实现日历、任务调度、批量回填、AI 调用或收益 UI。

## 2. 预测时点、参考价与交易日计数

仅覆盖已明确交易市场、币种的 JP/US 股票。其他工具或无法确认市场的资产返回
`unavailable / unsupported_instrument`，不使用默认纽约时区猜测市场。

`prediction_as_of` 是预测当前修订实际保存时间（UTC），不是新闻发布时间。
首次保存取 created_at；后续任何预测编辑产生新修订和新 as_of，旧结果不转移给新修订。
新闻发布时间只保留为背景，事后录入历史新闻不能获得早于录入时间的参考价。
现有 updated_at 可以作为迁移快照证据，但不能单独充当唯一修订号；#60 应保存不可变内容快照
或内容哈希及明确修订标识。原始新闻、预测 source、正文及预测内容不得被验证器改写。

v1 使用**严格晚于 prediction_as_of 的下一次常规交易日开盘价** P0。
这是一致的日线观测口径，有意不衡量盘中新闻到当日收盘的即时反应，也不用过去收盘价。

| 保存预测时点（资产市场当地时间） | 参考日 D0 / P0 |
| --- | --- |
| 交易日开盘前 | 当日 / 当日常规日线 Open |
| 恰好开盘、盘中、午间休市 | 下一个交易日 / 该日日线 Open |
| 恰好收盘、收盘后、盘前盘后扩展交易 | 下一次严格晚于保存时间的常规开盘 / 对应日线 Open |
| 周末、交易所假日 | 下一交易日 / 对应日线 Open |

开盘瞬间也移到下一交易日，以避免仅秒级时间戳无法证明先后顺序。
停牌不改变 D0：D0 有效日线缺失时不顺延到复牌日，不以前收填充。

周期 H ∈ {1, 3, 5, 10, 20}；**D0 算第 1 个交易日**，DH 为 D0 起第 H 个交易日，
PH 为 DH 的常规日线 Close。1D 即 D0 开盘至 D0 收盘；3D 即 D0 开盘至第 3 个交易日收盘。
周期按交易所开市日计数，不按日历天或实际返回 bar 数计数。个股停牌仍计为交易日。
每个周期独立计算、独立状态，不等到最长周期才发布短周期结果。

## 3. 市场日期、完整性与可用时间

- JP 使用 `Asia/Tokyo`，US 使用 `America/New_York`；UTC 保存瞬间，交易所当地
  `MarketDate` 保存交易日期。US 夏令时交给 IANA 时区处理，不固定 UTC 偏移。
- 开盘、收盘、假日、临时休市和提前收盘来自相应版本的市场日历；午间休市不产生第二个交易日。
  时间戳必须携带时区；缺失/非法时间戳为 `unavailable / invalid_prediction_time`。
- 不把日线 `Bar.Time` 直接当作收盘时刻；市场日历定义 DH 的收盘时刻。
  收盘之前，即使 provider 已返回当天日线，也不能用其 Close 验证。
- v1 规定收盘后 60 分钟为最早评估时刻 `eligible_at`（等待数据的产品规则，不是 provider
  完整性保证）。此前是 pending；此后仍缺失/不完整则进入下述失败状态，并允许后续重试。
  必须使用在 eligible_at 之后取得的行情快照；不能把盘中缓存命中时间冒充上游获取时间。
  当前 Response 不暴露 fetched_at/#60 需补齐共享来源证据或经共享服务刷新，不能旁路下载。
- D0…DH 的每个预期交易日都必须有且仅有一条有效日线；周末/假日不要求 bar。
  价格必须有限且大于零，OHLC 关系有效；零成交量本身不证明停牌，也不能据此伪造价格。
  冲突重复日期、错误市场日期、非交易日 bar 或已知盘中快照均不能作为完整输入。
- 缺少中间日线也标记 incomplete，即使两个端点可算数值；禁止插值、前向填充、跳过缺日
  或偷偷延长窗口。完全无行情和部分行情要区分。网络错误不解释成“股票没涨跌”。

## 4. 复权与公司行动

v1 只计算同币种、同 provider、同价格口径的**未复权价格收益**；不含分红再投资、费用、
滑点、税或汇率收益，不标作 total return。不能混用 adjusted close 和 unadjusted open。
`adjustment_status` 未知、非 unadjusted、两端不一致或来源发生拼接时，结果 incomplete。

公司行动检查应额外请求 D0 前一交易日，用于识别 D0 的边界异常（不计入收益周期）。
缺少这条检查用日线时标记 `incomplete / corporate_action_evidence_missing`。
D0…DH（含两端）出现拆股、合股、配股、合并等不可比较事件或未排除的候选时，保守标记
`incomplete / corporate_action_boundary`，不判方向、不自动改价格或数量；即使事件在 D0
且两个端点看似同一口径，也不例外。已明确否决的误报候选不阻断，但保留审计依据。

已知现金分红只作为警告记录，不把除息缺口当拆股修正；价格收益仍排除分红。
未知异常若触发现有候选检测，仍按 incomplete 处理，不能擅自解释为分红。
检测无候选不等于完整公司行动核验；保存检测方式及覆盖限制，不宣称权威事件覆盖。
跨拆股的经济收益计算留待另一个版本，不在 #60/#61 自动引入复权模型。

## 5. 收益和方向判定

收益以小数保存，展示时乘 100%。仅 validated 结果参与准确率；缺失值是 null，不是 0。

```
asset_return = PH / P0 - 1
benchmark_return = BH / B0 - 1
excess_return = asset_return - benchmark_return
neutral_epsilon = 0.001  # 10 bps，即 0.10%，固定于 v1
```

比较未展示舍入的收益；零和正负阈值边界均归 neutral。不按 confidence 加权，不按波动率
或周期动态改变阈值。实现应避免二进制浮点误差把数学上恰好 ±0.001 分到区间之外，
并以阈值边界和紧邻边界的测试固定行为（可直接按十进制价格比较 PH 与 P0 × (1 ± epsilon)）。

| 观察到的收益 r | 观察方向 | bullish 正确 | bearish 正确 | neutral 正确 |
| --- | --- | --- | --- | --- |
| r > 0.001 | bullish | 是 | 否 | 否 |
| r < -0.001 | bearish | 否 | 是 | 否 |
| -0.001 ≤ r ≤ 0.001 | neutral | 否 | 否 | 是 |

主结果 `direction_correct` 始终使用 asset_return；可选 `excess_direction_correct` 对
excess_return 使用同一规则，不能替换主结果。例如资产 +2%、基准 +3%，bullish 的
原始方向正确、超额方向不正确。未 validated 时观察方向与正确性均为 null。

## 6. 可选基准与窗口对齐（#61）

基准必须显式选定可由共享服务解析的证券、市场、币种；不自动猜指数或代理 ETF。
未指定时 `benchmark_status = not_requested`，所有基准值为 null，原始结果不受影响。

v1 仅支持基准与资产同币种，且 D0…DH 的预期交易日期、常规开盘和收盘 UTC 瞬间完全一致。
因此基准 B0 使用资产 D0 对应的开盘，BH 使用资产 DH 对应的收盘；基准不能自己另数 H 天。
整个窗口按资产和基准各自日历核对；同日期但跨时区/跨市场交易时段不同也算不对齐。
非重叠假日、额外基准交易日、提前收盘差异等返回 `unavailable / benchmark_calendar_mismatch`；
币种不同返回 `unavailable / benchmark_currency_mismatch`。不取交集、不找最近日期、
不填前值、不换汇；跨市场异步基准留待后续口径版本。

基准应用相同的获取时点、缺日、复权和公司行动规则。基准失败只影响 benchmark_status
和 excess 字段，已 validated 的原始资产收益仍保留。只有两侧都 validated 才发布 excess；
资产未验证时，基准自身结果可保存，但 excess 及其方向判定必须为 null。

## 7. 状态与重跑

每个预测修订 × 周期有一个资产状态；基准有独立状态（额外允许 not_requested）。
按下表顺序判断，reason_code 与说明必须保存，不把失败折叠成永久 pending。

| 状态 | 条件与示例 | 可公开的结果 |
| --- | --- | --- |
| unavailable | 输入不支持/无可靠日历，或到期后获取失败、完全无有效行情 | 结果 null；如 calendar_unavailable、provider_error、no_bars |
| pending | 输入和日历可解析，但尚未到 eligible_at | 已知计划日期；收益/正确性 null |
| incomplete | 已到期，有部分行情，但缺日、冲突、无效价、来源时点不足、口径或公司行动不确定 | 证据和缺失日期；正式收益/正确性 null |
| validated | 到期且全部完整性检查通过 | 参考价、结果价、收益、观察方向、正确性 |

时间未到时不必请求行情；provider_error/no_bars 只在到期评估后产生。
日历无法解析时 DH/eligible_at 为 null，不猜日期。每次重试先重新按同一版本判定，
pending/unavailable/incomplete 均可转为 validated；没有固定重试次数后自动改成错误预测。

同一输入修订、规则版本、日历版本及行情快照重复运行必须得到同一结果，且不能重复计入统计。
相同证据重试更新尝试信息即可；provider 修订历史或日历更新产生新的评估修订，保留旧证据和结果。
新的证据可能让 validated 降为 incomplete/unavailable；不能仅保留“最好”的一次结果。
预测编辑后旧结果只属于旧修订，当前视图回到新修订的状态；新增/删除周期不改写历史证据。
不要求永远保存已删除新闻：沿用现有 owner 隔离与父记录级联删除策略。

## 8. 必须持久化的审计字段

以下是语义字段，#60/#61 决定最小表结构；不要只保存百分比或缓存键。
原有行情缓存可能过期，结果证据应随结果保存，它不是另一个行情服务缓存。

| 分组 | 字段要求 |
| --- | --- |
| 身份 | owner/news/asset/prediction ID、预测修订 ID/内容哈希及快照、source、direction、confidence、周期 H |
| 时间与规则 | prediction_as_of、新闻 published_at（可空）、规则版本、epsilon、计算时间、eligible_at、日历来源/版本、时区、市场、币种 |
| 窗口 | D0、DH、参考/结果字段 open/close、对应交易时刻 UTC、预期交易日期列表、缺失日期 |
| 行情证据 | 规范化 symbol、provider symbol/instrument、interval、provider/source、adjustment_status、请求范围、真实 fetched_at、cache 标志、使用的日线快照/哈希 |
| 公司行动 | 检查范围（含前一日）、候选/事件、状态、来源、警告、检测与覆盖限制 |
| 资产结果 | P0、PH、asset_return、observed_direction、direction_correct、status、reason_code、说明 |
| 基准结果 | 可空基准身份、独立窗口/来源/价格/快照/公司行动证据、benchmark_status/reason、B0、BH、benchmark_return、excess_return、excess_direction_correct |
| 重跑 | 评估修订 ID、前次评估引用、输入/证据指纹、最后尝试时间；区分重新获取与相同快照重算 |

价格使用现有精度可保真的表示；收益不在持久化前按展示位数舍入。未知审计值使用 null
并说明原因，不能编造 fetched_at、完整性或 provider 保证。当前 Response 缺少的来源字段
需由共享市场数据边界补充。来源快照必须足以脱离易失缓存重算已有结果。

## 9. 后续实现的确定性验收样例

以下是 #60/#61 应转换成测试的输入/预期，不代表本 PR 已执行引擎验证。
时间示例使用显式测试日历，闭市日不依赖联网；真实日历集成须另行核对来源。

1. 测试日历 JP 周一开市、周二假日、周三/四开市。周一开盘前保存：D0 周一，
   1D 周一收盘、3D 周四收盘；周一盘中/午休/收盘保存均从周三开盘起算。
   周末保存从周一开始。恰好开盘使用下一交易日。停牌缺 bar 不移动这些日期。
2. 用开市日 S1…S20：D0=S1，1/3/5/10/20D 分别取 S1/S3/S5/S10/S20 Close。
   S1 的 1D 已 validated 时，尚未到期的 S20 仍 pending。
3. US 夏令时切换前后按纽约当地开盘选择 D0；对应 UTC 时间变化。提前收盘日的
   eligible_at 按实际收盘 +60 分钟计算。JP 的日期由东京时区解释，不取 UTC 日期切片。
4. `285A` 走现有日本 symbol 映射；无法判市场/非股票不默认为 US；非法时间明确失败。
5. P0=100：PH=101/99/100 分别 bullish/bearish/neutral；100.1 与 99.9 都 neutral，
   100.1001 为 bullish、99.8999 为 bearish。三个预测方向逐一核对布尔结果。
6. 到期前返回盘中 bar 仍 pending；到期后仅有盘中旧缓存为 incomplete。
   全无 bar 为 unavailable；缺中间日、停牌、冲突重复、NaN/零价为 incomplete。
   补齐证据后转 validated，保持原 D0/DH；完全无数据不得得到 0% 收益。
7. 两拆一使未复权价格 100→50：公司行动边界 incomplete，不判 bearish 正确。
   D0 发生事件也阻断；前一日证据缺失、未知复权口径阻断；候选被明确否决后可重新验证。
8. 基准未选不影响资产；资产 +2%、基准 +3% 得 excess=-1 个百分点。
   基准缺价/公司行动不销毁资产结果；不同币种或窗口/日历错位不得偷换日期计算。
9. 同快照重跑不重复生成有效统计记录；历史行情修订可撤回旧 validated，旧评估可追溯。
   编辑预测后使用新 as_of/修订；原新闻、User/AI 来源和另一用户记录均不被改写。

## 10. 交付边界

#59 完成为本规格审阅和一个文档 PR；#60 实现资产验证、审计存储及上述相关测试，
#61 在其上增加基准验证。共享接口的缺口属于后续实现，不在此声称已经补齐。
任何改变参考价、交易日计数、阈值、复权或对齐规则的后续工作必须升级口径版本，不能静默
改写旧结果。Web 显示真实验证结果时另按仓库要求完成真实 API 的端到端验收。
