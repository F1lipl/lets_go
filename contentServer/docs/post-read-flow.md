# 已发布帖子查询

`GET /api/v1/posts/:postId` 由 `GetPostLogic` 编排，不使用跨多项读取的事务，也不做联表查询。先按主键读取 Post，领域方法 `CurrentPublicRevision` 判断当前生命周期、可见范围和可用状态；通过 `post.published_revision_id` 再按主键读取不可变 Revision，并校验其所属 Post。发布事务保证 Revision 与 Post 的当前版本指针同时提交，因此详情可按已读取的 Post 状态查找对应版本。随后 Logic 组装回包。

卡片投影是异步更新的，只用于列表/推荐页；详情请求传入 `postId`，不查询卡片。即使卡片尚未生成或仍指向旧版本，详情也会从 `post.published_revision_id` 获取当前版本。当前公开接口只返回可见范围为 public 且可用状态为 normal 的帖子；关注者范围需要未来接入关注关系判断。

正文块和封面保留资源 ID，不依赖资源引用表查询才能展示文字。图片元数据按这些 ID 批量读取，使用独立的短超时（配置项 `PostMediaLookupTimeoutMs`，默认 300 毫秒）。查询失败或超时仍返回正文，对应的 `mediaAssets[].status` 为 `unavailable`；查询成功但找不到资源时为 `missing`。话题快照从 `post_revision_tag` 读取，互动数字从 `post_stats` 读取，缺失的统计行按零处理。这两项目前仍是同步读取，失败会使整个接口失败，后续可按产品需要继续拆成独立接口。

资源访问地址目前没有生成能力，因此返回资源 ID 和尺寸等元数据，`thumbnailUrl`、`largeUrl` 暂为空。发布链路目前也未写入 `post_revision_tag`，所以新发布帖子的 `tags` 暂为空；这两项需分别在媒体交付和话题发布链路中补齐，不能把当前接口视为完整展示能力。
