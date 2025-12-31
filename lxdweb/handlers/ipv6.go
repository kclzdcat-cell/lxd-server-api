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

// IPv6Page IPv6管理页面
// @Summary IPv6管理页面
// @Description 显示IPv6绑定管理页面
// @Tags IPv6管理
// @Produce html
// @Success 200 {string} string "HTML页面"
// @Router /ipv6 [get]
func IPv6Page(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	c.HTML(http.StatusOK, "ipv6.html", gin.H{
		"title":    "IPv6管理 - LXD管理后台",
		"username": username,
	})
}

// GetIPv6Bindings 获取IPv6绑定列表
// @Summary 获取IPv6绑定列表
// @Description 查询所有IPv6绑定信息，支持按节点过滤
// @Tags IPv6管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回IPv6绑定列表"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/ipv6 [get]
func GetIPv6Bindings(c *gin.Context) {
	var bindings []models.IPv6BindingCache
	query := database.DB.Order("created_at desc")

	if nodeID := c.Query("node_id"); nodeID != "" {
		query = query.Where("node_id = ?", nodeID)
	}

	if err := query.Find(&bindings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": bindings,
	})
}

// SyncIPv6Bindings 同步IPv6绑定
// @Summary 同步IPv6绑定
// @Description 从lxdapi同步指定节点的IPv6绑定信息
// @Tags IPv6管理
// @Produce json
// @Param node_id query string true "节点ID"
// @Success 200 {object} map[string]interface{} "同步任务已启动"
// @Failure 400 {object} map[string]interface{} "缺少参数"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/ipv6/sync [post]
func SyncIPv6Bindings(c *gin.Context) {
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

	go services.SyncNodeIPv6Bindings(node.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "同步任务已启动",
	})
}

// GetIPv6SyncTasks 获取IPv6同步任务列表
// @Summary 获取IPv6同步任务列表
// @Description 查询最近50条IPv6同步任务记录
// @Tags IPv6管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回任务列表"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/ipv6/tasks [get]
func GetIPv6SyncTasks(c *gin.Context) {
	var tasks []models.IPv6SyncTask
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

// GetIPv6BindingsFromCache 从缓存获取IPv6绑定
// @Summary 从缓存获取IPv6绑定
// @Description 直接从本地缓存数据库读取IPv6绑定信息
// @Tags IPv6管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Param hostname query string false "容器名称"
// @Success 200 {object} map[string]interface{} "成功返回缓存数据"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/ipv6/cache [get]
func GetIPv6BindingsFromCache(c *gin.Context) {
	var bindings []models.IPv6BindingCache
	query := database.DB.
		Joins("JOIN nodes ON nodes.id = ipv6_binding_caches.node_id").
		Where("nodes.status = ?", "active").
		Order("ipv6_binding_caches.last_sync desc")

	if nodeID := c.Query("node_id"); nodeID != "" {
		query = query.Where("ipv6_binding_caches.node_id = ?", nodeID)
	}
	if hostname := c.Query("hostname"); hostname != "" {
		query = query.Where("ipv6_binding_caches.hostname = ?", hostname)
	}

	if err := query.Find(&bindings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": bindings,
	})
}

// SyncAllIPv6 同步所有节点的IPv6绑定
// @Summary 同步所有节点的IPv6绑定
// @Description 启动所有活跃节点的IPv6绑定同步任务
// @Tags IPv6管理
// @Produce json
// @Success 200 {object} map[string]interface{} "同步任务已启动"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/ipv6-sync/all [post]
func SyncAllIPv6(c *gin.Context) {
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
		go services.SyncNodeIPv6Bindings(node.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "IPv6绑定同步任务已启动，请稍后刷新页面查看结果",
	})
}

// GetIPv6SyncStatus 获取IPv6同步状态
// @Summary 获取IPv6同步状态
// @Description 查询指定节点或所有节点的IPv6绑定同步状态
// @Tags IPv6管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回同步状态"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/ipv6-sync/status [get]
func GetIPv6SyncStatus(c *gin.Context) {
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
			var lastTask models.IPv6SyncTask
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
			var lastTask models.IPv6SyncTask
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

// CreateIPv6Binding 创建IPv6绑定
// @Summary 创建IPv6绑定
// @Description 为指定容器创建新的IPv6地址绑定
// @Tags IPv6管理
// @Accept json
// @Produce json
// @Param body body models.CreateIPv6Request true "IPv6绑定参数"
// @Success 200 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/ipv6 [post]
func CreateIPv6Binding(c *gin.Context) {
	var req models.CreateIPv6Request
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

	ipv6Data := map[string]interface{}{
		"hostname":    req.ContainerHostname,
		"description": req.Description,
	}

	result := callNodeAPIForIPv6Mgmt(node, "POST", "/api/ipv6/add", ipv6Data)
	if result["code"] != float64(200) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result["msg"],
		})
		return
	}

	time.Sleep(1 * time.Second)
	go services.SyncNodeIPv6Bindings(node.ID)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "IPv6绑定创建成功",
		"data": result["data"],
	})
}

// DeleteIPv6Binding 删除IPv6绑定
// @Summary 删除IPv6绑定
// @Description 删除指定的IPv6地址绑定并清理数据库
// @Tags IPv6管理
// @Produce json
// @Param id path string true "绑定ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 404 {object} map[string]interface{} "绑定或节点不存在"
// @Router /api/ipv6/{id} [delete]
func DeleteIPv6Binding(c *gin.Context) {
	id := c.Param("id")
	var binding models.IPv6BindingCache
	if err := database.DB.First(&binding, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "IPv6绑定不存在",
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, binding.NodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "节点不存在",
		})
		return
	}

	ipv6Data := map[string]interface{}{
		"hostname":    binding.Hostname,
		"public_ipv6": binding.IPv6Address,
	}

	result := callNodeAPIForIPv6Mgmt(node, "POST", "/api/ipv6/delete", ipv6Data)
	if result["code"] != float64(200) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result["msg"],
		})
		return
	}

	if err := database.DB.Unscoped().Delete(&binding).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "删除缓存失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}

func callNodeAPIForIPv6Mgmt(node models.Node, method, path string, data interface{}) map[string]interface{} {
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

