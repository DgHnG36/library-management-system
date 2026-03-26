package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

/* HELPER METHODS */
func setupTestDB(t *testing.T) *gorm.DB {
	host := os.Getenv("TEST_BOOK_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("TEST_BOOK_DB_PORT")
	if port == "" {
		port = "15433"
	}
	dbName := os.Getenv("TEST_BOOK_DB_NAME")
	if dbName == "" {
		dbName = "book_db"
	}
	user := os.Getenv("TEST_BOOK_DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("TEST_BOOK_DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, password, dbName, port)

	var db *gorm.DB
	var err error

	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !assert.NoError(t, err, "failed to connect to test database. check docker is running and TEST_BOOK_DB_* envs") {
		t.FailNow()
	}

	err = db.AutoMigrate(&models.Book{})
	if !assert.NoError(t, err, "failed to migrate test database") {
		t.FailNow()
	}

	return db
}

func setupBookService(t *testing.T) (*handlers.BookHandler, repository.BookRepository) {
	db := setupTestDB(t)

	tx := db.Begin()
	if !assert.NoError(t, tx.Error, "failed to begin test transaction") {
		t.FailNow()
	}
	if !assert.NoError(t, tx.Exec("TRUNCATE TABLE books RESTART IDENTITY CASCADE").Error, "failed to reset books table for test transaction") {
		t.FailNow()
	}
	t.Cleanup(func() {
		_ = tx.Rollback().Error
	})

	log := logger.DefaultNewLogger()
	repo := repository.NewBookRepo(tx)
	srv := applications.NewBookService(repo, log)
	handler := handlers.NewBookHandler(srv, log)
	return handler, repo
}

func TestBookService_GetBook_NotFound(t *testing.T) {
	handler, _ := setupBookService(t)

	req := &bookv1.GetBookRequest{
		Identifier: &bookv1.GetBookRequest_Id{
			Id: uuid.New().String(),
		},
	}

	resp, err := handler.GetBook(context.Background(), req)
	st, _ := status.FromError(err)

	assert.Error(t, err)
	assert.Equal(t, st.Code(), codes.NotFound, "expected NotFound error code")
	assert.Nil(t, resp)
}

func TestBookService_GetBook_Success(t *testing.T) {
	handler, repo := setupBookService(t)

	books := []*models.Book{
		{
			ID:          uuid.New().String(),
			Title:       "Test Book",
			Author:      "Test Author",
			Description: "Test Description",
		},
		{
			ID:          uuid.New().String(),
			Title:       "Another Book",
			Author:      "Another Author",
			Description: "Another Description",
			ISBN:        "1234567890",
		},
	}
	err := repo.Create(context.Background(), books)
	if !assert.NoError(t, err, "failed to create test book") {
		t.FailNow()
	}

	req := &bookv1.GetBookRequest{
		Identifier: &bookv1.GetBookRequest_Id{
			Id: books[0].ID,
		},
	}

	resp, err := handler.GetBook(context.Background(), req)
	st, _ := status.FromError(err)

	assert.NoError(t, err)
	assert.Equal(t, st.Code(), codes.OK, "expected OK error code")
	assert.NotNil(t, resp)
	assert.Equal(t, books[0].ID, resp.Book.Id)
	assert.Equal(t, books[0].Title, resp.Book.Title)
	assert.Equal(t, books[0].Author, resp.Book.Author)
	assert.Equal(t, books[0].Description, resp.Book.Description)
	assert.Equal(t, books[0].ISBN, resp.Book.Isbn)

	resp, err = handler.GetBook(context.Background(), &bookv1.GetBookRequest{
		Identifier: &bookv1.GetBookRequest_Title{
			Title: books[1].Title,
		},
	})
	st, _ = status.FromError(err)

	assert.NoError(t, err)
	assert.Equal(t, st.Code(), codes.OK, "expected OK error code")
	assert.NotNil(t, resp)
	assert.Equal(t, books[1].ID, resp.Book.Id)
	assert.Equal(t, books[1].Title, resp.Book.Title)
	assert.Equal(t, books[1].Author, resp.Book.Author)
	assert.Equal(t, books[1].Description, resp.Book.Description)
	assert.Equal(t, books[1].ISBN, resp.Book.Isbn)
}
func TestBookService_CreateBooks_Success(t *testing.T) {
	handler, _ := setupBookService(t)
	ctx := context.Background()

	books := []*bookv1.Book{
		{
			Title:             "Tittle1",
			Author:            "Author1",
			Isbn:              "12474563",
			Category:          "dict",
			Description:       "New bool first",
			TotalQuantity:     10,
			AvailableQuantity: 10,
		},
		{
			Title:             "Tittle2",
			Author:            "Author2",
			Isbn:              "0987654321",
			Category:          "fiction",
			Description:       "Another new book",
			TotalQuantity:     5,
			AvailableQuantity: 5,
		},
		{
			Title:             "Tittle3",
			Author:            "Author3",
			Isbn:              "1234567890",
			Category:          "science",
			Description:       "Another new book",
			TotalQuantity:     3,
			AvailableQuantity: 3,
		},
	}

	bookPayload := make([]*bookv1.CreateBookPayload, len(books))
	for i, b := range books {
		bookPayload[i] = &bookv1.CreateBookPayload{
			Title:         b.Title,
			Author:        b.Author,
			Isbn:          b.Isbn,
			Category:      b.Category,
			Description:   b.Description,
			TotalQuantity: b.TotalQuantity,
		}
	}

	req := &bookv1.CreateBooksRequest{
		Books: bookPayload,
	}

	resp, err := handler.CreateBooks(ctx, req)
	st, _ := status.FromError(err)
	assert.Equal(t, st.Code(), codes.OK, "expected OK error code")
	assert.Equal(t, len(books), len(resp.Books))
}

func TestBookService_ListBooks_FilterByCategory_Success(t *testing.T) {
	handler, repo := setupBookService(t)
	ctx := context.Background()

	err := repo.Create(ctx, []*models.Book{
		{ID: uuid.New().String(), Title: "Go In Action", Author: "A", ISBN: "ISBN-001-tech", Category: "tech", TotalQuantity: 4, AvailableQuantity: 4},
		{ID: uuid.New().String(), Title: "Vietnam History", Author: "B", ISBN: "ISBN-002-hist", Category: "history", TotalQuantity: 2, AvailableQuantity: 2},
	})
	if !assert.NoError(t, err) {
		return
	}

	resp, listErr := handler.ListBooks(ctx, &bookv1.ListBooksRequest{
		Pagination:  &commonv1.PaginationRequest{Page: 1, Limit: 10},
		Category:    "tech",
		SearchQuery: "",
	})
	if !assert.NoError(t, listErr) {
		return
	}

	assert.Equal(t, int32(1), resp.TotalCount)
	assert.Len(t, resp.Books, 1)
	assert.Equal(t, "tech", resp.Books[0].Category)
}

func TestBookService_UpdateAndCheckAvailability_Success(t *testing.T) {
	handler, repo := setupBookService(t)
	ctx := context.Background()

	bookID := uuid.New().String()
	err := repo.Create(ctx, []*models.Book{
		{ID: bookID, Title: "Distributed Systems", Author: "Old", ISBN: "ISBN-003-dist", Category: "tech", TotalQuantity: 3, AvailableQuantity: 3},
	})
	if !assert.NoError(t, err) {
		return
	}

	updateResp, updateErr := handler.UpdateBook(ctx, &bookv1.UpdateBookRequest{
		Id:     bookID,
		Author: "New Author",
		Title:  "Distributed Systems - 2nd",
	})
	if !assert.NoError(t, updateErr) {
		return
	}
	assert.Equal(t, "New Author", updateResp.Book.Author)
	assert.Equal(t, "Distributed Systems - 2nd", updateResp.Book.Title)

	qtyResp, qtyErr := handler.UpdateBookQuantity(ctx, &bookv1.UpdateBookQuantityRequest{
		BookId:       bookID,
		ChangeAmount: -2,
	})
	if !assert.NoError(t, qtyErr) {
		return
	}
	assert.True(t, qtyResp.Success)
	assert.Equal(t, int32(1), qtyResp.NewAvailableQuantity)

	availabilityResp, availabilityErr := handler.CheckAvailability(ctx, &bookv1.CheckAvailabilityRequest{BookId: bookID})
	if !assert.NoError(t, availabilityErr) {
		return
	}
	assert.True(t, availabilityResp.IsAvailable)
	assert.Equal(t, int32(1), availabilityResp.AvailableQuantity)
}

func TestBookService_DeleteBooks_Success(t *testing.T) {
	handler, repo := setupBookService(t)
	ctx := context.Background()

	b1 := uuid.New().String()
	b2 := uuid.New().String()
	err := repo.Create(ctx, []*models.Book{
		{ID: b1, Title: "Book 1", Author: "A", ISBN: "ISBN-004-book1", TotalQuantity: 1, AvailableQuantity: 1},
		{ID: b2, Title: "Book 2", Author: "B", ISBN: "ISBN-005-book2", TotalQuantity: 1, AvailableQuantity: 1},
	})
	if !assert.NoError(t, err) {
		return
	}

	deleteResp, deleteErr := handler.DeleteBooks(ctx, &bookv1.DeleteBooksRequest{Ids: []string{b1, b2}})
	if !assert.NoError(t, deleteErr) {
		return
	}
	assert.Equal(t, int32(200), deleteResp.Status)

	getResp, getErr := handler.GetBook(ctx, &bookv1.GetBookRequest{Identifier: &bookv1.GetBookRequest_Id{Id: b1}})
	assert.Error(t, getErr)
	assert.Nil(t, getResp)
	st, ok := status.FromError(getErr)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}
