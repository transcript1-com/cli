package config

func ResolveKey(getenv func(string) string) (string, error) {
	if key := getenv(EnvAPIKey); key != "" {
		return key, nil
	}

	return LoadKey()
}
