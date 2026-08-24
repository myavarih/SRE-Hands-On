package controllers

import (
	"context"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	orderv1 "github.com/myavarih/sre-hands-on/services/gen/go/order/v1"
	"github.com/myavarih/sre-hands-on/services/order-service/internal/application/dto"
	"github.com/myavarih/sre-hands-on/services/order-service/internal/application/usecase"
)

// OrderGrpcController is a thin transport adapter mapping Protobuf to Application DTOs.
type OrderGrpcController struct {
	orderv1.UnimplementedOrderServiceServer
	orderUC usecase.OrderUseCase
}

func NewOrderGrpcController(orderUC usecase.OrderUseCase) *OrderGrpcController {
	return &OrderGrpcController{orderUC: orderUC}
}

func (c *OrderGrpcController) ListProducts(ctx context.Context, req *orderv1.ListProductsRequest) (*orderv1.ListProductsResponse, error) {
	products, err := c.orderUC.ListProducts(ctx)
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	protoProducts := make([]*orderv1.ProductItem, len(products))
	for i, p := range products {
		protoProducts[i] = &orderv1.ProductItem{
			Id:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			PriceCents:    p.PriceCents,
			StockQuantity: p.StockQuantity,
		}
	}

	return &orderv1.ListProductsResponse{Products: protoProducts}, nil
}

func (c *OrderGrpcController) GetProduct(ctx context.Context, req *orderv1.GetProductRequest) (*orderv1.GetProductResponse, error) {
	p, err := c.orderUC.GetProductByID(ctx, req.GetProductId())
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &orderv1.GetProductResponse{
		Product: &orderv1.ProductItem{
			Id:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			PriceCents:    p.PriceCents,
			StockQuantity: p.StockQuantity,
		},
	}, nil
}

func (c *OrderGrpcController) CreateProduct(ctx context.Context, req *orderv1.CreateProductRequest) (*orderv1.CreateProductResponse, error) {
	p, err := c.orderUC.CreateProduct(ctx, dto.CreateProductInput{
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		PriceCents:   req.GetPriceCents(),
		InitialStock: req.GetInitialStock(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &orderv1.CreateProductResponse{
		Product: &orderv1.ProductItem{
			Id:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			PriceCents:    p.PriceCents,
			StockQuantity: p.StockQuantity,
		},
	}, nil
}

func (c *OrderGrpcController) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	items := make([]dto.OrderItemInput, len(req.GetItems()))
	for i, item := range req.GetItems() {
		items[i] = dto.OrderItemInput{
			ProductID: item.GetProductId(),
			Quantity:  item.GetQuantity(),
		}
	}

	order, err := c.orderUC.CreateOrder(ctx, dto.CreateOrderInput{
		UserID:        req.GetUserId(),
		Items:         items,
		PaymentMethod: req.GetPaymentMethod(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	protoItems := make([]*orderv1.OrderItemDetail, len(order.Items))
	for i, item := range order.Items {
		protoItems[i] = &orderv1.OrderItemDetail{
			ProductId:      item.ProductID,
			ProductName:    item.ProductName,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		}
	}

	return &orderv1.CreateOrderResponse{
		OrderId:         order.ID,
		UserId:          order.UserID,
		TotalPriceCents: order.TotalPriceCents,
		Status:          order.Status,
		TransactionId:   order.TransactionID,
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		Items:           protoItems,
	}, nil
}

func (c *OrderGrpcController) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	order, err := c.orderUC.GetOrderByID(ctx, req.GetOrderId())
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	protoItems := make([]*orderv1.OrderItemDetail, len(order.Items))
	for i, item := range order.Items {
		protoItems[i] = &orderv1.OrderItemDetail{
			ProductId:      item.ProductID,
			ProductName:    item.ProductName,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		}
	}

	return &orderv1.GetOrderResponse{
		OrderId:         order.ID,
		UserId:          order.UserID,
		TotalPriceCents: order.TotalPriceCents,
		Status:          order.Status,
		TransactionId:   order.TransactionID,
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		Items:           protoItems,
	}, nil
}

func (c *OrderGrpcController) ListUserOrders(ctx context.Context, req *orderv1.ListUserOrdersRequest) (*orderv1.ListUserOrdersResponse, error) {
	orders, err := c.orderUC.ListOrdersByUserID(ctx, req.GetUserId())
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	protoOrders := make([]*orderv1.GetOrderResponse, len(orders))
	for i, o := range orders {
		protoItems := make([]*orderv1.OrderItemDetail, len(o.Items))
		for j, item := range o.Items {
			protoItems[j] = &orderv1.OrderItemDetail{
				ProductId:      item.ProductID,
				ProductName:    item.ProductName,
				Quantity:       item.Quantity,
				UnitPriceCents: item.UnitPriceCents,
			}
		}
		protoOrders[i] = &orderv1.GetOrderResponse{
			OrderId:         o.ID,
			UserId:          o.UserID,
			TotalPriceCents: o.TotalPriceCents,
			Status:          o.Status,
			TransactionId:   o.TransactionID,
			CreatedAt:       o.CreatedAt.Format(time.RFC3339),
			Items:           protoItems,
		}
	}

	return &orderv1.ListUserOrdersResponse{Orders: protoOrders}, nil
}
