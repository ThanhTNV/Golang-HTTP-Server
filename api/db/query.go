package db

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name string
	Age  int
	Pets []Pet `gorm:"many2many:user_pets"`
}

type Pet struct {
	gorm.Model
	Name string
}

type Query[T any] interface {
	// SELECT * FROM @@table WHERE id=@id
	GetByID(id int) (T, error)
}
