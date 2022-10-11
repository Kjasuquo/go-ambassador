package models

type Link struct {
	Model
	Code     string    `json:"code"`
	UserId   uint      `json:"user_id"`
	Products []Product `json:"products" gorm:"many2many:link_products"`
	Orders   []Order   `json:"orders,omitempty" gorm:"-"`
}
