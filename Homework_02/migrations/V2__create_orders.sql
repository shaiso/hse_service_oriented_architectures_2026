DO $$ BEGIN
CREATE TYPE order_status AS ENUM (
        'CREATED', 'PAYMENT_PENDING', 'PAID', 'SHIPPED', 'COMPLETED', 'CANCELED'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
CREATE TYPE discount_type AS ENUM (
        'PERCENTAGE', 'FIXED_AMOUNT'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
CREATE TYPE operation_type AS ENUM (
        'CREATE_ORDER', 'UPDATE_ORDER'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

create table if not exists promo_codes(
    id uuid primary key default gen_random_uuid(),
    code varchar(20) not null unique,
    discount_type discount_type not null,
    discount_value decimal(12,2),
    min_order_amount decimal(12,2),
    max_uses int,
    current_uses int default 0,
    valid_from timestamptz not null default now(),
    valid_until timestamptz,
    active boolean default true
);

create table if not exists orders(
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null,
    status order_status not null,
    promo_code_id uuid references promo_codes(id),
    total_amount decimal(12,2),
    discount_amount decimal(12,2),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

alter table products
    alter column name type varchar(255),
    alter column description type varchar(4000),
    alter column category type varchar(100),
    add column seller_id uuid;

create table if not exists order_items(
    id uuid primary key default gen_random_uuid(),
    order_id uuid references orders(id),
    product_id uuid references products(id),
    quantity int not null,
    price_at_order decimal(12,2) not null
);

create table if not exists user_operations(
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null,
    operation_type operation_type not null,
    created_at timestamptz not null default now()
);

CREATE TRIGGER trg_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();