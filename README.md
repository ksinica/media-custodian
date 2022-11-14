# Media Custodian

![Mr. Bean examining the painting](images/mrbean.webp)

A simple tool that I’m using to organize photos and videos on a home NAS server. It scans the source directory for JPEG, DNG, and MP4 files and tries to extract file creation time from the metadata. If it’s possible, the new file is moved into the destination directory, into the ```Pictures``` or ```Videos``` subdirectory, and then organized into the ```Year-Month``` subdirectory. The file is renamed, so it contains a creation timestamp (extracted from metadata) and BLAKE3 hash of the file (in order to detect duplicates).
If the metadata cannot be retrieved, or the file is considered a duplicate, it’s not moved.
The permissions and stats are preserved, and subdirectories are created as needed, with `rwx-r-x-r-x` permissions.

Usage:
```
$ media-custodian /mnt/tank/dcim /mnt/tank/media/
Moved 20181224-163405-1545669245000.jpeg to /mnt/tank/media/Pictures/2018-12/20181224-173405-ca701c93f9180d4761cd223d7170790f584a25f771847ce2f21c96c35dd5b1cc.jpeg
Moved IMG_20220304_083818.dng to /mnt/tank/media/Pictures/2022-03/20220304-083818-81f47d52bcf3c2df095e49c137ec97a18a90af80a33d7b6e600e9198b391b295.dng
Moved VID_20220918_125410.mp4 to /mnt/tank/media/Videos/2022-09/20220918-125431-2f57c19c53d34d85b8da2f05d85a7c351a217da00713203d558842a697d5dd91.mp4
```

**USE AT YOUR OWN RISK!**

I personally use this tool as a Cron job that moves files from the ```DCIM``` Syncthing-synced folder to a family photo collection folder that is shared via Samba.

## License

Source code is available under the MIT [License](/LICENSE).

The "Mr. Bean examining the painting" photo is taken from [Mr. Bean Wiki](https://mrbean.fandom.com/wiki/Whistler%27s_Mother), which shares content under a [CC-BY-SA](https://www.fandom.com/licensing) license.