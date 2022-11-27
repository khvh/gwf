package config

import (
	"os"
	"path"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

// OASConfig ...
type OASConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// OAuthConfig ...
type OAuthConfig struct {
	Client           string `yaml:"client"`
	Secret           string `yaml:"secret"`
	Realm            string `yaml:"realm"`
	AuthorizationURL string `yaml:"authorizationUrl"`
	TokenURL         string `yaml:"tokenUrl"`
	RedirectURL      string `yaml:"redirectUrl"`
	IssuerURL        string `yaml:"issuerUrl"`
}

// DatabaseConfig ...
type DatabaseConfig struct {
	URL          []string `yaml:"url"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	DatabaseName string   `yaml:"databaseName"`
	Collections  []string `yaml:"collections"`
}

// Server is generic server config data
type Server struct {
	Port int  `yaml:"port" json:"port,omitempty"`
	Dev  bool `yaml:"dev" json:"dev,omitempty"`
	UI   bool `yaml:"ui" json:"ui"`
}

// Configuration is main config object
type Configuration struct {
	ID       string          `json:"id,omitempty" yaml:"id,omitempty"`
	Server   *Server         `json:"server,omitempty" yaml:"server,omitempty"`
	OAS      *OASConfig      `json:"oas" yaml:"oas"`
	OAuth    *OAuthConfig    `json:"oauth" yaml:"oauth"`
	Database *DatabaseConfig `json:"db" yaml:"db"`
}

var conf *Configuration

// Load loads the config with viper
func Load(location, name string) error {
	log.Trace().Msgf("Loading config [%s] from '%s'", name, location)

	if strings.HasPrefix(location, "http") {

	} else {
		f, err := os.ReadFile(path.Join(location, name+".yml"))
		if err != nil {
			log.Trace().Err(err).Msg("While reading config file")

			return err
		}

		err = yaml.Unmarshal(f, &conf)
		if err != nil {
			log.Trace().Err(err).Msg("While unmarshaling config")

			return err
		}
	}

	return nil
}

// Autoload inits .env and loads the configuration
func Autoload() error {
	if err := godotenv.Load(); err != nil {
		log.Trace().Err(err).Msg("While loading .env")

		return err
	}

	checkDefined([]string{"CONFIG_NAME", "CONFIG_LOCATION"})

	return Load(os.Getenv("CONFIG_LOCATION"), os.Getenv("CONFIG_NAME"))
}

// Get config
func Get() *Configuration {
	return conf
}

func checkDefined(arr []string) {
	for _, v := range arr {
		if os.Getenv(v) == "" {
			log.Error().Msgf("Missing [%s] env variable", v)
			log.Fatal().Msgf("Required variables [%s]", strings.Join(arr, ", "))
		}
	}
}