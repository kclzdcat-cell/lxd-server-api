package handlers

import (
	"net/http"
	"strconv"

	"lxdweb/database"
	"lxdweb/models"
	"lxdweb/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// SyncAllNAT 同步所有节点的NAT规则
// @Summary 同步所有节点的NAT规则
// @Description 启动所有活跃节点的NAT规则同步任务
// @Tags NAT管理
// @Produce json
// @Success 200 {object} map[string]interface{} "同步任务已启动"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/nat-sync/all [post]
func SyncAllNAT(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}

	go services.SyncAllNodesNATAsync()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "NAT规则同步任务已启动，请稍后刷新页面查看结果",
	})
}

// SyncNodeNAT 同步指定节点的NAT规则
// @Summary 同步指定节点的NAT规则
// @Description 启动指定节点的NAT规则同步任务
// @Tags NAT管理
// @Produce json
// @Param id path string true "节点ID"
// @Success 200 {object} map[string]interface{} "同步任务已启动"
// @Failure 400 {object} map[string]interface{} "节点ID格式错误或节点正在同步"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/nat-sync/node/{id} [post]
func SyncNodeNAT(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}

	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "节点ID格式错误",
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

	if services.IsNATSyncing(uint(nodeID)) {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "该节点NAT规则正在同步中，请稍后再试",
		})
		return
	}

	go services.SyncNodeNATRules(uint(nodeID), true)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "NAT规则同步任务已启动，请稍后刷新页面查看结果",
	})
}

// GetNATSyncTasks 获取NAT同步任务列表
// @Summary 获取NAT同步任务列表
// @Description 查询最近50条NAT规则同步任务记录
// @Tags NAT管理
// @Produce json
// @Success 200 {object} map[string]interface{} "成功返回任务列表"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/nat-sync/tasks [get]
func GetNATSyncTasks(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}

	var tasks []models.NATSyncTask
	database.DB.Order("created_at DESC").Limit(50).Find(&tasks)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": tasks,
	})
}

// GetNATSyncStatus 获取NAT同步状态
// @Summary 获取NAT同步状态
// @Description 查询指定节点或所有节点的NAT规则同步状态
// @Tags NAT管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回同步状态"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/nat-sync/status [get]
func GetNATSyncStatus(c *gin.Context) {
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
			isSyncing := services.IsNATSyncing(uint(nodeID))
			
			var lastTask models.NATSyncTask
			database.DB.Where("node_id = ?", nodeID).Order("created_at DESC").First(&lastTask)
			
			status = append(status, map[string]interface{}{
				"node_id":   uint(nodeID),
				"syncing":   isSyncing,
				"last_task": lastTask,
			})
		}
	} else {
		var nodes []models.Node
		database.DB.Find(&nodes)
		
		for _, node := range nodes {
			isSyncing := services.IsNATSyncing(node.ID)
			
			var lastTask models.NATSyncTask
			database.DB.Where("node_id = ?", node.ID).Order("created_at DESC").First(&lastTask)
			
			status = append(status, map[string]interface{}{
				"node_id":   node.ID,
				"node_name": node.Name,
				"syncing":   isSyncing,
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

// GetNATRulesFromCache 从缓存获取NAT规则列表
// @Summary 从缓存获取NAT规则列表
// @Description 直接从本地缓存数据库读取NAT规则信息
// @Tags NAT管理
// @Produce json
// @Success 200 {object} map[string]interface{} "成功返回缓存数据"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Router /api/nat/cache [get]
func GetNATRulesFromCache(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未登录",
		})
		return
	}

	var rules []models.NATRuleCache
	result := database.DB.
		Joins("JOIN nodes ON nodes.id = nat_rule_cache.node_id").
		Where("nodes.status = ?", "active").
		Order("nat_rule_cache.node_id ASC, nat_rule_cache.external_port ASC").
		Find(&rules)
	
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败: " + result.Error.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": rules,
	})
}

