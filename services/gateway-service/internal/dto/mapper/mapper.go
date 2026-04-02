package mapper

import (
	"strings"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/book_service_dto"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/order_service_dto"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/user_service_dto"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/user_token_dto"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
)

type MapperInterface interface {
	/* USER SERVICE MAPPER */
	MapPbRegisterRequest(*user_service_dto.RegisterRequestDTO) *userv1.RegisterRequest
	MapPbLoginRequest(*user_service_dto.LoginRequestDTO, bool) *userv1.LoginRequest
	MapPbGetProfileRequest(*user_service_dto.GetProfileRequestDTO) *userv1.GetProfileRequest
	MapPbUpdateProfileRequest(*user_service_dto.UpdateProfileRequestDTO) *userv1.UpdateProfileRequest
	MapPbUpdateVIPAccountRequest(*user_service_dto.UpdateVIPAccountRequestDTO) *userv1.UpdateVIPAccountRequest
	MapPbListUsersRequest(*user_service_dto.ListUsersRequestDTO) *userv1.ListUsersRequest
	MapPbDeleteUsersRequest(*user_service_dto.DeleteUsersRequestDTO) *userv1.DeleteUsersRequest

	MapDTORegisterResponse(*userv1.RegisterResponse) *user_service_dto.RegisterResponseDTO
	MapDTOLoginResponse(*userv1.LoginResponse) *user_service_dto.LoginResponseDTO
	MapDTOGetProfileResponse(*userv1.UserProfileResponse) *user_service_dto.UserProfileResponseDTO
	MapDTOUpdateProfileResponse(*userv1.UserProfileResponse) *user_service_dto.UserProfileResponseDTO
	MapDTOUpdateVIPAccountResponse(*userv1.UpdateVIPAccountResponse) *user_service_dto.UpdateVIPAccountResponseDTO
	MapDTOListUsersResponse(*userv1.ListUsersResponse) *user_service_dto.ListUsersResponseDTO

	/* BOOK SERVICE MAPPER */
	MapPbGetBookRequest(*book_service_dto.GetBookRequestDTO, bool) *bookv1.GetBookRequest
	MapPbListBooksRequest(*book_service_dto.ListBooksRequestDTO) *bookv1.ListBooksRequest
	MapPbCreateBooksRequest(*book_service_dto.CreateBooksRequestDTO) *bookv1.CreateBooksRequest
	MapPbUpdateBookRequest(*book_service_dto.UpdateBookRequestDTO) *bookv1.UpdateBookRequest
	MapPbDeleteBooksRequest(*book_service_dto.DeleteBooksRequestDTO) *bookv1.DeleteBooksRequest
	MapPbCheckBookAvailabilityRequest(*book_service_dto.CheckAvailabilityRequestDTO) *bookv1.CheckAvailabilityRequest
	MapPbUpdateBookQuantityRequest(*book_service_dto.UpdateBookQuantityRequestDTO) *bookv1.UpdateBookQuantityRequest

	MapDTOGetBookResponse(*bookv1.BookResponse) *book_service_dto.BookResponseDTO
	MapDTOListBooksResponse(*bookv1.ListBooksResponse) *book_service_dto.ListBooksResponseDTO
	MapDTOCreateBooksResponse(*bookv1.CreateBooksResponse) *book_service_dto.CreateBooksResponseDTO
	MapDTOUpdateBookResponse(*bookv1.BookResponse) *book_service_dto.BookResponseDTO
	MapDTOCheckBookAvailabilityResponse(*bookv1.CheckAvailabilityResponse) *book_service_dto.CheckAvailabilityResponseDTO
	MapDTOUpdateBookQuantityResponse(*bookv1.UpdateBookQuantityResponse) *book_service_dto.UpdateBookQuantityResponseDTO

	/* ORDER SERVICE MAPPER */
	MapPbCreateOrderRequest(*order_service_dto.CreateOrderRequestDTO) *orderv1.CreateOrderRequest
	MapPbGetOrderRequest(*order_service_dto.GetOrderRequestDTO) *orderv1.GetOrderRequest
	MapPbListMyOrdersRequest(*order_service_dto.ListMyOrdersRequestDTO) *orderv1.ListMyOrdersRequest
	MapPbCancelOrderRequest(*order_service_dto.CancelOrderRequestDTO) *orderv1.CancelOrderRequest
	MapPbListAllOrdersRequest(*order_service_dto.ListAllOrdersRequestDTO) *orderv1.ListAllOrdersRequest
	MapPbUpdateOrderStatusRequest(*order_service_dto.UpdateOrderStatusRequestDTO) *orderv1.UpdateOrderStatusRequest

	MapDTOCreateOrderResponse(*orderv1.OrderResponse) *order_service_dto.OrderResponseDTO
	MapDTOGetOrderResponse(*orderv1.OrderResponse) *order_service_dto.OrderResponseDTO
	MapDTOListMyOrdersResponse(*orderv1.ListOrdersResponse) *order_service_dto.ListOrdersResponseDTO
	MapDTOCancelOrderResponse(*orderv1.OrderResponse) *order_service_dto.OrderResponseDTO
	MapDTOListAllOrdersResponse(*orderv1.ListOrdersResponse) *order_service_dto.ListOrdersResponseDTO
	MapDTOUpdateOrderStatusResponse(*orderv1.OrderResponse) *order_service_dto.OrderResponseDTO
}

