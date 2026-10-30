package domain

// RatingCategory is the speed a rating belongs to. Every player has one
// rating per category; a game's category comes from its time control
// (rating.CategoryFor).
type RatingCategory string

const (
	CategoryBullet RatingCategory = "bullet"
	CategoryBlitz  RatingCategory = "blitz"
	CategoryRapid  RatingCategory = "rapid"
)

// PlayerRating is one player's standing in one category.
type PlayerRating struct {
	Rating      int32
	GamesPlayed int
}

// SideRating is one side's rating as of the game's start, plus what the
// game did to it. K is snapshotted at creation so the stakes shown when the
// game starts are exactly what's applied when it ends.
type SideRating struct {
	Before int32
	K      int32
	Delta  *int32 // nil until the game finishes
}

// GameRatings is the rating snapshot carried by every game created since
// ratings were introduced. Older games have none (Game.Ratings is nil).
type GameRatings struct {
	Category RatingCategory
	White    SideRating
	Black    SideRating
}
