// domain/product.go
package domain

type ProductRepository interface {
	CreateProduct(product *Product) (*Product, error)
	GetProductByID(id int) (*Product, error)

	UpdateProduct(id int, updates map[string]interface{}) (*Product, error)

	DeleteProduct(id int) error
	ListProducts(limit, offset int) ([]Product, error)
	ListProductsByCategory(categoryID, limit, offset int) ([]Product, error)
}