type Mapper struct{}

func NewMapper() MapperInterface {
	return &Mapper{}
}

/* USER SERVICE MAPPER */
func (m *Mapper) MapPbRegisterRequest(req *user_service_dto.RegisterRequestDTO) *userv1.RegisterRequest {
	if req == nil {
		return &userv1.RegisterRequest{}
	}

	return &userv1.RegisterRequest{
		Username:    req.Username,
		Password:    req.Password,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
	}
}

func (m *Mapper) MapPbLoginRequest(req *user_service_dto.LoginRequestDTO, isEmail bool) *userv1.LoginRequest {
	if req == nil {
		return &userv1.LoginRequest{}
	}

	if isEmail {
		return &userv1.LoginRequest{
			Identifier: &userv1.LoginRequest_Email{
				Email: req.Identifier,
			},
			Password: req.Password,
		}
	}
	return &userv1.LoginRequest{
		Identifier: &userv1.LoginRequest_Username{
			Username: req.Identifier,
		},
		Password: req.Password,
	}
}

func (m *Mapper) MapPbGetProfileRequest(req *user_service_dto.GetProfileRequestDTO) *userv1.GetProfileRequest {
	if req == nil {
		return &userv1.GetProfileRequest{}
	}

	return &userv1.GetProfileRequest{
		Id: req.UserID,
	}
}

func (m *Mapper) MapPbUpdateProfileRequest(req *user_service_dto.UpdateProfileRequestDTO) *userv1.UpdateProfileRequest {
	if req == nil {
		return &userv1.UpdateProfileRequest{}
	}

	return &userv1.UpdateProfileRequest{
		Id:          req.UserID,
		Username:    req.Username,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
	}
}

func (m *Mapper) MapPbUpdateVIPAccountRequest(req *user_service_dto.UpdateVIPAccountRequestDTO) *userv1.UpdateVIPAccountRequest {
	if req == nil {
		return &userv1.UpdateVIPAccountRequest{}
	}

	return &userv1.UpdateVIPAccountRequest{
		Id:    req.UserID,
		IsVip: req.IsVIP,
	}
}

func (m *Mapper) MapPbListUsersRequest(req *user_service_dto.ListUsersRequestDTO) *userv1.ListUsersRequest {
	if req == nil {
		return &userv1.ListUsersRequest{}
	}

	return &userv1.ListUsersRequest{
		Pagination: &commonv1.PaginationRequest{
			Page:   req.Pagination.Page,
			Limit:  req.Pagination.Limit,
			SortBy: req.Pagination.SortBy,
			IsDesc: req.Pagination.IsDesc,
		},
		Role: m.toPbRole(req.RoleFilter),
	}
}

