package common

type TestingDataYaml struct {
	MockFsdServers []FSDServer    `yaml:"mockFsdServers"`
	MockDnsQueries []MockDnsQuery `yaml:"mockDnsQueries"`
}

type MockDnsQuery struct {
	SourceIpAddress    string `yaml:"source_ip_address"`
	ExpectedIpReturned string `yaml:"expected_ip_returned"`
}
