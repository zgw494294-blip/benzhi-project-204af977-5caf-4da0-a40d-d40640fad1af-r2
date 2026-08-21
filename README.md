# 星测校准台

星测校准台面向天文台仪器工程师、观测值班员和质量复核员，提供从创建校准会话、登记标准样本、逐项录入测量，到偏差复核、返工补测、封存会话和生成校准证书的完整流程。

服务使用 Go 标准库提供 HTTP API 和响应式浏览器工作台。业务状态与审计事件保存到带 `schemaVersion` 的本地 JSON 账本，账本通过临时文件 `Sync`、原子 `Rename` 提交；审计事件使用哈希链校验完整性，封存证书保存摘要哈希并进入只读状态。读取证书时会基于账本快照重新校验会话摘要和审计哈希链，并在 API 与工作台中展示证书是否可验证及失败原因。

## 构建

```text
go build ./...
```

## 运行

默认监听 `http://localhost:8080`，账本写入 `data/ledger.json`：

```text
go run .
```

也可以使用 `--addr` 指定监听地址，使用 `--ledger` 指定本地账本路径：

```text
go run . --addr :8080 --ledger data/ledger.json
```

启动后打开浏览器访问 `http://localhost:8080/`，在会话列表中创建或继续校准会话。

会话列表支持按状态和设备编号精确筛选；同时提供时按 AND 条件匹配，未提供参数时返回全部会话并按更新时间倒序：

```text
GET /api/sessions?status=measuring&deviceID=AST-SELF
```

`status` 只接受 `draft`、`measuring`、`pending_review`、`rework`、`ready_to_seal` 和 `sealed`。筛选结果仍为 `CalibrationSession` 列表，不包含证书编号查询返回的证书扩展字段。

已封存会话可通过证书编号精确查询：

```text
GET /api/sessions?certificateNo=CAL-20260820-0001
```

查询只匹配已封存会话，结果中的 `certificate` 包含重新计算的 `summaryHash` 和 `verification`；`verification.status` 明确表示 `verified` 或 `invalid`，`auditVerified` 表示审计哈希链校验状态。无匹配时返回空的 `sessions` 数组。

## 测试

运行全部 Go 单元测试和 HTTP 适配测试：

```text
go test ./...
```

运行可自行结束的完整业务冒烟自检：

```text
go run . --selfcheck
```

自检会在临时账本中走通创建、顺序测量、幂等重试、版本冲突、返工补测、复核通过、封存、证书读取和审计完整性校验。