func (m *Mapper) MapPbDeleteUsersRequest(req *user_service_dto.DeleteUsersRequestDTO) *userv1.DeleteUsersRequest {
	if req == nil {
		return &userv1.DeleteUsersRequest{}
	}

	return &userv1.DeleteUsersRequest{
		Ids: req.UserIDs,
	}
}

func (m *Mapper) MapDTORegisterResponse(resp *userv1.RegisterResponse) *user_service_dto.RegisterResponseDTO {
	if resp == nil {
		return &user_service_dto.RegisterResponseDTO{}
	}

	return &user_service_dto.RegisterResponseDTO{
		UserID: resp.GetUserId(),
	}
}

func (m *Mapper) MapDTOLoginResponse(resp *userv1.LoginResponse) *user_service_dto.LoginResponseDTO {
	if resp == nil {
		return &user_service_dto.LoginResponseDTO{}
	}

	return &user_service_dto.LoginResponseDTO{
		TokenPair: &user_token_dto.TokenPairDTO{
			AccessToken:  resp.AccessToken,
			RefreshToken: resp.RefreshToken,
		},
		User: m.toDTOUser(resp.GetUser()),
	}
}

func (m *Mapper) MapDTOGetProfileResponse(resp *userv1.UserProfileResponse) *user_service_dto.UserProfileResponseDTO {
	if resp == nil {
		return &user_service_dto.UserProfileResponseDTO{}
	}

	return &user_service_dto.UserProfileResponseDTO{
		User: m.toDTOUser(resp.GetUser()),
	}
}

func (m *Mapper) MapDTOUpdateProfileResponse(resp *userv1.UserProfileResponse) *user_service_dto.UserProfileResponseDTO {
	if resp == nil {
		return &user_service_dto.UserProfileResponseDTO{}
	}

	return &user_service_dto.UserProfileResponseDTO{
		User: m.toDTOUser(resp.GetUser()),
	}
}

func (m *Mapper) MapDTOUpdateVIPAccountResponse(resp *userv1.UpdateVIPAccountResponse) *user_service_dto.UpdateVIPAccountResponseDTO {
	if resp == nil {
		return &user_service_dto.UpdateVIPAccountResponseDTO{}
	}

	return &user_service_dto.UpdateVIPAccountResponseDTO{
		CurrentVIPStatus: resp.GetCurrentVipStatus(),
	}
}

func (m *Mapper) MapDTOListUsersResponse(resp *userv1.ListUsersResponse) *user_service_dto.ListUsersResponseDTO {
	if resp == nil {
		return &user_service_dto.ListUsersResponseDTO{}
	}

	users := make([]*user_service_dto.UserDTO, 0, len(resp.GetUsers()))
	for _, user := range resp.GetUsers() {
		users = append(users, m.toDTOUser(user))
	}

	return &user_service_dto.ListUsersResponseDTO{
		Users:      users,
		TotalCount: resp.GetTotalCount(),
	}
}

/* BOOK SERVICE MAPPER */
func (m *Mapper) MapPbGetBookRequest(req *book_service_dto.GetBookRequestDTO, isTitle bool) *bookv1.GetBookRequest {
	if req == nil {
		return &bookv1.GetBookRequest{}
	}

	if isTitle {
		return &bookv1.GetBookRequest{
			Identifier: &bookv1.GetBookRequest_Title{
				Title: req.Identifier,
			},
		}
	}

	return &bookv1.GetBookRequest{
		Identifier: &bookv1.GetBookRequest_Id{
			Id: req.Identifier,
		},
	}
}

