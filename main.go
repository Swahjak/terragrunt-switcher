package main

/*
* Version 0.12.0
* Compatible with Mac OS X AND other LINUX OS ONLY
 */

/*** OPERATION WORKFLOW ***/
/*
* 1- Create /usr/local/terragrunt directory if does not exist
* 2- Download zip file from url to /usr/local/terragrunt
* 3- MoveFile the file to /usr/local/terragrunt
* 4- Rename the file from `terragrunt` to `terragrunt_version`
* 5- Remove the downloaded zip file
* 6- Read the existing symlink for terragrunt (Check if it's a homebrew symlink)
* 7- Remove that symlink (Check if it's a homebrew symlink)
* 8- Create new symlink to binary  `terragrunt_version`
 */

import (
	"fmt"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	semver "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/manifoldco/promptui"
	"github.com/pborman/getopt"
	"github.com/spf13/viper"

	lib "github.com/Swahjak/terragrunt-switcher/lib"
)

const (
	defaultMirror  = "https://github.com/gruntwork-io/terragrunt/releases/download/"
	defaultVersion = "https://warrensbox.github.io/terragunt-versions-list/index.json"
	defaultBin     = "/usr/local/bin/terragrunt" //default bin installation dir
	defaultLatest  = ""
	tgvFilename    = ".terragrunt-version"
	rcFilename     = ".tgswitchrc"
	tomlFilename   = ".tgswitch.toml"
	tgHclFilename  = "terragrunt.hcl"
	versionPrefix  = "terragrunt_"
)

var version = "0.12.0\n"

