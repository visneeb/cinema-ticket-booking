package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"cinema-ticket-booking/internal/model"
)

// BookingFilter controls which bookings ListBookings returns.
type BookingFilter struct {
	MovieID  string
	UserID   string
	DateFrom string
	DateTo   string
	Status   string
	Page     int64
	PageSize int64
}

// BookingRepository wraps all MongoDB access for bookings, seats, and audit logs.
type BookingRepository struct {
	db *mongo.Database
}

type SeatDoc struct {
	ID         bson.ObjectID `bson:"_id"`
	Label      string        `bson:"label"`
	ShowtimeID bson.ObjectID `bson:"showtime_id"`
	Status     string        `bson:"status"`
}

func NewBookingRepository(db *mongo.Database) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) FindSeatsByShowtime(ctx context.Context, showtimeID bson.ObjectID) ([]SeatDoc, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor, err := r.db.Collection("seats").Find(ctx, bson.D{{Key: "showtime_id", Value: showtimeID}}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []SeatDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *BookingRepository) ListShowtimes(ctx context.Context) ([]model.Showtime, error) {
	cursor, err := r.db.Collection("showtimes").Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var showtimes []struct {
		ID       bson.ObjectID `bson:"_id"`
		MovieID  bson.ObjectID `bson:"movie_id"`
		StartsAt time.Time     `bson:"starts_at"`
	}
	if err := cursor.All(ctx, &showtimes); err != nil {
		return nil, err
	}

	movieIDSet := make(map[bson.ObjectID]struct{}, len(showtimes))
	for _, st := range showtimes {
		movieIDSet[st.MovieID] = struct{}{}
	}

	movieIDs := make([]bson.ObjectID, 0, len(movieIDSet))
	for id := range movieIDSet {
		movieIDs = append(movieIDs, id)
	}

	movieMap := make(map[string]struct {
		Title       string
		Description string
	})
	if len(movieIDs) > 0 {
		mc, err := r.db.Collection("movies").Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: movieIDs}}}})
		if err == nil {
			defer mc.Close(ctx)
			var movies []struct {
				ID          bson.ObjectID `bson:"_id"`
				Title       string        `bson:"title"`
				Description string        `bson:"description"`
			}
			if err := mc.All(ctx, &movies); err == nil {
				for _, m := range movies {
					movieMap[m.ID.Hex()] = struct {
						Title       string
						Description string
					}{Title: m.Title, Description: m.Description}
				}
			}
		}
	}

	result := make([]model.Showtime, len(showtimes))
	for i, st := range showtimes {
		movie := movieMap[st.MovieID.Hex()]
		result[i] = model.Showtime{
			ID:          st.ID.Hex(),
			MovieID:     st.MovieID.Hex(),
			StartsAt:    st.StartsAt,
			Title:       movie.Title,
			Description: movie.Description,
		}
	}

	return result, nil
}

// InsertBooking writes a new booking document and returns its generated ID.
func (r *BookingRepository) InsertBooking(ctx context.Context, b *model.Booking) (bson.ObjectID, error) {
	result, err := r.db.Collection("bookings").InsertOne(ctx, b)
	if err != nil {
		return bson.ObjectID{}, err
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		return oid, nil
	}
	return bson.ObjectID{}, nil
}

// SetSeatStatus writes the seat's status field (AVAILABLE | LOCKED | BOOKED).
// This is a write-through cache so GetSeats stays correct even if Redis restarts.
// seatID is the hex string form of the seat's ObjectID; invalid IDs are ignored.
func (r *BookingRepository) SetSeatStatus(ctx context.Context, seatID, status string) {
	seatOID, err := bson.ObjectIDFromHex(seatID)
	if err != nil {
		log.Printf("[BookingRepository.SetSeatStatus] invalid seat id %q: %v", seatID, err)
		return
	}
	_, err = r.db.Collection("seats").UpdateOne(ctx,
		bson.D{{Key: "_id", Value: seatOID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}}}},
	)
	if err != nil {
		log.Printf("[BookingRepository.SetSeatStatus] update error seat=%s: %v", seatID, err)
	}
}

// InsertAuditLog persists an audit event. Errors are logged, not propagated
// audit logging must never block the primary booking flow.
func (r *BookingRepository) InsertAuditLog(ctx context.Context, eventType, userID, showtimeID, seatID, msg string) {
	doc := model.AuditLog{
		EventType:  eventType,
		UserID:     userID,
		ShowtimeID: showtimeID,
		SeatID:     seatID,
		Message:    msg,
		CreatedAt:  time.Now().UTC(),
	}
	if _, err := r.db.Collection("audit_logs").InsertOne(ctx, doc); err != nil {
		log.Printf("[BookingRepository.InsertAuditLog] write error: %v", err)
	}
}

// ListBookings returns a paginated, filtered list of bookings for the admin dashboard.
func (r *BookingRepository) ListBookings(ctx context.Context, f BookingFilter) ([]model.Booking, int64, error) {
	filter := bson.D{}

	if f.UserID != "" {
		filter = append(filter, bson.E{Key: "user_id", Value: f.UserID})
	}
	if f.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: f.Status})
	}

	// Date range on created_at
	if f.DateFrom != "" || f.DateTo != "" {
		dateFilter := bson.D{}
		if f.DateFrom != "" {
			if t, err := time.Parse("2006-01-02", f.DateFrom); err == nil {
				dateFilter = append(dateFilter, bson.E{Key: "$gte", Value: t.UTC()})
			}
		}
		if f.DateTo != "" {
			if t, err := time.Parse("2006-01-02", f.DateTo); err == nil {
				// include the full end day
				dateFilter = append(dateFilter, bson.E{Key: "$lte", Value: t.Add(24*time.Hour - time.Second).UTC()})
			}
		}
		if len(dateFilter) > 0 {
			filter = append(filter, bson.E{Key: "created_at", Value: dateFilter})
		}
	}

	// Movie filter
	if f.MovieID != "" {
		movieOID, err := bson.ObjectIDFromHex(f.MovieID)
		if err == nil {
			cur, err := r.db.Collection("showtimes").Find(ctx, bson.D{{Key: "movie_id", Value: movieOID}})
			if err == nil {
				defer cur.Close(ctx)
				var showtimeIDs []string
				for cur.Next(ctx) {
					var doc struct {
						ID bson.ObjectID `bson:"_id"`
					}
					if cur.Decode(&doc) == nil {
						showtimeIDs = append(showtimeIDs, doc.ID.Hex())
					}
				}
				if len(showtimeIDs) > 0 {
					filter = append(filter, bson.E{Key: "showtime_id", Value: bson.D{{Key: "$in", Value: showtimeIDs}}})
				}
			}
		}
	}

	total, err := r.db.Collection("bookings").CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (f.Page - 1) * f.PageSize
	opts := options.Find().
		SetSkip(skip).
		SetLimit(f.PageSize).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := r.db.Collection("bookings").Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var bookings []model.Booking
	if err := cur.All(ctx, &bookings); err != nil {
		return nil, 0, err
	}
	return bookings, total, nil
}
