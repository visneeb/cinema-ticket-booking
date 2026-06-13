// Seed script: runs once on first container start
db = db.getSiblingDB("cinema");

// Create indexes
db.seats.createIndex({ showtime_id: 1, status: 1 });
db.bookings.createIndex({ user_id: 1 });
db.bookings.createIndex({ showtime_id: 1 });
db.audit_logs.createIndex({ created_at: -1 });
db.users.createIndex({ google_id: 1 }, { unique: true });

// Seed one movie + showtime + 40 seats (rows A-D, cols 1-10)
const movieId = new ObjectId();
const showId = new ObjectId();

db.movies.insertOne({
  _id: movieId,
  title: "Inception",
  description: "A mind-bending thriller",
});

db.showtimes.insertOne({
  _id: showId,
  movie_id: movieId,
  starts_at: new Date(Date.now() + 86400000),
});

const rows = ["A", "B", "C", "D"];
const seats = [];
rows.forEach((row) => {
  for (let col = 1; col <= 10; col++) {
    seats.push({
      _id: new ObjectId(),
      showtime_id: showId,
      label: `${row}${col}`,
      status: "AVAILABLE",
    });
  }
});
db.seats.insertMany(seats);

print("Seed complete: 1 movie, 1 showtime, 40 seats");
