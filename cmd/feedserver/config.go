package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type serverConfig struct {
	addr         string
	feedsPath    string
	refresh      time.Duration
	fetchTimeout time.Duration
	workers      int
	logFormat    string
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return d, nil
}

func envInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return n, nil
}

func loadConfig() (serverConfig, error) {
	refreshDefault, err := envDuration("FEEDSTACK_REFRESH", 15*time.Minute)
	if err != nil {
		return serverConfig{}, err
	}
	timeoutDefault, err := envDuration("FEEDSTACK_FETCH_TIMEOUT", 5*time.Second)
	if err != nil {
		return serverConfig{}, err
	}
	workersDefault, err := envInt("FEEDSTACK_WORKERS", 4)
	if err != nil {
		return serverConfig{}, err
	}

	var cfg serverConfig
	flag.StringVar(&cfg.addr, "addr", envOr("FEEDSTACK_ADDR", ":8080"), "listen address")
	flag.StringVar(&cfg.feedsPath, "feeds", envOr("FEEDSTACK_FEEDS", "feeds.txt"), "path to the feed list")
	flag.DurationVar(&cfg.refresh, "refresh", refreshDefault, "time between refreshes")
	flag.DurationVar(&cfg.fetchTimeout, "fetch-timeout", timeoutDefault, "per-fetch timeout")
	flag.IntVar(&cfg.workers, "workers", workersDefault, "fetch worker count")
	flag.StringVar(&cfg.logFormat, "log-format", envOr("FEEDSTACK_LOG_FORMAT", "text"), "log format: text or json")
	flag.Parse()

	if cfg.workers < 1 {
		return serverConfig{}, fmt.Errorf("workers must be at least 1, got %d", cfg.workers)
	}
	if cfg.refresh < time.Second {
		return serverConfig{}, fmt.Errorf("refresh interval %s is below one second", cfg.refresh)
	}
	if cfg.logFormat != "text" && cfg.logFormat != "json" {
		return serverConfig{}, fmt.Errorf("log-format must be text or json, got %q", cfg.logFormat)
	}
	return cfg, nil
}
