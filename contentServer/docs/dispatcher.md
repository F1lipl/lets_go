# Dispatcher 使用

Dispatcher 负责将 `outbox_event` 扇出成每个订阅者一条 `event_delivery`。它不领取和执行任务。

```go
registry, err := event.NewConsumerRegistry(map[string][]string{
    "PostPublished": {
        "post_card",
        "post_search",
    },
})
if err != nil {
    return err
}

dispatchRepo := respository.NewDispatchRepository(svcCtx.DB)
dispatcher, err := event.NewDispatcher(
    dispatchRepo,
    registry,
    scheduler,
    event.DefaultDispatcherConfig(),
)
if err != nil {
    return err
}

go func() {
    if err := dispatcher.Run(serviceCtx); err != nil &&
        !errors.Is(err, context.Canceled) {
        logx.Error(err)
    }
}()
```

业务事务成功写入 outbox 并提交后，调用：

```go
dispatcher.Notify()
```

`Notify` 只是低延迟提示，连续提示会合并。默认每 2 秒扫描一次作为恢复手段，因此提交成功后即使没有发出提示，事件也不会遗留。

分发事务会锁定一批待处理 outbox，查找初始化期注册的 Consumer，为每个 Consumer 插入一条 pending delivery，再将 outbox 标记为 dispatched。多个实例可以并行分发不同事件。未知事件类型会被标记为 dispatch failed，不会阻塞后续事件。事务提交后 Dispatcher 才会通知 Scheduler。
