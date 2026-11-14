-- Миграция 002: флаги отправки напоминаний для исключения дубликатов
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS reminded_24h BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS reminded_2h BOOLEAN DEFAULT FALSE;

-- Удаляем возможные фиктивные слоты 00:00 от старой логики SetWorkday (если по ним нет заказов)
DELETE FROM schedule
WHERE time_slot = '00:00:00'
  AND is_occupied = FALSE;
