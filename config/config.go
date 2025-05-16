// This is temporary config file, in production we will get the data from ssm or any other secrets manager for the secrets and env from the environment variables

package config

import "github.com/spf13/viper"

type (
	DB  struct{ User, Password, Host, Port, Name string }
	App struct {
		Port int
		Mode string
	}
	CSV  struct{ Path string }
	Cron struct{ Spec string }

	Config struct {
		App  App
		DB   DB
		CSV  CSV
		Cron Cron
	}
)

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	var c Config
	if err := v.ReadInConfig(); err != nil {
		return c, err
	}
	if err := v.Unmarshal(&c); err != nil {
		return c, err
	}
	return c, nil
}
