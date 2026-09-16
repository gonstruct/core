package config

type App struct {
	Name        string
	Environment Environment
	AppKey      string
	Port        string
	Url         string
	Domain      string
}

type Environment string

const (
	LocalEnvironment      Environment = "local"
	TestingEnvironment    Environment = "testing"
	PreviewEnvironment    Environment = "preview"
	StagingEnvironment    Environment = "staging"
	ProductionEnvironment Environment = "production"
)

func (Environment) Values() []Environment {
	return []Environment{
		LocalEnvironment,
		TestingEnvironment,
		PreviewEnvironment,
		StagingEnvironment,
		ProductionEnvironment,
	}
}

func (environment Environment) String() string {
	return string(environment)
}
