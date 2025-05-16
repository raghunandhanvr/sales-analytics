package models

type ProductRevenue struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Revenue     float64 `json:"revenue"`
}

type CategoryRevenue struct {
	Category string  `json:"category"`
	Revenue  float64 `json:"revenue"`
}

type RegionRevenue struct {
	Region  string  `json:"region"`
	Revenue float64 `json:"revenue"`
}

type TopProduct struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	QuantitySold int     `json:"quantity_sold"`
	Revenue      float64 `json:"revenue"`
}
