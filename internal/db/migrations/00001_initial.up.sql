-- 1. Create a custom enum type for order status
CREATE TYPE order_status AS ENUM ('pending', 'shipped', 'delivered', 'cancelled');

-- 2. Create an addresses table
CREATE TABLE addresses (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- requires pgcrypto or extension for gen_random_uuid()
    line1              TEXT NOT NULL,
    line2              TEXT,
    city               TEXT NOT NULL,
    state              TEXT NOT NULL,
    postal_code        TEXT NOT NULL,
    country            TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),          -- timestamp of order creation
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()           -- timestamp of last update
);

-- 3. Create an orders table
CREATE TABLE orders (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- unique identifier of the order
    user_id            UUID NOT NULL,                               -- id referencing a users table (if you have one)
    shipping_address_id UUID REFERENCES addresses(id),       -- link to the shipping address
    billing_address_id  UUID REFERENCES addresses(id),       -- link to the billing address
    total_amount       NUMERIC(10,2) NOT NULL,                      -- total amount of the order
    status             order_status NOT NULL DEFAULT 'pending',     -- current status of the order
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),          -- timestamp of order creation
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()           -- timestamp of last update
);

-- 4. Create an order_items table
CREATE TABLE order_items (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id           UUID NOT NULL REFERENCES orders(id),   -- links back to the parent order
    product_id         UUID NOT NULL,                               -- links to a products table (if you have one)
    quantity           INT NOT NULL,
    price              NUMERIC(10,2) NOT NULL,                      -- price per unit
    total_price        NUMERIC(10,2) NOT NULL,                      -- total price for this item (quantity * price)
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),          -- timestamp of order creation
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()           -- timestamp of last update
);