func (m *Mapper) MapPbListBooksRequest(req *book_service_dto.ListBooksRequestDTO) *bookv1.ListBooksRequest {
	if req == nil {
		return &bookv1.ListBooksRequest{}
	}

	return &bookv1.ListBooksRequest{
		Pagination: &commonv1.PaginationRequest{
			Page:   req.Pagination.Page,
			Limit:  req.Pagination.Limit,
			SortBy: req.Pagination.SortBy,
			IsDesc: req.Pagination.IsDesc,
		},
		SearchQuery: req.SearchQuery,
		Category:    req.Category,
	}
}

func (m *Mapper) MapPbCreateBooksRequest(req *book_service_dto.CreateBooksRequestDTO) *bookv1.CreateBooksRequest {
	if req == nil {
		return &bookv1.CreateBooksRequest{}
	}

	books := make([]*bookv1.CreateBookPayload, 0, len(req.BookPayloads))
	for _, payload := range req.BookPayloads {
		books = append(books, &bookv1.CreateBookPayload{
			Title:         payload.Title,
			Author:        payload.Author,
			Isbn:          payload.ISBN,
			Category:      payload.Category,
			Description:   payload.Description,
			TotalQuantity: payload.Quantity,
		})
	}

	return &bookv1.CreateBooksRequest{
		Books: books,
	}
}

func (m *Mapper) MapPbUpdateBookRequest(req *book_service_dto.UpdateBookRequestDTO) *bookv1.UpdateBookRequest {
	if req == nil {
		return &bookv1.UpdateBookRequest{}
	}

	return &bookv1.UpdateBookRequest{
		Id:          req.ID,
		Title:       req.Title,
		Author:      req.Author,
		Isbn:        req.ISBN,
		Category:    req.Category,
		Description: req.Description,
	}
}

func (m *Mapper) MapPbDeleteBooksRequest(req *book_service_dto.DeleteBooksRequestDTO) *bookv1.DeleteBooksRequest {
	if req == nil {
		return &bookv1.DeleteBooksRequest{}
	}

	return &bookv1.DeleteBooksRequest{
		Ids: req.BookIDs,
	}
}

func (m *Mapper) MapPbCheckBookAvailabilityRequest(req *book_service_dto.CheckAvailabilityRequestDTO) *bookv1.CheckAvailabilityRequest {
	if req == nil {
		return &bookv1.CheckAvailabilityRequest{}
	}

	return &bookv1.CheckAvailabilityRequest{
		BookId: req.BookID,
	}
}

func (m *Mapper) MapPbUpdateBookQuantityRequest(req *book_service_dto.UpdateBookQuantityRequestDTO) *bookv1.UpdateBookQuantityRequest {
	if req == nil {
		return &bookv1.UpdateBookQuantityRequest{}
	}

	return &bookv1.UpdateBookQuantityRequest{
		BookId:       req.BookID,
		ChangeAmount: req.ChangeAmount,
	}
}

func (m *Mapper) MapDTOGetBookResponse(resp *bookv1.BookResponse) *book_service_dto.BookResponseDTO {
	if resp == nil {
		return &book_service_dto.BookResponseDTO{}
	}

	return &book_service_dto.BookResponseDTO{
		Book: m.toDTOBook(resp.GetBook()),
	}
}

func (m *Mapper) MapDTOListBooksResponse(resp *bookv1.ListBooksResponse) *book_service_dto.ListBooksResponseDTO {
	if resp == nil {
		return &book_service_dto.ListBooksResponseDTO{}
	}

	books := make([]*book_service_dto.BookDTO, 0, len(resp.GetBooks()))
	for _, book := range resp.GetBooks() {
		books = append(books, m.toDTOBook(book))
	}

	return &book_service_dto.ListBooksResponseDTO{
		Books:      books,
		TotalCount: resp.GetTotalCount(),
	}
}

