package main

import (
	"fmt"
	"os"


	tmc "github.com/firepear/thrasher-music-catalog"
	tmcu "github.com/firepear/thrasher-music-catalog/updater"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if fversion {
		fmt.Println(ver)
		os.Exit(0)
	}

	var err error
	if fdebug {
		fmt.Println("DEBUG> Config")
		fmt.Printf("\tDbFile: %s\n\tMusicDir: %s\n\tArtistCutoff: %d\n",
			conf.DbFile, conf.MusicDir, conf.ArtistCutoff)
	}

	// create an updater instance
	upd, err := tmcu.New(conf.DbFile)
	if err != nil {
		fmt.Printf("error creating updater: %s", err)
		os.Exit(1)
	}
	defer upd.Close()
	// then handle database creation, if asked
	if fcreate {
		// we've been asked to create the db; do so
		err := upd.CreateDB()
		if err != nil {
			fmt.Printf("couldn't create db: %s\n", err)
			os.Exit(2)
		}
		fmt.Printf("database initialized in %s\n", conf.DbFile)
		os.Exit(0)
	}

	// instantiate a catalog instance
	cat, err = tmc.New(conf, "tmctool")
	cat.TrimPrefix = ftrim
	if err != nil {
		fmt.Printf("error creating catalog: %s", err)
		os.Exit(1)
	}
	defer cat.Close()

	// handle setting filter, if we have a format string
	if ffilter != "" {
		err = cat.Filter(ffilter)
		if err != nil {
			fmt.Printf("error parsing filter: %s\n", err)
			os.Exit(3)
		}
		trks = []string{}
		if fdebug {
			fmt.Printf("DEBUG> filter: '%s', %v, %d\n", cat.FltrStr, cat.FltrVals, cat.FltrCount)
		}
	}

	switch {
	case fscan || fscanall:
		// scan for new tracks
		stat, err := os.Stat(conf.MusicDir)
		if err != nil {
			fmt.Printf("can't access musicdir '%s': %s\n", conf.MusicDir, err)
			os.Exit(3)
		}
		if !stat.IsDir() {
			fmt.Printf("%s is not a directory\n", conf.MusicDir)
			os.Exit(3)
		}

		err = scanmp3s(conf, fscanall)
		if err != nil {
			fmt.Printf("error during scan: %s\n", err)
			os.Exit(3)
		}
	case fqrecent:
		// display tracks on recently added albums
		trks, err = cat.QueryRecent()
		if err != nil {
			fmt.Printf("error getting recent tracks: %s\n", err)
			os.Exit(3)
		}
		for _, trk := range trks {
			fmt.Println(trk)
		}
	case fquery || fqquery:
		// query catalog and produce output
		if trks == nil {
			// unless trks isn't set, which means a filter hasn't been set
			fmt.Println("running a query requires a filter to be set; exiting")
			os.Exit(1)
		}
		trks, err = cat.Query(forder, flimit, foffset)
		if err != nil {
			fmt.Printf("error querying catalog: %s\n", err)
			os.Exit(2)
		}
		if fdebug {
			fmt.Printf("DEBUG> query: '%s', %v\n----\n", cat.QueryStr, cat.QueryVals)
		}
		if fqquery {
			// fetch and print track details
			for _, trk := range trks {
				i := cat.TrkInfo(trk)
				if len(i.Artist) > 30 {
					i.Artist = i.Artist[:29] + "…"
				}
				if len(i.Title) > 50 {
					i.Title = i.Title[:49] + "…"
				}
				if len(i.Album) > 30 {
					i.Album = i.Album[:29] + "…"
				}
				fmt.Printf("%3d | %-30s | %-50s | %-30s | %d |\t%s\n",
					i.Num, i.Artist, i.Title, i.Album, i.Year, i.Facets)
			}
		} else {
			// just print the track paths
			for _, trk := range trks {
				fmt.Println(trk)
			}
		}
	case ffadd != "":
	case ffls:
		if trks == nil {
			// unless trks isn't set, which means a filter hasn't been set
			fmt.Println("listing facets requires a filter to be set; exiting")
			os.Exit(1)
		}
	case ffrm != "":
	default:
		fmt.Println("No op requested")
	}
}