func main() {
	dir := lib.GetCurrentDirectory()
	custBinPath := getopt.StringLong("bin", 'b', lib.ConvertExecutableExt(defaultBin), "Custom binary path. Ex: tgswitch -b "+lib.ConvertExecutableExt("/Users/username/bin/terragrunt"))
	listAllFlag := getopt.BoolLong("list-all", 'l', "List all versions of terragrunt - including beta and rc")
	latestPre := getopt.StringLong("latest-pre", 'p', defaultLatest, "Latest pre-release implicit version. Ex: tgswitch --latest-pre 0.13 downloads 0.13.0-rc1 (latest)")
	showLatestPre := getopt.StringLong("show-latest-pre", 'P', defaultLatest, "Show latest pre-release implicit version. Ex: tgswitch --show-latest-pre 0.13 prints 0.13.0-rc1 (latest)")
	latestStable := getopt.StringLong("latest-stable", 's', defaultLatest, "Latest implicit version. Ex: tgswitch --latest-stable 0.13 downloads 0.13.7 (latest)")
	showLatestStable := getopt.StringLong("show-latest-stable", 'S', defaultLatest, "Show latest implicit version. Ex: tgswitch --show-latest-stable 0.13 prints 0.13.7 (latest)")
	latestFlag := getopt.BoolLong("latest", 'u', "Get latest stable version")
	showLatestFlag := getopt.BoolLong("show-latest", 'U', "Show latest stable version")
	mirrorURL := getopt.StringLong("mirror", 'm', defaultMirror, "Install from a remote API other than the default. Default: "+defaultMirror)
	versionURL := getopt.StringLong("version_url", 'z', defaultVersion, "List from a remote API other than the default. Default: "+defaultVersion)
	chDirPath := getopt.StringLong("chdir", 'c', dir, "Switch to a different working directory before executing the given command. Ex: tgswitch --chdir terragrunt_project will run tgswitch in the terragrunt_project directory")
	versionFlag := getopt.BoolLong("version", 'v', "Displays the version of tgswitch")
	helpFlag := getopt.BoolLong("help", 'h', "Displays help message")
	_ = versionFlag

	getopt.Parse()
	args := getopt.Args()

	homedir := lib.GetHomeDirectory()

	TGVersionFile := filepath.Join(*chDirPath, tgvFilename)    //settings for .terragrunt-version file in current directory (tgenv compatible)
	RCFile := filepath.Join(*chDirPath, rcFilename)            //settings for .tgswitchrc file in current directory (backward compatible purpose)
	TOMLConfigFile := filepath.Join(*chDirPath, tomlFilename)  //settings for .tgswitch.toml file in current directory (option to specify bin directory)
	HomeTOMLConfigFile := filepath.Join(homedir, tomlFilename) //settings for .tgswitch.toml file in home directory (option to specify bin directory)
	TGHACLFile := filepath.Join(*chDirPath, tgHclFilename)     //settings for terragrunt.hcl file in current directory (option to specify bin directory)

	switch {
	case *versionFlag:
		//if *versionFlag {
		fmt.Printf("\nVersion: %v\n", version)
	case *helpFlag:
		//} else if *helpFlag {
		usageMessage()
	/* Checks if the .tgswitch.toml file exist in home or current directory
	 * This block checks to see if the tgswitch toml file is provided in the current path.
	 * If the .tgswitch.toml file exist, it has a higher precedence than the .tgswitchrc file
	 * You can specify the custom binary path and the version you desire
	 * If you provide a custom binary path with the -b option, this will override the bin value in the toml file
	 * If you provide a version on the command line, this will override the version value in the toml file
	 */
	case fileExists(TOMLConfigFile) || fileExists(HomeTOMLConfigFile):
		version := ""
		binPath := *custBinPath
		if fileExists(TOMLConfigFile) { //read from toml from current directory
			version, binPath = getParamsTOML(binPath, *chDirPath)
		} else { // else read from toml from home directory
			version, binPath = getParamsTOML(binPath, homedir)
		}

		switch {
		/* GIVEN A TOML FILE, */
		/* show all terragrunt version including betas and RCs*/
		case *listAllFlag:
			listAll := true //set list all true - all versions including beta and rc will be displayed
			installOption(listAll, &binPath, mirrorURL, versionURL)
		/* latest pre-release implicit version. Ex: tgswitch --latest-pre 0.13 downloads 0.13.0-rc1 (latest) */
		case *latestPre != "":
			preRelease := true
			installLatestImplicitVersion(*latestPre, custBinPath, mirrorURL, versionURL, preRelease)
		/* latest implicit version. Ex: tgswitch --latest 0.13 downloads 0.13.5 (latest) */
		case *latestStable != "":
			preRelease := false
			installLatestImplicitVersion(*latestStable, custBinPath, mirrorURL, versionURL, preRelease)
		/* latest stable version */
		case *latestFlag:
			installLatestVersion(custBinPath, mirrorURL, versionURL)
		/* version provided on command line as arg */
		case len(args) == 1:
			installVersion(args[0], &binPath, mirrorURL, versionURL)
		/* provide an tgswitchrc file (IN ADDITION TO A TOML FILE) */
		case fileExists(RCFile) && len(args) == 0:
			readingFileMsg(rcFilename)
			tgversion := retrieveFileContents(RCFile)
			installVersion(tgversion, &binPath, mirrorURL, versionURL)
		/* if .terragrunt-version file found (IN ADDITION TO A TOML FILE) */
		case fileExists(TGVersionFile) && len(args) == 0:
			readingFileMsg(tgvFilename)
			tgversion := retrieveFileContents(TGVersionFile)
			installVersion(tgversion, &binPath, mirrorURL, versionURL)
		/* if versions.tg file found (IN ADDITION TO A TOML FILE) */
		case checkTGModuleFileExist(*chDirPath) && len(args) == 0:
			installTGProvidedModule(*chDirPath, &binPath, mirrorURL, versionURL)
		/* if Terragrunt Version environment variable is set */
		case checkTGEnvExist() && len(args) == 0 && version == "":
			tgversion := os.Getenv("TF_VERSION")
			fmt.Printf("Terragrunt version environment variable: %s\n", tgversion)
			installVersion(tgversion, custBinPath, mirrorURL, versionURL)
		/* if terragrunt.hcl file found (IN ADDITION TO A TOML FILE) */
		case fileExists(TGHACLFile) && checkVersionDefinedHCL(&TGHACLFile) && len(args) == 0:
			installTGHclFile(&TGHACLFile, &binPath, mirrorURL, versionURL)
		// if no arg is provided - but toml file is provided
		case version != "":
			installVersion(version, &binPath, mirrorURL, versionURL)
		default:
			listAll := false //set list all false - only official release will be displayed
			installOption(listAll, &binPath, mirrorURL, versionURL)
		}

	/* show all terragrunt version including betas and RCs*/
	case *listAllFlag:
		installWithListAll(custBinPath, mirrorURL, versionURL)

	/* latest pre-release implicit version. Ex: tgswitch --latest-pre 0.13 downloads 0.13.0-rc1 (latest) */
	case *latestPre != "":
		preRelease := true
		installLatestImplicitVersion(*latestPre, custBinPath, mirrorURL, versionURL, preRelease)

	/* show latest pre-release implicit version. Ex: tgswitch --latest-pre 0.13 downloads 0.13.0-rc1 (latest) */
	case *showLatestPre != "":
		preRelease := true
		showLatestImplicitVersion(*showLatestPre, custBinPath, mirrorURL, versionURL, preRelease)

	/* latest implicit version. Ex: tgswitch --latest 0.13 downloads 0.13.5 (latest) */
	case *latestStable != "":
		preRelease := false
		installLatestImplicitVersion(*latestStable, custBinPath, mirrorURL, versionURL, preRelease)

	/* show latest implicit stable version. Ex: tgswitch --latest 0.13 downloads 0.13.5 (latest) */
	case *showLatestStable != "":
		preRelease := false
		showLatestImplicitVersion(*showLatestStable, custBinPath, mirrorURL, versionURL, preRelease)

	/* latest stable version */
	case *latestFlag:
		installLatestVersion(custBinPath, mirrorURL, versionURL)

	/* show latest stable version */
	case *showLatestFlag:
		showLatestVersion(custBinPath, versionURL)

	/* version provided on command line as arg */
	case len(args) == 1:
		installVersion(args[0], custBinPath, mirrorURL, versionURL)

	/* provide an tgswitchrc file */
	case fileExists(RCFile) && len(args) == 0:
		readingFileMsg(rcFilename)
		tgversion := retrieveFileContents(RCFile)
		installVersion(tgversion, custBinPath, mirrorURL, versionURL)

	/* if .terragrunt-version file found */
	case fileExists(TGVersionFile) && len(args) == 0:
		readingFileMsg(tgvFilename)
		tgversion := retrieveFileContents(TGVersionFile)
		installVersion(tgversion, custBinPath, mirrorURL, versionURL)

	/* if versions.tg file found */
	case checkTGModuleFileExist(*chDirPath) && len(args) == 0:
		installTGProvidedModule(*chDirPath, custBinPath, mirrorURL, versionURL)

	/* if terragrunt.hcl file found */
	case fileExists(TGHACLFile) && checkVersionDefinedHCL(&TGHACLFile) && len(args) == 0:
		installTGHclFile(&TGHACLFile, custBinPath, mirrorURL, versionURL)

	/* if Terragrunt Version environment variable is set */
	case checkTGEnvExist() && len(args) == 0:
		tgversion := os.Getenv("TG_VERSION")
		fmt.Printf("Terragrunt version environment variable: %s\n", tgversion)
		installVersion(tgversion, custBinPath, mirrorURL, versionURL)

	// if no arg is provided
	default:
		listAll := false //set list all false - only official release will be displayed
		installOption(listAll, custBinPath, mirrorURL, versionURL)
	}
}

