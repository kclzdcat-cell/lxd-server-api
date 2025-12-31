package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"lxdweb/database"
	"lxdweb/models"
	"lxdweb/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// ProxyPage Proxy管理页面
// @Summary 反向代理管理页面
// @Description 显示反向代理配置管理页面
// @Tags 反向代理管理
// @Produce html
// @Success 200 {string} string "HTML页面"
// @Router /proxy [get]
func ProxyPage(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	c.HTML(http.StatusOK, "proxy.html", gin.H{
		"title":    "反向代理管理 - LXD管理后台",
		"username": username,
	})
}

// GetProxyConfigs 获取Proxy配置列表
// @Summary 获取反向代理配置列表
// @Description 查询所有反向代理配置信息，支持按节点过滤
// @Tags 反向代理管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回配置列表"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/proxy-configs [get]
func GetProxyConfigs(c *gin.Context) {
	var configs []models.ProxyConfigCache
	query := database.DB.Order("created_at desc")

	if nodeID := c.Query("node_id"); nodeID != "" {
		query = query.Where("node_id = ?", nodeID)
	}

	if err := query.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": configs,
	})
}

// SyncProxyConfigs 同步Proxy配置
// @Summary 同步反向代理配置
// @Description 从lxdapi同步指定节点的反向代理配置
// @Tags 反向代理管理
// @Produce json
// @Param node_id query string true "节点ID"
// @Success 200 {object} map[string]interface{} "同步任务已启动"
// @Failure 400 {object} map[string]interface{} "缺少参数"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/proxy-configs/sync [post]
func SyncProxyConfigs(c *gin.Context) {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "缺少node_id参数",
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "节点不存在",
		})
		return
	}

	go services.SyncNodeProxyConfigs(node.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "同步任务已启动",
	})
}

// GetProxySyncTasks 获取Proxy同步任务列表
// @Summary 获取反向代理同步任务列表
// @Description 查询最近50条反向代理同步任务记录
// @Tags 反向代理管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回任务列表"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/proxy-configs/tasks [get]
func GetProxySyncTasks(c *gin.Context) {
	var tasks []models.ProxySyncTask
	query := database.DB.Order("created_at desc").Limit(50)

	if nodeID := c.Query("node_id"); nodeID != "" {
		query = query.Where("node_id = ?", nodeID)
	}

	if err := query.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": tasks,
	})
}

// GetProxyConfigsFromCache 从缓存获取Proxy配置
// @Summary 从缓存获取反向代理配置
// @Description 直接从本地缓存数据库读取反向代理配置信息
// @Tags 反向代理管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Param hostname query string false "容器名称"
// @Success 200 {object} map[string]interface{} "成功返回缓存数据"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/proxy-configs/cache [get]
func GetProxyConfigsFromCache(c *gin.Context) {
	var configs []models.ProxyConfigCache
	query := database.DB.
		Joins("JOIN nodes ON nodes.id = proxy_config_caches.node_id").
		Where("nodes.status = ?", "active").
		Order("proxy_config_caches.last_sync desc")

	if nodeID := c.Query("node_id"); nodeID != "" {
		query = query.Where("proxy_config_caches.node_id = ?", nodeID)
	}
	if hostname := c.Query("hostname"); hostname != "" {
		query = query.Where("proxy_config_caches.hostname = ?", hostname)
	}

	if err := query.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": configs,
	})
}

// SyncAllProxy 同步所有节点的反向代理配置
// @Summary 同步所有节点的反向代理配置
// @Description 启动所有活跃节点的反向代理配置同步任务
// @Tags 反向代理管理
// @Produce json
// @Success 200 {object} map[string]interface{} "同步任务已启动"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/proxy-sync/all [post]
func SyncAllProxy(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}

	var nodes []models.Node
	if err := database.DB.Where("status = ?", "active").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询节点失败",
		})
		return
	}

	for _, node := range nodes {
		go services.SyncNodeProxyConfigs(node.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "反向代理配置同步任务已启动，请稍后刷新页面查看结果",
	})
}