func (m *Mapper) MapDTOCreateBooksResponse(resp *bookv1.CreateBooksResponse) *book_service_dto.CreateBooksResponseDTO {
	if resp == nil {
		return &book_service_dto.CreateBooksResponseDTO{}
	}

	createdBooks := make([]*book_service_dto.BookDTO, 0, len(resp.GetBooks()))
	for _, book := range resp.GetBooks() {
		createdBooks = append(createdBooks, m.toDTOBook(book))
	}

	return &book_service_dto.CreateBooksResponseDTO{
		CreatedBooks: createdBooks,
		SuccessCount: resp.GetSuccessCount(),
	}
}

func (m *Mapper) MapDTOUpdateBookResponse(resp *bookv1.BookResponse) *book_service_dto.BookResponseDTO {
	if resp == nil {
		return &book_service_dto.BookResponseDTO{}
	}

	return &book_service_dto.BookResponseDTO{
		Book: m.toDTOBook(resp.GetBook()),
	}
}

func (m *Mapper) MapDTOCheckBookAvailabilityResponse(resp *bookv1.CheckAvailabilityResponse) *book_service_dto.CheckAvailabilityResponseDTO {
	if resp == nil {
		return &book_service_dto.CheckAvailabilityResponseDTO{}
	}

	return &book_service_dto.CheckAvailabilityResponseDTO{
		IsAvailable:       resp.GetIsAvailable(),
		AvailableQuantity: resp.GetAvailableQuantity(),
	}
}

func (m *Mapper) MapDTOUpdateBookQuantityResponse(resp *bookv1.UpdateBookQuantityResponse) *book_service_dto.UpdateBookQuantityResponseDTO {
	if resp == nil {
		return &book_service_dto.UpdateBookQuantityResponseDTO{}
	}

	return &book_service_dto.UpdateBookQuantityResponseDTO{
		NewAvailableQuantity: resp.GetNewAvailableQuantity(),
	}
}

/* ORDER SERVICE MAPPER */
func (m *Mapper) MapPbCreateOrderRequest(req *order_service_dto.CreateOrderRequestDTO) *orderv1.CreateOrderRequest {
	if req == nil {
		return &orderv1.CreateOrderRequest{}
	}

	return &orderv1.CreateOrderRequest{
		UserId:     req.UserID,
		BookIds:    req.BookIDs,
		BorrowDays: req.BorrowDays,
	}
}

func (m *Mapper) MapPbGetOrderRequest(req *order_service_dto.GetOrderRequestDTO) *orderv1.GetOrderRequest {
	if req == nil {
		return &orderv1.GetOrderRequest{}
	}

	return &orderv1.GetOrderRequest{
		OrderId: req.OrderID,
	}
}

func (m *Mapper) MapPbListMyOrdersRequest(req *order_service_dto.ListMyOrdersRequestDTO) *orderv1.ListMyOrdersRequest {
	if req == nil {
		return &orderv1.ListMyOrdersRequest{}
	}

	return &orderv1.ListMyOrdersRequest{
		Pagination: &commonv1.PaginationRequest{
			Page:   req.Pagination.Page,
			Limit:  req.Pagination.Limit,
			SortBy: req.Pagination.SortBy,
			IsDesc: req.Pagination.IsDesc,
		},
		UserId:       req.UserID,
		FilterStatus: m.toPbOrderStatus(req.FilterStatus),
	}
}

func (m *Mapper) MapPbCancelOrderRequest(req *order_service_dto.CancelOrderRequestDTO) *orderv1.CancelOrderRequest {
	if req == nil {
		return &orderv1.CancelOrderRequest{}
	}

	return &orderv1.CancelOrderRequest{
		OrderId:      req.OrderID,
		UserId:       req.UserID,
		CancelReason: req.CancelReason,
	}
}

func (m *Mapper) MapPbListAllOrdersRequest(req *order_service_dto.ListAllOrdersRequestDTO) *orderv1.ListAllOrdersRequest {
	if req == nil {
		return &orderv1.ListAllOrdersRequest{}
	}

	return &orderv1.ListAllOrdersRequest{
		Pagination: &commonv1.PaginationRequest{
			Page:   req.Pagination.Page,
			Limit:  req.Pagination.Limit,
			SortBy: req.Pagination.SortBy,
			IsDesc: req.Pagination.IsDesc,
		},
		FilterStatus: m.toPbOrderStatus(req.FilterStatus),
		SearchUserId: req.SearchUserID,
	}
}

