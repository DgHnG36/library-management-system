package book_service_dto

import "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/common_dto"

type BookDTO struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Author            string `json:"author"`
	ISBN              string `json:"isbn"`
	Category          string `json:"category"`
	Description       string `json:"description"`
	TotalQuantity     int32  `json:"total_quantity"`
	AvailableQuantity int32  `json:"available_quantity"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type BookResponseDTO struct {
	Book *BookDTO `json:"book"`
}

type GetBookRequestDTO struct {
	Identifier string `uri:"id" binding:"required"`
}

type ListBooksRequestDTO struct {
	Pagination  *common_dto.PaginationDTO `form:",inline"`
	SearchQuery string                    `form:"search_query"`
	Category    string                    `form:"category"`
}

type ListBooksResponseDTO struct {
	Books      []*BookDTO `json:"books"`
	TotalCount int32      `json:"total_count"`
}

type CreateBookPayload struct {
	Title       string `json:"title" binding:"required"`
	Author      string `json:"author" binding:"required"`
	ISBN        string `json:"isbn" binding:"required"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	Quantity    int32  `json:"quantity" binding:"required"`
}

type CreateBooksRequestDTO struct {
	BookPayloads []*CreateBookPayload `json:"books_payload" binding:"required,dive"`
}

type CreateBooksResponseDTO struct {
	CreatedBooks []*BookDTO `json:"created_books"`
	SuccessCount int32      `json:"success_count"`
}

type UpdateBookRequestDTO struct {
	ID          string `uri:"id" binding:"required"`
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	ISBN        string `json:"isbn,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
}

type DeleteBooksRequestDTO struct {
	BookIDs []string `json:"book_ids" binding:"required,dive,required"`
}

type CheckAvailabilityRequestDTO struct {
	BookID string `uri:"id" binding:"required"`
}

type CheckAvailabilityResponseDTO struct {
	IsAvailable       bool  `json:"is_available"`
	AvailableQuantity int32 `json:"available_quantity"`
}

type UpdateBookQuantityRequestDTO struct {
	BookID       string `uri:"id" binding:"required"`
	ChangeAmount int32  `json:"change_amount" binding:"required"`
}

type UpdateBookQuantityResponseDTO struct {
	NewAvailableQuantity int32 `json:"new_available_quantity"`
}
