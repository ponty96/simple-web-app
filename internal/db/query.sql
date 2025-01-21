-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: ListOrders :many
SELECT * FROM orders
WHERE user_id = $1
ORDER BY updated_at;

-- name: CreateOrder :one
INSERT INTO orders (
  user_id, total_amount, status,
  shipping_address_id, billing_address_id
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: CreateAddress :one
INSERT INTO addresses (
 line1, city, state, postal_code,
 country, line2
) VALUES (
 $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (
 order_id, product_id, quantity,
 price, total_price
) VALUES (
 $1, $2, $3, $4, $5
)
RETURNING *;
