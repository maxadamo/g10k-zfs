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
	logMsg     string
	//DebugInformation use this to print debug
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
		logMsg = "created filesystem " + zfsDevice
	} else {
		logMsg = "filesystem " + zfsDevice + "already existing"
	}
	printString(logMsg)
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
			logMsg = "mounted filesytem " + zfsDevice
		}
	} else {
		logMsg = "filesytem " + zfsDevice + "already mounted"
	}
	printString(logMsg)
}

func checkSnapshots(zfsDevice string) [2]bool {
	evenStatus := true
	oddStatus := true
	_, evenErr := zfs.GetDataset(zfsDevice + "@Even")
	if evenErr != nil {
		evenStatus = false
	} else {
		logMsg = "found snapshot " + zfsDevice + "@Even"
	}
	_, oddErr := zfs.GetDataset(zfsDevice + "@Odd")
	if oddErr != nil {
		oddStatus = false
	} else {
		logMsg = "found snapshot " + zfsDevice + "@Odd"
	}
	printString(logMsg)

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

func umountSnapshots(mountPoint string) {
	status, _ := mount.Mounted(mountPoint)
	if status == true {
		err := mount.Unmount(mountPoint)
		if err != nil {
			panic(err)
		} else {
			printString("destroyed snapshot " + mountPoint)
		}
	}
}

func mountSnapshots(mountPoint, zfsDevice, nextSnap string) {
	mountDevice := zfsDevice + "@" + nextSnap
	err := mount.Mount(mountDevice, mountPoint, "zfs", "defaults")
	if err != nil {
		panic(err)
	} else {
		printString("mounted " + zfsDevice + "@" + nextSnap)
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
		panic(userErr)
	}
}

// ChownR walks inside the directory and assign every file to given user and group
func ChownR(path, userName, groupName string) {
	printString("fixing file permissions under " + path)
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
  g10k-zfs --pool=POOL [--mountpoint=MOUNTPOINT] [--owner=OWNER] [--group=GROUP] [--debug]
  g10k-zfs -v | --version
  g10k-zfs -b | --build
  g10k-zfs -h | --help

Options:
  -h --help           Show this screen
  -p --pool=POOL               ZFS Pool
  -m --mountpoint=MOUNTPOINT   Output file [default: /etc/puppetlabs/code]
  -o --owner=OWNER             Files owner [default: puppet]
  -g --group=GROUP             Files group [default: puppet]
  -gk --g10k=G10K              G10k mount point [default: /g10k]
  -d --debug                   Print password and full key path
  -v --version                 Print version exit
  -b --build                   Print version and build information and exit`

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
	g10kMountpoint := fmt.Sprintf("%v", arguments["--g10k"])
	zPool := fmt.Sprintf("%v", arguments["--pool"])
	zDevice := zPool + g10kMountpoint
	username := fmt.Sprintf("%v", arguments["--owner"])
	groupname := fmt.Sprintf("%v", arguments["--group"])

	checkUserGroupExistence(username, groupname)
	createZdevice(zDevice)
	ensureG10kMounted(zDevice, g10kMountpoint)
	ChownR(g10kMountpoint, username, groupname)
	SnapshotStatus := checkSnapshots(zDevice)

	nextSnapshot := "Odd"
	if SnapshotStatus[0] == false && SnapshotStatus[1] == false {
		nextSnapshot = "Even"
	} else if SnapshotStatus[0] == true && SnapshotStatus[1] == false {
		nextSnapshot = "Odd"
	} else if SnapshotStatus[0] == false && SnapshotStatus[1] == true {
		nextSnapshot = "Even"
	} else if SnapshotStatus[0] == true && SnapshotStatus[1] == true {
		// this is anomalous: we kill 'em all and start with Odd
		umountSnapshots(mountpoint)
		destroySnapshot("Even", zDevice)
		destroySnapshot("Odd", zDevice)
	}

	createSnapshot(nextSnapshot, zDevice)
	umountSnapshots(mountpoint)
	mountSnapshots(mountpoint, zDevice, nextSnapshot)
	destroySnapshot(nextSnapshot, zDevice)

}
