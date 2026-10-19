-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE expense_category AS ENUM (
    'еда',
    'транспорт',
    'жильё',
    'развлечения',
    'здоровье',
    'покупки',
    'другое'
);

CREATE TABLE expenses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     BIGINT NOT NULL,
    amount      BIGINT NOT NULL, -- сумма в копейках
    category    expense_category NOT NULL DEFAULT 'другое',
    spent_at    DATE NOT NULL,
    comment     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT expenses_amount_positive CHECK (amount > 0)
);



-- Бот: выборки по владельцу и периоду («за неделю», «в этом месяце»)
CREATE INDEX idx_expenses_user_spent_at ON expenses (user_id, spent_at DESC);

-- Веб: фильтр по дате / категории / члену семьи
CREATE INDEX idx_expenses_spent_at ON expenses (spent_at DESC);
CREATE INDEX idx_expenses_category ON expenses (category);
CREATE INDEX idx_expenses_user_id ON expenses (user_id);

-- +goose Down
DROP TABLE expenses;
DROP TYPE expense_category;
DROP EXTENSION pgcrypto;