# nexen-viper

A config parser for the [nexen](http://nexen.io/) framework, built on [spf13/viper](https://github.com/spf13/viper) that centralises configuration parsing, file-watching and environment binding.

## How to use it

Import the package

	import "github.com/krakendio/krakend-viper"

And you are ready for building a parser and get the config from any format supported by viper

	parser := viper.New()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
