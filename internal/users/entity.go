package users

type User struct {
	ID        string `json:"id" bson:"_id"`
	Name      string `json:"name" bson:"name"`
	Email     string `json:"email" bson:"email"`
	Password  string `json:"-" bson:"password"`
	CreatedAt string `json:"created_at" bson:"created_at"`
}

type UpdateFields struct {
	Name  *string
	Email *string
}
