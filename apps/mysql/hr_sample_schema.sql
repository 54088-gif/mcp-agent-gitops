-- 创建数据库
CREATE DATABASE IF NOT EXISTS hr_db;
USE hr_db;

-- 创建部门表
CREATE TABLE IF NOT EXISTS departments (
    dept_id INT PRIMARY KEY,
    dept_name VARCHAR(50) NOT NULL,
    location VARCHAR(50)
);

-- 创建员工表
CREATE TABLE IF NOT EXISTS employees (
    emp_id INT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    role VARCHAR(50),
    salary DECIMAL(10, 2),
    dept_id INT,
    FOREIGN KEY (dept_id) REFERENCES departments(dept_id)
);

-- 插入样本数据
INSERT INTO departments VALUES (1, 'Engineering', 'Building A'), (2, 'Sales', 'Building B'), (3, 'HR', 'Building A');
INSERT INTO employees VALUES 
(101, 'Alice', 'Engineer', 90000, 1),
(102, 'Bob', 'Manager', 120000, 1),
(103, 'Charlie', 'Salesperson', 70000, 2),
(104, 'David', 'Recruiter', 60000, 3),
(105, 'Eve', 'Engineer', 95000, 1);
