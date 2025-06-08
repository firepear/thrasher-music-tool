package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"strconv"
	"syscall"

	"github.com/bogem/id3v2/v2"

	tmc "github.com/firepear/thrasher-music-catalog"
	_ "github.com/mattn/go-sqlite3"
)

// scanmp3s is the function which walks conf.MusicDir and imports
// tracks to the database.
//
// It does not use the updater instance, but opens its own connection
// and uses it directly, in order to run without the synchronous
// pragma. It does use the catalog instance to see if tracks are
// already in it.
func scanmp3s(conf *tmc.Config, scanall bool) error {
	f, err := os.OpenFile("scanlog", os.O_RDWR | os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening scanlog: %s", err)
	}
	defer f.Close()
	log.SetOutput(f)

	db, err := sql.Open("sqlite3", conf.DbFile)
	if err != nil {
		return err
	}
	defer db.Close()
	db.Exec("PRAGMA synchronous=0")

	var seen = 0
	var added = 0
	var updated = 0
	var clean = false
	var genart = true
	var genre = ""
	var ctime int64
	var mtime int64

	stmt, _ := db.Prepare("INSERT INTO tracks VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")

	// add new tracks
	err = filepath.WalkDir(conf.MusicDir, func(path string, info fs.DirEntry, err error) error {
		// if looking at a dir check mtime and mark clean
		// unless it's newer than lastscan. also, check for
		// cover art
		if info.IsDir() {
			if scanall {
				clean = false
			} else {
				stat, _ := info.Info()
				if stat.ModTime().Unix() <= int64(cat.Lastscan) {
					clean = true
				} else {
					clean = false
				}
			}

			_, err := os.Stat(path + "/cover.jpg")
			if err != nil {
				genart = true
			} else {
				genart = false
			}
			return nil
		}

		if strings.HasSuffix(info.Name(), ".mp3") {
			seen++
			// do nothing if our parent dir is clean
			if clean {
				return nil
			}

			// see if track is already in DB. return
			// unless we're in force scan mode
			inDB := cat.TrkExists(path)
			if inDB && !scanall {
				// for now ignore it. maybe in the
				// future do some kind of update? but
				// also maybe we handle that in-DB
				return nil
			}

			// set create and modified time; ensure that
			// ctime is set to the lowest value for
			// ingestion purposes
			stat, _ := info.Info()
			ctime = int64(stat.Sys().(*syscall.Stat_t).Ctim.Sec)
			mtime = stat.ModTime().Unix()
			if ctime > mtime {
				ctime = mtime
			}

			// get tag data
			tag, err := readTag(path)
			if err != nil {
				log.Printf("tag error %s: %s", path, err)
				return err
			}

			// generate a cover art file if needed
			if genart {
				err = genCoverFile(path, tag)
				if err != nil {
					log.Println(err)
				}
				// success or failure, mark the
				// directory as checked to reduce log
				// spam. i found no instances where
				// the first file didn't have APIC
				// data and subsequent ones did
				genart = false
			}

			// munge genre, if it's numeric
			genid := string(genreg.Find([]byte(tag.Genre())))
			if len(genid) == 0 {
				genre = tag.Genre()
			} else {
				gi, _ := strconv.Atoi(genid)
				genre = genres[gi]
			}

			// get track number
			tnum := tag.GetTextFrame("TRCK").Text
			tnum = strings.Split(tnum, "/")[0]
			if tnum == "" {
				// no empty track numbers; they create
				// spurious errs later on
				tnum = "99"
			}

			// fixup year
			year := tag.Year()
			if year == "" {
				// no blank years
				year = "9999"
			}
			ychunks := strings.Split(year, "-")
			if len(ychunks) == 3 {
				// no ISO formatted datestamps
				year = ychunks[0]
			}

			// log if artist or title or album is blank
			if tag.Artist() == "" || tag.Album() == "" || tag.Title() == "" {
				log.Printf("%s :: missing tags: t '%s', a '%s', b '%s'\n",
					path, tag.Title(), tag.Artist(), tag.Album())
			}

			// only add tracks if they aren't in the
			// DB. in the future update logic may go here
			if !inDB {
				fmt.Printf("+ %s '%s' '%s' (%s; %s; %s)\n",
					strings.TrimSpace(tag.Artist()), strings.TrimSpace(tag.Album()),
					strings.TrimSpace(tag.Title()), tnum, year, genre)
				_, err = stmt.Exec(path, ctime, mtime,
					tnum, strings.TrimSpace(tag.Artist()), strings.TrimSpace(tag.Title()),
					strings.TrimSpace(tag.Album()), year, fmt.Sprintf(`["%s"]`, genre))
				if err != nil {
					return err
				}
				added++
			}
		}
		return err
	})
	if err != nil {
		return err
	}

	fmt.Printf("Totals: seen %d, added, %d, updated %d\n", seen, added, updated)
	_, err = db.Exec("UPDATE meta SET lastscan=?", mtime)
	return err
}


// genCoverFile attemps to extract a `cover.jpg` image from mp3 APIC
// frames
func genCoverFile(path string, tag *id3v2.Tag) error {
	pathchunks := strings.Split(path, "/")
	cvrpath := strings.Join(pathchunks[:len(pathchunks)-1], "/")
	cvrjpg := cvrpath + "/cover.jpg"
	pictures := tag.GetFrames(tag.CommonID("Attached picture"))
	if len(pictures) == 0 {
		return fmt.Errorf("%s :: no APIC tags", cvrpath)
	}
	pic, ok := pictures[0].(id3v2.PictureFrame)
	fmt.Println(ok)
	if !ok {
		return fmt.Errorf("%s :: Bad APIC data", path)
	}
	err := os.WriteFile(cvrjpg, pic.Picture, 0644)
	if err != nil {
		return err
	}
	return nil
}


// readTag takes a file path and returns the ID3 tags contained in
// that file
func readTag(path string) (*id3v2.Tag, error) {
	tag, err := id3v2.Open(path, id3opts)
	if err != nil {
		return nil, fmt.Errorf("'%s': %s", path, err)
	}
	tag.Close()
	return tag, err
}
