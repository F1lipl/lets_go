# 本地事件投递表设计

## 三张表

- outbox_event：业务事务写入事件事实及 payload。status=1 待分发，3 已分发，4 分发失败；2 保留兼容，新的本地分发流程不提交这个中间状态。published_at 记录分发事务完成时间，不代表订阅者全部执行成功。
- event_delivery：每个事件与订阅者对应一条任务，主键为 (event_id, consumer_name)。payload 只从 outbox 读取。
- inbox_event：处理器去重及业务结果记录，沿用原主键 (consumer_name, event_id)，新增 status=4 表示 superseded。processed_at 同时用于已处理和已取代的终态。

## 分发

短事务中锁定待分发的 outbox 行，依据启动时注册的订阅关系插入全部 delivery，然后更新 outbox.status=3 和 published_at，再提交。支持 FOR UPDATE SKIP LOCKED。
未注册的事件类型按分发失败处理，不要悄悄视为零订阅成功；明确允许零订阅的事件才可直接完成分发。
已有事件的任务集合在分发时固定。新增订阅者是否补历史任务需要明确安排，不能靠每次扫描改变已分发事件的任务集合。

## 任务状态

1 pending；2 processing；3 succeeded；4 superseded；5 failed。

- attempt_count 每次领取时递增，包括上一次进程退出后的重新领取。max_attempts 默认 10，可按处理器调整。
- pending 到 next_attempt_at 后才可领取，且 attempt_count < max_attempts。达到上限的待执行/到期任务改为 failed。
- processing 必须同时设置 claim_token 和 locked_until。领取、到期判断和续期使用数据库时间。
- 完成/失败确认、续期都携带主键和 claim_token 条件；确认还检查 status=2。禁止无条件更新或使用通用全字段 Update 确认任务。
- 超时回收检查 status=2 AND locked_until<=NOW(3)，清除领取字段，安排退避重试或转 failed。超时不等于业务已失败。
- succeeded、superseded、failed 必须设置 completed_at，并清空领取字段。
- 人工重试 failed 时，重置 completed_at，明确重置次数或增加 max_attempts，并设置新的 next_attempt_at；保留终态不是自动循环重试。

## 业务一致性

claim_token 只保护任务进度。处理器仍需在同一业务事务内完成 inbox 去重记录和业务更新。
过期 worker 仍可能执行，因此卡片等完整快照用 source_post_version 条件写入防止旧版本覆盖新结果。
superseded 由处理器根据业务版本判断，不由调度器仅凭事件时间判定。增量业务不能直接丢弃旧事件。
本地任务的重试次数以 event_delivery 为准，旧 inbox 的 attempt_count 不作为调度依据。

## 事件处理完成判断

不额外持久化汇总状态：

- outbox 尚未分发：待分发或分发失败。
- outbox 已分发且存在 pending/processing：处理中。
- 全部任务是 succeeded/superseded：完成。
- 没有 pending/processing，但存在 failed：处理结束且有失败。

不要把 outbox.status=3 解释为业务全部完成。

## 索引和保留

支持全局到期任务领取、指定 consumer 领取、超时回收和终态归档四种扫描。
event_delivery 外键阻止有子任务时删除 outbox，避免丢失仍需重试的 payload。
按保留策略先归档/清理终态任务，再清理事件；inbox 去重记录保留时间必须覆盖可能的重投窗口。

本次仅建立表和约束，尚未实现分发器、worker、续期、回收及处理器。
