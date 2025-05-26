package models

type User struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Birthdate string `json:"birthdate"`
	Biography string `json:"biography"`
	City      string `json:"city"`
	Password  string `json:"password"`
}

type Credentials struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

type Post struct {
	ID           string `json:"id"`
	Text         string `json:"text"`
	CreatedAt    string `json:"created_at"`
	AuthorUserID string `json:"author_user_id"`
}
