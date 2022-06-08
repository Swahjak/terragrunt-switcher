package lib

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
)

const (
	installFile               = "terragrunt"
	versionPrefix             = "terragrunt_"
	installPath               = ".terragrunt.versions"
	recentFile                = "RECENT"
	defaultBin                = "/usr/local/bin/terragrunt" //default bin installation dir
	tfDarwinArm64StartVersion = "1.0.2"
)

var (
	installLocation = "/tmp"
)

// initialize : removes existing symlink to terragrunt binary// I Don't think this is needed
func initialize() {

	/* Step 1 */
	/* initilize default binary path for terragrunt */
	/* assumes that terragrunt is installed here */
	/* we will find the terragrunt path instalation later and replace this variable with the correct installed bin path */
	installedBinPath := "/usr/local/bin/terragrunt"

	/* find terragrunt binary location if terragrunt is already installed*/
	cmd := NewCommand("terragrunt")
	next := cmd.Find()

	/* overrride installation default binary path if terragrunt is already installed */
	/* find the last bin path */
	for path := next(); len(path) > 0; path = next() {
		installedBinPath = path
	}

	/* check if current symlink to terragrunt binary exist */
	symlinkExist := CheckSymlink(installedBinPath)

	/* remove current symlink if exist*/
	if symlinkExist {
		RemoveSymlink(installedBinPath)
	}

}

// GetInstallLocation : get location where the terragrunt binary will be installed,
// will create a directory in the home location if it does not exist
func GetInstallLocation() string {
	/* get current user */
	usr, errCurr := user.Current()
	if errCurr != nil {
		log.Fatal(errCurr)
	}

	userCommon := usr.HomeDir

	/* set installation location */
	installLocation = filepath.Join(userCommon, installPath)

	/* Create local installation directory if it does not exist */
	CreateDirIfNotExist(installLocation)

	return installLocation

}

//Install : Install the provided version in the argument
func Install(tgVersion string, binPath string, mirrorURL string) {

	// if !ValidVersionFormat(tgVersion) {
	// 	fmt.Printf("The provided terragrunt version format does not exist - %s. Try `tgswitch -l` to see all available versions.\n", tgVersion)
	// 	os.Exit(1)
	// }

	/* Check to see if user has permission to the default bin location which is  "/usr/local/bin/terragrunt"
	 * If user does not have permission to default bin location, proceed to create $HOME/bin and install the tgswitch there
	 * Inform user that they dont have permission to default location, therefore tgswitch was installed in $HOME/bin
	 * Tell users to add $HOME/bin to their path
	 */
	binPath = InstallableBinLocation(binPath)

	initialize()                           //initialize path
	installLocation = GetInstallLocation() //get installation location -  this is where we will put our terragrunt binary file

	goarch := runtime.GOARCH
	goos := runtime.GOOS

	// terragrunt darwin arm64 comes with 1.0.2 and next version
	tfver, _ := version.NewVersion(tgVersion)
	tf102, _ := version.NewVersion(tfDarwinArm64StartVersion)
	if goos == "darwin" && goarch == "arm64" && tfver.LessThan(tf102) {
		goarch = "amd64"
	}

	/* check if selected version already downloaded */
	installFileVersionPath := ConvertExecutableExt(filepath.Join(installLocation, versionPrefix+tgVersion))
	fileExist := CheckFileExist(installFileVersionPath)

	/* if selected version already exist, */
	if fileExist {

		/* remove current symlink if exist*/
		symlinkExist := CheckSymlink(binPath)

		if symlinkExist {
			RemoveSymlink(binPath)
		}

		/* set symlink to desired version */
		CreateSymlink(installFileVersionPath, binPath)
		fmt.Printf("Switched terragrunt to version %q \n", tgVersion)
		AddRecent(tgVersion) //add to recent file for faster lookup
		os.Exit(0)
	}

	//if does not have slash - append slash
	hasSlash := strings.HasSuffix(mirrorURL, "/")
	if !hasSlash {
		mirrorURL = fmt.Sprintf("%s/", mirrorURL)
	}

	/* if selected version already exist, */
	/* proceed to download it from the hashicorp release page */
	url := mirrorURL + "v" + tgVersion + "/" + versionPrefix + goos + "_" + goarch
	downloadedFile, errDownload := DownloadFromURL(installLocation, url)

	/* If unable to download file from url, exit(1) immediately */
	if errDownload != nil {
		fmt.Println(errDownload)
		os.Exit(1)
	}

	/* unzip the downloaded zipfile */
	errMove := MoveFile(downloadedFile, installFileVersionPath)
	if errMove != nil {
		fmt.Println("[Error] : Unable to unzip downloaded zip file")
		log.Fatal(errMove)
		os.Exit(1)
	}

	err := os.Chmod(installFileVersionPath, 0755)
	if err != nil {
		log.Println(err)
	}

	/* remove current symlink if exist*/
	symlinkExist := CheckSymlink(binPath)

	if symlinkExist {
		RemoveSymlink(binPath)
	}

	/* set symlink to desired version */
	CreateSymlink(installFileVersionPath, binPath)
	fmt.Printf("Switched terragrunt to version %q \n", tgVersion)
	AddRecent(tgVersion) //add to recent file for faster lookup
	os.Exit(0)
}

