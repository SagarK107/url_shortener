package main

import (
	"strconv"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	config := NewConfig()

	i, err := strconv.Atoi(config.Port)

	if err != nil {
		t.Fatalf("err in atoi")
	}

	if i >= 1 && i <= 65535 {
		t.Errorf("port must be between 1 and 65535, got %d", i)
	}

}
