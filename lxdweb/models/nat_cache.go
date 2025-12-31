package models

import (
	"time"
	"gorm.io/gorm"
)

type NATRuleCache struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	NodeID            uint           `json:"node_id" gorm:"not null;index:idx_nat_cache;uniqueIndex:idx_unique_nat_cache"`
	NodeName          string         `json:"node_name" gorm:"size:200"`
	ContainerHostname string         `json:"container_hostname" gorm:"size:200;not null;uniqueIndex:idx_unique_nat_cache"`
	ExternalPort      int            `json:"external_port" gorm:"not null;uniqueIndex:idx_unique_nat_cache"`
	InternalPort      int            `json:"internal_port" gorm:"not null"`
	Protocol          string         `json:"protocol" gorm:"size:10;default:'tcp';uniqueIndex:idx_unique_nat_cache"`
	Description       string         `json:"description" gorm:"type:text"`
	Status            string         `json:"status" gorm:"size:50;default:'active'"`
	
	LastSync          time.Time      `json:"last_sync"`
	SyncError         string         `json:"sync_error" gorm:"type:text"`
	
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

type NATSyncTask struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	NodeID         uint           `json:"node_id" gorm:"index"`
	NodeName       string         `json:"node_name" gorm:"size:200"`
	Status         string         `json:"status" gorm:"size:50;default:'pending'"` 
	TotalCount     int            `json:"total_count"`
	SuccessCount   int            `json:"success_count"`
	FailedCount    int            `json:"failed_count"`
	StartTime      *time.Time     `json:"start_time"`
	EndTime        *time.Time     `json:"end_time"`
	ErrorMessage   string         `json:"error_message" gorm:"type:text"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (NATRuleCache) TableName() string {
	return "nat_rule_cache"
}

func (NATSyncTask) TableName() string {
	return "nat_sync_tasks"
}

