package usecase

import (
	"errors"
	"fmt"
	"inventory_service/internal/domain"

	"github.com/sirupsen/logrus"
)

type ProductUseCase interface {
	CreateProduct(product *domain.Product) (*domain.Product, error)
	GetProductByID(id int) (*domain.Product, error)
	UpdateProduct(id int, updates map[string]interface{}) (*domain.Product, error)
	DeleteProduct(id int) error
	ListProducts(limit, offset int) ([]domain.Product, error)
	ListProductsByCategory(categoryID, limit, offset int) ([]domain.Product, error)
}

type productUseCase struct {
	productRepo  domain.ProductRepository
	categoryRepo domain.CategoryRepository
	log          *logrus.Logger
}

// NewProductUseCase (без изменений)
func NewProductUseCase(pRepo domain.ProductRepository, cRepo domain.CategoryRepository, logger *logrus.Logger) ProductUseCase {
	return &productUseCase{
		productRepo:  pRepo,
		categoryRepo: cRepo,
		log:          logger,
	}
}

func (uc *productUseCase) CreateProduct(product *domain.Product) (*domain.Product, error) {
	if product.Name == "" {
		uc.log.Warn("Use Case: Attempted to create product with empty name")
		return nil, errors.New("product name cannot be empty")
	}
	if product.Price <= 0 {
		uc.log.Warnf("Use Case: Attempted to create product '%s' with invalid price: %f", product.Name, product.Price)
		return nil, errors.New("product price must be positive")
	}
	if product.Stock < 0 {
		uc.log.Warnf("Use Case: Attempted to create product '%s' with negative stock: %d", product.Name, product.Stock)
		return nil, errors.New("product stock cannot be negative")
	}
	if product.CategoryID != 0 {
		_, err := uc.categoryRepo.GetCategoryByID(product.CategoryID)
		if err != nil {
			uc.log.Warnf("Use Case: Category ID %d not found during product creation: %v", product.CategoryID, err)
			return nil, fmt.Errorf("category with id %d does not exist", product.CategoryID)
		}
	}

	uc.log.Infof("Use Case: Attempting to create product '%s'", product.Name)
	createdProduct, err := uc.productRepo.CreateProduct(product)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to create product '%s': %v", product.Name, err)
		return nil, err
	}

	uc.log.Infof("Use Case: Product '%s' created successfully with ID %d", createdProduct.Name, createdProduct.ID)
	return createdProduct, nil
}

func (uc *productUseCase) GetProductByID(id int) (*domain.Product, error) {
	if id <= 0 {
		uc.log.Warnf("Use Case: Attempted to get product with invalid ID: %d", id)
		return nil, errors.New("invalid product ID")
	}

	uc.log.Infof("Use Case: Attempting to get product with ID %d", id)
	product, err := uc.productRepo.GetProductByID(id)
	if err != nil {
		uc.log.Warnf("Use Case: Repository failed to get product ID %d: %v", id, err)
		return nil, err
	}

	uc.log.Infof("Use Case: Product retrieved successfully for ID %d", id)
	return product, nil
}

