/*
Lalamove intern challege.
Jimu Ryoo, 2019/03/15

PROGRAM DESCRIPTION:
Program gives highest patch version of every release between a minimum version and the highest released version.
It reads the Github Releases list,
uses SemVer for comparison and takes a path to a txt file as its first argument when executed.

Gitbuh has request limit per hour.
Authentication extends that limit to 5000 requests/hour
This program does not implement features for user to be able to Authenticate.

INPUT FILE:
only reads flie format specified in readme.
if there is more than one -------/------:#.#.# in one line, only first one will be read

OUTPUTS:
Outputs to stdout with format below. (if there is no matching version, program returns empty list)
If there is an error, it will display why

If specified file does not exist or is wrong format, or file is not at all specified
program terminates with error message.

GENERAL FORMAT:
latest versions of kubernetes/kubernetes: [1.13.4 1.12.6 1.11.8 1.10.13 1.9.11 1.8.15]
latest versions of prometheus/prometheus: [2.7.2 2.6.1 2.5.0 2.4.3 2.3.2 2.2.1]
Incorrect format : etc....
*/
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"

	"github.com/coreos/go-semver/semver"
	"github.com/google/go-github/github"
)

/*LatestVersions returns a sorted slice with the highest version as its first element
and the highest version of the smaller minor versions in a descending order
*/
func LatestVersions(releases []*semver.Version, minVersion *semver.Version) []*semver.Version {
	var versionSlice []*semver.Version

	sort.Sort(Versions1(releases))
	for _, elem := range releases {
		//if elem is bigger than minversion and versionSlice is empty just puts element in
		if elem.Compare(*minVersion) >= 0 && versionSlice == nil {
			versionSlice = append(versionSlice, elem)
		} else {
			//swaps last element in versionSlice with elem if
			//they are of same Major and Minor - First if condition
			//Second if statement compares two versions down to pre-releases
			if elem.Major == versionSlice[len(versionSlice)-1].Major && elem.Minor == versionSlice[len(versionSlice)-1].Minor {
				if elem.Compare(*versionSlice[len(versionSlice)-1]) >= 0 {
					versionSlice[len(versionSlice)-1] = elem
				}
			} else {
				//if last element in versionSlice and elem are different Major and minor, just appends elem
				versionSlice = append(versionSlice, elem)
			}
		}

	}
	return versionSlice
}

/* Versions1
This type, and 3 functions down below
 was necessary to sort releases slice above in decending order.
One included in semver sorts in increasing order
*/
type Versions1 []*semver.Version

func (s Versions1) Len() int {
	return len(s)
}

func (s Versions1) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Versions1) Less(i, j int) bool {
	//this part is changed from one included in semver: i,j place switched
	return s[j].LessThan(*s[i])
}

/*FetchGitVer implements the basics of communicating with github through
the library as well as printing the version
It will read first 20 releases only.
Github only allows 60 requests/hr without authentication.
Quits program when hitting Github rate limit.
*/
func FetchGitVer(apiName1 string, apiName2 string, minVersion *semver.Version) {

	client := github.NewClient(nil)
	ctx := context.Background()
	opt := &github.ListOptions{PerPage: 20}

	releases, _, err := client.Repositories.ListReleases(ctx, apiName1, apiName2, opt)

	//Checks if github rate limit is met. If it is program quits.

	//When such api does not exist in github.
	if err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			log.Fatal("\nGithub Rate limit was met. Please try again in an hour:(")

		} else {
			fmt.Printf("Sorry! Could not find: %s\\%s in Github\n", apiName1, apiName2)
		}

	} else {
		allReleases := make([]*semver.Version, len(releases))
		for i, release := range releases {
			versionString := *release.TagName
			if versionString[0] == 'v' {
				versionString = versionString[1:]
			}
			allReleases[i] = semver.New(versionString)
		}
		versionSlice := LatestVersions(allReleases, minVersion)
		fmt.Printf("latest versions of %s\\%s: %s\n", apiName1, apiName2, versionSlice)
	}
}

//ProcessString - function to Process the line.
//If all conditions are met, calls FetchGitVer to find release list.
func ProcessString(input string) {
	//regex to read line
	r, _ := regexp.Compile(`([a-zA-z]+)/([a-zA-z]+),(\S*)`)
	line := r.FindStringSubmatch(input)

	/*This is when format of line is wrong, except when version is wronly formated
	ex: asdfasdf asdf asdf
	asdf/asdf 1.2.4   will trigger this but not
	asdf/asdf,1.2.4.5.4.2.3.1.2.
	*/
	if len(line) == 0 {
		fmt.Printf("Wrong formated line. Could not process: %s\n", input)
	} else {
		//version extracted from input. semver automatically makes corresponding error message
		version, err := semver.NewVersion(line[3])

		//When version is not semver correct
		if err != nil {
			fmt.Println("Incorrect Version:", err)
		} else {
			//if there is no error calls FetchGitVer to actually do git stuff
			//and use that as an argument of LatestVersions
			FetchGitVer(line[1], line[2], version)
		}
	}
}

//main func.
//scans file and calls FetchGitVer() on each line read while handling some errors.
func main() {

	//first check if program was run with correct # of argument = 1
	if len(os.Args) != 2 {
		log.Fatal("\nInvalid Argument: Please provide one and only one file path for argument")
	}

	//open file
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal("\nInvalid Argument: Please make sure file name/directory is correct")
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	//checks if first line of the reading file is `repository,min_version`
	//if not quits the program.
	scanner.Scan()
	if scanner.Text() != `repository,min_version` {
		log.Fatal("\nInvalid Argument: Please make sure file format is correct\nFirst line of the file should be `repository,min_version`")
	}

	//Then scan file and call ProcessString() on each line
	for scanner.Scan() {
		ProcessString(scanner.Text())
	}

	//for when there is an error while doing scanning and processing for some reason, although unlikely
	if err != nil {
		log.Fatal("\nFile Error: There was error reading file. Please check your file and restart")
	}
}
