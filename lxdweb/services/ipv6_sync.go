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

// RefreshNodeIPv6Bindings 刷新节点的IPv6绑定信息（从缓存获取）
func RefreshNodeIPv6Bindings(nodeID uint) error {
	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("节点不存在: %v", err)
	}

	now := time.Now()
	task := models.IPv6SyncTask{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Status:    "running",
		StartTime: &now,
	}
	database.DB.Create(&task)

	log.Printf("[IPv6-REFRESH] 开始刷新节点 %s (ID: %d) IPv6绑定", node.Name, node.ID)

	result := callNodeAPIForIPv6(node, "GET", "/api/cache/ipv6", nil)
	if result["code"] != float64(200) {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("获取IPv6缓存失败: %v", result["msg"])
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[IPv6-REFRESH] 节点 %s 获取缓存失败，清理旧IPv6绑定缓存", node.Name)
		database.DB.Unscoped().Where("node_id = ?", node.ID).Delete(&models.IPv6BindingCache{})

		return fmt.Errorf("获取IPv6缓存失败")
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		task.Status = "failed"
		task.ErrorMessage = "IPv6绑定列表格式错误"
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[IPv6-REFRESH] 节点 %s 返回数据格式错误，清理旧IPv6绑定缓存", node.Name)
		database.DB.Unscoped().Where("node_id = ?", node.ID).Delete(&models.IPv6BindingCache{})

		return fmt.Errorf("IPv6绑定列表格式错误")
	}

	task.TotalCount = len(data)
	database.DB.Save(&task)

	successCount := 0
	failedCount := 0

	for _, item := range data {
		bindingData, ok := item.(map[string]interface{})
		if !ok {
			failedCount++
			continue
		}

		if err := updateIPv6Cache(node, bindingData); err != nil {
			log.Printf("[IPv6-REFRESH] 更新IPv6绑定缓存失败: %v", err)
			failedCount++
		} else {
			successCount++
		}
	}

	var cachedBindings []models.IPv6BindingCache
	database.DB.Where("node_id = ?", node.ID).Find(&cachedBindings)

	existingBindings := make(map[string]bool)
	for _, item := range data {
		if binding, ok := item.(map[string]interface{}); ok {
			if hostname, ok := binding["hostname"].(string); ok {
				if ipv6, ok := binding["ipv6_address"].(string); ok {
					key := hostname + ":" + ipv6
					existingBindings[key] = true
				}
			}
		}
	}

	for _, cached := range cachedBindings {
		key := cached.Hostname + ":" + cached.IPv6Address
		if !existingBindings[key] {
			database.DB.Unscoped().Delete(&cached)
			log.Printf("[IPv6-REFRESH] 删除不存在的IPv6绑定缓存: %s -> %s", cached.Hostname, cached.IPv6Address)
		}
	}

	task.Status = "completed"
	task.SuccessCount = successCount
	task.FailedCount = failedCount
	endTime := time.Now()
	task.EndTime = &endTime
	database.DB.Save(&task)

	log.Printf("[IPv6-REFRESH] 节点 %s IPv6绑定刷新完成: 成功 %d, 失败 %d, 总计 %d",
		node.Name, successCount, failedCount, task.TotalCount)

	return nil
}

