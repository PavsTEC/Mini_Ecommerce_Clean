package usecase_test

import (
	"context"
	"errors"
	"testing"

	prodDto "ecommerce_clean/internals/product/controller/dto"
	productEntity "ecommerce_clean/internals/product/entity"
	"ecommerce_clean/internals/product/usecase"
	"ecommerce_clean/pkgs/paging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// -------------------
// Mocks
// -------------------

type MockProductRepository struct {
	mock.Mock
}

// ListProducts maneja nil sin panic.
func (m *MockProductRepository) ListProducts(ctx context.Context, req *prodDto.ListProductRequest) ([]*productEntity.Product, *paging.Pagination, error) {
	args := m.Called(ctx, req)
	// Productos
	var products []*productEntity.Product
	if v := args.Get(0); v != nil {
		products = v.([]*productEntity.Product)
	}
	// Paginación
	var page *paging.Pagination
	if v := args.Get(1); v != nil {
		page = v.(*paging.Pagination)
	}
	return products, page, args.Error(2)
}

func (m *MockProductRepository) GetProductById(ctx context.Context, id string) (*productEntity.Product, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*productEntity.Product), args.Error(1)
}

func (m *MockProductRepository) CreatedProduct(ctx context.Context, p *productEntity.Product) error {
	return nil
}
func (m *MockProductRepository) UpdateProduct(ctx context.Context, p *productEntity.Product) error {
	return nil
}
func (m *MockProductRepository) DeleteProduct(ctx context.Context, p *productEntity.Product) error {
	return nil
}

// -------------------------------------
// Tests de ProductUseCase
// -------------------------------------

// TestListProducts_Success verifica que ListProducts:
// 1) Llama al repositorio con la solicitud correcta.
// 2) Devuelve la lista de productos y la paginación proporcionada.
func TestListProducts_Success(t *testing.T) {
	mockRepo := new(MockProductRepository)
	uc := usecase.NewProductUseCase(nil, mockRepo, nil)

	req := &prodDto.ListProductRequest{Page: 1, Limit: 2}
	expected := []*productEntity.Product{{ID: "p1"}, {ID: "p2"}}
	pagination := paging.NewPagination(1, 2, 2)

	mockRepo.On("ListProducts", mock.Anything, req).Return(expected, pagination, nil)

	products, page, err := uc.ListProducts(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, expected, products)
	assert.Equal(t, pagination, page)
	mockRepo.AssertExpectations(t)
}

// TestListProducts_RepoError verifica que ListProducts propaga errores
// cuando el repositorio falla.
func TestListProducts_RepoError(t *testing.T) {
	mockRepo := new(MockProductRepository)
	uc := usecase.NewProductUseCase(nil, mockRepo, nil)

	req := &prodDto.ListProductRequest{Page: 1, Limit: 2}
	mockRepo.On("ListProducts", mock.Anything, req).Return(nil, nil, errors.New("db error"))

	products, page, err := uc.ListProducts(context.Background(), req)

	assert.Nil(t, products)
	assert.Nil(t, page)
	assert.EqualError(t, err, "db error")
	mockRepo.AssertExpectations(t)
}

// TestGetProductById_Success verifica que GetProductById devuelve
// correctamente un producto cuando existe.
func TestGetProductById_Success(t *testing.T) {
	mockRepo := new(MockProductRepository)
	uc := usecase.NewProductUseCase(nil, mockRepo, nil)

	expected := &productEntity.Product{ID: "p1"}
	mockRepo.On("GetProductById", mock.Anything, "p1").Return(expected, nil)

	product, err := uc.GetProductById(context.Background(), "p1")

	assert.NoError(t, err)
	assert.Equal(t, expected, product)
	mockRepo.AssertExpectations(t)
}

// TestGetProductById_RepoError verifica que GetProductById propaga error
// cuando el repositorio falla.
func TestGetProductById_RepoError(t *testing.T) {
	mockRepo := new(MockProductRepository)
	uc := usecase.NewProductUseCase(nil, mockRepo, nil)

	mockRepo.On("GetProductById", mock.Anything, "p1").Return((*productEntity.Product)(nil), errors.New("not found"))

	product, err := uc.GetProductById(context.Background(), "p1")

	assert.Nil(t, product)
	assert.EqualError(t, err, "not found")
	mockRepo.AssertExpectations(t)
}
