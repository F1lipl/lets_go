# Worker 使用

Worker 只消费已经领取并提交到 event_delivery 的任务。它不负责扫描 outbox、创建任务或回收过期任务。Scheduler 根据 Worker 队列容量，通过短事务领取任务并提交给 Worker。

启动时先注册全部 Handler，再注入 TaskRepository：

```go
controller := event.NewTaskController()
if err := controller.Register("post_card", cardHandler); err != nil {
    return err
}
repo := respository.NewTaskRepository(svcCtx.DB)
worker, err := event.NewWorker(4, repo, controller)
if err != nil {
    return err
}
go func() {
    if err := worker.Run(serviceCtx); err != nil &&
        !errors.Is(err, context.Canceled) {
        logx.Error(err)
    }
}()
<-worker.Ready()

scheduler, err := event.NewScheduler(repo, worker, event.DefaultSchedulerConfig())
if err != nil {
    return err
}
go func() {
    if err := scheduler.Run(serviceCtx); err != nil &&
        !errors.Is(err, context.Canceled) {
        logx.Error(err)
    }
}()

// Dispatcher 的事务提交成功后调用；不要在提交前通知。
scheduler.Notify()
```

`TaskController` 的回调表采用“启动时写入、运行时只读”的生命周期约束，因此不使用锁。所有 `Register` 必须在 `NewWorker` 和 `Run` 之前完成；Worker 开始运行后禁止继续注册或替换 Handler。

Run 只能调用一次，阻塞到 worker 全部退出；Done 可等待退出。Submit 是有界阻塞操作，可由传入 context 取消。禁止外部关闭内部队列。
取消 serviceCtx 会停止继续取任务并取消正在执行的任务。入队成功不等于任务成功；退出时未开始的任务等待调度器按数据库领取期限恢复。

Scheduler 必须是 Worker 队列的唯一生产者。Dispatcher 提交新增任务的事务后调用 `scheduler.Notify()`，Scheduler 会立即尝试领取；连续通知会合并，不会阻塞 Dispatcher。Worker 从队列取走任务时也会发送容量可用信号，让 Scheduler 继续填充队列。

通知只负责唤醒，数据库仍然是任务事实来源。即使服务在事务提交后、调用 `Notify` 前退出，兜底扫描也会找到任务。默认兜底扫描周期为 2 秒，领取失败后等待 1 秒，任务领取期限为 30 秒。Scheduler 使用 `QueueStats` 观察固定容量，每批领取数取剩余容量与 `MaxClaimBatch` 的较小值；队列满时暂停领取。领取期限必须覆盖最坏排队时间和 Handler 超时时间。

默认每个任务执行超时 10 秒，失败状态更新超时 3 秒。处理器必须响应 context，Go 不会强制结束不响应取消的函数。

## 事务

TaskRepository.Execute 锁定任务行，核对 claim_token、processing 状态和数据库领取期限；再读取 outbox 的真实 payload，避免依赖队列中的旧参数。
随后调用 Handler，并在同一事务里更新任务成功状态。Handler 的业务 Repository 必须使用回调传入的 session，不得换成普通连接、自行提交或执行长时间外部调用。

任务行锁一直保持到执行事务结束。回收器应 SKIP LOCKED 或等待后重新检查条件，不能无条件回收。
claim_token 检查失败时跳过处理；不能修改另一次领取。失败回滚后，独立事务安排重试。
同库业务和任务成功已经原子提交，同一个 delivery 的成功状态可阻止重复执行；跨任务或跨入口去重仍由 Handler 按需要维护 inbox。

## 错误

- nil：业务写入与 succeeded 一起提交。
- event.ErrSuperseded：处理器已确认被取代；与 superseded 状态一起提交。此分支应只做去重记录等操作，不得写入旧业务状态。
- 普通 error：回滚，按领取次数指数退避重试，加入随机错峰，上限五分钟；达到 max_attempts 后 failed。
- event.Permanent(err)：回滚并标记 failed。
- Handler panic：转换为不可重试错误，事务回滚；其他任务继续处理。
- 确认提交发生不确定错误：重试更新只允许原 claim 且仍 processing；已提交成功的任务不会被改回 pending。
- 更新重试状态也失败：记录错误，保留数据库状态，由调度器到期恢复。

领取时由 Scheduler 递增 attempt_count，Worker 不会重复递增。队列容量默认等于 worker 数量。事件分发、到期回收及具体业务 Handler 尚未接入程序启动入口。
