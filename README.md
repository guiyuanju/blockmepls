# Block Me PLS

A cross platform utility to block websites to stay focused.
Works on MacOS, Windows and Linux.

**Only tested on MacOS. Windows and Linux not tested, if you found any problem, please open a issue!**

## Installation

```bash
go install github.com/guiyuanju/blockmepls@latest
```

## Usage

```bash
# block
blockmepls -sites="x.com,instagram.com"

# reset
blockmepls -reset
```

```
Usage of blockmepls:
  -reset
        reset the block list
  -sites value
        the sites to be blocked, comma separated
```
