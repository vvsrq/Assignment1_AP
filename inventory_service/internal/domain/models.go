package domain

type Product struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Stock      int     `json:"stock"`
	CategoryID int     `json:"category_id"`
}

type Category struct {
	ID   int    `json:"id"`   // Category id
	Name string `json:"name"` // Category nma
}
