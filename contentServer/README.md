# contentServer

基于 go-zero 的 REST 服务基础工程。

## 启动

```powershell
$env:CONTENT_DB_DSN = '<内容服务专用账号>:<口令>@tcp(127.0.0.1:3309)/content_server?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai&timeout=5s&readTimeout=3s&writeTimeout=3s'
$env:CONTENT_ACCESS_SECRET = '<与用户服务一致的签发配置>'
go run . -f etc/content-api.yaml
```

`CONTENT_DB_DSN` 必须指向内容服务自己的 MySQL 实例和 `content_server`
数据库。不要填写用户服务的 `user_server` 连接地址；生产环境还应为内容服务使用独立数据库账号。

服务默认监听 `8889` 端口，可用下面的地址检查运行状态：

```text
GET http://localhost:8889/api/v1/health
```

预期响应：

```json
{"status":"ok"}
```

## 开发命令

```powershell
# 重新生成 API 代码
go tool goctl api go --api api/content.api --dir . --style go_zero

# 整理依赖
go mod tidy

# 运行测试
go test ./...

# 编译
go build -o bin/contentserver.exe .
```

接口定义位于 `api/content.api`，服务配置位于 `etc/content-api.yaml`。

## 内容与旅行方案拆分

ContentServer 仅管理普通图文帖子。旅行方案、地点节点、交通路线及节点说明由
TripServer 独立管理。内容服务没有路线引用，不需要旅行服务参与保存或发布。
当前仍以图片和文本为 MVP；本次没有增加视频上传或处理功能。

接口契约版本为 4.1，URL 前缀仍为 /api/v1。以下是破坏性调整，调用方需要同时更新：

- 删除创建、解除路线草稿的两个 /posts/:postId/route-draft 接口。
- 创建、保存、详情和卡片移除 presentationMode、hasRoute、route 等字段。
- 发布只检查 expectedPostVersion、expectedDraftVersion，不再接收 expectedRouteVersion。
- ContentBlock 保留 blockId、parentBlockId、blockType、sortOrder、title、text、assetIds；
  删除 placeId、bindings。parentBlockId 表示正文内部嵌套，与行程节点无关。
- 帖子列表删除 placeId 筛选，保留话题筛选和分页。
- 保存草稿删除 saveRequestId；使用 postId + expectedDraftVersion 做条件更新。
  旧版本重试与其他编辑导致的冲突统一返回 DraftVersionConflict（800302 / HTTP 409）。
  客户端保留本地编辑内容，重新获取草稿后再处理冲突，不自动替换版本号覆盖。
  创建帖子、发布和申请图片上传的请求标识继续保留。

SavePostDraftLogic 当前仍是模板；以上描述的是接口契约。
实现保存时必须在同一事务中按预期版本更新草稿并维护图片引用，
检查影响行数，为零时返回冲突（帖子不存在等情况由相应检查区分）。
goctl 默认生成的 Model.Update 仅按主键更新，不具备这个条件检查，
不能直接用它实现草稿的版本并发控制。

post、post_draft、post_revision 仍分别承担生命周期、当前编辑内容、发布版本职责。
发布版本用于让线上内容与未发布编辑分离，普通帖子同样需要这一能力。

## 数据库迁移与 Model

`schema/content.sql` 是当前最终表结构快照，供审阅和对比；实际安装和升级统一运行迁移脚本：

```powershell
./scripts/start-content-mysql.ps1
./scripts/migrate-content-db.ps1
./scripts/generate-content-models.ps1
```

历史 001、002 保留原样，003_detach_trip.sql 删除 post_draft、post_revision、
post_card_projection 的路线列与展示模式，并删除 post_revision_place。
其余 13 张业务表保留。schema_migration 是额外的迁移记录表，不属于业务领域。
004_remove_draft_save_request_id.sql 删除 post_draft.last_save_request_id，
保留草稿内容和 draft_version；已有请求标识只在迁移前备份中保留。
已移除错误码的数字继续保留，不能分配给其他含义。

迁移脚本在执行前备份到 .runtime/mysql/backups，记录文件名称和校验值；
重复执行会跳过已完成的迁移，不会重新添加旧字段。
执行迁移时应停止内容写入，并且只运行一个迁移进程。
MySQL 表结构操作逐步提交；003 支持中断后重试，不能依靠事务整体撤销。

003 面向当前空库：如果受影响的表已有内容，会停止执行。
已有作品必须先明确导出、转换和正文 JSON 中旧节点绑定的处理方案，
不能直接丢弃字段。恢复时停止写入，将备份先恢复到独立实例核对，
再切回配套旧版代码与数据库；备份后的新增数据需要另行处理。

可选的数据库集成测试：

```powershell
$env:CONTENT_MODEL_TEST_DSN = $env:CONTENT_DB_DSN
go test ./internal/model -run TestContentModelsAgainstMySQL -count=1
```

测试验证迁移后的字段及草稿、发布版本、卡片的实际读写，测试数据全部通过事务回滚。

## 业务响应约定

业务接口统一返回一个信封，具体业务字段只出现在 `data` 中：

```json
{
  "errorCode": 0,
  "message": "操作成功",
  "data": {},
  "requestId": ""
}
```

分层约定：

- `internal/logic` 只返回对应的 `*Data` 和 `error`，不组装响应码或提示文本。
- `internal/handler` 解析请求并调用 `writeBusinessResponse`。
- `internal/httpresponse` 统一生成成功或失败回包，并完成 HTTP 状态映射。
- 新增业务接口时必须分别定义 `XxxData` 和 `XxxResponse`；运行状态接口不使用业务信封。
