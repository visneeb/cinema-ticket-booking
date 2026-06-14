package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// GetSeatsByIDs returns a map of seatID -> SeatDoc for the provided hex IDs.
func (r *BookingRepository) GetSeatsByIDs(ctx context.Context, seatIDs []string) (map[string]SeatDoc, error) {
	if len(seatIDs) == 0 {
		return map[string]SeatDoc{}, nil
	}
	ids := make([]bson.ObjectID, 0, len(seatIDs))
	for _, s := range seatIDs {
		if oid, err := bson.ObjectIDFromHex(s); err == nil {
			ids = append(ids, oid)
		}
	}
	if len(ids) == 0 {
		return map[string]SeatDoc{}, nil
	}
	cur, err := r.db.Collection("seats").Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []SeatDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	m := make(map[string]SeatDoc, len(docs))
	for _, d := range docs {
		m[d.ID.Hex()] = d
	}
	return m, nil
}

// GetShowtimesMovieIDs returns a map showtimeID -> movieID (hex strings).
func (r *BookingRepository) GetShowtimesMovieIDs(ctx context.Context, showtimeIDs []string) (map[string]string, error) {
	if len(showtimeIDs) == 0 {
		return map[string]string{}, nil
	}
	ids := make([]bson.ObjectID, 0, len(showtimeIDs))
	for _, s := range showtimeIDs {
		if oid, err := bson.ObjectIDFromHex(s); err == nil {
			ids = append(ids, oid)
		}
	}
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	cur, err := r.db.Collection("showtimes").Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var sts []struct {
		ID      bson.ObjectID `bson:"_id"`
		MovieID bson.ObjectID `bson:"movie_id"`
	}
	if err := cur.All(ctx, &sts); err != nil {
		return nil, err
	}
	m := make(map[string]string, len(sts))
	for _, s := range sts {
		m[s.ID.Hex()] = s.MovieID.Hex()
	}
	return m, nil
}
