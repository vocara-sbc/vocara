package database

import (
	"context"

	"gorm.io/gorm"
)

// PageQuery 分页查询参数
type PageQuery struct {
	Page     int `form:"page" json:"page"`         // 页码，从 1 开始
	PageSize int `form:"page_size" json:"page_size"` // 每页数量
}

// PageResult 分页查询结果
type PageResult struct {
	Total    int64       `json:"total"`     // 总记录数
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"page_size"` // 每页数量
	Pages    int         `json:"pages"`     // 总页数
	List     interface{} `json:"list"`      // 数据列表
}

// DefaultPageSize 默认每页数量
const DefaultPageSize = 20

// MaxPageSize 最大每页数量
const MaxPageSize = 200

// Normalize 规范化分页参数
func (p *PageQuery) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// Offset 计算偏移量
func (p *PageQuery) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Paginate 分页查询
// db: 查询条件（已经加了 where 等条件的 *gorm.DB）
// query: 分页参数
// dest: 结果切片指针
func Paginate(ctx context.Context, db *gorm.DB, query PageQuery, dest interface{}) (*PageResult, error) {
	query.Normalize()

	// 统计总数
	var total int64
	if err := db.WithContext(ctx).Count(&total).Error; err != nil {
		return nil, err
	}

	// 如果总数为 0，直接返回
	if total == 0 {
		return &PageResult{
			Total:    0,
			Page:     query.Page,
			PageSize: query.PageSize,
			Pages:    0,
			List:     dest,
		}, nil
	}

	// 分页查询
	err := db.WithContext(ctx).
		Limit(query.PageSize).
		Offset(query.Offset()).
		Find(dest).Error
	if err != nil {
		return nil, err
	}

	// 计算总页数
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize != 0 {
		pages++
	}

	return &PageResult{
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
		Pages:    pages,
		List:     dest,
	}, nil
}

// PaginateWithOrder 分页查询（带排序）
func PaginateWithOrder(ctx context.Context, db *gorm.DB, query PageQuery, order string, dest interface{}) (*PageResult, error) {
	if order != "" {
		db = db.Order(order)
	}
	return Paginate(ctx, db, query, dest)
}

// PaginateByDB 使用全局 DB 进行分页查询
func PaginateByDB(ctx context.Context, db *DB, model interface{}, query PageQuery, dest interface{}) (*PageResult, error) {
	return Paginate(ctx, db.WithContext(ctx).Model(model), query, dest)
}
