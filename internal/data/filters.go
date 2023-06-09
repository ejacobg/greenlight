package data

import (
	"github.com/ejacobg/greenlight/internal/validator"
	"math"
	"strings"
)

type Filters struct {
	Page     int
	PageSize int
	Sort     string
	// SortSafelist defines all the valid values the Sort field can take.
	SortSafelist []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

// sortColumn will extract the column value the Sort term refers to.
// If the Sort term is invalid, this routine will panic.
func (f Filters) sortColumn() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}
	// The ValidateFilters call will check for correctness, but this is here just in case.
	panic("unsafe sort parameter: " + f.Sort)
}

// sortDirection will return the appropriate direction keyword depending on if the '-' character is present.
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

func (f Filters) limit() int {
	return f.PageSize
}
func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

// Metadata holds extra fields that will be returned for paginated requests.
type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		// Note that we return an empty Metadata struct if there are no records.
		return Metadata{}
	}
	return Metadata{
		CurrentPage: page,
		PageSize:    pageSize,
		FirstPage:   1,
		// The number of pages is the total number of records divided by the page size, rounded up.
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
}
