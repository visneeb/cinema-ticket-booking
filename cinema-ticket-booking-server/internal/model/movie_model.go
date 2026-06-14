package model

// Movie is stored in the cinema.movies collection.
type Movie struct {
	ID          interface{} `bson:"_id"         json:"id"`
	Title       string      `bson:"title"       json:"title"`
	Description string      `bson:"description" json:"description"`
}
