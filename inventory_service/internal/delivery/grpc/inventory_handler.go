package grpc

import (
	"context"
	"inventory_service/internal/domain"
	"inventory_service/internal/usecase"
	inventorypb "inventory_service/proto"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type InventoryHandler struct {
	inventorypb.UnimplementedInventoryServiceServer
	productUseCase  usecase.ProductUseCase
	categoryUseCase usecase.CategoryUseCase
	log             *logrus.Logger
}

func NewInventoryHandler(puc usecase.ProductUseCase, cuc usecase.CategoryUseCase, logger *logrus.Logger) *InventoryHandler {
	return &InventoryHandler{
		productUseCase:  puc,
		categoryUseCase: cuc,
		log:             logger,
	}
}

func mapDomainCategoryToProto(cat *domain.Category) *inventorypb.Category {
	if cat == nil {
		return nil
	}
	return &inventorypb.Category{
		Id:   int64(cat.ID),
		Name: cat.Name,
	}
}

func mapDomainProductToProto(prod *domain.Product) *inventorypb.Product {
	if prod == nil {
		return nil
	}
	return &inventorypb.Product{
		Id:         int64(prod.ID),
		Name:       prod.Name,
		Price:      prod.Price,
		Stock:      int32(prod.Stock),
		CategoryId: int64(prod.CategoryID),
	}
}

func (h *InventoryHandler) CreateCategory(ctx context.Context, req *inventorypb.CreateCategoryRequest) (*inventorypb.Category, error) {
	h.log.Infof("gRPC Handler: Received CreateCategory request: Name=%s", req.GetName())
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Category name cannot be empty")
	}

	domainCat := &domain.Category{Name: req.GetName()}
	createdCat, err := h.categoryUseCase.CreateCategory(domainCat)
	if err != nil {
		h.log.Errorf("gRPC Handler: CreateCategory use case error: %v", err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Category created successfully: ID=%d", createdCat.ID)
	return mapDomainCategoryToProto(createdCat), nil
}

func (h *InventoryHandler) GetCategory(ctx context.Context, req *inventorypb.GetCategoryRequest) (*inventorypb.Category, error) {
	id := int(req.GetId())
	h.log.Infof("gRPC Handler: Received GetCategory request: ID=%d", id)
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid category ID")
	}

	cat, err := h.categoryUseCase.GetCategoryByID(id)
	if err != nil {
		h.log.Warnf("gRPC Handler: GetCategory use case error for ID %d: %v", id, err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Category retrieved successfully: ID=%d", cat.ID)
	return mapDomainCategoryToProto(cat), nil
}

func (h *InventoryHandler) UpdateCategory(ctx context.Context, req *inventorypb.UpdateCategoryRequest) (*inventorypb.Category, error) {
	protoCat := req.GetCategory()
	if protoCat == nil || protoCat.GetId() <= 0 || protoCat.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Valid category ID and name are required for update")
	}
	id := int(protoCat.GetId())
	h.log.Infof("gRPC Handler: Received UpdateCategory request: ID=%d, NewName=%s", id, protoCat.GetName())

	domainCat := &domain.Category{
		ID:   id,
		Name: protoCat.GetName(),
	}

	updatedCat, err := h.categoryUseCase.UpdateCategory(domainCat)
	if err != nil {
		h.log.Errorf("gRPC Handler: UpdateCategory use case error for ID %d: %v", id, err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Category updated successfully: ID=%d", updatedCat.ID)
	return mapDomainCategoryToProto(updatedCat), nil
}

func (h *InventoryHandler) DeleteCategory(ctx context.Context, req *inventorypb.DeleteCategoryRequest) (*empty.Empty, error) {
	id := int(req.GetId())
	h.log.Infof("gRPC Handler: Received DeleteCategory request: ID=%d", id)
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid category ID")
	}

	err := h.categoryUseCase.DeleteCategory(id)
	if err != nil {
		h.log.Warnf("gRPC Handler: DeleteCategory use case error for ID %d: %v", id, err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Category deleted successfully: ID=%d", id)
	return &emptypb.Empty{}, nil
}

func (h *InventoryHandler) ListCategories(ctx context.Context, req *inventorypb.ListCategoriesRequest) (*inventorypb.ListCategoriesResponse, error) {
	h.log.Info("gRPC Handler: Received ListCategories request")

	cats, err := h.categoryUseCase.ListCategories()
	if err != nil {
		h.log.Errorf("gRPC Handler: ListCategories use case error: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to list categories: %v", err)
	}

	resp := &inventorypb.ListCategoriesResponse{
		Categories: make([]*inventorypb.Category, 0, len(cats)),
	}
	for i := range cats {
		resp.Categories = append(resp.Categories, mapDomainCategoryToProto(&cats[i]))
	}

	h.log.Infof("gRPC Handler: Listed %d categories successfully", len(resp.Categories))
	return resp, nil
}

func (h *InventoryHandler) CreateProduct(ctx context.Context, req *inventorypb.CreateProductRequest) (*inventorypb.Product, error) {
	h.log.Infof("gRPC Handler: Received CreateProduct request: Name=%s", req.GetName())
	if req.GetName() == "" || req.GetPrice() <= 0 || req.GetStock() < 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid product data: Name required, price must be positive, stock cannot be negative")
	}

	domainProd := &domain.Product{
		Name:       req.GetName(),
		Price:      req.GetPrice(),
		Stock:      int(req.GetStock()),
		CategoryID: int(req.GetCategoryId()),
	}

	createdProd, err := h.productUseCase.CreateProduct(domainProd)
	if err != nil {
		h.log.Errorf("gRPC Handler: CreateProduct use case error: %v", err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Product created successfully: ID=%d", createdProd.ID)
	return mapDomainProductToProto(createdProd), nil
}

func (h *InventoryHandler) GetProduct(ctx context.Context, req *inventorypb.GetProductRequest) (*inventorypb.Product, error) {
	id := int(req.GetId())
	h.log.Infof("gRPC Handler: Received GetProduct request: ID=%d", id)
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid product ID")
	}

	prod, err := h.productUseCase.GetProductByID(id)
	if err != nil {
		h.log.Warnf("gRPC Handler: GetProduct use case error for ID %d: %v", id, err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Product retrieved successfully: ID=%d", prod.ID)
	return mapDomainProductToProto(prod), nil
}

func (h *InventoryHandler) UpdateProduct(ctx context.Context, req *inventorypb.UpdateProductRequest) (*inventorypb.Product, error) {
	protoProd := req.GetProduct()
	mask := req.GetUpdateMask()

	if protoProd == nil || protoProd.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Valid product with ID is required for update")
	}
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Field mask is required for update")
	}

	id := int(protoProd.GetId())
	h.log.Infof("gRPC Handler: Received UpdateProduct request: ID=%d, MaskPaths=%v", id, mask.GetPaths())

	updates := make(map[string]interface{})
	for _, path := range mask.GetPaths() {
		switch path {
		case "name":
			if protoProd.GetName() == "" {
				return nil, status.Errorf(codes.InvalidArgument, "Product name cannot be empty if included in mask")
			}
			updates["name"] = protoProd.GetName()
		case "price":
			if protoProd.GetPrice() <= 0 {
				return nil, status.Errorf(codes.InvalidArgument, "Product price must be positive if included in mask")
			}
			updates["price"] = protoProd.GetPrice()
		case "stock":
			if protoProd.GetStock() < 0 {
				return nil, status.Errorf(codes.InvalidArgument, "Product stock cannot be negative if included in mask")
			}
			updates["stock"] = int(protoProd.GetStock())
		case "category_id":

			catID := protoProd.GetCategoryId()
			if catID < 0 {
				return nil, status.Errorf(codes.InvalidArgument, "Category ID must be non-negative if included in mask")
			}
			updates["category_id"] = int(catID)
		default:
			h.log.Warnf("gRPC Handler: UpdateProduct ignoring unknown path in mask: %s", path)
		}
	}

	if len(updates) == 0 {
		h.log.Warnf("gRPC Handler: UpdateProduct request for ID %d resulted in empty valid updates map after processing mask.", id)
		currentProd, err := h.productUseCase.GetProductByID(id)
		if err != nil {
			return nil, mapDomainErrorToGrpcStatus(err)
		}
		return mapDomainProductToProto(currentProd), nil
	}

	updatedProd, err := h.productUseCase.UpdateProduct(id, updates)
	if err != nil {
		h.log.Errorf("gRPC Handler: UpdateProduct use case error for ID %d: %v", id, err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Product updated successfully: ID=%d", updatedProd.ID)
	return mapDomainProductToProto(updatedProd), nil
}

func (h *InventoryHandler) DeleteProduct(ctx context.Context, req *inventorypb.DeleteProductRequest) (*empty.Empty, error) {
	id := int(req.GetId())
	h.log.Infof("gRPC Handler: Received DeleteProduct request: ID=%d", id)
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid product ID")
	}

	err := h.productUseCase.DeleteProduct(id)
	if err != nil {
		h.log.Warnf("gRPC Handler: DeleteProduct use case error for ID %d: %v", id, err)
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Product deleted successfully: ID=%d", id)
	return &emptypb.Empty{}, nil
}

func (h *InventoryHandler) ListProducts(ctx context.Context, req *inventorypb.ListProductsRequest) (*inventorypb.ListProductsResponse, error) {
	limit := int(req.GetLimit())
	offset := int(req.GetOffset())
	categoryIDFilter := req.GetCategoryIdFilter()

	h.log.Infof("gRPC Handler: Received ListProducts request: Limit=%d, Offset=%d, CategoryFilterPresent=%t", limit, offset, categoryIDFilter != nil)

	var products []domain.Product
	var err error

	if categoryIDFilter != nil {
		catID := int(categoryIDFilter.GetValue())
		if catID <= 0 {
			return nil, status.Error(codes.InvalidArgument, "Invalid category ID filter value")
		}
		h.log.Infof("gRPC Handler: Listing products by category: %d", catID)
		products, err = h.productUseCase.ListProductsByCategory(catID, limit, offset)
	} else {
		h.log.Info("gRPC Handler: Listing all products")
		products, err = h.productUseCase.ListProducts(limit, offset)
	}

	if err != nil {
		h.log.Errorf("gRPC Handler: ListProducts use case error: %v", err)
		if categoryIDFilter != nil && strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Failed to list products: category %d not found", categoryIDFilter.GetValue())
		}
		return nil, mapDomainErrorToGrpcStatus(err)
	}

	resp := &inventorypb.ListProductsResponse{
		Products: make([]*inventorypb.Product, 0, len(products)),
	}
	for i := range products {
		resp.Products = append(resp.Products, mapDomainProductToProto(&products[i]))
	}

	h.log.Infof("gRPC Handler: Listed %d products successfully", len(resp.Products))
	return resp, nil
}

func mapDomainErrorToGrpcStatus(err error) error {
	if err == nil {
		return nil
	}
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "not found"):
		return status.Error(codes.NotFound, err.Error())
	case strings.Contains(errMsg, "already exists"),
		strings.Contains(errMsg, "duplicate key"),
		strings.Contains(errMsg, "unique constraint"):
		return status.Error(codes.AlreadyExists, err.Error())
	case strings.Contains(errMsg, "invalid"),
		strings.Contains(errMsg, "cannot be empty"),
		strings.Contains(errMsg, "must be positive"),
		strings.Contains(errMsg, "cannot be negative"),
		strings.Contains(errMsg, "constraint violation"),
		strings.Contains(errMsg, "does not exist") && strings.Contains(errMsg, "category"):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "Internal server error: %v", err)
	}
}
