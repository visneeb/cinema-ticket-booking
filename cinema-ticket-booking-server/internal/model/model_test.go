package model_test

import (
    "testing"
    "cinema-ticket-booking/internal/model"
)

func TestErrConst(t *testing.T) {
    if model.ErrSeatBooked.Error() != "seat already booked" {
        t.Fatalf("unexpected error message: %s", model.ErrSeatBooked.Error())
    }
    if model.ErrSeatLocked.Error() != "seat already locked by another user" {
        t.Fatalf("unexpected error message: %s", model.ErrSeatLocked.Error())
    }
}

func TestSeatStatusConstants(t *testing.T) {
    if model.SeatAvailable != "AVAILABLE" { t.Error("wrong AVAILABLE") }
    if model.SeatLocked    != "LOCKED"    { t.Error("wrong LOCKED") }
    if model.SeatBooked    != "BOOKED"    { t.Error("wrong BOOKED") }
}
