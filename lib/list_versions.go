package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type tgVersionList struct {
	tgList []string
}

//GetTGList :  Get the list of available terraform version given the hashicorp url
func GetTGList(versionUrl string, preRelease bool) ([]string, error) {

	var tgVersionList tgVersionList
	result, error := GetTGURLBody(versionUrl)

	if error != nil {
		log.Println(error)
		os.Exit(1)

		return tgVersionList.tgList, error
	}

	var semver string
	if preRelease == true {
		// Getting versions from body; should return match /X.X.X-@/ where X is a number,@ is a word character between a-z or A-Z
		semver = `^(\d+\.\d+\.\d+)(-[a-zA-z]+\d*)?$`
	} else if preRelease == false {
		// Getting versions from body; should return match /X.X.X/ where X is a number
		semver = `^(\d+\.\d+\.\d+)$`
	}

	r, _ := regexp.Compile(semver)
	for i := range result {
		if r.MatchString(result[i]) {
			str := r.FindString(result[i])
			trimstr := strings.Trim(str, "/\"") //remove '/' or '"' from /X.X.X/" or /X.X.X"
			tgVersionList.tgList = append(tgVersionList.tgList, trimstr)
		}
	}

	if len(tgVersionList.tgList) == 0 {
		fmt.Printf("Cannot get list from mirror: %s\n", versionUrl)
	}

	return tgVersionList.tgList, nil

}

//GetTGLatest :  Get the latest terraform version given the hashicorp url
func GetTGLatest(versionUrl string) (string, error) {

	result, error := GetTGURLBody(versionUrl)
	if error != nil {
		return "", error
	}
	// Getting versions from body; should return match /X.X.X/ where X is a number
	semver := `^(\d+\.\d+\.\d+)$`
	r, _ := regexp.Compile(semver)
	for i := range result {
		if r.MatchString(result[i]) {
			str := r.FindString(result[i])
			return str, nil
		}
	}

	return "", nil
}

//GetTGLatestImplicit :  Get the latest implicit terraform version given the hashicorp url
func GetTGLatestImplicit(versionUrl string, preRelease bool, version string) (string, error) {

	if preRelease == true {
		//TODO: use GetTGList() instead of GetTGURLBody
		versions, error := GetTGURLBody(versionUrl)
		if error != nil {
			return "", error
		}
		// Getting versions from body; should return match /X.X.X-@/ where X is a number,@ is a word character between a-z or A-Z
		semver := fmt.Sprintf(`^(%s{1}\.\d+\-[a-zA-z]+\d*)$`, version)
		r, err := regexp.Compile(semver)
		if err != nil {
			return "", err
		}
		for i := range versions {
			if r.MatchString(versions[i]) {
				str := r.FindString(versions[i])
				return str, nil
			}
		}
	} else if preRelease == false {
		listAll := false
		tgList, _ := GetTGList(versionUrl, listAll) //get list of versions
		version = fmt.Sprintf("~> %v", version)
		semv, err := SemVerParser(&version, tgList)
		if err != nil {
			return "", err
		}
		return semv, nil
	}
	return "", nil
}

//GetTGURLBody : Get list of terraform versions from hashicorp releases
func GetTGURLBody(versionUrl string) ([]string, error) {
	gswitch := http.Client{
		Timeout: time.Second * 10, // Maximum of 10 secs [decresing this seem to fail]
	}

	req, err := http.NewRequest(http.MethodGet, versionUrl, nil)
	if err != nil {
		log.Println("[Error] Unable to make request. Please try again.")
		os.Exit(1)
		return nil, err
	}

	req.Header.Set("User-Agent", "github-appinstaller")

	res, getErr := gswitch.Do(req)
	if getErr != nil {
		log.Println("[Error] Unable to make request Please try again.")
		os.Exit(1)
		return nil, getErr
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Println("[Error] Unable to get release from repo ", string(body))
		os.Exit(1)
		return nil, readErr
	}

	var repo ListVersion
	jsonErr := json.Unmarshal(body, &repo)
	if jsonErr != nil {
		log.Println("[Error] Unable to get release from repo ", string(body))
		os.Exit(1)
		return nil, jsonErr
	}

	return repo.Versions, nil
}

type ListVersion struct {
	Versions []string `json:"Versions"`
}

//VersionExist : check if requested version exist
func VersionExist(val interface{}, array interface{}) (exists bool) {

	exists = false
	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(val, s.Index(i).Interface()) == true {
				exists = true
				return exists
			}
		}
	}

	return exists
}

//RemoveDuplicateVersions : remove duplicate version
func RemoveDuplicateVersions(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for _, val := range elements {
		versionOnly := strings.Trim(val, " *recent")
		if encountered[versionOnly] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[versionOnly] = true
			// Append to result slice.
			result = append(result, val)
		}
	}
	// Return the new slice.
	return result
}

// ValidVersionFormat : returns valid version format
/* For example: 0.1.2 = valid
// For example: 0.1.2-beta1 = valid
// For example: 0.1.2-alpha = valid
// For example: a.1.2 = invalid
// For example: 0.1. 2 = invalid
*/
func ValidVersionFormat(version string) bool {

	// Getting versions from body; should return match /X.X.X-@/ where X is a number,@ is a word character between a-z or A-Z
	// Follow https://semver.org/spec/v1.0.0-beta.html
	// Check regular expression at https://rubular.com/r/ju3PxbaSBALpJB
	semverRegex := regexp.MustCompile(`^(\d+\.\d+\.\d+)(-[a-zA-z]+\d*)?$`)

	return semverRegex.MatchString(version)
}

// ValidMinorVersionFormat : returns valid MINOR version format
/* For example: 0.1 = valid
// For example: a.1.2 = invalid
// For example: 0.1.2 = invalid
*/
func ValidMinorVersionFormat(version string) bool {

	// Getting versions from body; should return match /X.X./ where X is a number
	semverRegex := regexp.MustCompile(`^(\d+\.\d+)$`)

	return semverRegex.MatchString(version)
}
