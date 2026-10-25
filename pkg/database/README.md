# database - 数据库封装包

基于 GORM v2 + MySQL 的数据库封装，支持连接管理、连接池配置、自动迁移、分页查询、软删除、上下文透传。

## 功能特性

- 基于 GORM v2，支持 MySQL
- 完整的连接池配置
- 支持自动迁移（可选）
- 分页查询封装
- 上下文透传
- 全局单例 + 实例化两种使用方式

## 使用方法

### 初始化

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
    ConnMaxLifetime: 60,  // 分钟
    ConnMaxIdleTime: 10,  // 分钟
    AutoMigrate:     false,
    LogLevel:        "warn",
    SlowThreshold:   1000, // 毫秒
}

// 方式一：初始化全局连接
if err := database.Init(cfg); err != nil {
    log.Fatal(err)
}

// 方式二：创建独立连接
db, err := database.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### 基本 CRUD

```go
import "vocara/pkg/database"
import "gorm.io/gorm"

db := database.Get()
ctx := context.Background()

// 创建
user := &User{Name: "张三", Age: 25}
db.Create(ctx, user)

// 查询单条
var user User
db.First(ctx, &user, "id = ?", 1)

// 查询多条
var users []User
db.Find(ctx, &users, "age > ?", 20)

// 更新
db.Update(ctx, &user, "name", "李四")
db.Updates(ctx, &user, map[string]interface{}{
    "name": "李四",
    "age":  26,
})

// 删除
db.Delete(ctx, &user)

// 统计
count, _ := db.Count(ctx, &User{})
```

### 使用 GORM 原生方法

```go
// 获取底层 *gorm.DB
gormDB := db.DB

// 链式调用
var users []User
gormDB.Where("age > ?", 18).
    Order("created_at DESC").
    Limit(10).
    Find(&users)

// 带上下文
db.WithContext(ctx).Where("status = ?", 1).Find(&users)
```

### 分页查询

```go
import "vocara/pkg/database"

// 方式一：使用 Paginate 函数
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

// 方式二：带排序
result, err = database.PaginateWithOrder(
    ctx,
    db.WithContext(ctx).Model(&User{}),
    query,
    "created_at DESC",
    &users,
)
```

### 事务

```go
db := database.Get()

err := db.Transaction(ctx, func(tx *gorm.DB) error {
    // 在事务中执行操作
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    if err := tx.Create(&order).Error; err != nil {
        return err
    }
    return nil
})

if err != nil {
    // 事务已回滚
}
```

### 自动迁移

```go
// 配置中开启 AutoMigrate
cfg.AutoMigrate = true

// 初始化时传入模型
db, _ := database.New(cfg)
db.AutoMigrate(&User{}, &Order{}, &Product{})
```

### 软删除

GORM 原生支持软删除，在模型中使用 `gorm.DeletedAt` 字段即可：

```go
type User struct {
    ID        uint           `gorm:"primaryKey"`
    Name      string
    Age       int
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除字段
}
```

## 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| Host | 主机地址 | 127.0.0.1 |
| Port | 端口 | 3306 |
| User | 用户名 | root |
| Password | 密码 | - |
| Database | 数据库名 | vocara |
| Charset | 字符集 | utf8mb4 |
| MaxIdleConns | 最大空闲连接数 | 10 |
| MaxOpenConns | 最大打开连接数 | 100 |
| ConnMaxLifetime | 连接最大生命周期(分钟) | 60 |
| ConnMaxIdleTime | 连接最大空闲时间(分钟) | 10 |
| AutoMigrate | 自动迁移 | false |
| LogLevel | 日志级别: silent/error/warn/info | warn |
| SlowThreshold | 慢查询阈值(毫秒) | 1000 |
