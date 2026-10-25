# errors - 错误码封装包

自定义业务错误类型，包含错误码、错误消息、HTTP 状态码，支持错误链和错误判断。

## 功能特性

- 自定义业务错误 `BizError`，包含错误码、消息、详情
- 自动根据错误码推断 HTTP 状态码
- 支持 `errors.Is` / `errors.As` 错误判断
- 支持错误包装（Wrap）和原始错误保留
- 丰富的预定义错误常量
- 按业务分类的错误码规范

## 错误码规范

| 错误码范围 | 类别 | 对应 HTTP 状态码 |
|-----------|------|----------------|
| 0 | 成功 | 200 |
| 40000-40099 | 参数错误 | 400 |
| 40100-40199 | 未授权 | 401 |
| 40300-40399 | 禁止访问 | 403 |
| 40400-40499 | 资源不存在 | 404 |
| 40900-40999 | 资源冲突 | 409 |
| 42900-42999 | 请求过多 | 429 |
| 50000-59999 | 系统错误 | 500 |
| 10000-10099 | 座席相关 | 200 |
| 20000-20099 | 队列相关 | 200 |
| 30000-30099 | 通话相关 | 200 |
| 35000-35099 | ESL 相关 | 200 |
| 60000-60099 | NATS 相关 | 200 |

## 使用方法

### 创建业务错误

```go
import "vocara/pkg/errors"

// 简单创建
err := errors.NewBizError(errors.CodeParamError, "参数错误")

// 带详细信息
err := errors.NewBizError(errors.CodeParamError, "参数错误").
    WithDetail("字段 name 不能为空")

// 格式化详细信息
err := errors.NewBizError(errors.CodeParamError, "参数错误").
    WithDetailf("字段 %s 不能为空", "name")

// 自定义 HTTP 状态码
err := errors.NewBizErrorWithStatus(10001, "自定义错误", 200)
```

### 包装普通错误

```go
// 包装为系统错误
result, err := db.Query()
if err != nil {
    return errors.Wrap(err)
}

// 包装为指定错误码
if err != nil {
    return errors.WrapWithCode(err, errors.CodeDBError, "数据库查询失败")
}
```

### 判断错误类型

```go
// 判断是否为业务错误
bizErr, ok := errors.IsBizError(err)
if ok {
    // 处理业务错误
    fmt.Println(bizErr.Code, bizErr.Message)
}

// 判断错误码
if bizErr.Is(errors.CodeParamError) {
    // 参数错误处理
}

// 使用标准库 errors.As
var bizErr *errors.BizError
if errors.As(err, &bizErr) {
    // ...
}
```

### 使用预定义错误

```go
// 直接返回预定义错误
return errors.ErrParamError
return errors.ErrUnauthorized
return errors.ErrNotFound
return errors.ErrSystemError

// 基于预定义错误添加详情
return errors.ErrParamError.WithDetail("缺少必要参数")
```

### 常用预定义错误

#### 通用错误

| 常量 | 错误码 | 消息 |
|------|--------|------|
| `ErrSuccess` | 0 | success |
| `ErrParamError` | 40000 | 参数错误 |
| `ErrUnauthorized` | 40100 | 未授权 |
| `ErrForbidden` | 40300 | 禁止访问 |
| `ErrNotFound` | 40400 | 资源不存在 |
| `ErrSystemError` | 50000 | 系统错误 |
| `ErrDBError` | 50002 | 数据库错误 |
| `ErrRedisError` | 50003 | Redis 错误 |
| `ErrTimeoutError` | 50005 | 请求超时 |

#### 座席相关

| 常量 | 错误码 | 消息 |
|------|--------|------|
| `ErrAgentNotFound` | 10000 | 座席不存在 |
| `ErrAgentNotCheckedIn` | 10001 | 座席未签入 |
| `ErrAgentAlreadyCheckedIn` | 10002 | 座席已签入 |
| `ErrAgentBusy` | 10003 | 座席忙 |
| `ErrAgentOffline` | 10004 | 座席已离线 |
| `ErrAgentPasswordWrong` | 10006 | 密码错误 |
| `ErrAgentNoIdle` | 10010 | 无空闲座席 |

#### 队列相关

| 常量 | 错误码 | 消息 |
|------|--------|------|
| `ErrQueueNotFound` | 20001 | 队列不存在 |
| `ErrQueueFull` | 20002 | 队列已满 |
| `ErrSkillNotFound` | 20006 | 技能组不存在 |

#### 通话相关

| 常量 | 错误码 | 消息 |
|------|--------|------|
| `ErrCallNotFound` | 30001 | 通话不存在 |
| `ErrCallEnded` | 30003 | 通话已结束 |
| `ErrCallBridgeFail` | 30005 | 通话桥接失败 |
| `ErrCallOriginateFail` | 30009 | 外呼失败 |

#### ESL 相关

| 常量 | 错误码 | 消息 |
|------|--------|------|
| `ErrESLNotConnected` | 35001 | ESL 未连接 |
| `ErrESLConnectFail` | 35002 | ESL 连接失败 |

#### NATS 相关

| 常量 | 错误码 | 消息 |
|------|--------|------|
| `ErrNATSNotConnected` | 60001 | NATS 未连接 |
| `ErrNATSPublishFail` | 60002 | NATS 发布失败 |
