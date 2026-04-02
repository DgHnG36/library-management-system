package common_dto

type PaginationDTO struct {
	Page   int32  `form:"page" json:"page"`
	Limit  int32  `form:"limit" json:"limit"`
	SortBy string `form:"sort_by" json:"sort_by"`
	IsDesc bool   `form:"is_desc" json:"is_desc"`
}
