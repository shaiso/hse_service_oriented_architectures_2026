ALTER TABLE promo_codes
    ALTER COLUMN discount_value SET NOT NULL,
    ALTER COLUMN min_order_amount SET NOT NULL,
    ALTER COLUMN max_uses SET NOT NULL,
    ALTER COLUMN valid_until SET NOT NULL;

ALTER TABLE orders
    ALTER COLUMN total_amount SET NOT NULL,
    ALTER COLUMN discount_amount SET NOT NULL,
    ALTER COLUMN discount_amount SET DEFAULT 0;

DO $$ BEGIN
CREATE TYPE user_role AS ENUM ('USER', 'SELLER', 'ADMIN');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'USER',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
