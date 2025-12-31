package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"lxdweb/database"
	"lxdweb/models"
	"gorm.io/gorm/clause"
)

var (
	natSyncMutex    sync.Mutex
	natSyncRunning  = make(map[uint]bool) 
)

func StartNATSyncService() {
	log.Println("[NAT-SYNC] NAT规则同步服务就绪")
}

// SyncAllNodesNATAsync 同步所有活动节点的NAT规则
func SyncAllNodesNATAsync() {
	var nodes []models.Node
	database.DB.Where("status = ?", "active").Find(&nodes)
	
	log.Printf("[NAT-SYNC] 开始实时同步 %d 个活动节点的NAT规则", len(nodes))
	
	for i, node := range nodes {
		log.Printf("[NAT-SYNC] 处理节点 %d/%d: %s", i+1, len(nodes), node.Name)
		SyncNodeNATRules(node.ID, false)
		
		if i < len(nodes)-1 {
			interval := time.Duration(node.BatchInterval) * time.Second
			log.Printf("[NAT-SYNC] 等待 %v 后处理下一个节点", interval)
			time.Sleep(interval)
		}
	}
	
	log.Printf("[NAT-SYNC] 所有节点NAT规则实时同步完成")
}

func RefreshNodeNATRules(nodeID uint, manual bool) error {
	natSyncMutex.Lock()
	if natSyncRunning[nodeID] {
		natSyncMutex.Unlock()
		return fmt.Errorf("节点 %d NAT规则正在同步中", nodeID)
	}
	natSyncRunning[nodeID] = true
	natSyncMutex.Unlock()
	
	defer func() {
		natSyncMutex.Lock()
		natSyncRunning[nodeID] = false
		natSyncMutex.Unlock()
	}()
	
	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("节点不存在: %v", err)
	}

	now := time.Now()
	task := models.NATSyncTask{
		NodeID:     node.ID,
		NodeName:   node.Name,
		Status:     "running",
		StartTime:  &now,
	}
	database.DB.Create(&task)
	
	log.Printf("[NAT-REFRESH] 开始刷新节点 %s (ID: %d)%s NAT规则", node.Name, node.ID, map[bool]string{true: " [手动]", false: ""}[manual])

	result := callNodeAPIForNAT(node, "GET", "/api/cache/nat", nil)
	if result["code"] != float64(200) {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("获取NAT缓存失败: %v", result["msg"])
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[NAT-REFRESH] 节点 %s 获取缓存失败，清理旧NAT规则缓存", node.Name)
		database.DB.Unscoped().Where("node_id = ?", node.ID).Delete(&models.NATRuleCache{})
		
		return fmt.Errorf("获取NAT缓存失败")
	}
	
	data, ok := result["data"].([]interface{})
	if !ok {
		task.Status = "failed"
		task.ErrorMessage = "NAT规则列表格式错误"
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)

		log.Printf("[NAT-REFRESH] 节点 %s 返回数据格式错误，清理旧NAT规则缓存", node.Name)
		database.DB.Unscoped().Where("node_id = ?", node.ID).Delete(&models.NATRuleCache{})
		
		return fmt.Errorf("NAT规则列表格式错误")
	}
	
	task.TotalCount = len(data)
	database.DB.Save(&task)

	successCount := 0
	failedCount := 0
	
	existingRules := make(map[string]bool)
	
	for _, item := range data {
		ruleData, ok := item.(map[string]interface{})
		if !ok {
			failedCount++
			continue
		}
		
		if err := updateNATCache(node, ruleData); err != nil {
			log.Printf("[NAT-REFRESH] 更新NAT规则缓存失败: %v", err)
			failedCount++
		} else {
			successCount++
			hostname, _ := ruleData["hostname"].(string)
			external, _ := ruleData["external"].(float64)
			protocol, _ := ruleData["protocol"].(string)
			key := fmt.Sprintf("%d-%s-%d-%s", node.ID, hostname, int(external), protocol)
			existingRules[key] = true
		}
	}

	var cachedRules []models.NATRuleCache
	database.DB.Where("node_id = ?", node.ID).Find(&cachedRules)
	
	for _, cached := range cachedRules {
		key := fmt.Sprintf("%d-%s-%d-%s", node.ID, cached.ContainerHostname, cached.ExternalPort, cached.Protocol)
		if !existingRules[key] {
			database.DB.Unscoped().Delete(&cached)
			log.Printf("[NAT-REFRESH] 删除不存在的NAT规则缓存: %s:%d/%s", cached.ContainerHostname, cached.ExternalPort, cached.Protocol)
		}
	}

	task.Status = "completed"
	task.SuccessCount = successCount
	task.FailedCount = failedCount
	endTime := time.Now()
	task.EndTime = &endTime
	database.DB.Save(&task)
	
	log.Printf("[NAT-REFRESH] 节点 %s NAT规则刷新完成: 成功 %d, 失败 %d, 总计 %d", 
		node.Name, successCount, failedCount, task.TotalCount)
	
	return nil
}

