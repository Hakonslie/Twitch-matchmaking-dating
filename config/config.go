package config

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Environment struct {
	Logger  logrus.FieldLogger
	Context context.Context
	Config  Config
}

type Config struct {
	Irc struct {
		Nick    string `yaml:"nick"`
		Pass    string `yaml:"pass"`
		Channel string `yaml:"channel"`
	} `yaml:"irc"`
	Twitch struct {
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		Token        string `yaml:"token"`
	} `yaml:"twitch"`
}

func OpenConfig(logger logrus.FieldLogger) Config {
	f, err := os.Open("./config.yml")
	if err != nil {
		logger.WithError(err).Error()
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		logger.WithError(err).Error()
	}

	return cfg
}
