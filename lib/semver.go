package lib

import (
	"fmt"
	"sort"

	semver "github.com/hashicorp/go-version"
)

// GetSemver : returns version that will be installed based on server constaint provided
func GetSemver(tgConstraint *string, mirrorURL *string) (string, error) {

	listAll := true
	tflist, _ := GetTGList(*mirrorURL, listAll) //get list of versions
	fmt.Printf("Reading required version from constraint: %s\n", *tgConstraint)
	tgVersion, err := SemVerParser(tgConstraint, tflist)
	return tgVersion, err
}

// ValidateSemVer : Goes through the list of terragrunt version, return a valid tf version for contraint provided
func SemVerParser(tgConstraint *string, tflist []string) (string, error) {
	tgVersion := ""
	constraints, err := semver.NewConstraint(*tgConstraint) //NewConstraint returns a Constraints instance that a Version instance can be checked against
	if err != nil {
		return "", fmt.Errorf("error parsing constraint: %s", err)
	}
	versions := make([]*semver.Version, len(tflist))
	//put tgVersion into semver object
	for i, tfvals := range tflist {
		version, err := semver.NewVersion(tfvals) //NewVersion parses a given version and returns an instance of Version or an error if unable to parse the version.
		if err != nil {
			return "", fmt.Errorf("error parsing constraint: %s", err)
		}
		versions[i] = version
	}

	sort.Sort(sort.Reverse(semver.Collection(versions)))

	for _, element := range versions {
		if constraints.Check(element) { // Validate a version against a constraint
			tgVersion = element.String()
			fmt.Printf("Matched version: %s\n", tgVersion)
			if ValidVersionFormat(tgVersion) { //check if version format is correct
				return tgVersion, nil
			}
		}
	}

	PrintInvalidTGVersion()
	return "", fmt.Errorf("error parsing constraint: %s", *tgConstraint)
}

// Print invalid TF version
func PrintInvalidTGVersion() {
	fmt.Println("Version does not exist or invalid terragrunt version format.\n Format should be #.#.# or #.#.#-@# where # are numbers and @ are word characters.\n For example, 0.11.7 and 0.11.9-beta1 are valid versions")
}

// Print invalid TF version
func PrintInvalidMinorTGVersion() {
	fmt.Println("Invalid minor terragrunt version format. Format should be #.# where # are numbers. For example, 0.11 is valid version")
}