// SyncNodeNATRules 实时同步单个节点的NAT规则
func SyncNodeNATRules(nodeID uint, manual bool) error {
	natSyncMutex.Lock()
	if natSyncRunning[nodeID] {
		natSyncMutex.Unlock()
		return fmt.Errorf("节点 %d NAT规则正在同步中", nodeID)
	}
	natSyncRunning[nodeID] = true
	natSyncMutex.Unlock()
	
	defer func() {
		natSyncMutex.Lock()
		natSyncRunning[nodeID] = false
		natSyncMutex.Unlock()
	}()
	
	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		return fmt.Errorf("节点不存在: %v", err)
	}

	now := time.Now()
	task := models.NATSyncTask{
		NodeID:     node.ID,
		NodeName:   node.Name,
		Status:     "running",
		StartTime:  &now,
	}
	database.DB.Create(&task)
	
	log.Printf("[NAT-SYNC] 开始实时同步节点 %s (ID: %d)%s NAT规则", node.Name, node.ID, map[bool]string{true: " [手动]", false: ""}[manual])

	containerResult := callNodeAPIForNAT(node, "GET", "/api/cache/containers", nil)
	if containerResult["code"] != float64(200) {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("获取容器列表失败: %v", containerResult["msg"])
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)
		
		log.Printf("[NAT-SYNC] 节点 %s 获取容器列表失败", node.Name)
		return fmt.Errorf("获取容器列表失败")
	}
	
	containers, ok := containerResult["data"].([]interface{})
	if !ok {
		task.Status = "failed"
		task.ErrorMessage = "容器列表格式错误"
		endTime := time.Now()
		task.EndTime = &endTime
		database.DB.Save(&task)
		
		log.Printf("[NAT-SYNC] 节点 %s 容器列表格式错误", node.Name)
		return fmt.Errorf("容器列表格式错误")
	}

	successCount := 0
	failedCount := 0
	totalRules := 0
	existingRules := make(map[string]bool)

	log.Printf("[NAT-SYNC] 节点 %s 开始实时同步，共 %d 个容器", node.Name, len(containers))

	for _, item := range containers {
		container, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		
		hostname, _ := container["hostname"].(string)
		if hostname == "" {
			continue
		}
		
		natResult := callNodeAPIForNAT(node, "GET", fmt.Sprintf("/api/natlist?hostname=%s", hostname), nil)
		
		if natResult["code"] != float64(200) {
			log.Printf("[NAT-SYNC] 容器 %s NAT规则同步失败: %v", hostname, natResult["msg"])
			failedCount++
			continue
		}
		
		if natData, ok := natResult["data"].([]interface{}); ok {
			for _, ruleItem := range natData {
				if ruleData, ok := ruleItem.(map[string]interface{}); ok {
					totalRules++
					if err := updateNATCache(node, ruleData); err != nil {
						log.Printf("[NAT-SYNC] NAT规则缓存更新失败: %v", err)
						failedCount++
					} else {
						successCount++
						external, _ := ruleData["external_port"].(float64)
						protocol, _ := ruleData["protocol"].(string)
						key := fmt.Sprintf("%d-%s-%d-%s", node.ID, hostname, int(external), protocol)
						existingRules[key] = true
					}
				}
			}
		}
		
		time.Sleep(100 * time.Millisecond)
	}

	var cachedRules []models.NATRuleCache
	database.DB.Where("node_id = ?", node.ID).Find(&cachedRules)
	
	for _, cached := range cachedRules {
		key := fmt.Sprintf("%d-%s-%d-%s", node.ID, cached.ContainerHostname, cached.ExternalPort, cached.Protocol)
		if !existingRules[key] {
			database.DB.Unscoped().Delete(&cached)
			log.Printf("[NAT-SYNC] 删除不存在的NAT规则缓存: %s:%d/%s", cached.ContainerHostname, cached.ExternalPort, cached.Protocol)
		}
	}

	task.Status = "completed"
	task.TotalCount = totalRules
	task.SuccessCount = successCount
	task.FailedCount = failedCount
	endTime := time.Now()
	task.EndTime = &endTime
	database.DB.Save(&task)
	
	log.Printf("[NAT-SYNC] 节点 %s NAT规则实时同步完成: 成功 %d, 失败 %d, 总计 %d", 
		node.Name, successCount, failedCount, task.TotalCount)
	
	return nil
}

func updateNATCache(node models.Node, data map[string]interface{}) error {
	hostname, _ := data["container_name"].(string)
	external, _ := data["external_port"].(float64)
	internal, _ := data["internal_port"].(float64)
	protocol, _ := data["protocol"].(string)
	
	if hostname == "" || protocol == "" {
		return fmt.Errorf("缺少必要字段")
	}

	updates := map[string]interface{}{
		"node_name":     node.Name,
		"internal_port": int(internal),
		"last_sync":     time.Now(),
		"sync_error":    "",
	}
	
	if desc, ok := data["description"].(string); ok {
		updates["description"] = desc
	}
	if status, ok := data["status"].(string); ok {
		updates["status"] = status
	} else {
		updates["status"] = "active"
	}

	cache := models.NATRuleCache{
		NodeID:            node.ID,
		NodeName:          node.Name,
		ContainerHostname: hostname,
		ExternalPort:      int(external),
		Protocol:          protocol,
		InternalPort:      int(internal),
		LastSync:          time.Now(),
		SyncError:         "",
	}

	if desc, ok := updates["description"].(string); ok {
		cache.Description = desc
	}
	if status, ok := updates["status"].(string); ok {
		cache.Status = status
	}

	result := database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "node_id"},
			{Name: "container_hostname"},
			{Name: "external_port"},
			{Name: "protocol"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"node_name", "internal_port", "description", "status",
			"last_sync", "sync_error",
		}),
	}).Create(&cache)
	
	return result.Error
}

func callNodeAPIForNAT(node models.Node, method, path string, data interface{}) map[string]interface{} {
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

func IsNATSyncing(nodeID uint) bool {
	natSyncMutex.Lock()
	defer natSyncMutex.Unlock()
	return natSyncRunning[nodeID]
}

