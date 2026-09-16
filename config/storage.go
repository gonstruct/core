package config

import "strings"

type StorageDriver string

const (
	LocalStorageDriver StorageDriver = "local"
	S3StorageDriver    StorageDriver = "s3"
)

func (driver StorageDriver) Values() []StorageDriver {
	return []StorageDriver{
		LocalStorageDriver,
		S3StorageDriver,
	}
}

func (driver StorageDriver) String() string {
	return string(driver)
}

type Storage struct {
	Driver   StorageDriver
	Adapters StorageAdapters
}

type storageAdapter interface {
	BaseURL() string
}

type StorageAdapters struct {
	Local LocalStorageAdapter
	S3    S3StorageAdapter
}

type LocalStorageAdapter struct {
	BasePath string
}

func (adapter LocalStorageAdapter) BaseURL() string {
	return "file://" + adapter.BasePath
}

type S3StorageAdapter struct {
	AccessKey       string
	SecretAccessKey string
	Region          string
	Bucket          string
	Url             string
	Endpoint        string
	UsePathStyle    bool
}

func (adapter S3StorageAdapter) BaseURL() string {
	return strings.TrimSuffix(adapter.Url, "/") + "/"
}

func (s Storage) Adapter() storageAdapter {
	switch s.Driver {
	case LocalStorageDriver:
		return s.Adapters.Local
	case S3StorageDriver:
		return s.Adapters.S3
	default:
		panic("unknown storage driver: " + s.Driver.String())
	}
}