// AddRecent : add to recent file
func AddRecent(requestedVersion string) {

	installLocation = GetInstallLocation() //get installation location -  this is where we will put our terragrunt binary file
	versionFile := filepath.Join(installLocation, recentFile)

	fileExist := CheckFileExist(versionFile)
	if fileExist {
		lines, errRead := ReadLines(versionFile)

		if errRead != nil {
			fmt.Printf("[Error] : %s\n", errRead)
			return
		}

		for _, line := range lines {
			if !ValidVersionFormat(line) {
				fmt.Println("File dirty. Recreating cache file.")
				RemoveFiles(versionFile)
				CreateRecentFile(requestedVersion)
				return
			}
		}

		versionExist := VersionExist(requestedVersion, lines)

		if !versionExist {
			if len(lines) >= 3 {
				_, lines = lines[len(lines)-1], lines[:len(lines)-1]

				lines = append([]string{requestedVersion}, lines...)
				WriteLines(lines, versionFile)
			} else {
				lines = append([]string{requestedVersion}, lines...)
				WriteLines(lines, versionFile)
			}
		}

	} else {
		CreateRecentFile(requestedVersion)
	}
}

// GetRecentVersions : get recent version from file
func GetRecentVersions() ([]string, error) {

	installLocation = GetInstallLocation() //get installation location -  this is where we will put our terragrunt binary file
	versionFile := filepath.Join(installLocation, recentFile)

	fileExist := CheckFileExist(versionFile)
	if fileExist {

		lines, errRead := ReadLines(versionFile)
		outputRecent := []string{}

		if errRead != nil {
			fmt.Printf("Error: %s\n", errRead)
			return nil, errRead
		}

		for _, line := range lines {
			/* 	checks if versions in the recent file are valid.
			If any version is invalid, it will be consider dirty
			and the recent file will be removed
			*/
			if !ValidVersionFormat(line) {
				RemoveFiles(versionFile)
				return nil, errRead
			}

			/* 	output can be confusing since it displays the 3 most recent used terragrunt version
			append the string *recent to the output to make it more user friendly
			*/
			outputRecent = append(outputRecent, fmt.Sprintf("%s *recent", line))
		}

		return outputRecent, nil
	}

	return nil, nil
}

//CreateRecentFile : create a recent file
func CreateRecentFile(requestedVersion string) {

	installLocation = GetInstallLocation() //get installation location -  this is where we will put our terragrunt binary file

	WriteLines([]string{requestedVersion}, filepath.Join(installLocation, recentFile))
}

//ConvertExecutableExt : convert excutable with local OS extension
func ConvertExecutableExt(fpath string) string {
	switch runtime.GOOS {
	case "windows":
		if filepath.Ext(fpath) == ".exe" {
			return fpath
		}
		return fpath + ".exe"
	default:
		return fpath
	}
}

//InstallableBinLocation : Checks if terragrunt is installable in the location provided by the user.
//If not, create $HOME/bin. Ask users to add  $HOME/bin to $PATH and return $HOME/bin as install location
func InstallableBinLocation(userBinPath string) string {

	usr, errCurr := user.Current()
	if errCurr != nil {
		log.Fatal(errCurr)
	}

	binDir := Path(userBinPath)           //get path directory from binary path
	binPathExist := CheckDirExist(binDir) //the default is /usr/local/bin but users can provide custom bin locations

	if binPathExist == true { //if bin path exist - check if we can write to to it

		binPathWritable := false //assume bin path is not writable
		if runtime.GOOS != "windows" {
			binPathWritable = CheckDirWritable(binDir) //check if is writable on ( only works on LINUX)
		}

		// IF: "/usr/local/bin" or `custom bin path` provided by user is non-writable, (binPathWritable == false), we will attempt to install terragrunt at the ~/bin location. See ELSE
		if binPathWritable == false {

			homeBinExist := CheckDirExist(filepath.Join(usr.HomeDir, "bin")) //check to see if ~/bin exist
			if homeBinExist {                                                //if ~/bin exist, install at ~/bin/terragrunt
				fmt.Printf("Installing terragrunt at %s\n", filepath.Join(usr.HomeDir, "bin"))
				return filepath.Join(usr.HomeDir, "bin", "terragrunt")
			} else { //if ~/bin directory does not exist, create ~/bin for terragrunt installation
				fmt.Printf("Unable to write to: %s\n", userBinPath)
				fmt.Printf("Creating bin directory at: %s\n", filepath.Join(usr.HomeDir, "bin"))
				CreateDirIfNotExist(filepath.Join(usr.HomeDir, "bin")) //create ~/bin
				fmt.Printf("RUN `export PATH=$PATH:%s` to append bin to $PATH\n", filepath.Join(usr.HomeDir, "bin"))
				return filepath.Join(usr.HomeDir, "bin", "terragrunt")
			}
		} else { // ELSE: the "/usr/local/bin" or custom path provided by user is writable, we will return installable location
			return filepath.Join(userBinPath)
		}
	}
	fmt.Printf("[Error] : Binary path does not exist: %s\n", userBinPath)
	fmt.Printf("[Error] : Manually create bin directory at: %s and try again.\n", binDir)
	os.Exit(1)
	return ""
}
