package handlers
import (
	"fmt"
	"lxdweb/database"
	"lxdweb/models"
	"net/http"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)
// NATPage NAT规则管理页面
// @Summary NAT规则管理页面
// @Description 显示NAT端口转发规则管理页面
// @Tags NAT管理
// @Produce html
// @Success 200 {string} string "HTML页面"
// @Router /nat [get]
func NATPage(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	c.HTML(http.StatusOK, "nat.html", gin.H{
		"title":    "NAT规则管理 - LXD管理后台",
		"username": username,
	})
}
// GetNATRules 获取NAT规则列表
// @Summary 获取NAT规则列表
// @Description 查询所有NAT端口转发规则，支持按节点过滤
// @Tags NAT管理
// @Produce json
// @Param node_id query string false "节点ID"
// @Success 200 {object} map[string]interface{} "成功返回NAT规则列表"
// @Failure 500 {object} map[string]interface{} "查询失败"
// @Router /api/nat [get]
func GetNATRules(c *gin.Context) {
	var rules []models.NATRule
	query := database.DB.Preload("Node")
	if nodeID := c.Query("node_id"); nodeID != "" {
		query = query.Where("node_id = ?", nodeID)
	}
	if err := query.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": rules,
	})
}
// GetNATRule 获取单个NAT规则
// @Summary 获取单个NAT规则
// @Description 根据ID获取NAT规则详情
// @Tags NAT管理
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} map[string]interface{} "成功返回NAT规则"
// @Failure 404 {object} map[string]interface{} "NAT规则不存在"
// @Router /api/nat/{id} [get]
func GetNATRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.NATRule
	if err := database.DB.Preload("Node").First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "NAT规则不存在",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": rule,
	})
}
// CreateNATRule 创建NAT规则
// @Summary 创建NAT规则
// @Description 创建新的NAT端口转发规则
// @Tags NAT管理
// @Accept json
// @Produce json
// @Param body body models.CreateNATRequest true "NAT规则参数"
// @Success 200 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Failure 409 {object} map[string]interface{} "端口已被占用"
// @Router /api/nat [post]
func CreateNATRule(c *gin.Context) {
	var req models.CreateNATRequest
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
	var existingRule models.NATRule
	if err := database.DB.Where("node_id = ? AND external_port = ? AND protocol = ?", 
		req.NodeID, req.ExternalPort, req.Protocol).First(&existingRule).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"code": 409,
			"msg":  "该端口已被占用",
		})
		return
	}
	natData := map[string]interface{}{
		"hostname": req.ContainerHostname,
		"dport":    req.ExternalPort,
		"sport":    req.InternalPort,
		"dtype":    req.Protocol,
	}
	if req.Description != "" {
		natData["description"] = req.Description
	}
	result := callNodeAPI(node, "POST", "/api/addport", natData)
	if result["code"] != float64(200) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result["msg"],
		})
		return
	}
	rule := models.NATRule{
		NodeID:            req.NodeID,
		ContainerHostname: req.ContainerHostname,
		ExternalPort:      req.ExternalPort,
		InternalPort:      req.InternalPort,
		Protocol:          req.Protocol,
		Description:       req.Description,
		Status:            "active",
	}
	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "保存失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "NAT规则创建成功",
		"data": rule,
	})
}
// UpdateNATRule 更新NAT规则
// @Summary 更新NAT规则
// @Description 更新NAT规则的描述或状态
// @Tags NAT管理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param body body models.UpdateNATRequest true "更新参数"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "NAT规则不存在"
// @Router /api/nat/{id} [put]
func UpdateNATRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.NATRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "NAT规则不存在",
		})
		return
	}
	var req models.UpdateNATRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
		return
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.Status != "" {
		rule.Status = req.Status
	}
	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "更新失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新成功",
		"data": rule,
	})
}
// DeleteNATRule 删除NAT规则
// @Summary 删除NAT规则
// @Description 删除指定的NAT端口转发规则并清理数据库
// @Tags NAT管理
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 404 {object} map[string]interface{} "NAT规则或节点不存在"
// @Router /api/nat/{id} [delete]
func DeleteNATRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.NATRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "NAT规则不存在",
		})
		return
	}
	var node models.Node
	if err := database.DB.First(&node, rule.NodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "节点不存在",
		})
		return
	}
	natData := map[string]interface{}{
		"hostname": rule.ContainerHostname,
		"dport":    rule.ExternalPort,
		"sport":    rule.InternalPort,
		"dtype":    rule.Protocol,
	}
	result := callNodeAPI(node, "POST", "/api/delport", natData)
	if result["code"] != float64(200) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  result["msg"],
		})
		return
	}
	if err := database.DB.Unscoped().Delete(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "删除失败: " + err.Error(),
		})
		return
	}

	database.DB.Unscoped().Where("node_id = ? AND container_hostname = ? AND external_port = ? AND protocol = ?",
		rule.NodeID, rule.ContainerHostname, rule.ExternalPort, rule.Protocol).
		Delete(&models.NATRuleCache{})

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}
// SyncNATRules 同步NAT规则
// @Summary 同步NAT规则
// @Description 同步指定节点的NAT规则信息
// @Tags NAT管理
// @Produce json
// @Param node_id query string true "节点ID"
// @Success 200 {object} map[string]interface{} "同步任务已启动"
// @Failure 400 {object} map[string]interface{} "缺少参数"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/nat/sync [post]
func SyncNATRules(c *gin.Context) {
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
	result := callNodeAPI(node, "GET", "/api/nat/list", nil)
	if result["code"] != float64(200) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "同步失败: " + result["msg"].(string),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "同步成功",
		"data": result["data"],
	})
}

// CheckNATPort 检查NAT端口是否可用
// @Summary 检查NAT端口是否可用
// @Description 检查指定端口和协议是否可用于NAT转发
// @Tags NAT管理
// @Produce json
// @Param node_id query string true "节点ID"
// @Param hostname query string true "容器名称"
// @Param protocol query string true "协议类型(tcp/udp)"
// @Param port query int true "端口号"
// @Success 200 {object} map[string]interface{} "检查结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "节点不存在"
// @Router /api/nat/check [get]
func CheckNATPort(c *gin.Context) {
	nodeID := c.Query("node_id")
	hostname := c.Query("hostname")
	protocol := c.Query("protocol")
	port := c.Query("port")

	if nodeID == "" || hostname == "" || protocol == "" || port == "" {
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

	result := callNodeAPI(node, "GET", fmt.Sprintf("/api/nat/check?hostname=%s&protocol=%s&port=%s", hostname, protocol, port), nil)
	c.JSON(http.StatusOK, result)
}
