package config

// Demo policy overrides environment mistakes. Normal installations never enter it.
func (c *Config) applyDemoPolicy() {
	c.AllowRegistration = false
	c.AllowInsecureJWT = false
	c.OCREnabled = false
	c.CoachEnabled = false
	c.ShareLinksEnabled = false
	c.EconCalendarEnabled = false
	c.JobsEnabled = false
	c.CORSOrigins = nil
	c.MarketDataEnabled = true
	c.MarketDataProvider = "http"
	c.MarketDataProviders = ""
	c.MarketDataHTTPBaseURL = "http://127.0.0.1:18964"
	c.MarketDataAPIKey = ""
	c.MarketDataHTTPAPIKey = ""
	c.OCRVisionAPIKey = ""
	c.CoachAPIKey = ""
}
