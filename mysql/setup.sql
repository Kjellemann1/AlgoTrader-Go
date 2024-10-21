
-- Main database used for logging positions and trades from the program

CREATE DATABASE algo;

USE algo;


CREATE TABLE positions (
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  PRIMARY KEY (symbol, asset_class),
  position_id VARCHAR(255),
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
  position_id VARCHAR(255),
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


-- Test database
-- Equivalent to the above, but used for running tests

CREATE DATABASE algo_test;

USE algo_test;


CREATE TABLE positions (
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  PRIMARY KEY (symbol, asset_class),
  position_id VARCHAR(255),
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
  position_id VARCHAR(255),
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
