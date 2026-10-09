# 已发布帖子查询

`GET /api/v1/posts/:postId` 不开启跨读取事务，也不做联表查询。先按主键读取 Post，并由 `CurrentPublicRevision` 判断生命周期、可见范围和可用状态；然后按当前版本 ID 读取不可变 Revision，并核对其所属 Post。发布事务保证版本与当前指针同时提交。详情不依赖异步的卡片投影。

正文在保存草稿时验证结构，发布时复制为不可变快照。详情将 `document_json` 作为 `json.RawMessage` 原样输出为 JSON 对象，不在读取链路反序列化块列表。封面从 Revision 返回资源 ID，正文块保留资源 ID；详情不等待图片元数据，页面可先显示文字。资源 ID 转展示地址仍需要独立的资源读取能力。

发布事务同步写入 `post_revision_tag`，详情按版本读取话题快照。话题和互动数字是附加信息，共用最多 `PostSupplementLookupTimeoutMs`（默认 200 毫秒）的读取预算；任一读取失败不影响正文返回。`tagsStatus` 和 `statsStatus` 为 `ready` 或 `unavailable`。互动统计行不存在时 `statsStatus=unavailable`，数值字段暂为 0，不误称为已确认的零。当前统计写入链路尚未接通，因此该状态通常为 `unavailable`。

当前公开接口仅返回可见范围 public 且可用状态 normal 的帖子。关注者范围需要未来接入关注关系判断。
