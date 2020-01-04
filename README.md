# G10k ZFS

## Description

Add ZFS snapshot capability to G10K.

* creates ZFS snapshots on an existing ZFS pool
* rotates in a loop two snapshots, called `Even` and `Odd`
* mounts either `Even` or `Odd` depending on the rotation position
  
## Requirements

1. ZOL (ZFS On Linux) configured on Linux
1. One ZFS Pool configured (this tool will create a filesystem called `g10k` inside your pool and mount it on `/g10k`). You can read the instructions on this page: [setup-zfs-storage-pool](https://tutorials.ubuntu.com/tutorial/setup-zfs-storage-pool).
1. The default mount point is `/etc/puppetlabs/code`, which means that you need to populate the filesystem with all the files that you had under `/etc/puppetlabs/code`
1. **g10k will be pointing to the mount point /g10k**. This is the read/write mount point. The mount point on `/etc/puppetlabs/code` is a read only mount point for the snapshot (i.e. g10k could not write there)

## Usage

```sh
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
  -b --build                   Print version and build information and exit
```

## Examples

These are valid examples:

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
./build.sh
```