func (m *Mapper) MapPbUpdateOrderStatusRequest(req *order_service_dto.UpdateOrderStatusRequestDTO) *orderv1.UpdateOrderStatusRequest {
	if req == nil {
		return &orderv1.UpdateOrderStatusRequest{}
	}

	return &orderv1.UpdateOrderStatusRequest{
		OrderId:   req.OrderID,
		NewStatus: m.toPbOrderStatus(req.NewStatus),
		Note:      req.Note,
	}
}

func (m *Mapper) MapDTOCreateOrderResponse(resp *orderv1.OrderResponse) *order_service_dto.OrderResponseDTO {
	if resp == nil {
		return &order_service_dto.OrderResponseDTO{}
	}

	return &order_service_dto.OrderResponseDTO{
		Order: m.toDTOOrder(resp.GetOrder()),
	}
}

func (m *Mapper) MapDTOGetOrderResponse(resp *orderv1.OrderResponse) *order_service_dto.OrderResponseDTO {
	if resp == nil {
		return &order_service_dto.OrderResponseDTO{}
	}

	return &order_service_dto.OrderResponseDTO{
		Order: m.toDTOOrder(resp.GetOrder()),
	}
}

func (m *Mapper) MapDTOListMyOrdersResponse(resp *orderv1.ListOrdersResponse) *order_service_dto.ListOrdersResponseDTO {
	if resp == nil {
		return &order_service_dto.ListOrdersResponseDTO{}
	}

	orders := make([]*order_service_dto.OrderDTO, 0, len(resp.GetOrders()))
	for _, order := range resp.GetOrders() {
		orders = append(orders, m.toDTOOrder(order))
	}

	return &order_service_dto.ListOrdersResponseDTO{
		Orders:     orders,
		TotalCount: resp.GetTotalCount(),
	}
}

func (m *Mapper) MapDTOCancelOrderResponse(resp *orderv1.OrderResponse) *order_service_dto.OrderResponseDTO {
	if resp == nil {
		return &order_service_dto.OrderResponseDTO{}
	}

	return &order_service_dto.OrderResponseDTO{
		Order: m.toDTOOrder(resp.GetOrder()),
	}
}

func (m *Mapper) MapDTOListAllOrdersResponse(resp *orderv1.ListOrdersResponse) *order_service_dto.ListOrdersResponseDTO {
	if resp == nil {
		return &order_service_dto.ListOrdersResponseDTO{}
	}

	orders := make([]*order_service_dto.OrderDTO, 0, len(resp.GetOrders()))
	for _, order := range resp.GetOrders() {
		orders = append(orders, m.toDTOOrder(order))
	}

	return &order_service_dto.ListOrdersResponseDTO{
		Orders:     orders,
		TotalCount: resp.GetTotalCount(),
	}
}

func (m *Mapper) MapDTOUpdateOrderStatusResponse(resp *orderv1.OrderResponse) *order_service_dto.OrderResponseDTO {
	if resp == nil {
		return &order_service_dto.OrderResponseDTO{}
	}

	return &order_service_dto.OrderResponseDTO{
		Order: m.toDTOOrder(resp.GetOrder()),
	}
}

/* HELPER METHODS */

func (m *Mapper) toPbRole(role string) userv1.UserRole {
	role = strings.ToUpper(role)
	switch role {
	case "GUEST":
		return userv1.UserRole_GUEST
	case "REGISTERED_USER":
		return userv1.UserRole_REGISTERED_USER
	case "MANAGER":
		return userv1.UserRole_MANAGER
	case "ADMIN":
		return userv1.UserRole_ADMIN
	default:
		return userv1.UserRole_GUEST // Default role
	}
}

