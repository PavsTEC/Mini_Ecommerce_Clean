package usecase_test

import (
	"context"
	"errors"
	"testing"

	cartDto "ecommerce_clean/internals/cart/controller/dto"
	cartEntity "ecommerce_clean/internals/cart/entity"
	"ecommerce_clean/internals/cart/usecase"
	prodDto "ecommerce_clean/internals/product/controller/dto"
	productEntity "ecommerce_clean/internals/product/entity"
	"ecommerce_clean/pkgs/paging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) GetCartByUserID(ctx context.Context, userID string) (*cartEntity.Cart, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*cartEntity.Cart), args.Error(1)
}

func (m *MockCartRepository) GetCartLineByProductIDAndCartID(ctx context.Context, cartID, productID string) (*cartEntity.CartLine, error) {
	args := m.Called(ctx, cartID, productID)
	return args.Get(0).(*cartEntity.CartLine), args.Error(1)
}

func (m *MockCartRepository) CreateCartLine(ctx context.Context, cl *cartEntity.CartLine) error {
	args := m.Called(ctx, cl)
	return args.Error(0)
}

func (m *MockCartRepository) UpdateCartLine(ctx context.Context, cl *cartEntity.CartLine) error {
	args := m.Called(ctx, cl)
	return args.Error(0)
}

func (m *MockCartRepository) RemoveCartLine(ctx context.Context, cl *cartEntity.CartLine) error {
	args := m.Called(ctx, cl)
	return args.Error(0)
}

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) ListProducts(ctx context.Context, req *prodDto.ListProductRequest) ([]*productEntity.Product, *paging.Pagination, error) {
	return nil, nil, nil
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

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) ValidateStruct(i interface{}) error {
	args := m.Called(i)
	return args.Error(0)
}

// --- Tests ---

// -------------------------------------
// Tests de AddProduct
// -------------------------------------

// TestAddProduct_Success verifica que AddProduct:
// 1) Valida correctamente la petición.
// 2) Recupera el producto existente.
// 3) Crea la línea de carrito con el precio calculado.
// 4) No devuelve error.
func TestAddProduct_Success(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	req := &cartDto.AddProductRequest{
		CartID:    "cart123",
		ProductID: "prod456",
		Quantity:  2,
	}
	product := &productEntity.Product{ID: "prod456", Price: 10.0}

	mockValidator.On("ValidateStruct", req).Return(nil)
	mockProductRepo.On("GetProductById", mock.Anything, "prod456").Return(product, nil)
	mockCartRepo.On("CreateCartLine", mock.Anything, mock.Anything).Return(nil)

	err := uc.AddProduct(context.Background(), req)

	assert.NoError(t, err)
	mockValidator.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
	mockCartRepo.AssertExpectations(t)
}

// TestAddProduct_ValidationError verifica que AddProduct devuelve un error
// cuando la validación de la petición falla.
func TestAddProduct_ValidationError(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	req := &cartDto.AddProductRequest{
		CartID:    "",
		ProductID: "prod456",
		Quantity:  0,
	}
	mockValidator.On("ValidateStruct", req).Return(errors.New("invalid input"))

	err := uc.AddProduct(context.Background(), req)

	assert.Error(t, err)
	mockValidator.AssertExpectations(t)
}

// -------------------------------------
// Tests de GetCartByUserID
// -------------------------------------

// TestGetCartByUserID_Success verifica que GetCartByUserID:
// 1) Llama al repositorio con el userID correcto.
// 2) Devuelve el carrito esperado sin error.
func TestGetCartByUserID_Success(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	expected := &cartEntity.Cart{
		ID:     "c1",
		UserID: "u1",
		Lines:  []*cartEntity.CartLine{},
	}
	mockCartRepo.On("GetCartByUserID", mock.Anything, "u1").Return(expected, nil)

	cart, err := uc.GetCartByUserID(context.Background(), "u1")

	assert.NoError(t, err)
	assert.Equal(t, expected, cart)
	mockCartRepo.AssertExpectations(t)
}

