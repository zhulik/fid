package main

type Config struct {
	Functions map[string]*Function `yaml:"functions"`
}