/* Helper functions */

// install with all possible versions, including beta and rc
func installWithListAll(custBinPath, mirrorURL *string, versionURL *string) {
	listAll := true //set list all true - all versions including beta and rc will be displayed
	installOption(listAll, custBinPath, mirrorURL, versionURL)
}

// install latest stable tg version
func installLatestVersion(custBinPath, mirrorURL *string, versionURL *string) {
	tgversion, _ := lib.GetTGLatest(*versionURL)
	lib.Install(tgversion, *custBinPath, *mirrorURL)
}

// show install latest stable tg version
func showLatestVersion(custBinPath, versionURL *string) {
	tgversion, _ := lib.GetTGLatest(*versionURL)
	fmt.Printf("%s\n", tgversion)
}

// install latest - argument (version) must be provided
func installLatestImplicitVersion(requestedVersion string, custBinPath, mirrorURL *string, versionURL *string, preRelease bool) {
	_, err := semver.NewConstraint(requestedVersion)
	if err != nil {
		fmt.Printf("error parsing constraint: %s\n", err)
	}
	//if lib.ValidMinorVersionFormat(requestedVersion) {
	tgversion, err := lib.GetTGLatestImplicit(*versionURL, preRelease, requestedVersion)
	if err == nil && tgversion != "" {
		lib.Install(tgversion, *custBinPath, *mirrorURL)
	}
	fmt.Printf("Error parsing constraint: %s\n", err)
	lib.PrintInvalidMinorTGVersion()
}

