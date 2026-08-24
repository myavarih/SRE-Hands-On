package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/entities"
	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	"github.com/myavarih/sre-hands-on/pkg/ports"
	paymentv1 "github.com/myavarih/sre-hands-on/services/gen/go/payment/v1"
	"github.com/myavarih/sre-hands-on/services/order-service/internal/application/dto"
)

type OrderUseCase interface {
	ListProducts(ctx context.Context) ([]dto.ProductDTO, error)
	GetProductByID(ctx context.Context, id string) (*dto.ProductDTO, error)
	CreateProduct(ctx context.Context, input dto.CreateProductInput) (*dto.ProductDTO, error)
	CreateOrder(ctx context.Context, input dto.CreateOrderInput) (*dto.OrderDTO, error)
	GetOrderByID(ctx context.Context, orderID string) (*dto.OrderDTO, error)
	ListOrdersByUserID(ctx context.Context, userID string) ([]dto.OrderDTO, error)
}

type orderUseCase struct {
	uow           ports.UnitOfWork
	paymentClient paymentv1.PaymentServiceClient
	logger        *slog.Logger
}

func NewOrderUseCase(uow ports.UnitOfWork, paymentClient paymentv1.PaymentServiceClient, logger *slog.Logger) OrderUseCase {
	return &orderUseCase{
		uow:           uow,
		paymentClient: paymentClient,
		logger:        logger,
	}
}

func (uc *orderUseCase) ListProducts(ctx context.Context) ([]dto.ProductDTO, error) {
	products, err := uc.uow.Products().FindAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ProductDTO, len(products))
	for i, p := range products {
		result[i] = dto.ProductDTO{
			ID:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			PriceCents:    p.PriceCents,
			StockQuantity: p.StockQuantity,
		}
	}
	return result, nil
}

func (uc *orderUseCase) GetProductByID(ctx context.Context, id string) (*dto.ProductDTO, error) {
	if id == "" {
		return nil, exceptions.NewValidationError("product_id is required")
	}

	p, err := uc.uow.Products().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &dto.ProductDTO{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		PriceCents:    p.PriceCents,
		StockQuantity: p.StockQuantity,
	}, nil
}

func (uc *orderUseCase) CreateProduct(ctx context.Context, input dto.CreateProductInput) (*dto.ProductDTO, error) {
	if input.Name == "" || input.PriceCents < 0 || input.InitialStock < 0 {
		return nil, exceptions.NewValidationError("name, non-negative price, and stock are required")
	}

	p := &entities.Product{
		Name:          input.Name,
		Description:   input.Description,
		PriceCents:    input.PriceCents,
		StockQuantity: input.InitialStock,
	}

	if err := uc.uow.Products().Create(ctx, p); err != nil {
		return nil, err
	}

	return &dto.ProductDTO{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		PriceCents:    p.PriceCents,
		StockQuantity: p.StockQuantity,
	}, nil
}

