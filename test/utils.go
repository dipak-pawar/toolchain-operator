package test

import "os"

type Environment struct {
	k string
	v string
}

func Env(k, v string) Environment {
	return Environment{k, v}
}

func SetEnv(environments ...Environment) func() {
	originalValues := make([]Environment, 0, len(environments))

	for _, env := range environments {
		originalValues = append(originalValues, Env(env.k, os.Getenv(env.k)))
		os.Setenv(env.k, env.v)
	}
	return func() {
		for _, env := range originalValues {
			os.Setenv(env.k, env.v)
		}
	}
}
