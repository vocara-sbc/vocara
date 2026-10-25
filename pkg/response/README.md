# response - 统一响应封装包

基于 Hertz 框架的统一 HTTP 响应封装，提供成功/失败响应、分页响应等。

## 功能特性

- 统一的响应格式（code, message, data）
- 与 errors 包无缝集成
- 支持分页响应
- 丰富的快捷方法
- 自动根据错误码设置 HTTP 状态码

## 响应格式

```json
{
    "code": 0,
    "message": "success",
    "data": {}
}
```

## 使用方法

### 成功响应

```go
import "vocara/pkg/response"

// 带数据
response.OK(c, data)

// 无数据
response.OKNoContent(c)

// 别名
response.Success(c, data)
```

### 失败响应

```go
import "vocara/pkg/errors"
import "vocara/pkg/response"

// 使用预定义错误
response.Fail(c, errors.ErrParamError)
response.Fail(c, errors.ErrUnauthorized)
response.Fail(c, errors.ErrNotFound)
response.Fail(c, errors.ErrSystemError)

// 自定义错误码和消息
response.FailWithMsg(c, 40001, "参数 name 不能为空")

// 带详情
response.FailWithDetail(c, errors.ErrParamError, "name 字段长度不能超过 50")

// 自动判断错误类型
if err != nil {
    response.Error(c, err)
}
```

### 快捷失败方法

```go
// 参数错误
response.FailParamError(c, "参数无效")

// 未授权
response.FailUnauthorized(c, "请先登录")

// 禁止访问
response.FailForbidden(c, "无权限")

// 资源不存在
response.FailNotFound(c, "用户不存在")

// 系统错误
response.FailSystemError(c, "服务异常")
```

### 分页响应

```go
// 分页成功响应
response.PageSuccess(c, list, total, page, pageSize, pages)
// 或
response.PageOK(c, list, total, page, pageSize, pages)
```

分页响应格式：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [...],
        "total": 100,
        "page": 1,
        "page_size": 20,
        "pages": 5
    }
}
```

### 与 errors 包配合使用

```go
import "vocara/pkg/errors"
import "vocara/pkg/response"

func someHandler(c context.Context, ctx *app.RequestContext) {
    result, err := someService()
    if err != nil {
        // 自动判断是否为业务错误
        response.Error(ctx, err)
        return
    }
    response.OK(ctx, result)
}

func someService() (*Result, error) {
    if invalid {
        return nil, errors.ErrParamError
    }
    if dbErr != nil {
        return nil, errors.Wrap(dbErr)
    }
    return result, nil
}
```

### HTTP 状态码映射

response 包会自动根据错误码设置 HTTP 状态码：

| 错误码范围 | HTTP 状态码 |
|-----------|------------|
| 0 | 200 |
| 40000-40099 | 400 Bad Request |
| 40100-40199 | 401 Unauthorized |
| 40300-40399 | 403 Forbidden |
| 40400-40499 | 404 Not Found |
| 50000-59999 | 500 Internal Server Error |
| 其他业务码 | 200 |
