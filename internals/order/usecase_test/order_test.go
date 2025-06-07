package usecase_test

import (
	"context"
	"errors"
	"testing"

	orderDto "ecommerce_clean/internals/order/controller/dto"
	orderEntity "ecommerce_clean/internals/order/entity"
	"ecommerce_clean/internals/order/usecase"
	prodDto "ecommerce_clean/internals/product/controller/dto"
	productEntity "ecommerce_clean/internals/product/entity"
	"ecommerce_clean/pkgs/paging"
	"ecommerce_clean/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// -------------------
// Mocks
// -------------------

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, userID string, lines []*orderEntity.OrderLine) (*orderEntity.Order, error) {
	args := m.Called(ctx, userID, lines)
	return args.Get(0).(*orderEntity.Order), args.Error(1)
}

func (m *MockOrderRepository) GetOrderByID(ctx context.Context, id string, preload bool) (*orderEntity.Order, error) {
	args := m.Called(ctx, id, preload)
	return args.Get(0).(*orderEntity.Order), args.Error(1)
}

func (m *MockOrderRepository) GetMyOrders(ctx context.Context, req *orderDto.ListOrdersRequest) ([]*orderEntity.Order, *paging.Pagination, error) {
	args := m.Called(ctx, req)
	var orders []*orderEntity.Order
	if v := args.Get(0); v != nil {
		orders = v.([]*orderEntity.Order)
	}
	var page *paging.Pagination
	if v := args.Get(1); v != nil {
		page = v.(*paging.Pagination)
	}
	return orders, page, args.Error(2)
}

func (m *MockOrderRepository) UpdateOrder(ctx context.Context, order *orderEntity.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) ListProducts(ctx context.Context, req *prodDto.ListProductRequest) ([]*productEntity.Product, *paging.Pagination, error) {
	return nil, nil, nil
}

// GetProductById ahora maneja return nil sin panic.
func (m *MockProductRepository) GetProductById(ctx context.Context, id string) (*productEntity.Product, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*productEntity.Product), args.Error(1)
	}
	return nil, args.Error(1)
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
	return m.Called(i).Error(0)
}

// -------------------------------------
// Tests de PlaceOrder
// -------------------------------------

// TestPlaceOrder_Success verifica que PlaceOrder:
// 1) Valida la entrada.
// 2) Recupera el producto.
// 3) Calcula precio de líneas.
// 4) Crea el pedido con líneas y total correctos.
func TestPlaceOrder_Success(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewOrderUseCase(mockValidator, mockOrderRepo, mockProductRepo)

	req := &orderDto.PlaceOrderRequest{
		UserID: "u1",
		Lines: []orderDto.PlaceOrderLineRequest{
			{ProductID: "p1", Quantity: 2},
		},
	}
	prod := &productEntity.Product{ID: "p1", Price: 50.0}

	mockValidator.On("ValidateStruct", req).Return(nil)
	mockProductRepo.On("GetProductById", mock.Anything, "p1").Return(prod, nil)
	mockOrderRepo.
		On("CreateOrder", mock.Anything, "u1", mock.Anything).
		Return(&orderEntity.Order{
			UserID:     "u1",
			Lines:      []*orderEntity.OrderLine{{ProductID: "p1", Quantity: 2, Price: 100.0}},
			TotalPrice: 100.0,
		}, nil)

	order, err := uc.PlaceOrder(context.Background(), req)

	assert.NoError(t, err)
	if assert.Len(t, order.Lines, 1) {
		assert.Equal(t, prod, order.Lines[0].Product)
		assert.Equal(t, 100.0, order.Lines[0].Price)
	}
}

// TestPlaceOrder_ValidationError verifica que PlaceOrder devuelve error
// cuando la validación de la petición falla.
func TestPlaceOrder_ValidationError(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewOrderUseCase(mockValidator, mockOrderRepo, mockProductRepo)

	req := &orderDto.PlaceOrderRequest{UserID: "", Lines: nil}
	mockValidator.On("ValidateStruct", req).Return(errors.New("invalid input"))

	order, err := uc.PlaceOrder(context.Background(), req)

	assert.Nil(t, order)
	assert.EqualError(t, err, "invalid input")
}

