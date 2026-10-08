package config

import "testing"

func TestDemoOffPreservesSelfHostedConfiguration(t *testing.T) {
	t.Setenv("TM_DEMO_MODE", "false")
	t.Setenv("TM_ALLOW_REGISTRATION", "true")
	t.Setenv("TM_COACH_ENABLED", "true")
	t.Setenv("TM_MARKET_DATA_PROVIDER", "yahoo")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.AllowRegistration || !c.CoachEnabled || c.MarketDataProvider != "yahoo" {
		t.Fatal("demo policy changed a normal installation")
	}
}

func TestDemoOverridesUnsafeEnvironment(t *testing.T) {
	t.Setenv("TM_DEMO_MODE", "true")
	t.Setenv("TM_ALLOW_REGISTRATION", "true")
	t.Setenv("TM_ALLOW_INSECURE_JWT", "true")
	t.Setenv("TM_COACH_ENABLED", "true")
	t.Setenv("TM_OCR_ENABLED", "true")
	t.Setenv("TM_SHARE_LINKS_ENABLED", "true")
	t.Setenv("TM_MARKET_DATA_PROVIDERS", "yahoo,finnhub")
	t.Setenv("TM_MARKET_DATA_HTTP_BASE_URL", "https://example.com")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AllowRegistration || c.AllowInsecureJWT || c.OCREnabled || c.CoachEnabled || c.ShareLinksEnabled || c.JobsEnabled || c.EconCalendarEnabled {
		t.Fatal("demo enabled an unsafe feature")
	}
	if c.MarketDataProvider != "http" || c.MarketDataProviders != "" || c.MarketDataHTTPBaseURL != "http://127.0.0.1:18964" {
		t.Fatal("demo escaped fictional market provider")
	}
}
