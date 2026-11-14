-- Миграция 001:.initial схема базы данных для zapisBot
-- Таблица администраторов (Telegram ID из конфига)
CREATE TABLE IF NOT EXISTS admins (
    telegram_id BIGINT PRIMARY KEY,
    full_name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'admin'
);

-- Таблица клиентов
CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20) UNIQUE NOT NULL,
    telegram_id BIGINT UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Таблица услуг (категория, тип изделия, параметры, цвет, покрытие)
CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    category VARCHAR(50) NOT NULL,
    item_type VARCHAR(100) NOT NULL,
    parameters JSONB DEFAULT '{}'::jsonb,
    color VARCHAR(20),
    coating VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Таблица прайс-листа
CREATE TABLE IF NOT EXISTS price_list (
    id SERIAL PRIMARY KEY,
    item_type VARCHAR(100) NOT NULL,
    parameters VARCHAR(255),
    coating VARCHAR(20),
    price DECIMAL(10, 2) NOT NULL,
    UNIQUE(item_type, parameters, coating)
);

-- Таблица расписания (даты и временные слоты)
CREATE TABLE IF NOT EXISTS schedule (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    time_slot TIME NOT NULL,
    is_occupied BOOLEAN DEFAULT FALSE,
    is_workday BOOLEAN DEFAULT TRUE,
    UNIQUE(date, time_slot)
);

-- Таблица заказов
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    service_id INTEGER REFERENCES services(id) ON DELETE SET NULL,
    date DATE NOT NULL,
    time_slot TIME NOT NULL,
    status VARCHAR(20) DEFAULT 'confirmed',
    price DECIMAL(10, 2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для ускорения выборок
CREATE INDEX IF NOT EXISTS idx_orders_client_id ON orders(client_id);
CREATE INDEX IF NOT EXISTS idx_orders_date_status ON orders(date, status);
CREATE INDEX IF NOT EXISTS idx_clients_phone ON clients(phone);
CREATE INDEX IF NOT EXISTS idx_clients_telegram_id ON clients(telegram_id);
CREATE INDEX IF NOT EXISTS idx_schedule_date ON schedule(date);
CREATE INDEX IF NOT EXISTS idx_price_list_item ON price_list(item_type, coating);