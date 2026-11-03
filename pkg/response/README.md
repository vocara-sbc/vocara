# response - Unified Response Wrapper Package

A unified HTTP response wrapper based on the Hertz framework, providing success and failure responses, paginated responses, and more.

## Features

- Unified response format (code, message, data)
- Seamless integration with the errors package
- Paginated response support
- Extensive convenience methods
- Automatic HTTP status code setting based on error codes

## Response Format

```json
{
    "code": 0,
    "message": "success",
    "data": {}
}
```

## Usage

### Success Responses

```go
import "vocara/pkg/response"

// With data
response.OK(c, data)

// Without data
response.OKNoContent(c)

// Alias
response.Success(c, data)
```

### Failure Responses

```go
import "vocara/pkg/errors"
import "vocara/pkg/response"

// Use predefined errors
response.Fail(c, errors.ErrParamError)
response.Fail(c, errors.ErrUnauthorized)
response.Fail(c, errors.ErrNotFound)
response.Fail(c, errors.ErrSystemError)

// Custom error code and message
response.FailWithMsg(c, 40001, "The name parameter is required")

// With details
response.FailWithDetail(c, errors.ErrParamError, "The name field must not exceed 50 characters")

// Automatically determine the error type
if err != nil {
    response.Error(c, err)
}
```

### Convenience Failure Methods

```go
// Parameter error
response.FailParamError(c, "Invalid parameters")

// Unauthorized
response.FailUnauthorized(c, "Please log in")

// Forbidden
response.FailForbidden(c, "Permission denied")

// Resource not found
response.FailNotFound(c, "User does not exist")

// System error
response.FailSystemError(c, "Service error")
```

### Paginated Responses

```go
// Paginated success response
response.PageSuccess(c, list, total, page, pageSize, pages)
// Or
response.PageOK(c, list, total, page, pageSize, pages)
```

Paginated response format:
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

### Using with the errors Package

```go
import "vocara/pkg/errors"
import "vocara/pkg/response"

func someHandler(c context.Context, ctx *app.RequestContext) {
    result, err := someService()
    if err != nil {
        // Automatically determine whether the error is a business error
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

### HTTP Status Code Mapping

The response package automatically sets the HTTP status code based on the error code:

| Error Code Range | HTTP Status Code |
|-----------|------------|
| 0 | 200 |
| 40000-40099 | 400 Bad Request |
| 40100-40199 | 401 Unauthorized |
| 40300-40399 | 403 Forbidden |
| 40400-40499 | 404 Not Found |
| 50000-59999 | 500 Internal Server Error |
| Other business codes | 200 |
