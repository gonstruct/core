package config

import "fmt"

type SearchDriver string

const (
	TypesenseSearchDriver   SearchDriver = "typesense"
	MeilisearchSearchDriver SearchDriver = "meilisearch"
)

func (driver SearchDriver) Values() []SearchDriver {
	return []SearchDriver{
		TypesenseSearchDriver,
		MeilisearchSearchDriver,
	}
}

func (driver SearchDriver) String() string {
	return string(driver)
}

type IndexSettings struct {
	FilterableAttributes []string
	SortableAttributes   []string
	SearchableAttributes []string
}

type Search struct {
	Driver      SearchDriver
	Indexes     map[string]IndexSettings
	Typesense   Typesense
	Meilisearch Meilisearch
}

type Meilisearch struct {
	ApiKey                         string
	Host                           string
	TaskWebhookAuthorizationSecret string
}

type Typesense struct {
	ApiKey                     string
	Nodes                      TypesenseNodeSlice
	NearestNode                TypesenseNode
	ConnectionTimeoutSeconds   int
	HealthcheckIntervalSeconds int
	NumRetries                 int
	RetryIntervalSeconds       int
}

type TypesenseNodeSlice []TypesenseNode

func (nodes TypesenseNodeSlice) Addresses() (addresses []string) {
	for _, node := range nodes {
		addresses = append(addresses, node.Address())
	}
	return addresses
}

type TypesenseNode struct {
	Host     string
	Port     int
	Protocol string
	Path     string
}

func (node TypesenseNode) Address() string {
	return node.Protocol + "://" + node.Host + ":" + fmt.Sprint(node.Port) + node.Path
}