// TestPlaceOrder_ProductRepoError verifica que PlaceOrder propaga el error
// cuando GetProductById falla.
func TestPlaceOrder_ProductRepoError(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewOrderUseCase(mockValidator, mockOrderRepo, mockProductRepo)

	req := &orderDto.PlaceOrderRequest{
		UserID: "u1",
		Lines:  []orderDto.PlaceOrderLineRequest{{ProductID: "p1", Quantity: 1}},
	}
	mockValidator.On("ValidateStruct", req).Return(nil)
	mockProductRepo.On("GetProductById", mock.Anything, "p1").Return(nil, errors.New("not found"))

	order, err := uc.PlaceOrder(context.Background(), req)

	assert.Nil(t, order)
	assert.EqualError(t, err, "not found")
}

// TestPlaceOrder_MultipleLines verifica que PlaceOrder maneja varias líneas
// y suma correctamente todos los precios.
func TestPlaceOrder_MultipleLines(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockValidator)

	uc := usecase.NewOrderUseCase(mockValidator, mockOrderRepo, mockProductRepo)

	req := &orderDto.PlaceOrderRequest{
		UserID: "u1",
		Lines: []orderDto.PlaceOrderLineRequest{
			{ProductID: "p1", Quantity: 1},
			{ProductID: "p2", Quantity: 3},
		},
	}
	p1 := &productEntity.Product{ID: "p1", Price: 10.0}
	p2 := &productEntity.Product{ID: "p2", Price: 20.0}

	mockValidator.On("ValidateStruct", req).Return(nil)
	mockProductRepo.On("GetProductById", mock.Anything, "p1").Return(p1, nil)
	mockProductRepo.On("GetProductById", mock.Anything, "p2").Return(p2, nil)
	mockOrderRepo.
		On("CreateOrder", mock.Anything, "u1", mock.Anything).
		Return(&orderEntity.Order{
			UserID: "u1",
			Lines: []*orderEntity.OrderLine{
				{ProductID: "p1", Quantity: 1, Price: 10.0},
				{ProductID: "p2", Quantity: 3, Price: 60.0},
			},
			TotalPrice: 70.0,
		}, nil)

	order, err := uc.PlaceOrder(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, 70.0, order.TotalPrice)
	assert.Equal(t, p1, order.Lines[0].Product)
	assert.Equal(t, p2, order.Lines[1].Product)
}

// -------------------------------------
// Tests de ListMyOrders
// -------------------------------------

// TestListMyOrders_Success verifica que ListMyOrders devuelve lista no vacía
// y una paginación correcta.
func TestListMyOrders_Success(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	req := &orderDto.ListOrdersRequest{UserID: "u1", Page: 1, Limit: 10}
	expectedOrders := []*orderEntity.Order{{ID: "o1"}, {ID: "o2"}}
	expectedPage := paging.NewPagination(1, 10, 2)

	mockOrderRepo.
		On("GetMyOrders", mock.Anything, req).
		Return(expectedOrders, expectedPage, nil)

	orders, page, err := uc.ListMyOrders(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, expectedOrders, orders)
	assert.Equal(t, expectedPage, page)
}

// TestListMyOrders_Empty verifica que ListMyOrders devuelve lista vacía
// cuando no hay pedidos y la paginación refleja cero elementos.
func TestListMyOrders_Empty(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	req := &orderDto.ListOrdersRequest{UserID: "u1", Page: 2, Limit: 5}
	expectedPage := paging.NewPagination(2, 5, 0)

	mockOrderRepo.
		On("GetMyOrders", mock.Anything, req).
		Return(nil, expectedPage, nil)

	orders, page, err := uc.ListMyOrders(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, orders)
	assert.Equal(t, expectedPage, page)
}

// TestListMyOrders_RepoError verifica que ListMyOrders propaga error
// cuando el repositorio falla.
func TestListMyOrders_RepoError(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	req := &orderDto.ListOrdersRequest{UserID: "u1"}
	mockOrderRepo.
		On("GetMyOrders", mock.Anything, req).
		Return(nil, nil, errors.New("db error"))

	orders, page, err := uc.ListMyOrders(context.Background(), req)

	assert.Nil(t, orders)
	assert.Nil(t, page)
	assert.EqualError(t, err, "db error")
}

// -------------------------------------
// Tests de GetOrderByID
// -------------------------------------

