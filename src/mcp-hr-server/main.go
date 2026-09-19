package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var db *sql.DB

// --- 工具 1: 获取数据库 Schema ---
// AI 需要先知道有哪些表、哪些字段，才能写出正确的 SQL
func listSchemaHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 1. 查所有表
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error listing tables: %v", err)), nil
	}
	defer rows.Close()

	var output string
	var tables []string

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}

	// 2. 查每个表的字段结构
	for _, tableName := range tables {
		output += fmt.Sprintf("Table: %s\n", tableName)
		cols, err := db.Query(fmt.Sprintf("DESCRIBE %s", tableName))
		if err != nil {
			continue
		}
		
		output += "Columns:\n"
		for cols.Next() {
			var field, type_ string
			var null, key, default_, extra sql.NullString
			cols.Scan(&field, &type_, &null, &key, &default_, &extra)
			output += fmt.Sprintf(" - %s (%s)\n", field, type_)
		}
		cols.Close()
		output += "\n"
	}

	return mcp.NewToolResultText(output), nil
}

// --- 工具 2: 执行 SQL 查询 ---
// AI 生成 SQL 后，通过这个工具在数据库真正执行
func queryHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 修正点：先将 Arguments 断言为 map 类型
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments format"), nil
	}

	// 然后再从 map 中获取 query
	query, ok := args["query"].(string)
	if !ok {
		return mcp.NewToolResultError("Query argument missing"), nil
	}

	log.Printf("Executing SQL: %s", query)

	rows, err := db.Query(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SQL Error: %v", err)), nil
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	result := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		rows.Scan(valuePtrs...)
		
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			rowMap[col] = v
		}
		result = append(result, rowMap)
	}

	jsonBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func main() {
	// 1. 初始化数据库连接
	// 从环境变量 DSN 读取连接串，格式: user:pass@tcp(host:port)/dbname
	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "root:native@tcp(127.0.0.1:3306)/hr_db"
	}
	
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}
	// 测试连接
	if err := db.Ping(); err != nil {
		log.Printf("Warning: Database unreachable at startup: %v", err)
	} else {
		log.Println("Successfully connected to MySQL")
	}

	// 2. 创建 MCP Server
	s := server.NewMCPServer(
		"HR-Database-MCP",
		"1.0.0",
	)

	// 3. 注册工具
	// Tool: read_schema
	s.AddTool(mcp.NewTool("read_schema",
		mcp.WithDescription("获取数据库的所有表名和字段结构，这是编写SQL前的必要步骤"),
	), listSchemaHandler)

	// Tool: execute_query
	s.AddTool(mcp.NewTool("execute_query",
		mcp.WithDescription("执行SQL语句 (SELECT, INSERT, UPDATE, etc.)"),
		mcp.WithString("query", mcp.Required(), mcp.Description("要执行的SQL语句")),
	), queryHandler)

	// 4. 启动 SSE (Server-Sent Events) 服务
	// MCP 支持 Stdio 和 SSE 两种模式，Web场景下必须用 SSE
	log.Println("Starting MCP Server on :8080/sse")
	sseServer := server.NewSSEServer(s)
	if err := sseServer.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
