package candy

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func LoadConfig(cfgFile string, cmd *cobra.Command, requiredFlags []string, opts interface{}) error {
	v := viper.New()

	cmd.Flags().VisitAll(func(flag *flag.Flag) {
		flagName := flag.Name
		if flagName != "config" && flagName != "help" {
			if err := v.BindPFlag(flagName, flag); err != nil {
				panic(fmt.Errorf("error binding flag '%s': %w", flagName, err).Error())
			}
		}
	})

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.SetEnvPrefix("CANDY")

	if _, err := os.Stat(cfgFile); err == nil {
		v.SetConfigFile(cfgFile)
		v.SetConfigType("json")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("Error loading config file %s: %w", cfgFile, err)
		}
	} else {
		Log().Info("using config file", zap.String("file", v.ConfigFileUsed()))
	}

	if err := v.Unmarshal(opts); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	elem := reflect.ValueOf(opts).Elem()
	for _, requiredFlag := range requiredFlags {
		fieldName := kebabCaseToTitleCamelCase(requiredFlag)
		f := elem.FieldByName(fieldName)
		value := fmt.Sprintf("%v", f.Interface())
		if value == "" {
			return fmt.Errorf("'--%s' is required", requiredFlag)
		}
	}

	return nil
}

func kebabCaseToTitleCamelCase(input string) (result string) {
	nextToUpper := true
	for _, runeValue := range input {
		if nextToUpper {
			result += strings.ToUpper(string(runeValue))
			nextToUpper = false
		} else {
			if runeValue == '-' {
				nextToUpper = true
			} else {
				result += string(runeValue)
			}
		}
	}
	return
}
