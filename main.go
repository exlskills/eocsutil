package main

import (
	"fmt"
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocs"
	"github.com/exlskills/eocsutil/eocsuri"
	"github.com/exlskills/eocsutil/extfmt"
	"github.com/exlskills/eocsutil/ghserver"
	"github.com/exlskills/eocsutil/gitutils"
	"github.com/exlskills/eocsutil/mdutils"
	"github.com/exlskills/eocsutil/olx"
	"gopkg.in/alecthomas/kingpin.v2"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	serveGitHubHooks  = kingpin.Command("serve-gh-hook", "Serve GitHub course update hooks to auto-update courses")
	convertCmd        = kingpin.Command("convert", "Convert a course from one supported format to another")
	convertForce      = convertCmd.Flag("force", "Ignore if the destination already exists, just write the files/folders").Default("false").Bool()
	convertFromFormat = convertCmd.Flag("from-format", "The source format to convert from").Required().String()
	convertFromURI    = convertCmd.Flag("from-uri", "The URI to the source").Required().String()
	convertToFormat   = convertCmd.Flag("to-format", "The destination format to convert to").Required().String()
	convertToURI      = convertCmd.Flag("to-uri", "The destination URI").Required().String()
	verifyCmd         = kingpin.Command("verify", "Check that a course conforms to a supported format")
	verifyFormat      = verifyCmd.Flag("format", "The format to which the course should conform to").Default("eocs").String()
	verifyURI         = verifyCmd.Flag("uri", "The URI of the source of the course").Required().String()
)

var Log = config.Cfg().GetLogger()

func init() {
	extfmt.RegisterExtFmt("eocs", eocs.NewEOCSFormat())
	extfmt.RegisterExtFmt("olx", olx.NewOLXExtFmt())
}

func main() {
	// Do this to ensure that our on exit traps work
	run()
	time.Sleep(time.Second * 1)
}

func run() {
	defer mdutils.GracefulTeardown()
	go gracefulShutdown()
	rand.Seed(time.Now().UnixNano())
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("0.1").Author("EXL Inc.")
	kingpin.CommandLine.Help = "EXL Open Courseware Standard - Utilities"
	switch kingpin.Parse() {
	case "convert":
		if strings.HasPrefix(*convertToURI, "mongodb://") {
			err := eocs.NewEOCSFormat().Push(*convertFromURI, *convertToURI)
			if err != nil {
				Log.Errorf("Course push failed: %s", err.Error())
				return
			}
			return
		}

		// This is non-MongoDB only flow below
		Log.Info("Importing course for conversion ...")
		ir, err := getExtFmtF(*convertFromFormat).Import(verifyAndCleanURIF(*convertFromURI))
		if err != nil {
			Log.Errorf("Course import failed with: %s", err.Error())
			return
		}
		Log.Info("Successfully imported course %s for conversion, now exporting ...", ir.GetDisplayName())

		err = gitutils.SetCourseComponentsTimestamps(*convertFromURI, ir)
		if err != nil {
			Log.Errorf("Git reader failed with: %s", err.Error())
			return
		}


		err = getExtFmtF(*convertToFormat).Export(ir, verifyAndCleanURIF(*convertToURI), *convertForce)
		if err != nil {
			Log.Errorf("Course export failed with: %s", err.Error())
			return
		}
		Log.Infof("Successfully exported course: %s", ir.GetDisplayName())
	case "verify":
		Log.Info("Importing course for verification ...")
		ir, err := getExtFmtF(*verifyFormat).Import(verifyAndCleanURIF(*verifyURI))
		if err != nil {
			Log.Errorf("Course import verification failed with: %s", err.Error())
			return
		}
		Log.Infof("Successfully verified course: %s", ir.GetDisplayName())
		return
	case "serve-gh-hook":
		Log.Info("Serve GitHub Hooks ...")
		ghserver.ServeGH()
		return
	default:
		Log.Fatal("Unknown command")
	}
}

func getExtFmtF(key string) extfmt.ExtFmt {
	impl := extfmt.GetImplementation(key)
	if impl == nil {
		Log.Fatalf(fmt.Sprintf("invalid format type: %s", key))
	}
	return impl
}

func verifyAndCleanURIF(uri string) string {
	var err error
	uri, err = eocsuri.VerifyAndClean(uri)
	if err != nil {
		Log.Fatalf("invalid uri: %s", err.Error())
	}
	return uri
}

func gracefulShutdown() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	go func() {
		<-s
		fmt.Println("Shutting down gracefully.")
		mdutils.GracefulTeardown()
		os.Exit(0)
	}()
}
