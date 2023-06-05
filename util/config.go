package util

import "github.com/aichy126/igo"

func ConfGetbool(path string) bool {
	return igo.App.Conf.GetBool(path)
}
func ConfGetString(path string) string {
	return igo.App.Conf.GetString(path)
}
func ConfGetStringSlice(path string) []string {
	return igo.App.Conf.GetStringSlice(path)
}
func ConfGetInt(path string) int {
	return igo.App.Conf.GetInt(path)
}
func ConfGetInt64(path string) int64 {
	return igo.App.Conf.GetInt64(path)
}
