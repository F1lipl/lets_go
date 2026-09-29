# 发布流程

入口：现有 PublishPost API → PublishPostLogic → PostRepository.PublishPost。

- Logic 读取操作者、解析 ID、检查请求版本，在 Repository 回调里调用 Post.Publish。
- Repository 开启事务，依次锁定 Post、Draft，各读取一次并恢复领域对象。
- Post.Publish 校验作者、生命周期、可用状态和草稿条件，构造新版本并更新内存状态。正文保持 JSON 字符串，不重新解析或编码。
- Repository 校验资源当前状态，写入 Revision、Post、版本资源引用和 PostPublished outbox 事件。全部使用同一个事务连接。
- 只有事务提交成功才返回发布结果。失败不返回回调中产生的对象，也不自动重试发布。

## 草稿保存约定

新正文的格式、结构和资源引用应在保存草稿时校验。正文、统计信息、资源关系和 draft_version 必须在同一事务里更新。

发布检查正文图片数量与 usage_type=2 的草稿资源引用数量一致，并检查每条 source_version 与 draft_version 一致。不一致时返回 DraftContentInvalid，不发布缺少资源引用的快照。

当前保留原业务已有的封面必填规则。封面及已登记的正文资源必须属于作者、状态为 ready 且未删除。

其他同时修改 Post、Draft 的操作应保持先 Post 后 Draft 的锁定顺序。资源删除流程应在检查引用和改变资源状态时持有相应资源行锁。

## 事件边界

发布事务写入 outbox 后，Logic 在提交成功时唤醒 EventPipeline 的 Dispatcher；Dispatcher 将 PostPublished 拆分为 post_card delivery，Scheduler 领取任务，Worker 在一个事务里执行 PostCardHandler 并将 delivery 标记完成。进程重启或通知丢失时，定时扫描继续处理数据库中未完成的事件与任务。

PostPublished 的 payload 包含 schemaVersion、postId、authorId、revisionId、revisionNumber、sourceDraftVersion、postVersion、tagNames；发生时间在 occurred_at。
tagNames 是发布时复制的值。消费者必须读取指定 Revision，不得读取后续变化的 Draft。

PostCardHandler 只读取当前 Post 和指定 Revision 的卡片字段，不加载正文。它锁定 Post 行、校验当前发布版本，按 postVersion 避免旧事件覆盖新结果，并在与 delivery 完成标记相同的事务里更新 post_card_projection。过期版本任务标记为 superseded。删除 Post 时，同一事务删除卡片；尚未处理的旧发布事件不能将其重新创建。

当前还没有实现话题检索、搜索、推荐、通知消费者，也没有实现卡片列表及批量读取 API。资源 URL、互动计数、作者资料等需要在读取链路中另行组合；投影只保存内容服务拥有的卡片字段。
删除 outbox 历史前，应确保话题快照等下游数据已经可靠保存。

## 验证

go test ./...

设置 CONTENT_MODEL_TEST_DSN 为独立测试数据库连接后运行：

go test ./internal/logic ./internal/model ./internal/taskhandler -count=1

集成测试覆盖首次发布、再次发布、旧快照保留、版本冲突、作者不符、删除状态、资源未就绪、资源关系缺失、并发发布和 outbox 写入失败的整笔回滚；卡片测试覆盖首次投影、重复投递、旧版本事件和删除后不重建。测试仅清理自己生成的 ID 对应数据。
