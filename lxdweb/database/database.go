package database
import (
	"database/sql"
	"log"
	"lxdweb/config"
	"lxdweb/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)
var DB *gorm.DB
func InitDB() {
	var err error
	dsn := config.AppConfig.Database.Path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("[ERROR] 数据库连接失败: %v", err)
	}
	DB, err = gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("[ERROR] 数据库连接失败: %v", err)
	}
	err = DB.AutoMigrate(
		&models.Admin{},
		&models.Node{},
		&models.Container{},
		&models.ContainerCache{},
		&models.SyncTask{},
		&models.NodeInfoCache{},
		&models.NATRule{},
		&models.NATRuleCache{},
		&models.NATSyncTask{},
		&models.IPv6BindingCache{},
		&models.IPv6SyncTask{},
		&models.ProxyConfigCache{},
		&models.ProxySyncTask{},
		&models.OperationLog{},
		&models.Image{},
	)
	if err != nil {
		log.Fatalf("[ERROR] 数据库迁移失败: %v", err)
	}
	log.Printf("[DB] 数据库初始化完成")
}
func CheckAdminExists() {
	var count int64
	DB.Model(&models.Admin{}).Count(&count)
	if count == 0 {
		log.Printf("")
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("[WARN] 未检测到管理员账号")
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("")
		log.Printf("请使用以下命令创建管理员账号:")
		log.Printf("  lxdweb admin create")
		log.Printf("")
		log.Printf("服务已启动，但无法登录，请先创建管理员账号")
		log.Printf("")
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}
}
