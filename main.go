package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/pkg/mount"
	"github.com/docopt/docopt-go"
	"github.com/mistifyio/go-zfs"
)

var (
	appVersion       string
	buildTime        string
	debugInformation bool
)

func printString(logMessage string) {
	if debugInformation == true {
		log.Printf(logMessage)
	}
}

func createZdevice(zfsDevice string) {
	_, err := zfs.GetDataset(zfsDevice)
	if err != nil {
		_, createErr := zfs.CreateFilesystem(zfsDevice, nil)
		if err != nil {
			panic(createErr)
		}
		printString("created filesystem " + zfsDevice)
	} else {
		printString("filesystem " + zfsDevice + " already existing")
	}
}

func ensureG10kMounted(zfsDevice, gkMountpoint string) {
	if _, err := os.Stat(gkMountpoint); os.IsNotExist(err) {
		os.Mkdir(gkMountpoint, 0755)
	}

	status, _ := mount.Mounted(gkMountpoint)
	if status == false {
		g10kDataset, _ := zfs.GetDataset(zfsDevice)
		_, err := g10kDataset.Mount(false, nil)
		if err != nil {
			panic(err)
		} else {
			printString("mounted filesytem " + zfsDevice)
		}
	} else {
		printString("filesytem " + zfsDevice + " already mounted")
	}
}

func createSnapshot(nextSnap, zfsDevice string) {
	zfsDataset, _ := zfs.GetDataset(zfsDevice)
	_, err := zfsDataset.Snapshot(nextSnap, false)
	if err != nil {
		panic(err)
	} else {
		printString("created snapshot " + zfsDevice + "@" + nextSnap)
	}
}

func destroySnapshots(snapList []*zfs.Dataset, zfsPool string) {
	mountedLine := "unmounted"
	for _, eachDataset := range snapList {
		zfsDevName := fmt.Sprintf("%v", eachDataset.Name)
		match, _ := regexp.MatchString("^"+zfsPool+"/g10k@+", zfsDevName)
		if match == true {
			f, err := os.Open("/proc/self/mountinfo")
			if err != nil {
				panic(err)
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), " "+zfsDevName+" ") {
					mountedLine = scanner.Text()
				}
			}
			if mountedLine != "unmounted" {
				snapMount := strings.Split(mountedLine, " ")[4]
				umountSnapshot(snapMount)
				mountedLine = "unmounted"
			}
			zfsDataset, err := zfs.GetDataset(zfsDevName)
			if err == nil {
				err := zfsDataset.Destroy(16)
				if err != nil {
					panic(err)
				} else {
					printString("destroyed snapshot " + zfsDevName)
				}
			}
		}
	}
}

func umountSnapshot(mountPoint string) {
	status, _ := mount.Mounted(mountPoint)
	if status == true {
		err := mount.Unmount(mountPoint)
		if err != nil {
			panic(err)
		} else {
			printString("unmounted " + mountPoint)
		}
	}
}

func mountSnapshot(mountPoint, zfsDevice, nextSnap string) {
	mountDevice := zfsDevice + "@" + nextSnap
	err := mount.Mount(mountDevice, mountPoint, "zfs", "defaults")
	if err != nil {
		panic(err)
	} else {
		printString("mounted " + zfsDevice + "@" + nextSnap + " on " + mountPoint)
	}
}

func checkUserGroupExistence(userName, groupName string) {
	_, userErr := user.Lookup(userName)
	if userErr != nil {
		printString("the user " + userName + " does not exist")
		panic(userErr)
	}
	_, groupErr := user.LookupGroup(groupName)
	if groupErr != nil {
		printString("the group " + groupName + " does not exist")
		panic(groupErr)
	}
}

func chownR(path, userName, groupName string) {
	printString("fixing files ownership under " + path)
	uidgid := fmt.Sprintf("%v:%v", userName, groupName)
	cmd := exec.Command("chown", "-R", uidgid, path)
	cmd.Run()
}

func main() {

	usage := `G10k ZFS:
  - creates a ZFS r/o snapshot whose name contains a timestamp
  - unmount the currently mounted snapshot
  - mount the new snapshot
  - deletes the old snapshot

Usage:
  g10k-zfs --pool=POOL [--mountpoint=MOUNTPOINT] [--owner=OWNER] [--group=GROUP] [--g10k-mount=G10KMOUNT] [--fix-owner] [--debug]
  g10k-zfs -v | --version
  g10k-zfs -b | --build
  g10k-zfs -h | --help

Options:
  -h --help                   Show this screen
  -p --pool=POOL              ZFS Pool
  -m --mountpoint=MOUNTPOINT  Puppet code mount point [default: /etc/puppetlabs/code]
  -f --fix-owner              Whether to fix file ownership
  -o --owner=OWNER            Files owner [default: puppet]
  -g --group=GROUP            Files group [default: puppet]
  -k --g10k-mount=G10KMOUNT   G10k mount point [default: /g10k]
  -d --debug                  Print messages to the console
  -v --version                Print version exit
  -b --build                  Print version and build information and exit`

	arguments, _ := docopt.Parse(usage, nil, true, appVersion, false)

	if arguments["--build"] == true {
		fmt.Printf("g10k-zfs version: %v, built on: %v\n", appVersion, buildTime)
		os.Exit(0)
	}

	debugInformation = false
	if arguments["--debug"] == true {
		debugInformation = true
	}

	mountpoint := fmt.Sprintf("%v", arguments["--mountpoint"])
	g10kMountpoint := fmt.Sprintf("%v", arguments["--g10k-mount"])
	zPool := fmt.Sprintf("%v", arguments["--pool"])
	zDevice := zPool + g10kMountpoint

	currentTime := time.Now()
	nextSnapshot := fmt.Sprintf(currentTime.Format("Date-02-Jan-2006_Time-15.4.5"))
	snapshotList, _ := zfs.Snapshots("")

	username := fmt.Sprintf("%v", arguments["--owner"])
	groupname := fmt.Sprintf("%v", arguments["--group"])
	if arguments["--fix-owner"] != true {
		if username != "puppet" || groupname != "puppet" {
			fmt.Printf("you have set either owner or group without using --fix-owner\n")
			os.Exit(1)
		}
	}
	if arguments["--fix-owner"] == true {
		checkUserGroupExistence(username, groupname)
	}
	createZdevice(zDevice)
	ensureG10kMounted(zDevice, g10kMountpoint)
	if arguments["--fix-owner"] == true {
		chownR(g10kMountpoint, username, groupname)
	}
	createSnapshot(nextSnapshot, zDevice)
	umountSnapshot(mountpoint)
	mountSnapshot(mountpoint, zDevice, nextSnapshot)
	destroySnapshots(snapshotList, zPool)

}