// TestGetCartByUserID_RepoError verifica que GetCartByUserID devuelve un error
// y un carrito nulo cuando el repositorio falla.
func TestGetCartByUserID_RepoError(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	mockCartRepo.On("GetCartByUserID", mock.Anything, "u1").
		Return((*cartEntity.Cart)(nil), errors.New("db error"))

	cart, err := uc.GetCartByUserID(context.Background(), "u1")

	assert.Nil(t, cart)
	assert.EqualError(t, err, "db error")
	mockCartRepo.AssertExpectations(t)
}

// -------------------------------------
// Tests de UpdateCartLine
// -------------------------------------

// TestUpdateCartLine_Success verifica que UpdateCartLine:
// 1) Valida la petición.
// 2) Recupera el producto y la línea existente.
// 3) Re-calcula el precio correctamente.
// 4) Llama a UpdateCartLine sin error.
func TestUpdateCartLine_Success(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	req := &cartDto.UpdateCartLineRequest{CartID: "c1", ProductID: "p1", Quantity: 5}
	original := &cartEntity.CartLine{CartID: "c1", ProductID: "p1", Quantity: 2, Price: 20.0}
	prod := &productEntity.Product{ID: "p1", Price: 3.0}

	mockValidator.On("ValidateStruct", req).Return(nil)
	mockProductRepo.On("GetProductById", mock.Anything, "p1").Return(prod, nil)
	mockCartRepo.On("GetCartLineByProductIDAndCartID", mock.Anything, "c1", "p1").Return(original, nil)
	mockCartRepo.On("UpdateCartLine", mock.Anything, original).Return(nil)

	err := uc.UpdateCartLine(context.Background(), req)

	assert.NoError(t, err)
	// Verificamos que el precio haya sido recalculado: 3.0 * 5
	assert.Equal(t, 15.0, original.Price)
	mockValidator.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
	mockCartRepo.AssertExpectations(t)
}

// TestUpdateCartLine_ValidationError verifica que UpdateCartLine devuelve un error
// cuando la validación de la petición falla antes de cualquier otra operación.
func TestUpdateCartLine_ValidationError(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	req := &cartDto.UpdateCartLineRequest{CartID: "", ProductID: "p1", Quantity: 0}
	mockValidator.On("ValidateStruct", req).Return(errors.New("invalid"))

	err := uc.UpdateCartLine(context.Background(), req)

	assert.EqualError(t, err, "invalid")
	mockValidator.AssertExpectations(t)
}

// -------------------------------------
// Tests de RemoveProduct
// -------------------------------------

// TestRemoveProduct_Success verifica que RemoveProduct:
// 1) Recupera la línea de carrito correcta.
// 2) Llama a RemoveCartLine sin error.
func TestRemoveProduct_Success(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	req := &cartDto.RemoveProductRequest{CartID: "c1", ProductID: "p1"}
	cl := &cartEntity.CartLine{CartID: "c1", ProductID: "p1"}

	mockCartRepo.On("GetCartLineByProductIDAndCartID", mock.Anything, "c1", "p1").Return(cl, nil)
	mockCartRepo.On("RemoveCartLine", mock.Anything, cl).Return(nil)

	err := uc.RemoveProduct(context.Background(), req)

	assert.NoError(t, err)
	mockCartRepo.AssertExpectations(t)
}

// TestRemoveProduct_GetCartLineError verifica que RemoveProduct devuelve un error
// cuando no se puede recuperar la línea de carrito.
func TestRemoveProduct_GetCartLineError(t *testing.T) {
	mockCartRepo := new(MockCartRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewCartUseCase(mockValidator, mockCartRepo, mockProductRepo)

	req := &cartDto.RemoveProductRequest{CartID: "c1", ProductID: "p1"}
	mockCartRepo.On("GetCartLineByProductIDAndCartID", mock.Anything, "c1", "p1").
		Return((*cartEntity.CartLine)(nil), errors.New("not found"))

	err := uc.RemoveProduct(context.Background(), req)

	assert.EqualError(t, err, "not found")
	mockCartRepo.AssertExpectations(t)
}