// show latest - argument (version) must be provided
func showLatestImplicitVersion(requestedVersion string, custBinPath, mirrorURL *string, versionURL *string, preRelease bool) {
	if lib.ValidMinorVersionFormat(requestedVersion) {
		tgversion, _ := lib.GetTGLatestImplicit(*versionURL, preRelease, requestedVersion)
		if len(tgversion) > 0 {
			fmt.Printf("%s\n", tgversion)
		} else {
			fmt.Println("The provided terragrunt version does not exist. Try `tgswitch -l` to see all available versions.")
			os.Exit(1)
		}
	} else {
		lib.PrintInvalidMinorTGVersion()
	}
}

// install with provided version as argument
func installVersion(arg string, custBinPath *string, mirrorURL *string, versionURL *string) {
	if lib.ValidVersionFormat(arg) {
		requestedVersion := arg

		//check to see if the requested version has been downloaded before
		installLocation := lib.GetInstallLocation()
		installFileVersionPath := lib.ConvertExecutableExt(filepath.Join(installLocation, versionPrefix+requestedVersion))
		recentDownloadFile := lib.CheckFileExist(installFileVersionPath)
		if recentDownloadFile {
			lib.ChangeSymlink(installFileVersionPath, *custBinPath)
			fmt.Printf("Switched terragrunt to version %q \n", requestedVersion)
			lib.AddRecent(requestedVersion) //add to recent file for faster lookup
			os.Exit(0)
		}

		//if the requested version had not been downloaded before
		listAll := true                                     //set list all true - all versions including beta and rc will be displayed
		tglist, _ := lib.GetTGList(*versionURL, listAll)    //get list of versions
		exist := lib.VersionExist(requestedVersion, tglist) //check if version exist before downloading it

		if exist {
			lib.Install(requestedVersion, *custBinPath, *mirrorURL)
		} else {
			fmt.Println("The provided terragrunt version does not exist. Try `tgswitch -l` to see all available versions.")
			os.Exit(1)
		}

	} else {
		lib.PrintInvalidTGVersion()
		fmt.Println("Args must be a valid terragrunt version")
		usageMessage()
		os.Exit(1)
	}
}

//retrive file content of regular file
func retrieveFileContents(file string) string {
	fileContents, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Failed to read %s file. Follow the README.md instructions for setup. https://github.com/Swahjak/terragrunt-switcher/blob/master/README.md\n", file)
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	tgversion := strings.TrimSuffix(string(fileContents), "\n")
	return tgversion
}

// Print message reading file content of :
func readingFileMsg(filename string) {
	fmt.Printf("Reading file %s \n", filename)
}

// fileExists checks if a file exists and is not a directory before we try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func checkTGModuleFileExist(dir string) bool {

	module, _ := tfconfig.LoadModule(dir)
	if len(module.RequiredCore) >= 1 {
		return true
	}
	return false
}

// checkTGEnvExist - checks if the TG_VERSION environment variable is set
func checkTGEnvExist() bool {
	tgversion := os.Getenv("TG_VERSION")
	if tgversion != "" {
		return true
	}
	return false
}

/* parses everything in the toml file, return required version and bin path */
func getParamsTOML(binPath string, dir string) (string, string) {
	path := lib.GetHomeDirectory()
	if dir == path {
		path = "home directory"
	} else {
		path = "current directory"
	}
	fmt.Printf("Reading configuration from %s\n", path+" for "+tomlFilename) //takes the default bin (defaultBin) if user does not specify bin path
	configfileName := lib.GetFileName(tomlFilename)                          //get the config file
	viper.SetConfigType("toml")
	viper.SetConfigName(configfileName)
	viper.AddConfigPath(dir)

	errs := viper.ReadInConfig() // Find and read the config file
	if errs != nil {
		fmt.Printf("Unable to read %s provided\n", tomlFilename) // Handle errors reading the config file
		fmt.Println(errs)
		os.Exit(1) // exit immediately if config file provided but it is unable to read it
	}

	bin := viper.Get("bin")                                            // read custom binary location
	if binPath == lib.ConvertExecutableExt(defaultBin) && bin != nil { // if the bin path is the same as the default binary path and if the custom binary is provided in the toml file (use it)
		binPath = os.ExpandEnv(bin.(string))
	}
	//fmt.Println(binPath) //uncomment this to debug
	version := viper.Get("version") //attempt to get the version if it's provided in the toml
	if version == nil {
		version = ""
	}

	return version.(string), binPath
}

