package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"lxdweb/database"
	"lxdweb/models"
	"net/http"
	"time"

	"gorm.io/gorm/clause"
)

// RefreshNodeProxyConfigs 刷新节点的反向代理配置（从缓存获取）
func RefreshNodeProxyConfigs(nodeID uint) error {
	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("节点不存在: %v", err)
	}

	now := time.Now()
	task := models.ProxySyncTask{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Status:    "running",
		StartTime: &now,
	}
	database.DB.Create(&task)

	log.Printf("[PROXY-REFRESH] 开始刷新节点 %s (ID: %d) Proxy配置", node.Name, node.ID)

	result := callNodeAPIForProxy(node, "GET", "/api/cache/proxy", nil)
	if result["code"] != float64(200) {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("获取Proxy缓存失败: %v", result["msg"])
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[PROXY-REFRESH] 节点 %s 获取缓存失败，清理旧Proxy配置缓存", node.Name)
		database.DB.Unscoped().Where("node_id = ?", node.ID).Delete(&models.ProxyConfigCache{})

		return fmt.Errorf("获取Proxy缓存失败")
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		task.Status = "failed"
		task.ErrorMessage = "Proxy配置列表格式错误"
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[PROXY-REFRESH] 节点 %s 返回数据格式错误，清理旧Proxy配置缓存", node.Name)
		database.DB.Unscoped().Where("node_id = ?", node.ID).Delete(&models.ProxyConfigCache{})

		return fmt.Errorf("Proxy配置列表格式错误")
	}

	task.TotalCount = len(data)
	database.DB.Save(&task)

	successCount := 0
	failedCount := 0

	for _, item := range data {
		configData, ok := item.(map[string]interface{})
		if !ok {
			failedCount++
			continue
		}

		if err := updateProxyCache(node, configData); err != nil {
			log.Printf("[PROXY-REFRESH] 更新Proxy配置缓存失败: %v", err)
			failedCount++
		} else {
			successCount++
		}
	}

	var cachedConfigs []models.ProxyConfigCache
	database.DB.Where("node_id = ?", node.ID).Find(&cachedConfigs)

	existingConfigs := make(map[string]bool)
	for _, item := range data {
		if config, ok := item.(map[string]interface{}); ok {
			if hostname, ok := config["hostname"].(string); ok {
				if domain, ok := config["domain"].(string); ok {
					key := hostname + ":" + domain
					existingConfigs[key] = true
				}
			}
		}
	}

	for _, cached := range cachedConfigs {
		key := cached.Hostname + ":" + cached.Domain
		if !existingConfigs[key] {
			database.DB.Unscoped().Delete(&cached)
			log.Printf("[PROXY-REFRESH] 删除不存在的Proxy配置缓存: %s -> %s", cached.Hostname, cached.Domain)
		}
	}

	task.Status = "completed"
	task.SuccessCount = successCount
	task.FailedCount = failedCount
	endTime := time.Now()
	task.EndTime = &endTime
	database.DB.Save(&task)

	log.Printf("[PROXY-REFRESH] 节点 %s Proxy配置刷新完成: 成功 %d, 失败 %d, 总计 %d",
		node.Name, successCount, failedCount, task.TotalCount)

	return nil
}

