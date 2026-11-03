# database - Database Wrapper Package

A database wrapper based on GORM v2 and MySQL, supporting connection management, connection pool configuration, automatic migration, paginated queries, soft deletes, and context propagation.

## Features

- Based on GORM v2, with MySQL support
- Comprehensive connection pool configuration
- Optional automatic migration
- Paginated query wrapper
- Context propagation
- Both global singleton and instance-based usage

## Usage

### Initialization

```go
import "vocara/pkg/config"
import "vocara/pkg/database"

cfg := config.MySQLConfig{
    Host:            "127.0.0.1",
    Port:            3306,
    User:            "root",
    Password:        "password",
    Database:        "vocara",
    Charset:         "utf8mb4",
    ParseTime:       true,
    Loc:             "Local",
    MaxIdleConns:    10,
    MaxOpenConns:    100,
    ConnMaxLifetime: 60,  // minutes
    ConnMaxIdleTime: 10,  // minutes
    AutoMigrate:     false,
    LogLevel:        "warn",
    SlowThreshold:   1000, // milliseconds
}

// Option 1: Initialize the global connection
if err := database.Init(cfg); err != nil {
    log.Fatal(err)
}

// Option 2: Create an independent connection
db, err := database.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### Basic CRUD

```go
import "vocara/pkg/database"
import "gorm.io/gorm"

db := database.Get()
ctx := context.Background()

// Create
user := &User{Name: "John Doe", Age: 25}
db.Create(ctx, user)

// Query a single record
var user User
db.First(ctx, &user, "id = ?", 1)

// Query multiple records
var users []User
db.Find(ctx, &users, "age > ?", 20)

// Update
db.Update(ctx, &user, "name", "Jane Smith")
db.Updates(ctx, &user, map[string]interface{}{
    "name": "Jane Smith",
    "age":  26,
})

// Delete
db.Delete(ctx, &user)

// Count
count, _ := db.Count(ctx, &User{})
```

### Using Native GORM Methods

```go
// Get the underlying *gorm.DB
gormDB := db.DB

// Chained calls
var users []User
gormDB.Where("age > ?", 18).
    Order("created_at DESC").
    Limit(10).
    Find(&users)

// With context
db.WithContext(ctx).Where("status = ?", 1).Find(&users)
```

### Paginated Queries

```go
import "vocara/pkg/database"

// Option 1: Use the Paginate function
query := database.PageQuery{
    Page:     1,
    PageSize: 20,
}

var users []User
db := database.Get()

result, err := database.Paginate(
    ctx,
    db.WithContext(ctx).Model(&User{}).Where("status = ?", 1),
    query,
    &users,
)

// result:
// {
//     Total: 100,
//     Page: 1,
//     PageSize: 20,
//     Pages: 5,
//     List: &users,
// }

// Option 2: With ordering
result, err = database.PaginateWithOrder(
    ctx,
    db.WithContext(ctx).Model(&User{}),
    query,
    "created_at DESC",
    &users,
)
```

### Transactions

```go
db := database.Get()

err := db.Transaction(ctx, func(tx *gorm.DB) error {
    // Perform operations within the transaction
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    if err := tx.Create(&order).Error; err != nil {
        return err
    }
    return nil
})

if err != nil {
    // The transaction is rolled back
}
```

### Automatic Migration

```go
// Enable AutoMigrate in the configuration
cfg.AutoMigrate = true

// Pass models during initialization
db, _ := database.New(cfg)
db.AutoMigrate(&User{}, &Order{}, &Product{})
```

### Soft Deletes

GORM supports soft deletes natively. Add a `gorm.DeletedAt` field to the model:

```go
type User struct {
    ID        uint           `gorm:"primaryKey"`
    Name      string
    Age       int
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"` // Soft delete field
}
```

## Configuration Reference

| Configuration Item | Description | Default Value |
|--------|------|--------|
| Host | Host address | 127.0.0.1 |
| Port | Port | 3306 |
| User | Username | root |
| Password | Password | - |
| Database | Database name | vocara |
| Charset | Character set | utf8mb4 |
| MaxIdleConns | Maximum number of idle connections | 10 |
| MaxOpenConns | Maximum number of open connections | 100 |
| ConnMaxLifetime | Maximum connection lifetime (minutes) | 60 |
| ConnMaxIdleTime | Maximum connection idle time (minutes) | 10 |
| AutoMigrate | Automatic migration | false |
| LogLevel | Log level: silent/error/warn/info | warn |
| SlowThreshold | Slow query threshold (milliseconds) | 1000 |
