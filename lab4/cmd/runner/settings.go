package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"lab4/internal/stand"
)

type settings struct {
	dir       string
	results   string
	variant   string
	apiImage  string
	afterLoad string
	env       list
	profiles  list
}

type list []string

func (l *list) String() string {
	return strings.Join(*l, ",")
}

func (l *list) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func parseSettings(args []string) (settings, []string, error) {
	var s settings
	flags := flag.NewFlagSet("runner", flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flags.PrintDefaults()
	}
	flags.StringVar(&s.dir, "lab", ".", "каталог со stand/compose.yaml и scenarios.json")
	flags.StringVar(&s.results, "results", "", "каталог результатов, по умолчанию <lab>/results")
	flags.StringVar(&s.variant, "variant", "", "название варианта конфигурации стенда")
	flags.StringVar(&s.apiImage, "api-image", stand.APIBaseImage, "базовый образ API")
	flags.StringVar(&s.afterLoad, "after-load", "", "команда после нагрузки, до удаления стенда; каталог прогона в RUN_DIR")
	flags.Var(&s.env, "env", "переменная стенда KEY=VALUE, можно несколько раз")
	flags.Var(&s.profiles, "profile", "профиль docker compose, можно несколько раз")
	if err := flags.Parse(args); err != nil {
		return s, nil, err
	}
	for _, pair := range s.env {
		if !strings.Contains(pair, "=") {
			return s, nil, fmt.Errorf("-env %q: ожидается KEY=VALUE", pair)
		}
	}
	return s, flags.Args(), nil
}

func settingsMap(env []string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	result := make(map[string]string, len(env))
	for _, pair := range env {
		key, value, _ := strings.Cut(pair, "=")
		result[key] = value
	}
	return result
}
