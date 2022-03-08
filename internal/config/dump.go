package config

import "github.com/clevyr/kubedb/internal/database/sqlformat"

type Dump struct {
	Database         string
	Username         string
	Password         string
	Directory        string
	OutputFormat     sqlformat.Format
	IfExists         bool
	Clean            bool
	NoOwner          bool
	Tables           []string
	ExcludeTable     []string
	ExcludeTableData []string
}
