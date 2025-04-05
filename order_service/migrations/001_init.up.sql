-- Используем тип ENUM для статуса для большей строгости
CREATE TYPE order_status AS ENUM ('pending', 'completed', 'cancelled');

CREATE TABLE orders (
                        id SERIAL PRIMARY KEY,
    -- В реальном приложении здесь может быть внешний ключ к таблице users
                        user_id INT NOT NULL,
                        status order_status NOT NULL DEFAULT 'pending',
                        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для быстрого поиска заказов по пользователю
CREATE INDEX idx_orders_user_id ON orders(user_id);

CREATE TABLE order_items (
                             id SERIAL PRIMARY KEY,
                             order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE, -- При удалении заказа удаляются и его позиции
                             product_id INT NOT NULL, -- ID продукта из другого сервиса, без внешнего ключа БД
                             quantity INT NOT NULL CHECK (quantity > 0), -- Количество должно быть положительным
                             price DECIMAL(10, 2) NOT NULL CHECK (price >= 0) -- Цена на момент заказа, не может быть отрицательной
    -- Можно добавить UNIQUE (order_id, product_id), если один продукт может быть в заказе только один раз
);

-- Индекс для быстрого поиска позиций по заказу
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- Триггер для автоматического обновления поля updated_at в таблице orders
-- (Этот триггер можно использовать и для других таблиц, если он еще не создан)
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_orders_timestamp
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();