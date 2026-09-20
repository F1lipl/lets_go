# Worker 使用

Worker 只消费已经领取并提交到 event_delivery 的任务。它不负责扫描 outbox、创建任务、领取任务或回收过期任务；这些仍属于调度器。

启动时先注册 Handler，再注入 TaskRepository：

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
// 调度器成功提交领取事务后：
err = worker.Submit(serviceCtx, event.ClaimedTask{
    Context: event.TaskContext{
        EventId: eventID,
        ConsumerName: "post_card",
    },
    ClaimToken: claimToken,
    LockedUntil: lockedUntil,
})
```

Run 只能调用一次，阻塞到 worker 全部退出；Done 可等待退出。Submit 是有界阻塞操作，可由传入 context 取消。禁止外部关闭内部队列。
取消 serviceCtx 会停止继续取任务并取消正在执行的任务。入队成功不等于任务成功；退出时未开始的任务等待调度器按数据库领取期限恢复。

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

领取时由调度器递增 attempt_count，本模块不会重复递增。队列容量默认等于 worker 数量，领取期限要覆盖排队与执行时间。领取、回收、订阅分发及具体业务 Handler 尚未接入程序启动入口。