// GetProxySyncStatus 获取反向代理同步状态
// @Summary 获取反向代理同步状态
// @Description 查询指定节点或所有节点的反向代理配置同步状态
// @Tags 反向代理管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回同步状态"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/proxy-sync/status [get]
func GetProxySyncStatus(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}

	nodeIDStr := c.Query("node_id")
	
	var status []map[string]interface{}
	
	if nodeIDStr != "" {
		nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
		if err == nil {
			var lastTask models.ProxySyncTask
			database.DB.Where("node_id = ?", nodeID).Order("created_at DESC").First(&lastTask)
			
			status = append(status, map[string]interface{}{
				"node_id":   uint(nodeID),
				"syncing":   lastTask.Status == "running",
				"last_task": lastTask,
			})
		}
	} else {
		var nodes []models.Node
		database.DB.Find(&nodes)
		
		for _, node := range nodes {
			var lastTask models.ProxySyncTask
			database.DB.Where("node_id = ?", node.ID).Order("created_at DESC").First(&lastTask)
			
			status = append(status, map[string]interface{}{
				"node_id":   node.ID,
				"node_name": node.Name,
				"syncing":   lastTask.Status == "running",
				"last_task": lastTask,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": status,
	})
}

// CreateProxyConfig 创建反向代理配置
// @Summary 创建反向代理配置
// @Description 为指定容器创建新的反向代理配置
// @Tags 反向代理管理
// @Accept json
// @Produce json
// @Param body body models.CreateProxyRequest true "反向代理配置参数"
// @Success 200 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/proxy-configs [post]
func CreateProxyConfig(c *gin.Context) {
	var req models.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, req.NodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "节点不存在",
		})
		return
	}

	proxyData := map[string]interface{}{
		"hostname":       req.ContainerHostname,
		"domain":         req.Domain,
		"container_port": req.ContainerPort,
		"description":    req.Description,
		"ssl_enabled":    req.SSLEnabled,
		"ssl_type":       req.SSLType,
	}

	if req.SSLEnabled && req.SSLType == "custom" {
		proxyData["ssl_cert"] = req.SSLCert
		proxyData["ssl_key"] = req.SSLKey
	}

	result := callNodeAPIForProxyMgmt(node, "POST", "/api/proxy/add", proxyData)
	if result["code"] != float64(200) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result["msg"],
		})
		return
	}

	time.Sleep(1 * time.Second)
	go services.SyncNodeProxyConfigs(node.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "反向代理创建成功",
		"data": result["data"],
	})
}

// DeleteProxyConfig 删除反向代理配置
// @Summary 删除反向代理配置
// @Description 删除指定的反向代理配置并清理数据库
// @Tags 反向代理管理
// @Produce json
// @Param id path string true "配置ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 404 {object} map[string]interface{} "配置或节点不存在"
// @Router /api/proxy-configs/{id} [delete]
func DeleteProxyConfig(c *gin.Context) {
	id := c.Param("id")
	var config models.ProxyConfigCache
	if err := database.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "代理配置不存在: " + err.Error(),
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, config.NodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "节点不存在: " + err.Error(),
		})
		return
	}

	proxyData := map[string]interface{}{
		"hostname": config.Hostname,
		"domain":   config.Domain,
	}

	result := callNodeAPIForProxyMgmt(node, "POST", "/api/proxy/delete", proxyData)
	if result["code"] != float64(200) {
		msg := "未知错误"
		if msgStr, ok := result["msg"].(string); ok {
			msg = msgStr
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "删除失败: " + msg,
		})
		return
	}

	if err := database.DB.Unscoped().Delete(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "删除缓存失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "反向代理删除成功",
	})
}

// CheckProxyDomain 检查域名是否可用
// @Summary 检查反向代理域名是否可用
// @Description 检查指定域名是否可用于反向代理配置
// @Tags 反向代理管理
// @Produce json
// @Param node_id query string true "节点ID"
// @Param domain query string true "域名"
// @Success 200 {object} map[string]interface{} "检查结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/proxy/check [get]
func CheckProxyDomain(c *gin.Context) {
	nodeID := c.Query("node_id")
	domain := c.Query("domain")

	if nodeID == "" || domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "缺少必要参数",
			"data": map[string]interface{}{
				"available": false,
				"reason":    "缺少必要参数",
			},
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "节点不存在",
			"data": map[string]interface{}{
				"available": false,
				"reason":    "节点不存在",
			},
		})
		return
	}

	result := callNodeAPIForProxyMgmt(node, "GET", "/api/proxy/check?domain="+domain, nil)
	c.JSON(http.StatusOK, result)
}

func callNodeAPIForProxyMgmt(node models.Node, method, path string, data interface{}) map[string]interface{} {
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

