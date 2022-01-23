package configs

var RuntimeConfig *config

func NewRuntimeConfig() *config {
	if RuntimeConfig != nil {
		panic("Unexpected try to create second instance")
	}
	RuntimeConfig = &config{
		TradingEnabled: true,
	}
	return RuntimeConfig
}

type config struct {
	TradingEnabled bool
}
