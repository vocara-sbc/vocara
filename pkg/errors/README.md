# errors - Error Code Wrapper Package

Custom business error types containing an error code, error message, and HTTP status code, with support for error chains and error inspection.

## Features

- Custom business error `BizError` containing an error code, message, and details
- Automatic HTTP status code inference from the error code
- Supports error inspection with `errors.Is` / `errors.As`
- Supports error wrapping (Wrap) while preserving the original error
- Extensive predefined error constants
- Error code conventions organized by business domain

## Error Code Conventions

| Error Code Range | Category | Corresponding HTTP Status Code |
|-----------|------|----------------|
| 0 | Success | 200 |
| 40000-40099 | Parameter errors | 400 |
| 40100-40199 | Unauthorized | 401 |
| 40300-40399 | Forbidden | 403 |
| 40400-40499 | Resource not found | 404 |
| 40900-40999 | Resource conflict | 409 |
| 42900-42999 | Too many requests | 429 |
| 50000-59999 | System errors | 500 |
| 10000-10099 | Agent-related | 200 |
| 20000-20099 | Queue-related | 200 |
| 30000-30099 | Call-related | 200 |
| 35000-35099 | ESL-related | 200 |
| 60000-60099 | NATS-related | 200 |

## Usage

### Creating Business Errors

```go
import "vocara/pkg/errors"

// Simple creation
err := errors.NewBizError(errors.CodeParamError, "Invalid parameters")

// With detailed information
err := errors.NewBizError(errors.CodeParamError, "Invalid parameters").
    WithDetail("The name field is required")

// Formatted detailed information
err := errors.NewBizError(errors.CodeParamError, "Invalid parameters").
    WithDetailf("The %s field is required", "name")

// Custom HTTP status code
err := errors.NewBizErrorWithStatus(10001, "Custom error", 200)
```

### Wrapping Standard Errors

```go
// Wrap as a system error
result, err := db.Query()
if err != nil {
    return errors.Wrap(err)
}

// Wrap with a specified error code
if err != nil {
    return errors.WrapWithCode(err, errors.CodeDBError, "Database query failed")
}
```

### Inspecting Error Types

```go
// Check whether the error is a business error
bizErr, ok := errors.IsBizError(err)
if ok {
    // Handle the business error
    fmt.Println(bizErr.Code, bizErr.Message)
}

// Check the error code
if bizErr.Is(errors.CodeParamError) {
    // Handle the parameter error
}

// Use the standard library errors.As
var bizErr *errors.BizError
if errors.As(err, &bizErr) {
    // ...
}
```

### Using Predefined Errors

```go
// Return a predefined error directly
return errors.ErrParamError
return errors.ErrUnauthorized
return errors.ErrNotFound
return errors.ErrSystemError

// Add details to a predefined error
return errors.ErrParamError.WithDetail("A required parameter is missing")
```

### Common Predefined Errors

#### General Errors

| Constant | Error Code | Message |
|------|--------|------|
| `ErrSuccess` | 0 | success |
| `ErrParamError` | 40000 | Invalid parameters |
| `ErrUnauthorized` | 40100 | Unauthorized |
| `ErrForbidden` | 40300 | Forbidden |
| `ErrNotFound` | 40400 | Resource not found |
| `ErrSystemError` | 50000 | System error |
| `ErrDBError` | 50002 | Database error |
| `ErrRedisError` | 50003 | Redis error |
| `ErrTimeoutError` | 50005 | Request timeout |

#### Agent-Related

| Constant | Error Code | Message |
|------|--------|------|
| `ErrAgentNotFound` | 10000 | Agent does not exist |
| `ErrAgentNotCheckedIn` | 10001 | Agent is not checked in |
| `ErrAgentAlreadyCheckedIn` | 10002 | Agent is already checked in |
| `ErrAgentBusy` | 10003 | Agent is busy |
| `ErrAgentOffline` | 10004 | Agent is offline |
| `ErrAgentPasswordWrong` | 10006 | Incorrect password |
| `ErrAgentNoIdle` | 10010 | No idle agents available |

#### Queue-Related

| Constant | Error Code | Message |
|------|--------|------|
| `ErrQueueNotFound` | 20001 | Queue does not exist |
| `ErrQueueFull` | 20002 | Queue is full |
| `ErrSkillNotFound` | 20006 | Skill group does not exist |

#### Call-Related

| Constant | Error Code | Message |
|------|--------|------|
| `ErrCallNotFound` | 30001 | Call does not exist |
| `ErrCallEnded` | 30003 | Call has ended |
| `ErrCallBridgeFail` | 30005 | Call bridging failed |
| `ErrCallOriginateFail` | 30009 | Outbound call failed |

#### ESL-Related

| Constant | Error Code | Message |
|------|--------|------|
| `ErrESLNotConnected` | 35001 | ESL is not connected |
| `ErrESLConnectFail` | 35002 | ESL connection failed |

#### NATS-Related

| Constant | Error Code | Message |
|------|--------|------|
| `ErrNATSNotConnected` | 60001 | NATS is not connected |
| `ErrNATSPublishFail` | 60002 | NATS publish failed |
