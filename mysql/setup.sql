
create database algo;
use algo;

create table positions (
	symbol varchar(50),
	strat_name varchar(255),
	primary key (symbol, strat_name),
	asset_class varchar(255),
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
	n_close_orders int
)

create table trades (
  id int primary key auto_increment,
  action varchar(50),
	position_id varchar(255),
	symbol varchar(50),
	asset_class varchar(255),
	side varchar(50),
	strat_name varchar(255),
	order_type varchar(50),
	qty decimal(19,9),
	price_time datetime(3),
	trigger_time datetime(3),
	trigger_price decimal(15,9),
	fill_time datetime(3),
	filled_avg_price decimal(15,9),
	bad_for_analysis tinyint(1),
	received_time datetime(3),
	n_close_orders int
)

create database algo_test;
use algo_test;

create table positions (
	symbol varchar(50),
	strat_name varchar(255),
	primary key (symbol, strat_name),
	asset_class varchar(255),
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
	n_close_orders int
)

create table trades (
  id int primary key auto_increment,
  action varchar(50),
	position_id varchar(255),
	symbol varchar(50),
	asset_class varchar(255),
	side varchar(50),
	strat_name varchar(255),
	order_type varchar(50),
	qty decimal(19,9),
	price_time datetime(3),
	trigger_time datetime(3),
	trigger_price decimal(15,9),
	fill_time datetime(3),
	filled_avg_price decimal(15,9),
	bad_for_analysis tinyint(1),
	received_time datetime(3),
	n_close_orders int
)