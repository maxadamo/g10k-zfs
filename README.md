# G10k ZFS

## Description

Add ZFS snapshot capability to [G10K](https://github.com/xorpaul/g10k/).

* creates a new ZFS r/o snapshot whose name contains a timestamp (i.e.: `code/g10k@Date-06-Jan-2020_Time-16.28.42`)
* unmount the currently mounted snapshot
* mount the new snapshot
* deletes the old snapshot

## Requirements

* 1\. ZOL (ZFS On Linux) configured on Linux
* 2\. One ZFS Pool configured (this tool will create a filesystem called `g10k` inside your pool and mount it, by default, on `/g10k`). You can follow one of these steps:
  * 2.1\. read this help page: [setup-zfs-storage-pool](https://tutorials.ubuntu.com/tutorial/setup-zfs-storage-pool)
  * 2.2\. use Puppet to create the pool: [zfs_core](https://forge.puppet.com/puppetlabs/zfs_core).
* 3\.  The default mount point is `/etc/puppetlabs/code`, which means that you need to populate the filesystem with all the files that you had under `/etc/puppetlabs/code`
* 4\. if you intend to use g10k hard-links, you need to add the cache directory inside this same filesystem.
* 5\.  **g10k will be pointing to the mount point /g10k**. This is the read/write mount point. The mount point on `/etc/puppetlabs/code` is a read only mount point for the snapshot (i.e. g10k could not write there)

## Usage

```sh
  g10k-zfs --pool=POOL [--mountpoint=MOUNTPOINT] [--owner=OWNER] [--group=GROUP] [--g10k-mount=G10KMOUNT] [--fix-owner] [--debug]
  g10k-zfs -v | --version
  g10k-zfs -b | --build
  g10k-zfs -h | --help

Options:
  -h --help                   Show this screen
  -p --pool=POOL              ZFS Pool
  -m --mountpoint=MOUNTPOINT  Output file [default: /etc/puppetlabs/code]
  -f --fix-owner              Whether to fix file ownership
  -o --owner=OWNER            Files owner [default: puppet]
  -g --group=GROUP            Files group [default: puppet]
  -k --g10k-mount=G10KMOUNT   G10k mount point [default: /g10k]
  -d --debug                  Print messages to console
  -v --version                Print version exit
  -b --build                  Print version and build information and exit
```

## Examples

Valid examples are:

```sh
g10k-zfs -p code -d
```

for testing purpose you may want to run it outside of your production code directory:

```sh
g10k-zfs -p code -m /my/destination -d
```

## Installation

to compile the latest tag:

```sh
git clone git@github.com:maxadamo/g10k-zfs.git
cd g10k-zfs
./build.sh
```
