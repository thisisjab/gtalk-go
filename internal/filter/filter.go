package filter

import (
	"errors"
	"strings"

	"slices"

	"github.com/thisisjab/gchat-go/internal/validator"
)

type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafeList []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	// Check sorting only if both sort and sort safe list are provided.
	if len(f.SortSafeList) != 0 && f.Sort != "" {
		v.Check(validator.PermittedValue(f.Sort, f.SortSafeList...), "sort", "invalid sort value")
	}
}

func (f Filters) SortColumn() string {
	if slices.Contains(f.SortSafeList, f.Sort) {
		return strings.TrimPrefix(f.Sort, "-")
	}

	panic("unsafe sort param: " + f.Sort)
}

func (f Filters) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f Filters) Limit() int {
	return f.PageSize
}

func (f Filters) Offset() int {
	return (f.Page - 1) * f.PageSize
}

type PaginationMetadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

var InvalidPageError = errors.New("invalid page")

func CalculatePaginationMetadata(totalRecords, page, pageSize int) (*PaginationMetadata, error) {
	if totalRecords == 0 {
		if page == 1 {
			return &PaginationMetadata{
				CurrentPage:  page,
				PageSize:     pageSize,
				FirstPage:    1,
				LastPage:     1,
				TotalRecords: 0,
			}, nil
		}

		return nil, InvalidPageError

	}

	return &PaginationMetadata{
		CurrentPage: page,
		PageSize:    pageSize,
		FirstPage:   1,
		// We prefer this formula over total_records / page_size because
		// when total_records is not divisible by page_size, it gets rounded down and shows one less page.
		LastPage:     (totalRecords + pageSize - 1) / pageSize,
		TotalRecords: totalRecords,
	}, nil
}
