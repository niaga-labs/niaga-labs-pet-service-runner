-- 003_create_pet_shops.sql
-- PetShopModel was only ever created by the development AutoMigrate branch in
-- cmd/server/main.go, so "pet_shops" did not exist in any environment that runs
-- the SQL migrations instead. See KPD-59.
--
-- owner_id is nullable on purpose: the model uses *uuid.UUID, and a directory
-- listing can exist before a merchant claims it. It references a user owned by
-- service-identity, in a different database, so it carries no foreign key.
--
-- The category CHECK mirrors petshop.Category in internal/domain.

CREATE TABLE IF NOT EXISTS pet_shops (
    id            UUID             PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id      UUID,
    name          VARCHAR(255)     NOT NULL,
    address       TEXT             NOT NULL,
    latitude      DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude     DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    phone         VARCHAR(20),
    email         VARCHAR(255),
    category      VARCHAR(20)      NOT NULL CHECK (category IN ('grooming', 'vet', 'boarding', 'pet_store')),
    services      JSONB            NOT NULL DEFAULT '[]',
    rating        DECIMAL(3,2)     NOT NULL DEFAULT 0.0,
    image_url     TEXT,
    opening_hours VARCHAR(100),
    description   TEXT,
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pet_shops_owner_id ON pet_shops(owner_id);
CREATE INDEX IF NOT EXISTS idx_pet_shops_category ON pet_shops(category);
