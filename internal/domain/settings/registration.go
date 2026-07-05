package settings

import "time"

type Registration func(*SettingsV3) error

func AString(name, defaultValue string) Registration {
	return func(s *SettingsV3) error {
		return s.register(name, StringType, defaultValue)
	}
}

func AUint64(name string, defaultValue uint64) Registration {
	return func(s *SettingsV3) error {
		return s.register(name, Uint64Type, defaultValue)
	}
}

func ADuration(name string, defaultValue time.Duration) Registration {
	return func(s *SettingsV3) error {
		return s.register(name, DurationType, defaultValue)
	}
}
