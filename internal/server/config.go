package server

type Config struct {
	httpPort     int
	interval     int
	payout       int
	payoutTokens int
	proxyCount   int
	queueCap     int
}

func NewConfig(httpPort, interval, payout, payoutTokens, proxyCount, queueCap int) *Config {
	return &Config{
		httpPort:     httpPort,
		interval:     interval,
		payout:       payout,
		payoutTokens: payoutTokens,
		proxyCount:   proxyCount,
		queueCap:     queueCap,
	}
}
