package conf

type Data struct {
	Database Database `mapstructure:"database" json:"database"`
	MinIO    MinIO    `mapstructure:"minio" json:"minio"`
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
