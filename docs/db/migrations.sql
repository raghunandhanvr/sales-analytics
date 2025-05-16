create table `products` (
  `id` varchar(50) not null,
  `name` varchar(100) not null,
  `category` varchar(50) default null,
  `unit_price` decimal(12,2) not null,
  primary key (`id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_0900_ai_ci;

create table `orders` (
  `id` varchar(50) not null,
  `customer_id` varchar(50) not null,
  `order_date` date not null,
  `total_amount` decimal(14,2) not null,
  `marketing_source` varchar(100) default null,
  `created_at` timestamp null default current_timestamp,
  `updated_at` timestamp null default current_timestamp on update current_timestamp,
  primary key (`id`),
  key `order_date_idx` (`order_date`),
  key `customer_idx` (`customer_id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_0900_ai_ci;

create table `order_items` (
  `order_id` varchar(50) not null,
  `product_id` varchar(50) not null,
  `quantity` int default null,
  `unit_price` decimal(12,2) default null,
  `discount` decimal(6,4) default null,
  `shipping_cost` decimal(12,2) default null,
  primary key (`order_id`,`product_id`),
  key `product_idx` (`product_id`),
  key `order_idx` (`order_id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_0900_ai_ci;

create table `customers` (
  `id` varchar(50) not null,
  `name` varchar(100) default null,
  `email` varchar(150) default null,
  `region` varchar(50) default null,
  `address` text,
  primary key (`id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_0900_ai_ci;

create table `ingestion_jobs` (
  `job_id` varchar(36) not null default (uuid()),
  `status` varchar(20) not null,
  `total_rows` int default '0',
  `processed_rows` int default '0',
  `error_message` text,
  `created_at` timestamp null default current_timestamp,
  `updated_at` timestamp null default current_timestamp on update current_timestamp,
  primary key (`job_id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_0900_ai_ci;