// TestGetOrderByID_Success verifica que GetOrderByID devuelve una orden válida.
func TestGetOrderByID_Success(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	expected := &orderEntity.Order{ID: "o123"}
	mockOrderRepo.
		On("GetOrderByID", mock.Anything, "o123", true).
		Return(expected, nil)

	order, err := uc.GetOrderByID(context.Background(), "o123")

	assert.NoError(t, err)
	assert.Equal(t, expected, order)
}

// TestGetOrderByID_RepoError verifica que GetOrderByID propaga error
// cuando el repositorio no encuentra la orden.
func TestGetOrderByID_RepoError(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	mockOrderRepo.
		On("GetOrderByID", mock.Anything, "o123", true).
		Return((*orderEntity.Order)(nil), errors.New("not found"))

	order, err := uc.GetOrderByID(context.Background(), "o123")

	assert.Nil(t, order)
	assert.EqualError(t, err, "not found")
}

// -------------------------------------
// Tests de UpdateOrder
// -------------------------------------

// TestUpdateOrder_Success verifica que UpdateOrder actualiza correctamente
// el estado de la orden cuando el usuario coincide y el estado es válido.
func TestUpdateOrder_Success(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	existing := &orderEntity.Order{ID: "o1", UserID: "u1", Status: utils.OrderStatusNew}
	mockOrderRepo.On("GetOrderByID", mock.Anything, "o1", false).Return(existing, nil)
	mockOrderRepo.On("UpdateOrder", mock.Anything, existing).Return(nil)

	updated, err := uc.UpdateOrder(context.Background(), "o1", "u1", string(utils.OrderStatusDone))

	assert.NoError(t, err)
	assert.Equal(t, utils.OrderStatusDone, updated.Status)
}

// TestUpdateOrder_PermissionDenied verifica que UpdateOrder falla
// cuando el userID no coincide con el de la orden.
func TestUpdateOrder_PermissionDenied(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	existing := &orderEntity.Order{ID: "o1", UserID: "u1", Status: utils.OrderStatusNew}
	mockOrderRepo.On("GetOrderByID", mock.Anything, "o1", false).Return(existing, nil)

	_, err := uc.UpdateOrder(context.Background(), "o1", "otherUser", string(utils.OrderStatusDone))
	assert.EqualError(t, err, "permission denied")
}

// TestUpdateOrder_InvalidState verifica que UpdateOrder rechaza cambios
// cuando la orden ya está en estado 'done' o 'canceled'.
func TestUpdateOrder_InvalidState(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	for _, s := range []utils.OrderStatus{utils.OrderStatusDone, utils.OrderStatusCanceled} {
		existing := &orderEntity.Order{ID: "o1", UserID: "u1", Status: s}
		mockOrderRepo.On("GetOrderByID", mock.Anything, "o1", false).Return(existing, nil)

		_, err := uc.UpdateOrder(context.Background(), "o1", "u1", string(utils.OrderStatusInProgress))
		assert.EqualError(t, err, "invalid order status")
		mockOrderRepo.ExpectedCalls = nil
	}
}

// TestUpdateOrder_InvalidStatusParam verifica que UpdateOrder devuelve error
// cuando se pasa un estado no válido en el parámetro.
func TestUpdateOrder_InvalidStatusParam(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	existing := &orderEntity.Order{ID: "o1", UserID: "u1", Status: utils.OrderStatusNew}
	mockOrderRepo.On("GetOrderByID", mock.Anything, "o1", false).Return(existing, nil)

	_, err := uc.UpdateOrder(context.Background(), "o1", "u1", "badstatus")
	assert.EqualError(t, err, "invalid status")
}

// TestUpdateOrder_UpdateError verifica que UpdateOrder propaga el error
// cuando el repositorio falla al actualizar la orden.
func TestUpdateOrder_UpdateError(t *testing.T) {
	mockOrderRepo := new(MockOrderRepository)
	uc := usecase.NewOrderUseCase(new(MockValidator), mockOrderRepo, new(MockProductRepository))

	existing := &orderEntity.Order{ID: "o1", UserID: "u1", Status: utils.OrderStatusNew}
	mockOrderRepo.On("GetOrderByID", mock.Anything, "o1", false).Return(existing, nil)
	mockOrderRepo.On("UpdateOrder", mock.Anything, existing).Return(errors.New("update failed"))

	_, err := uc.UpdateOrder(context.Background(), "o1", "u1", string(utils.OrderStatusInProgress))
	assert.EqualError(t, err, "update failed")
}
