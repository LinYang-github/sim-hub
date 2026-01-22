package conf

type Data struct {
	Database      Database       `mapstructure:"database" json:"database"`
	MinIO         MinIO          `mapstructure:"minio" json:"minio"`
	ResourceTypes []ResourceType `mapstructure:"resource_types" json:"resource_types"`
	Log           Log            `mapstructure:"log" json:"log"`
}

type Log struct {
	Level      string `mapstructure:"level" json:"level"`             // debug, info, warn, error
	Filename   string `mapstructure:"filename" json:"filename"`       // 日志文件路径
	MaxSize    int    `mapstructure:"max_size" json:"max_size"`       // 每个日志文件最大尺寸 (MB)
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups"` // 保留旧文件最大个数
	MaxAge     int    `mapstructure:"max_age" json:"max_age"`         // 保留旧文件最大天数
	Compress   bool   `mapstructure:"compress" json:"compress"`       // 是否压缩/归档旧文件
	Format     string `mapstructure:"format" json:"format"`           // json 或 text
}

type Database struct {
	Driver string `mapstructure:"driver" json:"driver"` // sqlite, mysql, postgres
	Source string `mapstructure:"source" json:"source"` // DSN
}

type MinIO struct {
	Endpoint  string `mapstructure:"endpoint" json:"endpoint"`
	AccessKey string `mapstructure:"access_key" json:"access_key"`
	SecretKey string `mapstructure:"secret_key" json:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl" json:"use_ssl"`
	Bucket    string `mapstructure:"bucket" json:"bucket"`
}

type ResourceType struct {
	TypeKey      string         `mapstructure:"type_key" json:"type_key"`
	TypeName     string         `mapstructure:"type_name" json:"type_name"`
	SchemaDef    map[string]any `mapstructure:"schema_def" json:"schema_def"`
	ViewerConf   map[string]any `mapstructure:"viewer_conf" json:"viewer_conf"`
	ProcessConf  map[string]any `mapstructure:"process_conf" json:"process_conf"`
	ProcessorCmd string         `mapstructure:"processor_cmd" json:"processor_cmd"`
	CategoryMode string         `mapstructure:"category_mode" json:"category_mode"` // "flat" or "tree"
}
