package domain

type CategoryRepository interface {
	CreateCategory(category *Category) (*Category, error)
	GetCategoryByID(id int) (*Category, error)
	UpdateCategory(category *Category) (*Category, error)
	DeleteCategory(id int) error
	ListCategories() ([]Category, error)
}
