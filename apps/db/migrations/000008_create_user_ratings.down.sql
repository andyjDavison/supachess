ALTER TABLE games
    DROP COLUMN white_rating,
    DROP COLUMN black_rating,
    DROP COLUMN white_k,
    DROP COLUMN black_k,
    DROP COLUMN white_rating_delta,
    DROP COLUMN black_rating_delta;

DROP TABLE user_ratings;
