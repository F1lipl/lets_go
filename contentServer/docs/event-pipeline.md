# EventPipeline 使用

`EventPipeline` 管理本地事件流水线的生命周期：一个 Dispatcher goroutine、一个 Scheduler goroutine、一个 Reclaimer goroutine，以及配置数量的 Worker goroutine。构造函数只组装依赖，`Run` 才启动后台工作。

```go
taskRepo := respository.NewTaskRepository(svcCtx.DB)
dispatchRepo := respository.NewDispatchRepository(svcCtx.DB)

registry, err := event.NewConsumerRegistry(map[string][]string{
    "PostPublished": {"post_card"},
})
if err != nil {
    return err
}

controller := event.NewTaskController()
if err := controller.Register("post_card", postCardHandler); err != nil {
    return err
}

pipeline, err := event.NewEventPipeline(
    dispatchRepo,
    taskRepo,
    registry,
    controller,
    event.DefaultEventPipelineConfig(),
)
if err != nil {
    return err
}

go func() {
    if err := pipeline.Run(serviceCtx); err != nil &&
        !errors.Is(err, context.Canceled) {
        logx.Error(err)
    }
}()

<-pipeline.Ready()
```

写入 outbox 的事务成功返回后调用：

```go
pipeline.NotifyOutboxCommitted()
```

流水线内部字段不导出。外部不能绕过 Dispatcher 或 Scheduler 直接投递任务，只能启动完整流水线、等待生命周期信号以及发送 outbox 已提交提示。

## 过期任务恢复

Scheduler 领取任务时会将 delivery 改为 `processing`，写入唯一的 `claim_token` 和 `locked_until`。Reclaimer 启动时立即检查一次，之后默认每 5 秒检查一批已经超过领取期限的任务：

- `attempt_count < max_attempts`：清除旧领取信息，恢复为 `pending`，然后唤醒 Scheduler。
- `attempt_count >= max_attempts`：清除旧领取信息并标记为 `failed`。
- `locked_until` 尚未到期：保持不变。

恢复事务使用 `FOR UPDATE SKIP LOCKED`，多个服务实例可以共同扫描而不会处理同一行。`attempt_count` 在 Scheduler 再次领取时增加，Reclaimer 本身不重复增加次数。

默认值可以通过 `EventPipelineConfig.Reclaimer` 调整：

```go
config.Reclaimer = event.ReclaimerConfig{
    PollInterval: 5 * time.Second,
    ErrorBackoff: time.Second,
    BatchSize:    50,
}
```

当一次正好取满一批时，Reclaimer 会立即继续下一批；扫描不到更多任务后才等待下一个定时周期。数据库依然是任务状态的最终依据，定时检查不依赖进程内通知。
