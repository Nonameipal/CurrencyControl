package configs

type PostgresParams struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type AppParams struct {
	ServerURL  string
	ServerName string
	PortRun    string
	GinMode    string
}

type ADParams struct {
	Server     string
	Domain     string
	SearchBase string
}

type ABSParams struct {
	Endpoint string
}
type AuthParams struct {
	AccessTokenTtlMinutes int
	RefreshTokenTtlDays   int
	JwtSecret             string
}

type Configs struct {
	AppParams      AppParams
	PostgresParams PostgresParams
	AuthParams     AuthParams
	ADParams       ADParams
	ABSParams      ABSParams
}
