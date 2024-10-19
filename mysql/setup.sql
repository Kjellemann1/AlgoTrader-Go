
CREATE DATABASE algo;
CREATE DATABASE algo_test;

USE algo;

CREATE TABLE positions (
  db_key VARCHAR(255) PRIMARY KEY,
  order_id VARCHAR(255),
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  side VARCHAR(50),
  strat_name VARCHAR(255),
  order_type VARCHAR(50),
  qty DECIMAL(15, 9),
  price_time DATETIME(3),
  trigger_time DATETIME(3),
  trigger_price DECIMAL(15, 9),
  fill_time DATETIME(3),
  filled_avg_price DECIMAL(15, 9),
  order_sent_time DATETIME(3),
  trailing_stop DECIMAL(15, 9),
  bad_for_analysis BOOL
);

CREATE TABLE trades (
  id INT AUTO_INCREMENT PRIMARY KEY,
  action VARCHAR(50),
  order_id VARCHAR(255),
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  side VARCHAR(50),
  strat_name VARCHAR(255),
  order_type VARCHAR(50),
  qty DECIMAL(15, 9),
  price_time DATETIME(3),
  trigger_time DATETIME(3),
  trigger_price DECIMAL(15, 9),
  fill_time DATETIME(3),
  filled_avg_price DECIMAL(15, 9),
  order_sent_time DATETIME(3),
  bad_for_analysis BOOL
);

USE algo_test;

CREATE TABLE positions (
  db_key VARCHAR(255) PRIMARY KEY,
  order_id VARCHAR(255),
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  side VARCHAR(50),
  strat_name VARCHAR(255),
  order_type VARCHAR(50),
  qty DECIMAL(15, 9),
  price_time DATETIME(3),
  trigger_time DATETIME(3),
  trigger_price DECIMAL(15, 9),
  fill_time DATETIME(3),
  filled_avg_price DECIMAL(15, 9),
  order_sent_time DATETIME(3),
  trailing_stop DECIMAL(15, 9),
  bad_for_analysis BOOL
);

CREATE TABLE trades (
  id INT AUTO_INCREMENT PRIMARY KEY,
  action VARCHAR(50),
  order_id VARCHAR(255),
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  side VARCHAR(50),
  strat_name VARCHAR(255),
  order_type VARCHAR(50),
  qty DECIMAL(15, 9),
  price_time DATETIME(3),
  trigger_time DATETIME(3),
  trigger_price DECIMAL(15, 9),
  fill_time DATETIME(3),
  filled_avg_price DECIMAL(15, 9),
  order_sent_time DATETIME(3),
  bad_for_analysis BOOL
);