// SyncNodeIPv6Bindings 实时同步节点的IPv6绑定信息（逐个容器调用 /api/ipv6/list 接口）
func SyncNodeIPv6Bindings(nodeID uint) error {
	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("节点不存在: %v", err)
	}

	now := time.Now()
	task := models.IPv6SyncTask{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Status:    "running",
		StartTime: &now,
	}
	database.DB.Create(&task)

	log.Printf("[IPv6-SYNC] 开始实时同步节点 %s (ID: %d) IPv6绑定", node.Name, node.ID)

	// 先获取容器列表
	containerResult := callNodeAPIForIPv6(node, "GET", "/api/cache/containers", nil)
	if containerResult["code"] != float64(200) {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("获取容器列表失败: %v", containerResult["msg"])
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[IPv6-SYNC] 节点 %s 获取容器列表失败", node.Name)
		return fmt.Errorf("获取容器列表失败")
	}

	containers, ok := containerResult["data"].([]interface{})
	if !ok {
		task.Status = "failed"
		task.ErrorMessage = "容器列表格式错误"
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[IPv6-SYNC] 节点 %s 容器列表格式错误", node.Name)
		return fmt.Errorf("容器列表格式错误")
	}

	successCount := 0
	failedCount := 0
	totalBindings := 0
	existingBindings := make(map[string]bool)

	log.Printf("[IPv6-SYNC] 节点 %s 开始实时同步，共 %d 个容器", node.Name, len(containers))

	// 逐个容器调用 /api/ipv6/list 接口
	for _, item := range containers {
		container, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		hostname, _ := container["hostname"].(string)
		if hostname == "" {
			continue
		}

		// 调用 /api/ipv6/list?hostname={name} 触发实时更新
		ipv6Result := callNodeAPIForIPv6(node, "GET", fmt.Sprintf("/api/ipv6/list?hostname=%s", hostname), nil)

		if ipv6Result["code"] != float64(200) {
			log.Printf("[IPv6-SYNC] 容器 %s IPv6绑定同步失败: %v", hostname, ipv6Result["msg"])
			failedCount++
			continue
		}

		// 获取返回的IPv6绑定并更新缓存
		if ipv6Data, ok := ipv6Result["data"].([]interface{}); ok {
			for _, bindingItem := range ipv6Data {
				if bindingData, ok := bindingItem.(map[string]interface{}); ok {
					totalBindings++
					if err := updateIPv6Cache(node, bindingData); err != nil {
						log.Printf("[IPv6-SYNC] IPv6绑定缓存更新失败: %v", err)
						failedCount++
					} else {
						successCount++
						ipv6Address, _ := bindingData["public_ipv6"].(string)
						if ipv6Address == "" {
							ipv6Address, _ = bindingData["ipv6_address"].(string)
						}
						if ipv6Address != "" {
							key := hostname + ":" + ipv6Address
							existingBindings[key] = true
						}
					}
				}
			}
		}

		// 避免请求过快
		time.Sleep(100 * time.Millisecond)
	}

	// 清理不存在的绑定
	var cachedBindings []models.IPv6BindingCache
	database.DB.Where("node_id = ?", node.ID).Find(&cachedBindings)

	for _, cached := range cachedBindings {
		key := cached.Hostname + ":" + cached.IPv6Address
		if !existingBindings[key] {
			database.DB.Unscoped().Delete(&cached)
			log.Printf("[IPv6-SYNC] 删除不存在的IPv6绑定缓存: %s -> %s", cached.Hostname, cached.IPv6Address)
		}
	}

	task.Status = "completed"
	task.TotalCount = totalBindings
	task.SuccessCount = successCount
	task.FailedCount = failedCount
	endTime := time.Now()
	task.EndTime = &endTime
	database.DB.Save(&task)

	log.Printf("[IPv6-SYNC] 节点 %s IPv6绑定实时同步完成: 成功 %d, 失败 %d, 总计 %d",
		node.Name, successCount, failedCount, task.TotalCount)

	return nil
}

func updateIPv6Cache(node models.Node, data map[string]interface{}) error {
	hostname, _ := data["container_name"].(string)
	ipv6Address, _ := data["public_ipv6"].(string)
	
	if hostname == "" || ipv6Address == "" {
		return fmt.Errorf("缺少必要字段: hostname=%s, ipv6=%s", hostname, ipv6Address)
	}

	updates := map[string]interface{}{
		"node_name":  node.Name,
		"last_sync":  time.Now(),
		"sync_error": "",
	}

	if iface, ok := data["interface"].(string); ok {
		updates["interface"] = iface
	}
	if status, ok := data["status"].(string); ok {
		updates["status"] = status
	} else {
		updates["status"] = "active"
	}

	cache := models.IPv6BindingCache{
		NodeID:      node.ID,
		NodeName:    node.Name,
		Hostname:    hostname,
		IPv6Address: ipv6Address,
		LastSync:    time.Now(),
		SyncError:   "",
	}

	if iface, ok := updates["interface"].(string); ok {
		cache.Interface = iface
	}
	if status, ok := updates["status"].(string); ok {
		cache.Status = status
	}

	result := database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "node_id"},
			{Name: "hostname"},
			{Name: "ipv6_address"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"node_name", "interface", "status",
			"last_sync", "sync_error",
		}),
	}).Create(&cache)

	return result.Error
}

func callNodeAPIForIPv6(node models.Node, method, path string, data interface{}) map[string]interface{} {
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