func (uc *orderUseCase) CreateOrder(ctx context.Context, input dto.CreateOrderInput) (*dto.OrderDTO, error) {
	if input.UserID == "" || len(input.Items) == 0 {
		return nil, exceptions.NewValidationError("user_id and at least one order item are required")
	}

	var createdOrder *entities.Order
	var totalPriceCents int64
	orderItems := make([]entities.OrderItem, 0, len(input.Items))
	itemDTOs := make([]dto.OrderItemDTO, 0, len(input.Items))

	// Step 1: Atomic Transaction (Deduct Stock & Create Order in PENDING)
	err := uc.uow.WithTransaction(ctx, func(rf ports.RepositoryFactory) error {
		for _, item := range input.Items {
			if item.Quantity <= 0 {
				return exceptions.NewValidationError("item quantity must be greater than zero")
			}

			product, err := rf.Products().FindByID(ctx, item.ProductID)
			if err != nil {
				return exceptions.NewNotFoundError(fmt.Sprintf("product not found: %s", item.ProductID))
			}

			if err := rf.Products().DeductStock(ctx, item.ProductID, item.Quantity); err != nil {
				return err
			}

			itemTotal := product.PriceCents * int64(item.Quantity)
			totalPriceCents += itemTotal

			orderItems = append(orderItems, entities.OrderItem{
				ProductID:      product.ID,
				ProductName:    product.Name,
				Quantity:       item.Quantity,
				UnitPriceCents: product.PriceCents,
			})

			itemDTOs = append(itemDTOs, dto.OrderItemDTO{
				ProductID:      product.ID,
				ProductName:    product.Name,
				Quantity:       item.Quantity,
				UnitPriceCents: product.PriceCents,
			})
		}

		createdOrder = &entities.Order{
			UserID:          input.UserID,
			TotalPriceCents: totalPriceCents,
			Status:          entities.OrderStatusPending,
			Items:           orderItems,
		}

		return rf.Orders().Create(ctx, createdOrder)
	})

	if err != nil {
		uc.logger.Warn("Failed to create pending order", "error", err, "user_id", input.UserID)
		return nil, err
	}

	// Step 2: Call Payment Service via gRPC
	payCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	payResp, payErr := uc.paymentClient.ProcessPayment(payCtx, &paymentv1.ProcessPaymentRequest{
		OrderId:       createdOrder.ID,
		UserId:        input.UserID,
		AmountCents:   totalPriceCents,
		PaymentMethod: input.PaymentMethod,
	})

	// Step 3: Handle Payment Outcome
	if payErr != nil || payResp.GetStatus() != "SUCCESS" {
		uc.logger.Error("Payment failed for order, restoring stock", "order_id", createdOrder.ID, "error", payErr)

		// Compensation: Restore deducted stock and mark order as FAILED
		for _, item := range orderItems {
			_ = uc.uow.Products().AddStock(ctx, item.ProductID, item.Quantity)
		}
		_ = uc.uow.Orders().UpdateStatus(ctx, createdOrder.ID, entities.OrderStatusFailed, "")

		return nil, exceptions.NewInternalError("payment processing failed, stock restored", payErr)
	}

	// Step 4: Confirm Order
	_ = uc.uow.Orders().UpdateStatus(ctx, createdOrder.ID, entities.OrderStatusCompleted, payResp.GetTransactionRef())
	uc.logger.Info("Order confirmed successfully", "order_id", createdOrder.ID, "total_cents", totalPriceCents)

	return &dto.OrderDTO{
		ID:              createdOrder.ID,
		UserID:          createdOrder.UserID,
		TotalPriceCents: createdOrder.TotalPriceCents,
		Status:          string(entities.OrderStatusCompleted),
		TransactionID:   payResp.GetTransactionRef(),
		CreatedAt:       createdOrder.CreatedAt,
		Items:           itemDTOs,
	}, nil
}

func (uc *orderUseCase) GetOrderByID(ctx context.Context, orderID string) (*dto.OrderDTO, error) {
	if orderID == "" {
		return nil, exceptions.NewValidationError("order_id is required")
	}

	order, err := uc.uow.Orders().FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	itemDTOs := make([]dto.OrderItemDTO, len(order.Items))
	for i, item := range order.Items {
		itemDTOs[i] = dto.OrderItemDTO{
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		}
	}

	return &dto.OrderDTO{
		ID:              order.ID,
		UserID:          order.UserID,
		TotalPriceCents: order.TotalPriceCents,
		Status:          string(order.Status),
		TransactionID:   order.TransactionID,
		CreatedAt:       order.CreatedAt,
		Items:           itemDTOs,
	}, nil
}

func (uc *orderUseCase) ListOrdersByUserID(ctx context.Context, userID string) ([]dto.OrderDTO, error) {
	if userID == "" {
		return nil, exceptions.NewValidationError("user_id is required")
	}

	orders, err := uc.uow.Orders().FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.OrderDTO, len(orders))
	for i, o := range orders {
		itemDTOs := make([]dto.OrderItemDTO, len(o.Items))
		for j, item := range o.Items {
			itemDTOs[j] = dto.OrderItemDTO{
				ProductID:      item.ProductID,
				Quantity:       item.Quantity,
				UnitPriceCents: item.UnitPriceCents,
			}
		}
		result[i] = dto.OrderDTO{
			ID:              o.ID,
			UserID:          o.UserID,
			TotalPriceCents: o.TotalPriceCents,
			Status:          string(o.Status),
			TransactionID:   o.TransactionID,
			CreatedAt:       o.CreatedAt,
			Items:           itemDTOs,
		}
	}
	return result, nil
}
