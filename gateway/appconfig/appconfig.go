package appconfig

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hoophq/hoop/common/envloader"
)

// TODO: it should include all runtime configuration

const (
	defaultPostgRESTRole             = "hoop_apiuser"
	defaultWebappStaticUiPath string = "/app/ui/public"
)

type pgCredentials struct {
	connectionString string
	username         string
	// Postgrest Role Name
	postgrestRole string
}
type Config struct {
	apiKey                          string
	askAICredentials                *url.URL
	authMethod                      string
	pgCred                          *pgCredentials
	gcpDLPJsonCredentials           string
	dlpProvider                     string
	hasRedactCredentials            bool
	msPresidioAnalyzerURL           string
	msPresidioAnonymizerURL         string
	webhookAppKey                   string
	webhookAppURL                   *url.URL
	licenseSigningKey               *rsa.PrivateKey
	licenseSignerOrgID              string
	migrationPathFiles              string
	orgMultitenant                  bool
	doNotTrack                      bool
	apiURL                          string
	grpcURL                         string
	apiHostname                     string
	apiHost                         string
	apiScheme                       string
	apiURLPath                      string
	webappUsersManagement           string
	jwtSecretKey                    []byte
	webappStaticUIPath              string
	disableSessionsDownload         bool
	gatewayTLSCa                    string
	gatewayTLSKey                   string
	gatewayTLSCert                  string
	sshClientHostKey                string
	integrationAWSInstanceRoleAllow bool

	isLoaded bool
}

var runtimeConfig Config

// Load validate for any errors and set the RuntimeConfig var
func Load() error {
	if runtimeConfig.isLoaded {
		return nil
	}
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://127.0.0.1:8009"
	}
	grpcURL := os.Getenv("GRPC_URL")
	if grpcURL == "" {
		grpcURL = "grpc://127.0.0.1:8010"
	}
	apiRawURL, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("failed parsing API_URL env, reason=%v", err)
	}
	askAICred, err := loadAskAICredentials()
	if err != nil {
		return err
	}
	pgCred, err := loadPostgresCredentials()
	if err != nil {
		return err
	}
	pgCred.postgrestRole = os.Getenv("PGREST_ROLE")
	if pgCred.postgrestRole == "" {
		pgCred.postgrestRole = defaultPostgRESTRole
	}
	migrationPathFiles := strings.TrimSuffix(os.Getenv("MIGRATION_PATH_FILES"), "/")
	if migrationPathFiles == "" {
		migrationPathFiles = "/app/migrations"
	}
	firstMigrationFilePath := fmt.Sprintf("%s/000001_init.up.sql", migrationPathFiles)
	if _, err := os.Stat(firstMigrationFilePath); err != nil {
		return fmt.Errorf("unable to find first migration file %v, err=%v", firstMigrationFilePath, err)
	}
	allowedOrgID, licensePrivKey, err := loadLicensePrivateKey()
	if err != nil {
		return err
	}
	gcpJsonCred, err := loadGcpDLPCredentials()
	if err != nil {
		return err
	}
	webappUsersManagement := os.Getenv("WEBAPP_USERS_MANAGEMENT")
	if webappUsersManagement == "" {
		webappUsersManagement = "on"
	}
	authMethod, err := loadAuthMethod()
	if err != nil {
		return err
	}
	webappStaticUiPath := os.Getenv("STATIC_UI_PATH")
	if webappStaticUiPath == "" {
		webappStaticUiPath = defaultWebappStaticUiPath
	}
	// it's important to coerce to empty string when the path is just a /
	// in most cases this is a typo provided by the user that will affect
	// every part of the application using the api url to construct links
	if apiRawURL.Path == "/" {
		apiRawURL.Path = ""
	}
	var webhookAppURL *url.URL
	if svixAppURL := os.Getenv("WEBHOOK_APPURL"); svixAppURL != "" {
		webhookAppURL, err = url.Parse(svixAppURL)
		if err != nil {
			return fmt.Errorf("failed parsing WEBHOOK_APPURL, reason=%v", err)
		}
	}

	gatewayTLSCa, err := envloader.GetEnv("TLS_CA")
	if err != nil {
		return fmt.Errorf("failed loading env TLS_CA, reason=%v", err)
	}
	gatewayTLSKey, err := envloader.GetEnv("TLS_KEY")
	if err != nil {
		return fmt.Errorf("failed loading env TLS_KEY, reason=%v", err)
	}
	gatewayTLSCert, err := envloader.GetEnv("TLS_CERT")
	if err != nil {
		return fmt.Errorf("failed loading env TLS_CERT, reason=%v", err)
	}

	hasRedactCredentials := func() bool {
		dlpProvider := os.Getenv("DLP_PROVIDER")
		if dlpProvider == "" || dlpProvider == "gcp" {
			return isEnvSet("GOOGLE_APPLICATION_CREDENTIALS_JSON")
		}

		if dlpProvider == "mspresidio" {
			return isEnvSet("MSPRESIDIO_ANALYZER_URL") && isEnvSet("MSPRESIDIO_ANONYMIZER_URL")
		}

		return false
	}()

	sshClientHostKey := os.Getenv("SSH_CLIENT_HOST_KEY")
	if sshClientHostKey != "" {
		if _, err := base64.StdEncoding.DecodeString(sshClientHostKey); err != nil {
			return fmt.Errorf("failed decoding env SSH_CLIENT_HOST_KEY, err=%v", err)
		}
	}

	runtimeConfig = Config{
		apiKey:                          os.Getenv("API_KEY"),
		apiURL:                          fmt.Sprintf("%s://%s", apiRawURL.Scheme, apiRawURL.Host),
		grpcURL:                         grpcURL,
		apiHostname:                     apiRawURL.Hostname(),
		apiScheme:                       apiRawURL.Scheme,
		apiHost:                         apiRawURL.Host,
		apiURLPath:                      apiRawURL.Path,
		authMethod:                      authMethod,
		askAICredentials:                askAICred,
		pgCred:                          pgCred,
		migrationPathFiles:              migrationPathFiles,
		licenseSigningKey:               licensePrivKey,
		licenseSignerOrgID:              allowedOrgID,
		gcpDLPJsonCredentials:           gcpJsonCred,
		orgMultitenant:                  os.Getenv("ORG_MULTI_TENANT") == "true",
		doNotTrack:                      os.Getenv("DO_NOT_TRACK") == "true",
		dlpProvider:                     os.Getenv("DLP_PROVIDER"),
		hasRedactCredentials:            hasRedactCredentials,
		msPresidioAnalyzerURL:           os.Getenv("MSPRESIDIO_ANALYZER_URL"),
		msPresidioAnonymizerURL:         os.Getenv("MSPRESIDIO_ANONYMIZER_URL"),
		webhookAppKey:                   os.Getenv("WEBHOOK_APPKEY"),
		webhookAppURL:                   webhookAppURL,
		webappUsersManagement:           webappUsersManagement,
		jwtSecretKey:                    []byte(os.Getenv("JWT_SECRET_KEY")),
		webappStaticUIPath:              webappStaticUiPath,
		isLoaded:                        true,
		disableSessionsDownload:         os.Getenv("DISABLE_SESSIONS_DOWNLOAD") == "true",
		gatewayTLSCa:                    gatewayTLSCa,
		gatewayTLSKey:                   gatewayTLSKey,
		gatewayTLSCert:                  gatewayTLSCert,
		sshClientHostKey:                sshClientHostKey,
		integrationAWSInstanceRoleAllow: os.Getenv("INTEGRATION_AWS_INSTANCE_ROLE_ALLOW") == "true",
	}
	return nil
}

