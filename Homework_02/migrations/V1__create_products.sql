DO $$ BEGIN
    CREATE TYPE product_status AS ENUM (
        'ACTIVE', 'INACTIVE', 'ARCHIVED'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;


create table if not exists products(
    id uuid primary key default gen_random_uuid(),
    name text not null,
    description text,
    price decimal(12,2) not null,
    stock int not null default 0,
    category text not null,
    status product_status not null default 'ACTIVE',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);


CREATE OR REPLACE FUNCTION update_updated_at()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();


CREATE INDEX IF NOT EXISTS idx_product_status ON products(status);