package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"

	"github.com/docker/docker/pkg/mount"
	"github.com/docopt/docopt-go"
	"github.com/mistifyio/go-zfs"
)

var (
	appVersion string
	buildTime  string
	//DebugInformation is used to print debug messages
	DebugInformation bool
)

func printString(logMessage string) {
	if DebugInformation == true {
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

func checkSnapshots(zfsDevice string) [2]bool {
	evenStatus := true
	oddStatus := true
	_, evenErr := zfs.GetDataset(zfsDevice + "@Even")
	if evenErr != nil {
		evenStatus = false
	} else {
		printString("found snapshot " + zfsDevice + "@Even")
	}
	_, oddErr := zfs.GetDataset(zfsDevice + "@Odd")
	if oddErr != nil {
		oddStatus = false
	} else {
		printString("found snapshot " + zfsDevice + "@Odd")
	}

	EvenOddStatus := [2]bool{evenStatus, oddStatus}
	return EvenOddStatus
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

func destroySnapshot(nextSnap, zfsDevice string) {
	killSnap := "Even"
	if nextSnap == "Even" {
		killSnap = "Odd"
	}
	zfsDataset, err := zfs.GetDataset(zfsDevice + "@" + killSnap)
	if err == nil {
		err := zfsDataset.Destroy(0)
		if err != nil {
			panic(err)
		} else {
			printString("destroyed snapshot " + zfsDevice + "@" + killSnap)
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
  - create ZFS snapshots on an existing ZFS pool
  - rotate the snapshots
  - mount the latest snapshot, by default on /etc/puppetlabs/code

Usage:
  g10k-zfs --pool=POOL [--mountpoint=MOUNTPOINT] [--owner=OWNER] [--group=GROUP] [--g10k-mount=G10KMOUNT] [--fix-owner] [--debug]
  g10k-zfs -v | --version
  g10k-zfs -b | --build
  g10k-zfs -h | --help

Options:
  -h --help           Show this screen
  -p --pool=POOL              ZFS Pool
  -m --mountpoint=MOUNTPOINT  Output file [default: /etc/puppetlabs/code]
  -f --fix-owner              Whether to fix file ownership
  -o --owner=OWNER            Files owner [default: puppet]
  -g --group=GROUP            Files group [default: puppet]
  -k --g10k-mount=G10KMOUNT   G10k mount point [default: /g10k]
  -d --debug                  Print password and full key path
  -v --version                Print version exit
  -b --build                  Print version and build information and exit`

	arguments, _ := docopt.Parse(usage, nil, true, appVersion, false)

	if arguments["--build"] == true {
		fmt.Printf("g10k-zfs version: %v, built on: %v\n", appVersion, buildTime)
		os.Exit(0)
	}

	DebugInformation = false
	if arguments["--debug"] == true {
		DebugInformation = true
	}

	mountpoint := fmt.Sprintf("%v", arguments["--mountpoint"])
	g10kMountpoint := fmt.Sprintf("%v", arguments["--g10k-mount"])
	zPool := fmt.Sprintf("%v", arguments["--pool"])
	zDevice := zPool + g10kMountpoint
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
	SnapshotStatus := checkSnapshots(zDevice)

	nextSnapshot := "Odd"
	if SnapshotStatus[0] == false && SnapshotStatus[1] == false {
		nextSnapshot = "Even"
	} else if SnapshotStatus[0] == false && SnapshotStatus[1] == true {
		nextSnapshot = "Even"
	} else if SnapshotStatus[0] == true && SnapshotStatus[1] == true {
		// this is an odd situation: we kill 'em all and start with Odd
		umountSnapshot(mountpoint)
		destroySnapshot("Even", zDevice)
		destroySnapshot("Odd", zDevice)
	}

	createSnapshot(nextSnapshot, zDevice)
	umountSnapshot(mountpoint)
	mountSnapshot(mountpoint, zDevice, nextSnapshot)
	destroySnapshot(nextSnapshot, zDevice)

}