func Get() Config { return runtimeConfig }

// loadAuthMethod() returns the auth method to use
// the possible values are: "local" and "idp".
// If not set, it defaults to "local"
// it also cross check the IDP_ISSUER, IDP_CLIENT_ID
// and IDP_CLIENT_SECRET envs to determine if it should
// the IDP configuration already set even without setting
// AUTH_METHOD to "idp".
// This last behavior ensures compatibility with the previous version
func loadAuthMethod() (authMethod string, err error) {
	authMethod = os.Getenv("AUTH_METHOD")
	switch authMethod {
	case "local":
		err = validateLocalAuthJwtKey()
	case "idp":
	default:
		if !hasIdpEnvs() {
			// default to local auth method
			return "local", validateLocalAuthJwtKey()

		}
		return "idp", nil
	}
	return
}

func validateLocalAuthJwtKey() error {
	if jwtSecretKey := os.Getenv("JWT_SECRET_KEY"); jwtSecretKey == "" {
		return fmt.Errorf("when AUTH_METHOD is set as `local`, you must configure a random string value at the JWT_SECRET_KEY environment variable")
	}
	return nil
}

func hasIdpEnvs() bool {
	return os.Getenv("IDP_ISSUER") != "" ||
		os.Getenv("IDP_CLIENT_ID") != "" ||
		os.Getenv("IDP_CLIENT_SECRET") != "" ||
		os.Getenv("IDP_URI") != ""
}

func loadPostgresCredentials() (*pgCredentials, error) {
	pgConnectionURI := os.Getenv("POSTGRES_DB_URI")
	pgURL, err := url.Parse(pgConnectionURI)
	if err != nil {
		return nil, fmt.Errorf("failed parsing POSTGRES_DB_URI, err=%v", err)
	}

	var pgUser, pgPassword string
	if pgURL.User != nil {
		pgUser = pgURL.User.Username()
		pgPassword, _ = pgURL.User.Password()
	}
	if pgUser == "" || pgPassword == "" {
		return nil, fmt.Errorf("missing user or password in POSTGRES_DB_URI env")
	}
	return &pgCredentials{connectionString: pgConnectionURI, username: pgUser}, nil
}

func loadAskAICredentials() (*url.URL, error) {
	askAICred := os.Getenv("ASK_AI_CREDENTIALS")
	if askAICred == "" {
		return nil, nil
	}
	u, err := url.Parse(askAICred)
	if err != nil {
		return nil, fmt.Errorf("ASK_AI_CREDENTIALS env is not in a valid configuration format: %v", err)
	}
	if u.User == nil {
		return nil, fmt.Errorf("ASK_AI_CREDENTIALS env is missing the api key")
	}
	if apiKey, _ := u.User.Password(); apiKey == "" {
		return nil, fmt.Errorf("ASK_AI_CREDENTIALS env is missing the api key")
	}
	return u, nil
}

