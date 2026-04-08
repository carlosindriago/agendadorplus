-- Reverse migration: drop all tables in dependency order
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS time_slots;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
