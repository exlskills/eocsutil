package main

import (
	"github.com/exlskills/eocsutil/config"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	fromOLXCmd       = kingpin.Command("from-olx", "Convert an OLX course to an EOCS course")
	fromOLXForceFlag = fromOLXCmd.Flag("force", "Ignore if the destination already exists, just write the files/folders").Default("false").Bool()
	toOLXCmd         = kingpin.Command("to-olx", "Convert an EOCS course to an OLX course")
)

var Log = config.Cfg().GetLogger()

func main() {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("0.1").Author("EXL Inc.")
	kingpin.CommandLine.Help = "EOCS Util"
	switch kingpin.Parse() {
	case "from-olx":
		Log.Info("Run from-olx command")
	case "to-olx":
		Log.Info("Run to-olx command")
	default:
		Log.Fatal("Unknown command")
	}
}