func loadGcpDLPCredentials() (string, error) {
	jsonCred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	if jsonCred == "" {
		return "", nil
	}
	var js json.RawMessage
	if err := json.Unmarshal([]byte(jsonCred), &js); err != nil {
		return "", fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS_JSON is not in json format, failed parsing: %v", err)
	}
	return jsonCred, nil
}

func loadLicensePrivateKey() (string, *rsa.PrivateKey, error) {
	signingKeyCredentials := os.Getenv("LICENSE_SIGNING_KEY")
	if signingKeyCredentials == "" {
		return "", nil, nil
	}
	allowedOrgID, b64EncPrivateKey, found := strings.Cut(signingKeyCredentials, ",")
	if !found || allowedOrgID == "" {
		return "", nil, nil
	}

	privKeyBytes, err := base64.StdEncoding.DecodeString(b64EncPrivateKey)
	if err != nil {
		return "", nil, fmt.Errorf("unable to load LICENSE_SIGNING_KEY, reason=%v", err)
	}
	block, _ := pem.Decode(privKeyBytes)
	obj, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	privkey, ok := obj.(*rsa.PrivateKey)
	if !ok {
		return "", nil, fmt.Errorf("unable to load LICENSE_SIGNING_KEY: it is not a private key, got=%T", obj)
	}
	return allowedOrgID, privkey, nil
}

func (c Config) LicenseSigningKey() (string, *rsa.PrivateKey) {
	return c.licenseSignerOrgID, c.licenseSigningKey
}

// FullApiURL returns the full url which contains the path of the URL
func (c Config) FullApiURL() string { return c.apiURL + c.apiURLPath }

// ApiURL is the base URL without any path segment or query strings (scheme://host:port)
func (c Config) ApiURL() string                        { return c.apiURL }
func (c Config) GrpcURL() string                       { return c.grpcURL }
func (c Config) WebappStaticUiPath() string            { return c.webappStaticUIPath }
func (c Config) ApiHostname() string                   { return c.apiHostname }
func (c Config) ApiHost() string                       { return c.apiHost } // ApiHost host or host:port
func (c Config) ApiScheme() string                     { return c.apiScheme }
func (c Config) ApiURLPath() string                    { return c.apiURLPath }
func (c Config) ApiKey() string                        { return c.apiKey }
func (c Config) AuthMethod() string                    { return c.authMethod }
func (c Config) WebhookAppKey() string                 { return c.webhookAppKey }
func (c Config) WebhookAppURL() *url.URL               { return c.webhookAppURL }
func (c Config) GcpDLPJsonCredentials() string         { return c.gcpDLPJsonCredentials }
func (c Config) DlpProvider() string                   { return c.dlpProvider }
func (c Config) HasRedactCredentials() bool            { return c.hasRedactCredentials }
func (c Config) MSPresidioAnalyzerURL() string         { return c.msPresidioAnalyzerURL }
func (c Config) MSPresidioAnomymizerURL() string       { return c.msPresidioAnonymizerURL }
func (c Config) PgUsername() string                    { return c.pgCred.username }
func (c Config) PgURI() string                         { return c.pgCred.connectionString }
func (c Config) PostgRESTRole() string                 { return c.pgCred.postgrestRole }
func (c Config) DoNotTrack() bool                      { return c.doNotTrack }
func (c Config) DisableSessionsDownload() bool         { return c.disableSessionsDownload }
func (c Config) MigrationPathFiles() string            { return c.migrationPathFiles }
func (c Config) OrgMultitenant() bool                  { return c.orgMultitenant }
func (c Config) WebappUsersManagement() string         { return c.webappUsersManagement }
func (c Config) IsAskAIAvailable() bool                { return c.askAICredentials != nil }
func (c Config) JWTSecretKey() []byte                  { return c.jwtSecretKey }
func (c Config) GatewayTLSCa() string                  { return c.gatewayTLSCa }
func (c Config) GatewayTLSKey() string                 { return c.gatewayTLSKey }
func (c Config) GatewayTLSCert() string                { return c.gatewayTLSCert }
func (c Config) SSHClientHostKey() string              { return c.sshClientHostKey }
func (c Config) IntegrationAWSInstanceRoleAllow() bool { return c.integrationAWSInstanceRoleAllow }
func (c Config) AskAIApiURL() (u string) {
	if c.IsAskAIAvailable() {
		return fmt.Sprintf("%s://%s", c.askAICredentials.Scheme, c.askAICredentials.Host)
	}
	return ""
}

func (c Config) AskAIAPIKey() (t string) {
	if c.IsAskAIAvailable() {
		t, _ = c.askAICredentials.User.Password()
	}
	return
}

func isEnvSet(key string) bool {
	val, isset := os.LookupEnv(key)
	return isset && val != ""
}
