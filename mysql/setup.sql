
-- Main database used for logging positions and trades from the program

-- TODO: Update this cause there missing/not used columns

CREATE DATABASE algo;

USE algo;

create table positions (
	asset_class varchar(255),
	symbol varchar(50),
	strat_name varchar(255),
	primary key (asset_class, symbol, strat_name),
	position_id varchar(255),
	side varchar(50),
	order_type varchar(50),
	qty decimal(19,9),
	price_time datetime(3),
	trigger_time datetime(3),
	trigger_price decimal(15, 9),
	fill_time datetime(3),
	filled_avg_price decimal(15,9),
	trailing_stop decimal(15, 9),
	bad_for_analysis tinyint(1),
	received_time datetime(3),
	process_time datetime(3)
) 

CREATE TABLE trades (
  id INT AUTO_INCREMENT PRIMARY KEY,
  action VARCHAR(50),
  position_id VARCHAR(255),
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  side VARCHAR(50),
  strat_name VARCHAR(255),
  order_type VARCHAR(50),
  qty DECIMAL(19, 9),
  price_time DATETIME(3),
  trigger_time DATETIME(3),
  trigger_price DECIMAL(15, 9),
  fill_time DATETIME(3),
  filled_avg_price DECIMAL(15, 9),
  bad_for_analysis BOOL
);


-- Test database
-- Equivalent to the above, but used for running tests

CREATE DATABASE algo_test;

USE algo_test;

create table positions (
	asset_class varchar(255),
	symbol varchar(50),
	strat_name varchar(255),
	primary key (asset_class, symbol, strat_name),
	position_id varchar(255),
	side varchar(50),
	order_type varchar(50),
	qty decimal(19,9),
	price_time datetime(3),
	trigger_time datetime(3),
	trigger_price decimal(15, 9),
	fill_time datetime(3),
	filled_avg_price decimal(15,9),
	trailing_stop decimal(15, 9),
	bad_for_analysis tinyint(1),
	received_time datetime(3),
) 

CREATE TABLE trades (
  id INT AUTO_INCREMENT PRIMARY KEY,
  action VARCHAR(50),
  position_id VARCHAR(255),
  symbol VARCHAR(50),
  asset_class VARCHAR(255),
  side VARCHAR(50),
  strat_name VARCHAR(255),
  order_type VARCHAR(50),
  qty DECIMAL(19, 9),
  price_time DATETIME(3),
  trigger_time DATETIME(3),
  trigger_price DECIMAL(15, 9),
  fill_time DATETIME(3),
  filled_avg_price DECIMAL(15, 9),
  bad_for_analysis BOOL
);
