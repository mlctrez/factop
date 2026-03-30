package surface

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseList parses the surface-list wire format: name:index,name:index,...
func ParseList(s string) ([]SurfaceEntry, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	entries := strings.Split(s, ",")
	result := make([]SurfaceEntry, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid surface entry %q", entry)
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid index in %q: %w", entry, err)
		}
		result = append(result, SurfaceEntry{Name: parts[0], Index: idx})
	}
	return result, nil
}

// ParseInfo parses the surface-info wire format: key:value,key:value,...
func ParseInfo(s string) (*Info, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty surface info")
	}
	info := &Info{}
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		switch key {
		case "name":
			info.Name = val
		case "index":
			info.Index, _ = strconv.Atoi(val)
		case "always_day":
			info.AlwaysDay = val == "true"
		case "daytime":
			info.Daytime, _ = strconv.ParseFloat(val, 64)
		case "darkness":
			info.Darkness, _ = strconv.ParseFloat(val, 64)
		case "freeze_daytime":
			info.FreezeDaytime = val == "true"
		case "peaceful_mode":
			info.PeacefulMode = val == "true"
		case "no_enemies_mode":
			info.NoEnemiesMode = val == "true"
		case "wind_speed":
			info.WindSpeed, _ = strconv.ParseFloat(val, 64)
		case "wind_orientation":
			info.WindOrientation, _ = strconv.ParseFloat(val, 64)
		case "solar_power_multiplier":
			info.SolarPowerMultiplier, _ = strconv.ParseFloat(val, 64)
		case "min_brightness":
			info.MinBrightness, _ = strconv.ParseFloat(val, 64)
		case "ticks_per_day":
			info.TicksPerDay, _ = strconv.Atoi(val)
		case "show_clouds":
			info.ShowClouds = val == "true"
		case "generate_with_lab_tiles":
			info.GenerateWithLabTiles = val == "true"
		}
	}
	return info, nil
}
