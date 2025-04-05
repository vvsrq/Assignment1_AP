DROP TRIGGER IF EXISTS set_orders_timestamp ON orders;
DROP FUNCTION IF EXISTS trigger_set_timestamp(); -- Удаляем только если она больше нигде не нужна

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_status;