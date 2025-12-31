package models
import (
	"time"
	"gorm.io/gorm"
)
type Container struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	NodeID     uint           `json:"node_id" gorm:"not null;index:idx_node_hostname"`
	Hostname   string         `json:"hostname" gorm:"size:200;not null;index:idx_node_hostname"`
	Status     string         `json:"status" gorm:"size:50"`      
	IPv4       string         `json:"ipv4" gorm:"size:50"`
	IPv6       string         `json:"ipv6" gorm:"size:200"`
	Image      string         `json:"image" gorm:"size:200"`
	CPUs       int            `json:"cpus"`
	Memory     string         `json:"memory" gorm:"size:50"`
	Disk       string         `json:"disk" gorm:"size:50"`
	LastSync   *time.Time     `json:"last_sync"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	Node Node `json:"node" gorm:"foreignKey:NodeID"`
}
type NATRule struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	NodeID            uint           `json:"node_id" gorm:"not null;index:idx_nat_unique"`
	ContainerHostname string         `json:"container_hostname" gorm:"size:200;not null"`
	ExternalPort      int            `json:"external_port" gorm:"not null;index:idx_nat_unique"`
	InternalPort      int            `json:"internal_port" gorm:"not null"`
	Protocol          string         `json:"protocol" gorm:"size:10;default:'tcp';index:idx_nat_unique"` 
	Status            string         `json:"status" gorm:"size:50;default:'active'"`
	Description       string         `json:"description" gorm:"type:text"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	Node Node `json:"node" gorm:"foreignKey:NodeID"`
}
type OperationLog struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AdminID       uint      `json:"admin_id" gorm:"index"`
	OperationType string    `json:"operation_type" gorm:"size:50;not null"` 
	TargetType    string    `json:"target_type" gorm:"size:50"`             
	TargetID      uint      `json:"target_id"`
	Details       string    `json:"details" gorm:"type:text"`
	IPAddress     string    `json:"ip_address" gorm:"size:100"`
	Status        string    `json:"status" gorm:"size:50"`       
	ErrorMessage  string    `json:"error_message" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at"`
}
type Image struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" gorm:"size:200;not null"`
	Alias        string         `json:"alias" gorm:"uniqueIndex;size:200;not null"`
	OS           string         `json:"os" gorm:"size:100"`
	Version      string         `json:"version" gorm:"size:100"`
	Architecture string         `json:"architecture" gorm:"size:50"`
	Description  string         `json:"description" gorm:"type:text"`
	IsActive     bool           `json:"is_active" gorm:"default:1"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
type CreateNATRequest struct {
	NodeID            uint   `json:"node_id" binding:"required"`
	ContainerHostname string `json:"container_hostname" binding:"required"`
	ExternalPort      int    `json:"external_port" binding:"required"`
	InternalPort      int    `json:"internal_port" binding:"required"`
	Protocol          string `json:"protocol" binding:"required,oneof=tcp udp"`
	Description       string `json:"description"`
}
type UpdateNATRequest struct {
	ExternalPort int    `json:"external_port"`
	InternalPort int    `json:"internal_port"`
	Protocol     string `json:"protocol" binding:"omitempty,oneof=tcp udp"`
	Description  string `json:"description"`
	Status       string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type CreateIPv6Request struct {
	NodeID            uint   `json:"node_id" binding:"required"`
	ContainerHostname string `json:"container_hostname" binding:"required"`
	Description       string `json:"description"`
}

type DeleteIPv6Request struct {
	PublicIPv6 string `json:"public_ipv6" binding:"required"`
}

type CreateProxyRequest struct {
	NodeID            uint   `json:"node_id" binding:"required"`
	ContainerHostname string `json:"container_hostname" binding:"required"`
	Domain            string `json:"domain" binding:"required"`
	ContainerPort     int    `json:"container_port" binding:"required"`
	Description       string `json:"description"`
	SSLEnabled        bool   `json:"ssl_enabled"`
	SSLType           string `json:"ssl_type"`
	SSLCert           string `json:"ssl_cert"`
	SSLKey            string `json:"ssl_key"`
}

type DeleteProxyRequest struct {
	Domain string `json:"domain" binding:"required"`
}
