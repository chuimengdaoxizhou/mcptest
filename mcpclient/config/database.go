package config

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

func InitDB() *gorm.DB {
	// 获取配置文件
	con := GetConfig()
	// 获取连接数据库的dsn
	dsn := con.GetDatabasedsn()
	// 连接mysql数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	fmt.Println("Database connection established.")
	return db
}

func ConnectMongoDB() (*mongo.Collection, error) {
	// 设置连接超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接 MongoDB（本地默认端口 27017）
	con := GetConfig()
	host, port, Databasename, Collectionname := con.Getnosqldatabase()
	url := "mongodb://" + host + ":" + port
	clientOptions := options.Client().ApplyURI(url)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// 检查是否连接成功
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}
	// 选择数据库和集合
	database := client.Database(Databasename)         // 数据库名
	collection := database.Collection(Collectionname) // 集合名
	fmt.Println("✅ 成功连接到 MongoDB！")
	return collection, nil
}
