package model

type ListBookingRequest struct {
	MovieID  string
	UserID   string
	DateFrom string
	DateTo   string
	Status   string
	Page     int64
	PageSize int64
}

type BookingResponse struct {
	Items    []Booking 		`json:"items"`
	Total    int64           `json:"total"`
	Page     int64           `json:"page"`
	PageSize int64           `json:"page_size"`
}