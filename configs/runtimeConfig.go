package configs

var RuntimeConfig *config

func NewRuntimeConfig() *config {
	if RuntimeConfig != nil {
		panic("Unexpected try to create second instance")
	}
	RuntimeConfig = &config{
		buyingEnabled: true,
	}
	return RuntimeConfig
}

type config struct {
	buyingEnabled bool
}

func (c *config) IsBuyingEnabled() bool {
	return c.buyingEnabled
}
func (c *config) EnableBuying() {
	c.buyingEnabled = true
}
func (c *config) DisableBuying() {
	c.buyingEnabled = false
}
