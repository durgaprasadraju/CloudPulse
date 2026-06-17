-- Copyright 2026 Durga Prasad Raju Nadimpalli
-- Licensed under the Apache License, Version 2.0

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Service-owned schemas (database-per-service pattern within one Postgres cluster for dev)
CREATE SCHEMA IF NOT EXISTS collector;
CREATE SCHEMA IF NOT EXISTS analyzer;
CREATE SCHEMA IF NOT EXISTS correlator;
CREATE SCHEMA IF NOT EXISTS gateway;
