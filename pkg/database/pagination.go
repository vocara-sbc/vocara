package database

import (
	"context"

	"gorm.io/gorm"
)

// PageQuery contains pagination query parameters.
type PageQuery struct {
	Page     int `form:"page" json:"page"`         // Page number, starting from 1
	PageSize int `form:"page_size" json:"page_size"` // Number of items per page
}

// PageResult contains paginated query results.
type PageResult struct {
	Total    int64       `json:"total"`     // Total record count
	Page     int         `json:"page"`      // Current page number
	PageSize int         `json:"page_size"` // Number of items per page
	Pages    int         `json:"pages"`     // Total page count
	List     interface{} `json:"list"`      // Data list
}

// DefaultPageSize is the default number of items per page.
const DefaultPageSize = 20

// MaxPageSize is the maximum number of items per page.
const MaxPageSize = 200

// Normalize normalizes pagination parameters.
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

// Offset calculates the offset.
func (p *PageQuery) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Paginate performs a paginated query.
// db: query conditions (a *gorm.DB with where and other conditions applied)
// query: pagination parameters
// dest: pointer to the result slice
func Paginate(ctx context.Context, db *gorm.DB, query PageQuery, dest interface{}) (*PageResult, error) {
	query.Normalize()

	// Count the total
	var total int64
	if err := db.WithContext(ctx).Count(&total).Error; err != nil {
		return nil, err
	}

	// Return immediately if the total is 0
	if total == 0 {
		return &PageResult{
			Total:    0,
			Page:     query.Page,
			PageSize: query.PageSize,
			Pages:    0,
			List:     dest,
		}, nil
	}

	// Execute the paginated query
	err := db.WithContext(ctx).
		Limit(query.PageSize).
		Offset(query.Offset()).
		Find(dest).Error
	if err != nil {
		return nil, err
	}

	// Calculate the total page count
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

// PaginateWithOrder performs a paginated query with ordering.
func PaginateWithOrder(ctx context.Context, db *gorm.DB, query PageQuery, order string, dest interface{}) (*PageResult, error) {
	if order != "" {
		db = db.Order(order)
	}
	return Paginate(ctx, db, query, dest)
}

// PaginateByDB performs a paginated query using the global DB.
func PaginateByDB(ctx context.Context, db *DB, model interface{}, query PageQuery, dest interface{}) (*PageResult, error) {
	return Paginate(ctx, db.WithContext(ctx).Model(model), query, dest)
}