func usageMessage() {
	fmt.Print("\n\n")
	getopt.PrintUsage(os.Stderr)
	fmt.Println("Supply the terragrunt version as an argument, or choose from a menu")
}

/* installOption : displays & installs tg version */
/* listAll = true - all versions including beta and rc will be displayed */
/* listAll = false - only official stable release are displayed */
func installOption(listAll bool, custBinPath, mirrorURL *string, versionURL *string) {
	tglist, _ := lib.GetTGList(*versionURL, listAll) //get list of versions
	recentVersions, _ := lib.GetRecentVersions()     //get recent versions from RECENT file
	tglist = append(recentVersions, tglist...)       //append recent versions to the top of the list
	tglist = lib.RemoveDuplicateVersions(tglist)     //remove duplicate version

	if len(tglist) == 0 {
		fmt.Println("[ERROR] : List is empty")
		os.Exit(1)
	}
	/* prompt user to select version of terragrunt */
	prompt := promptui.Select{
		Label: "Select Terragrunt version",
		Items: tglist,
	}

	_, tgversion, errPrompt := prompt.Run()
	tgversion = strings.Trim(tgversion, " *recent") //trim versions with the string " *recent" appended

	if errPrompt != nil {
		log.Printf("Prompt failed %v\n", errPrompt)
		os.Exit(1)
	}

	lib.Install(tgversion, *custBinPath, *mirrorURL)
	os.Exit(0)
}

// install when tf file is provided
func installTGProvidedModule(dir string, custBinPath, mirrorURL *string, versionURL *string) {
	fmt.Printf("Reading required version from terragrunt file\n")
	module, _ := tfconfig.LoadModule(dir)
	tgconstraint := module.RequiredCore[0] //we skip duplicated definitions and use only first one
	installFromConstraint(&tgconstraint, custBinPath, mirrorURL, versionURL)
}

// install using a version constraint
func installFromConstraint(tgconstraint *string, custBinPath, mirrorURL *string, versionURL *string) {

	tgversion, err := lib.GetSemver(tgconstraint, versionURL)
	if err == nil {
		lib.Install(tgversion, *custBinPath, *mirrorURL)
	}
	fmt.Println(err)
	fmt.Println("No version found to match constraint. Follow the README.md instructions for setup. https://github.com/Swahjak/terragrunt-switcher/blob/master/README.md")
	os.Exit(1)
}

// Install using version constraint from terragrunt file
func installTGHclFile(tgFile *string, custBinPath, mirrorURL *string, versionURL *string) {
	fmt.Printf("Terragrunt file found: %s\n", *tgFile)
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(*tgFile) //use hcl parser to parse HCL file
	if diags.HasErrors() {
		fmt.Println("Unable to parse HCL file")
		os.Exit(1)
	}
	var version terragruntVersionConstraints
	gohcl.DecodeBody(file.Body, nil, &version)
	installFromConstraint(&version.TerragruntVersionConstraint, custBinPath, mirrorURL, versionURL)
}

type terragruntVersionConstraints struct {
	TerragruntVersionConstraint string `hcl:"terragrunt_version_constraint"`
}

// check if version is defined in hcl file /* lazy-emergency fix - will improve later */
func checkVersionDefinedHCL(tgFile *string) bool {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(*tgFile) //use hcl parser to parse HCL file
	if diags.HasErrors() {
		fmt.Println("Unable to parse HCL file")
		os.Exit(1)
	}
	var version terragruntVersionConstraints
	gohcl.DecodeBody(file.Body, nil, &version)
	if version == (terragruntVersionConstraints{}) {
		return false
	}
	return true
}
