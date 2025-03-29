package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mcpclient/models"
)

// LoadMCPConfig 读取并解析 JSON 配置文件
func LoadMCPConfig(filePath string) ([]models.SSEServerConfig, error) {

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	// 定义一个 ServerConfig 类型的切片来解析 JSON 数组
	var servers []models.SSEServerConfig
	err = json.Unmarshal(data, &servers)
	if err != nil {
		log.Fatal(err)
	}

	//// 打印解析结果
	for _, server := range servers {
		fmt.Printf("Server Name: %s\n", server.Name)
		fmt.Printf("Server URL: %s\n", server.MCPServerURL)
	}
	fmt.Printf("")
	return servers, nil
}