func (uc *productUseCase) UpdateProduct(id int, updates map[string]interface{}) (*domain.Product, error) {
	if id <= 0 {
		uc.log.Warnf("Use Case: Attempted update with invalid product ID: %d", id)
		return nil, errors.New("invalid product ID for update")
	}
	if len(updates) == 0 {
		uc.log.Warnf("Use Case: Attempted update for product ID %d with no fields", id)

		return uc.productRepo.GetProductByID(id)
	}

	_, err := uc.productRepo.GetProductByID(id)
	if err != nil {
		uc.log.Warnf("Use Case: Product ID %d not found for update: %v", id, err)
		return nil, err
	}

	validUpdates := make(map[string]interface{})
	for key, value := range updates {
		switch key {
		case "name":
			name, ok := value.(string)
			if !ok || name == "" {
				uc.log.Warnf("Use Case: Invalid or empty 'name' provided for update ID %d", id)
				return nil, errors.New("product name cannot be empty if provided for update")
			}
			validUpdates[key] = name
		case "price":
			price, ok := value.(float64)
			if !ok || price <= 0 {
				uc.log.Warnf("Use Case: Invalid or non-positive 'price' provided for update ID %d", id)
				return nil, errors.New("product price must be positive if provided for update")
			}
			validUpdates[key] = price
		case "stock":

			var stock int
			var ok bool
			if stockFloat, okFloat := value.(float64); okFloat {
				stock = int(stockFloat)
				if float64(stock) != stockFloat {
					uc.log.Warnf("Use Case: Potential precision loss converting stock '%v' to int for update ID %d", value, id)
					return nil, errors.New("invalid type or precision for stock")
				}
				ok = true
			} else if stockInt, okInt := value.(int); okInt {
				stock = stockInt
				ok = true
			}

			if !ok || stock < 0 {
				uc.log.Warnf("Use Case: Invalid or negative 'stock' provided for update ID %d", id)
				return nil, errors.New("product stock cannot be negative if provided for update")
			}
			validUpdates[key] = stock
		case "category_id":
			var catID int
			var ok bool
			if catIDFloat, okFloat := value.(float64); okFloat {
				catID = int(catIDFloat)
				if float64(catID) != catIDFloat {
					uc.log.Warnf("Use Case: Potential precision loss converting category_id '%v' to int for update ID %d", value, id)
					return nil, errors.New("invalid type or precision for category_id")
				}
				ok = true
			} else if catIDInt, okInt := value.(int); okInt {
				catID = catIDInt
				ok = true
			} else if value == nil {
				catID = 0
				ok = true
			}

			if !ok {
				uc.log.Warnf("Use Case: Invalid type for 'category_id' provided for update ID %d", id)
				return nil, errors.New("invalid type for category_id")
			}

			if catID == 0 {
				validUpdates[key] = catID
			} else if catID > 0 {
				_, err := uc.categoryRepo.GetCategoryByID(catID)
				if err != nil {
					uc.log.Warnf("Use Case: Category ID %d not found during product update for ID %d: %v", catID, id, err)
					return nil, fmt.Errorf("category with id %d does not exist", catID)
				}
				validUpdates[key] = catID
			} else {
				uc.log.Warnf("Use Case: Invalid 'category_id' (%d) provided for update ID %d", catID, id)
				return nil, errors.New("category_id must be positive or 0/null")
			}

		default:
			uc.log.Warnf("Use Case: Attempted to update unknown or unsupported field '%s' for product ID %d", key, id)

		}
	}

	if len(validUpdates) == 0 {
		uc.log.Info("Use Case: No valid fields remaining after validation for update ID %d", id)
		return uc.productRepo.GetProductByID(id)
	}

	uc.log.Infof("Use Case: Attempting partial update for product ID %d with valid fields: %v", id, validUpdates)

	updatedProduct, err := uc.productRepo.UpdateProduct(id, validUpdates)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed partial update for product ID %d: %v", id, err)
		return nil, err
	}

	uc.log.Infof("Use Case: Product updated successfully for ID %d", updatedProduct.ID)
	return updatedProduct, nil
}

func (uc *productUseCase) DeleteProduct(id int) error {
	if id <= 0 {
		uc.log.Warnf("Use Case: Attempted delete with invalid product ID: %d", id)
		return errors.New("invalid product ID for delete")
	}
	uc.log.Infof("Use Case: Attempting to delete product ID %d", id)
	err := uc.productRepo.DeleteProduct(id)
	if err != nil {
		uc.log.Warnf("Use Case: Repository failed to delete product ID %d: %v", id, err)
		return err
	}
	uc.log.Infof("Use Case: Product deleted successfully for ID %d", id)
	return nil
}

func (uc *productUseCase) ListProducts(limit, offset int) ([]domain.Product, error) {
	if limit < 0 || offset < 0 {
		uc.log.Warnf("Use Case: Invalid pagination parameters (limit: %d, offset: %d)", limit, offset)
	}
	uc.log.Infof("Use Case: Attempting to list products (limit: %d, offset: %d)", limit, offset)
	products, err := uc.productRepo.ListProducts(limit, offset)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to list products: %v", err)
		return nil, fmt.Errorf("could not retrieve products: %w", err)
	}
	uc.log.Infof("Use Case: Retrieved %d products", len(products))
	return products, nil
}

func (uc *productUseCase) ListProductsByCategory(categoryID, limit, offset int) ([]domain.Product, error) {
	if categoryID <= 0 {
		uc.log.Warnf("Use Case: Attempted list by category with invalid category ID: %d", categoryID)
		return nil, errors.New("invalid category ID")
	}
	if limit < 0 || offset < 0 {
		uc.log.Warnf("Use Case: Invalid pagination parameters for category listing (limit: %d, offset: %d)", limit, offset)
	}
	_, err := uc.categoryRepo.GetCategoryByID(categoryID)
	if err != nil {
		uc.log.Warnf("Use Case: Category ID %d not found: %v", categoryID, err)
		return nil, fmt.Errorf("category with id %d not found", categoryID)
	}
	uc.log.Infof("Use Case: Attempting to list products for category %d (limit: %d, offset: %d)", categoryID, limit, offset)
	products, err := uc.productRepo.ListProductsByCategory(categoryID, limit, offset)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to list products for category %d: %v", categoryID, err)
		return nil, fmt.Errorf("could not retrieve products for category %d: %w", categoryID, err)
	}
	uc.log.Infof("Use Case: Retrieved %d products for category %d", len(products), categoryID)
	return products, nil
}
