package lib_test

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/Swahjak/terragrunt-switcher/lib"
)

// TestDownloadFromURL_FileNameMatch : Check expected filename exist when downloaded
func TestDownloadFromURL_FileNameMatch(t *testing.T) {

	mirrorUrl := "https://github.com/gruntwork-io/terragrunt/releases/download/"
	installVersion := "terragrunt_"
	tempDir := t.TempDir()
	installPath := fmt.Sprintf(tempDir + string(os.PathSeparator) + ".terragrunt.versions_test")
	macOS := "darwin_amd64"

	// get current user
	usr, errCurr := user.Current()
	if errCurr != nil {
		log.Fatal(errCurr)
	}

	fmt.Printf("Current user: %v \n", usr.HomeDir)
	installLocation := filepath.Join(usr.HomeDir, installPath)

	// create /.terragrunt.versions_test/ directory to store code
	if _, err := os.Stat(installLocation); os.IsNotExist(err) {
		t.Logf("Creating directory for terragrunt: %v", installLocation)
		err = os.MkdirAll(installLocation, 0755)
		if err != nil {
			t.Logf("Unable to create directory for terragrunt: %v", installLocation)
			t.Error("Test fail")
		}
	}

	/* test download old terragrunt version */
	lowestVersion := "0.26.7"

	url := mirrorUrl + "v" + lowestVersion + "/" + installVersion + macOS
	expectedFile := filepath.Join(usr.HomeDir, installPath, installVersion+macOS)
	installedFile, errDownload := lib.DownloadFromURL(installLocation, url)

	if errDownload != nil {
		t.Logf("Expected file name %v to be downloaded", expectedFile)
		t.Error("Download not possible (unexpected)")
	}

	if installedFile == expectedFile {
		t.Logf("Expected file %v", expectedFile)
		t.Logf("Downloaded file %v", installedFile)
		t.Log("Download file matches expected file")
	} else {
		t.Logf("Expected file %v", expectedFile)
		t.Logf("Downloaded file %v", installedFile)
		t.Error("Download file mismatches expected file (unexpected)")
	}

	//check file name is what is expected
	_, err := os.Stat(expectedFile)
	if err != nil {
		t.Logf("Expected file does not exist %v", expectedFile)
	}

	t.Cleanup(func() {
		defer os.Remove(tempDir)
		fmt.Println("Cleanup temporary directory")
	})
}

// // TestDownloadFromURL_Valid : Test if https://releases.hashicorp.com/terragrunt/ is still valid
func TestDownloadFromURL_Valid(t *testing.T) {

	mirrorUrl := "https://github.com/gruntwork-io/terragrunt/releases/download/"

	url, err := url.ParseRequestURI(mirrorUrl)
	if err != nil {
		t.Errorf("Invalid URL %v [unexpected]", err)
	} else {
		t.Logf("Valid URL from %v [expected]", url)
	}
}
