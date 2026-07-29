package main

import (
	"testing"
)

func TestParseSize_Bytes(t *testing.T) {
	result, err := parseSize("1024")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 1024 {
		t.Errorf("expected 1024, got %d", result)
	}
}

func TestParseSize_Empty(t *testing.T) {
	result, err := parseSize("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Errorf("expected 0, got %d", result)
	}
}

func TestParseSize_GiB(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"1G", 1073741824},
		{"10G", 10737418240},
		{"1GB", 1073741824},
		{"10GB", 10737418240},
		{"1g", 1073741824},
		{"10g", 10737418240},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseSize(tc.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("parseSize(%s): expected %d, got %d", tc.input, tc.expected, result)
			}
		})
	}
}

func TestParseSize_MiB(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"512M", 536870912},
		{"1024M", 1073741824},
		{"512MB", 536870912},
		{"1024mb", 1073741824},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseSize(tc.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("parseSize(%s): expected %d, got %d", tc.input, tc.expected, result)
			}
		})
	}
}

func TestParseSize_TiB(t *testing.T) {
	result, err := parseSize("1T")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 1099511627776 {
		t.Errorf("expected 1TiB, got %d", result)
	}
}

func TestParseSize_WithSpaces(t *testing.T) {
	result, err := parseSize("  10G  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 10737418240 {
		t.Errorf("expected 10GiB, got %d", result)
	}
}

func TestParseSize_Invalid(t *testing.T) {
	_, err := parseSize("abc")
	if err == nil {
		t.Error("expected error for invalid size")
	}
}

func TestParseSize_InvalidNumber(t *testing.T) {
	_, err := parseSize("abcG")
	if err == nil {
		t.Error("expected error for invalid number with suffix")
	}
}

func TestParseNetworkFlag_SingleNetwork(t *testing.T) {
	nets := parseNetworkFlag("default:e1000")
	if len(nets) != 1 {
		t.Fatalf("expected 1 network, got %d", len(nets))
	}
	if nets[0].Name != "default" {
		t.Errorf("expected name default, got %s", nets[0].Name)
	}
	if nets[0].Model != "e1000" {
		t.Errorf("expected model e1000, got %s", nets[0].Model)
	}
}

func TestParseNetworkFlag_MultipleNetworks(t *testing.T) {
	nets := parseNetworkFlag("default:e1000,mynet:virtio")
	if len(nets) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(nets))
	}
	if nets[0].Name != "default" || nets[0].Model != "e1000" {
		t.Errorf("unexpected first network: %+v", nets[0])
	}
	if nets[1].Name != "mynet" || nets[1].Model != "virtio" {
		t.Errorf("unexpected second network: %+v", nets[1])
	}
}

func TestParseNetworkFlag_DefaultModel(t *testing.T) {
	nets := parseNetworkFlag("mynet")
	if len(nets) != 1 {
		t.Fatalf("expected 1 network, got %d", len(nets))
	}
	if nets[0].Name != "mynet" {
		t.Errorf("expected name mynet, got %s", nets[0].Name)
	}
	if nets[0].Model != "e1000" {
		t.Errorf("expected default model e1000, got %s", nets[0].Model)
	}
}

func TestParseNetworkFlag_Empty(t *testing.T) {
	nets := parseNetworkFlag("")
	if nets != nil {
		t.Errorf("expected nil for empty string, got %v", nets)
	}
}

func TestParseNetworkFlag_TrailingComma(t *testing.T) {
	nets := parseNetworkFlag("default:e1000,")
	if len(nets) != 1 {
		t.Fatalf("expected 1 network (trailing comma ignored), got %d", len(nets))
	}
}

func TestParseNetworkFlag_WithSpaces(t *testing.T) {
	nets := parseNetworkFlag(" default:e1000 , mynet:virtio ")
	if len(nets) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(nets))
	}
	if nets[0].Name != "default" {
		t.Errorf("expected name default, got %s", nets[0].Name)
	}
	if nets[1].Name != "mynet" {
		t.Errorf("expected name mynet, got %s", nets[1].Name)
	}
}