func (m *Mapper) toDTOUser(user *userv1.User) *user_service_dto.UserDTO {
	if user == nil {
		return &user_service_dto.UserDTO{}
	}

	createdAtStr := ""
	lastUpdatedStr := ""
	if user.GetCreatedAt() != nil {
		createdAtStr = user.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if user.GetLastUpdated() != nil {
		lastUpdatedStr = user.GetLastUpdated().AsTime().Format(time.RFC3339)
	}

	return &user_service_dto.UserDTO{
		ID:          user.GetId(),
		Username:    user.GetUsername(),
		Email:       user.GetEmail(),
		PhoneNumber: user.GetPhoneNumber(),
		IsActive:    user.GetIsActive(),
		CreatedAt:   createdAtStr,
		LastUpdated: lastUpdatedStr,
	}
}

func (m *Mapper) toDTOBook(book *bookv1.Book) *book_service_dto.BookDTO {
	if book == nil {
		return &book_service_dto.BookDTO{}
	}

	createdAtStr := ""
	updatedAtStr := ""

	if book.GetCreatedAt() != nil {
		createdAtStr = book.GetCreatedAt().AsTime().Format(time.RFC3339)
	}

	if book.GetUpdatedAt() != nil {
		updatedAtStr = book.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	return &book_service_dto.BookDTO{
		ID:                book.GetId(),
		Title:             book.GetTitle(),
		Author:            book.GetAuthor(),
		ISBN:              book.GetIsbn(),
		Category:          book.GetCategory(),
		Description:       book.GetDescription(),
		TotalQuantity:     book.GetTotalQuantity(),
		AvailableQuantity: book.GetAvailableQuantity(),
		CreatedAt:         createdAtStr,
		UpdatedAt:         updatedAtStr,
	}
}

func (m *Mapper) toPbOrderStatus(status string) orderv1.OrderStatus {
	status = strings.ToUpper(status)
	switch status {
	case "PENDING":
		return orderv1.OrderStatus_PENDING
	case "APPROVED":
		return orderv1.OrderStatus_APPROVED
	case "BORROWED":
		return orderv1.OrderStatus_BORROWED
	case "RETURNED":
		return orderv1.OrderStatus_RETURNED
	case "CANCELED":
		return orderv1.OrderStatus_CANCELED
	case "OVERDUE":
		return orderv1.OrderStatus_OVERDUE
	default:
		return orderv1.OrderStatus_STATUS_UNSPECIFIED
	}
}

func (m *Mapper) toDTOOrder(order *orderv1.Order) *order_service_dto.OrderDTO {
	if order == nil {
		return &order_service_dto.OrderDTO{}
	}

	borrowDateStr := ""
	dueDateStr := ""
	var returnDateStr *string
	if order.GetBorrowDate() != nil {
		borrowDateStr = order.GetBorrowDate().AsTime().Format(time.RFC3339)
	}
	if order.GetDueDate() != nil {
		dueDateStr = order.GetDueDate().AsTime().Format(time.RFC3339)
	}
	if order.GetReturnDate() != nil {
		str := order.GetReturnDate().AsTime().Format(time.RFC3339)
		returnDateStr = &str
	}

	createdAtStr := ""
	updatedAtStr := ""
	if order.GetCreatedAt() != nil {
		createdAtStr = order.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if order.GetUpdatedAt() != nil {
		updatedAtStr = order.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	return &order_service_dto.OrderDTO{
		ID:            order.GetId(),
		UserID:        order.GetBorrower().GetId(),
		Status:        order.GetStatus().String(),
		BorrowDate:    borrowDateStr,
		DueDate:       dueDateStr,
		ReturnDate:    returnDateStr,
		Note:          order.GetNote(),
		PenaltyAmount: order.GetPenaltyAmount(),
		CreatedAt:     createdAtStr,
		UpdatedAt:     updatedAtStr,
	}
}
