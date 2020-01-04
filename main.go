package main

import (
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/pkg/mount"
	"github.com/docopt/docopt-go"
	"github.com/mistifyio/go-zfs"
)

var (
	appVersion string
	buildTime  string
)

func createZdevice(zfsDevice string, debuginfo bool) {
	_, err := zfs.GetDataset(zfsDevice)

	if err != nil {
		_, createErr := zfs.CreateFilesystem(zfsDevice, nil)
		if err != nil {
			panic(createErr)
		}
		if debuginfo == true {
			log.Printf("created filesystem %v", zfsDevice)
		}
	} else {
		if debuginfo == true {
			log.Printf("filesystem %v already existing", zfsDevice)
		}
	}

}

func ensureG10kMounted(zfsDevice string, debuginfo bool) {
	if _, err := os.Stat("/g10k"); os.IsNotExist(err) {
		os.Mkdir("/g10k", 0755)
	}

	status, _ := mount.Mounted("/g10k")
	if status == false {
		g10kDataset, _ := zfs.GetDataset(zfsDevice)
		_, err := g10kDataset.Mount(false, nil)
		if err != nil {
			panic(err)
		} else {
			if debuginfo == true {
				log.Printf("mounted filesytem %v", zfsDevice)
			}
		}
	} else {
		if debuginfo == true {
			log.Printf("filesystem %v already mounted", zfsDevice)
		}
	}
}

func checkSnapshots(zfsDevice string, debuginfo bool) [2]bool {
	evenStatus := true
	oddStatus := true
	_, evenErr := zfs.GetDataset(zfsDevice + "@Even")
	if evenErr != nil {
		evenStatus = false
	} else {
		if debuginfo == true {
			log.Printf("found snapshot %v@Even", zfsDevice)
		}
	}
	_, oddErr := zfs.GetDataset(zfsDevice + "@Odd")
	if oddErr != nil {
		oddStatus = false
	} else {
		if debuginfo == true {
			log.Printf("found snapshot %v@Odd", zfsDevice)
		}
	}

	EvenOddStatus := [2]bool{evenStatus, oddStatus}
	return EvenOddStatus
}

func createSnapshot(nextSnap string, zfsDevice string, debuginfo bool) {
	zfsDataset, _ := zfs.GetDataset(zfsDevice)
	_, err := zfsDataset.Snapshot(nextSnap, false)
	if err != nil {
		panic(err)
	} else {
		if debuginfo == true {
			log.Printf("created snapshot filesytem %v@%v", zfsDevice, nextSnap)
		}
	}
}

func destroySnapshot(nextSnap string, zfsDevice string, debuginfo bool) {
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
			if debuginfo == true {
				log.Printf("destroyed snapshot %v@%v", zfsDevice, killSnap)
			}
		}
	}
}

func umountSnapshots(mountPoint string, debuginfo bool) {
	status, _ := mount.Mounted(mountPoint)
	if status == true {
		err := mount.Unmount(mountPoint)
		if err != nil {
			panic(err)
		} else {
			if debuginfo == true {
				log.Printf("unmounted %v", mountPoint)
			}
		}
	}
}

func mountSnapshots(mountPoint string, zfsDevice string, nextSnap string, debuginfo bool) {
	// func Mount(device, target, mType, options string) error
	mountDevice := zfsDevice + "@" + nextSnap
	err := mount.Mount(mountDevice, mountPoint, "zfs", "defaults")
	if err != nil {
		panic(err)
	} else {
		if debuginfo == true {
			log.Printf("mounted %v@%v on %v", zfsDevice, nextSnap, mountPoint)
		}
	}
}

func main() {

	usage := `G10k ZFS:
  - create ZFS snapshots on an existing ZFS pool
  - rotate the snapshots
  - mount the latest snapshot, by default on /etc/puppetlabs/code

Usage:
  g10k-zfs --pool=POOL [--mountpoint=MOUNTPOINT] [--debug]
  g10k-zfs -v | --version
  g10k-zfs -b | --build
  g10k-zfs -h | --help

Options:
  -h --help           Show this screen
  -p --pool=POOL               ZFS Pool
  -m --mountpoint=MOUNTPOINT   Output file [default: '/etc/puppetlabs/code']
  -d --debug                   Print password and full key path
  -v --version                 Print version exit
  -b --build                   Print version and build information and exit`

	arguments, _ := docopt.Parse(usage, nil, true, appVersion, false)

	if arguments["--build"] == true {
		fmt.Printf("g10k-zfs version: %v, built on: %v\n", appVersion, buildTime)
		os.Exit(0)
	}

	debugInformation := false
	if arguments["--debug"] == true {
		debugInformation = true
	}

	mountpoint := fmt.Sprintf("%v", arguments["--mountpoint"])
	zPool := fmt.Sprintf("%v", arguments["--pool"])
	zDevice := zPool + "/g10k"

	createZdevice(zDevice, debugInformation)
	ensureG10kMounted(zDevice, debugInformation)
	SnapshotStatus := checkSnapshots(zDevice, debugInformation)

	nextSnapshot := "Odd"
	if SnapshotStatus[0] == false && SnapshotStatus[1] == false {
		nextSnapshot = "Even"
	} else if SnapshotStatus[0] == true && SnapshotStatus[1] == false {
		nextSnapshot = "Odd"
	} else if SnapshotStatus[0] == false && SnapshotStatus[1] == true {
		nextSnapshot = "Even"
	} else if SnapshotStatus[0] == true && SnapshotStatus[1] == true {
		// this is anomalous: we kill 'em all and start with Odd
		log.Printf("here we are")
		umountSnapshots(mountpoint, debugInformation)
		destroySnapshot("Even", zDevice, debugInformation)
		destroySnapshot("Odd", zDevice, debugInformation)
	}

	createSnapshot(nextSnapshot, zDevice, debugInformation)
	umountSnapshots(mountpoint, debugInformation)
	mountSnapshots(mountpoint, zDevice, nextSnapshot, debugInformation)
	destroySnapshot(nextSnapshot, zDevice, debugInformation)

}