// SyncNodeProxyConfigs 实时同步节点的反向代理配置（逐个容器调用 /api/proxy/list 接口）
func SyncNodeProxyConfigs(nodeID uint) error {
	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("节点不存在: %v", err)
	}

	now := time.Now()
	task := models.ProxySyncTask{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Status:    "running",
		StartTime: &now,
	}
	database.DB.Create(&task)

	log.Printf("[PROXY-SYNC] 开始实时同步节点 %s (ID: %d) Proxy配置", node.Name, node.ID)

	// 先获取容器列表
	containerResult := callNodeAPIForProxy(node, "GET", "/api/cache/containers", nil)
	if containerResult["code"] != float64(200) {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("获取容器列表失败: %v", containerResult["msg"])
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[PROXY-SYNC] 节点 %s 获取容器列表失败", node.Name)
		return fmt.Errorf("获取容器列表失败")
	}

	containers, ok := containerResult["data"].([]interface{})
	if !ok {
		task.Status = "failed"
		task.ErrorMessage = "容器列表格式错误"
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[PROXY-SYNC] 节点 %s 容器列表格式错误", node.Name)
		return fmt.Errorf("容器列表格式错误")
	}

	successCount := 0
	failedCount := 0
	totalConfigs := 0
	existingConfigs := make(map[string]bool)

	log.Printf("[PROXY-SYNC] 节点 %s 开始实时同步，共 %d 个容器", node.Name, len(containers))

	// 逐个容器调用 /api/proxy/list 接口
	for _, item := range containers {
		container, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		hostname, _ := container["hostname"].(string)
		if hostname == "" {
			continue
		}

		// 调用 /api/proxy/list?hostname={name} 触发实时更新
		proxyResult := callNodeAPIForProxy(node, "GET", fmt.Sprintf("/api/proxy/list?hostname=%s", hostname), nil)

		if proxyResult["code"] != float64(200) {
			log.Printf("[PROXY-SYNC] 容器 %s Proxy配置同步失败: %v", hostname, proxyResult["msg"])
			failedCount++
			continue
		}

		// 获取返回的Proxy配置并更新缓存
		if proxyData, ok := proxyResult["data"].([]interface{}); ok {
			for _, configItem := range proxyData {
				if configData, ok := configItem.(map[string]interface{}); ok {
					totalConfigs++
					if err := updateProxyCache(node, configData); err != nil {
						log.Printf("[PROXY-SYNC] Proxy配置缓存更新失败: %v", err)
						failedCount++
					} else {
						successCount++
						domain, _ := configData["domain"].(string)
						if domain != "" {
							key := hostname + ":" + domain
							existingConfigs[key] = true
						}
					}
				}
			}
		}

		// 避免请求过快
		time.Sleep(100 * time.Millisecond)
	}

	// 清理不存在的配置
	var cachedConfigs []models.ProxyConfigCache
	database.DB.Where("node_id = ?", node.ID).Find(&cachedConfigs)

	for _, cached := range cachedConfigs {
		key := cached.Hostname + ":" + cached.Domain
		if !existingConfigs[key] {
			database.DB.Unscoped().Delete(&cached)
			log.Printf("[PROXY-SYNC] 删除不存在的Proxy配置缓存: %s -> %s", cached.Hostname, cached.Domain)
		}
	}

	task.Status = "completed"
	task.TotalCount = totalConfigs
	task.SuccessCount = successCount
	task.FailedCount = failedCount
	endTime := time.Now()
	task.EndTime = &endTime
	database.DB.Save(&task)

	log.Printf("[PROXY-SYNC] 节点 %s Proxy配置实时同步完成: 成功 %d, 失败 %d, 总计 %d",
		node.Name, successCount, failedCount, task.TotalCount)

	return nil
}

func updateProxyCache(node models.Node, data map[string]interface{}) error {
	hostname, _ := data["container_name"].(string)
	domain, _ := data["domain"].(string)
	
	if hostname == "" || domain == "" {
		return fmt.Errorf("缺少必要字段: hostname=%s, domain=%s", hostname, domain)
	}

	updates := map[string]interface{}{
		"node_name":  node.Name,
		"last_sync":  time.Now(),
		"sync_error": "",
	}

	var backendPort int
	if port, ok := data["container_port"].(float64); ok {
		backendPort = int(port)
	}
	updates["backend_port"] = backendPort

	var sslEnabled bool
	if ssl, ok := data["ssl_enabled"].(bool); ok {
		sslEnabled = ssl
	} else if ssl, ok := data["ssl"].(bool); ok {
		sslEnabled = ssl
	}
	updates["ssl_enabled"] = sslEnabled

	var sslType string
	if st, ok := data["ssl_type"].(string); ok {
		sslType = st
	} else {
		sslType = "none"
	}
	updates["ssl_type"] = sslType

	if status, ok := data["status"].(string); ok {
		updates["status"] = status
	} else {
		updates["status"] = "active"
	}

	cache := models.ProxyConfigCache{
		NodeID:      node.ID,
		NodeName:    node.Name,
		Hostname:    hostname,
		Domain:      domain,
		BackendPort: backendPort,
		SSLEnabled:  sslEnabled,
		SSLType:     sslType,
		LastSync:    time.Now(),
		SyncError:   "",
	}

	if status, ok := updates["status"].(string); ok {
		cache.Status = status
	}

	result := database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "node_id"},
			{Name: "hostname"},
			{Name: "domain"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"node_name", "backend_port", "ssl_enabled", "status",
			"last_sync", "sync_error",
		}),
	}).Create(&cache)

	return result.Error
}

func callNodeAPIForProxy(node models.Node, method, path string, data interface{}) map[string]interface{} {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var body io.Reader
	if data != nil {
		jsonData, _ := json.Marshal(data)
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, node.Address+path, body)
	if err != nil {
		return map[string]interface{}{
			"code": 500,
			"msg":  "请求创建失败: " + err.Error(),
		}
	}

	if node.APIKey != "" {
		req.Header.Set("apikey", node.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"code": 500,
			"msg":  "请求失败: " + err.Error(),
		}
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return map[string]interface{}{
			"code": 500,
			"msg":  "响应解析失败: " + err.Error(),
		}
	}

	return result
}

